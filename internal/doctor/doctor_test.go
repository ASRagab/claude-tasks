package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsTruthy(t *testing.T) {
	truthy := []string{"1", "true", "TRUE", " yes ", "on"}
	for _, v := range truthy {
		if !isTruthy(v) {
			t.Fatalf("expected %q to be truthy", v)
		}
	}

	falsy := []string{"", "0", "false", "no", "off", "random"}
	for _, v := range falsy {
		if isTruthy(v) {
			t.Fatalf("expected %q to be falsy", v)
		}
	}
}

func TestRunnerPassesWithUsageDisabledAndFakeClaude(t *testing.T) {
	dataDir := t.TempDir()
	binDir := t.TempDir()
	claudePath := filepath.Join(binDir, "claude")
	if err := os.WriteFile(claudePath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+oldPath)
	t.Setenv("CLAUDE_TASKS_DISABLE_USAGE_CHECK", "1")

	report := NewRunner(dataDir).Run()
	if report.ExitCode() != 0 {
		t.Fatalf("expected healthy report, got failures=%d results=%+v", report.CriticalFailures, report.Results)
	}
}

func TestRunnerFailsWhenClaudeMissing(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CLAUDE_TASKS_DISABLE_USAGE_CHECK", "1")

	report := NewRunner(dataDir).Run()
	if report.ExitCode() == 0 {
		t.Fatalf("expected failure when claude binary is missing")
	}

	var found bool
	for _, result := range report.Results {
		if result.Name == "claude_binary" {
			found = true
			if result.Status != StatusFail {
				t.Fatalf("expected claude_binary check to fail, got %s", result.Status)
			}
		}
	}
	if !found {
		t.Fatalf("expected claude_binary check result")
	}
}
