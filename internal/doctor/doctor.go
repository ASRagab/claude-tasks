package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ASRagab/claude-tasks/internal/db"
	"github.com/ASRagab/claude-tasks/internal/usage"
)

type Status string

const (
	StatusPass Status = "PASS"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
)

type CheckResult struct {
	Name   string
	Status Status
	Detail string
	Hint   string
}

type Report struct {
	DataDir          string
	DBPath           string
	Results          []CheckResult
	CriticalFailures int
}

func (r Report) ExitCode() int {
	if r.CriticalFailures > 0 {
		return 1
	}
	return 0
}

type Runner struct {
	DataDir string
}

func NewRunner(dataDir string) Runner {
	return Runner{DataDir: dataDir}
}

func (r Runner) Run() Report {
	report := Report{
		DataDir: r.DataDir,
		DBPath:  filepath.Join(r.DataDir, "tasks.db"),
	}

	report.add(r.checkClaudeBinary())
	report.add(r.checkUsageCredentials())
	report.add(r.checkDataDirWritable())
	report.add(r.checkLogsDirWritable())
	database, dbResult := r.checkDBWritable(report.DBPath)
	report.add(dbResult)

	if database != nil {
		defer database.Close()
		report.add(checkSchedulerLeaseVisibility(database))
	}

	return report
}

func (r *Report) add(result CheckResult) {
	r.Results = append(r.Results, result)
	if result.Status == StatusFail {
		r.CriticalFailures++
	}
}

func (r Runner) checkClaudeBinary() CheckResult {
	path, err := exec.LookPath("claude")
	if err != nil {
		return CheckResult{
			Name:   "claude_binary",
			Status: StatusFail,
			Detail: "`claude` executable not found in PATH",
			Hint:   "Install Claude CLI or prepend its bin directory to PATH",
		}
	}
	return CheckResult{Name: "claude_binary", Status: StatusPass, Detail: fmt.Sprintf("found at %s", path)}
}

func (r Runner) checkUsageCredentials() CheckResult {
	if isTruthy(os.Getenv("CLAUDE_TASKS_DISABLE_USAGE_CHECK")) {
		return CheckResult{
			Name:   "usage_credentials",
			Status: StatusPass,
			Detail: "usage check disabled via CLAUDE_TASKS_DISABLE_USAGE_CHECK",
		}
	}

	if _, err := usage.NewClient(); err != nil {
		return CheckResult{
			Name:   "usage_credentials",
			Status: StatusFail,
			Detail: fmt.Sprintf("usage credentials unavailable: %v", err),
			Hint:   "Login Claude CLI or set CLAUDE_TASKS_DISABLE_USAGE_CHECK=1",
		}
	}

	return CheckResult{Name: "usage_credentials", Status: StatusPass, Detail: "credentials available"}
}

func (r Runner) checkDataDirWritable() CheckResult {
	if err := os.MkdirAll(r.DataDir, 0o755); err != nil {
		return CheckResult{
			Name:   "data_dir",
			Status: StatusFail,
			Detail: fmt.Sprintf("cannot create data dir %s: %v", r.DataDir, err),
			Hint:   "Fix CLAUDE_TASKS_DATA permissions",
		}
	}

	probePath := filepath.Join(r.DataDir, ".doctor-write-probe")
	if err := os.WriteFile(probePath, []byte("ok"), 0o644); err != nil {
		return CheckResult{
			Name:   "data_dir",
			Status: StatusFail,
			Detail: fmt.Sprintf("cannot write data dir %s: %v", r.DataDir, err),
			Hint:   "Fix CLAUDE_TASKS_DATA permissions",
		}
	}
	_ = os.Remove(probePath)

	return CheckResult{Name: "data_dir", Status: StatusPass, Detail: "writable"}
}

func (r Runner) checkLogsDirWritable() CheckResult {
	logsDir := filepath.Join(r.DataDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return CheckResult{
			Name:   "logs_dir",
			Status: StatusFail,
			Detail: fmt.Sprintf("cannot create logs dir %s: %v", logsDir, err),
			Hint:   "Fix logs directory permissions",
		}
	}

	probePath := filepath.Join(logsDir, ".doctor-write-probe")
	if err := os.WriteFile(probePath, []byte("ok"), 0o644); err != nil {
		return CheckResult{
			Name:   "logs_dir",
			Status: StatusFail,
			Detail: fmt.Sprintf("cannot write logs dir %s: %v", logsDir, err),
			Hint:   "Fix logs directory permissions",
		}
	}
	_ = os.Remove(probePath)

	return CheckResult{Name: "logs_dir", Status: StatusPass, Detail: "writable"}
}

func (r Runner) checkDBWritable(dbPath string) (*db.DB, CheckResult) {
	database, err := db.New(dbPath)
	if err != nil {
		return nil, CheckResult{
			Name:   "database",
			Status: StatusFail,
			Detail: fmt.Sprintf("cannot open DB %s: %v", dbPath, err),
			Hint:   "Verify database path and filesystem permissions",
		}
	}

	holderID := fmt.Sprintf("doctor-%d", time.Now().UnixNano())
	if _, _, err := database.TryAcquireSchedulerLease(holderID, 50*time.Millisecond); err != nil {
		database.Close()
		return nil, CheckResult{
			Name:   "database",
			Status: StatusFail,
			Detail: fmt.Sprintf("database write check failed: %v", err),
			Hint:   "Verify SQLite file permissions and locks",
		}
	}
	_ = database.ReleaseSchedulerLease(holderID)

	return database, CheckResult{Name: "database", Status: StatusPass, Detail: "open and writable"}
}

func checkSchedulerLeaseVisibility(database *db.DB) CheckResult {
	lease, err := database.GetSchedulerLease()
	if err != nil {
		return CheckResult{
			Name:   "scheduler_lease",
			Status: StatusWarn,
			Detail: fmt.Sprintf("unable to read lease: %v", err),
		}
	}
	if lease == nil {
		return CheckResult{Name: "scheduler_lease", Status: StatusWarn, Detail: "no lease holder recorded"}
	}

	now := time.Now()
	state := "expired"
	if lease.LeaseExpiresAt.After(now) {
		state = "active"
	}
	return CheckResult{
		Name:   "scheduler_lease",
		Status: StatusPass,
		Detail: fmt.Sprintf("holder=%s lease_expires_at=%s (%s)", lease.HolderID, lease.LeaseExpiresAt.Format(time.RFC3339), state),
	}
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
