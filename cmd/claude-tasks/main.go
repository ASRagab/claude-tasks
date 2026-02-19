package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ASRagab/claude-tasks/internal/api"
	"github.com/ASRagab/claude-tasks/internal/db"
	"github.com/ASRagab/claude-tasks/internal/doctor"
	"github.com/ASRagab/claude-tasks/internal/scheduler"
	"github.com/ASRagab/claude-tasks/internal/tui"
	"github.com/ASRagab/claude-tasks/internal/upgrade"
	"github.com/ASRagab/claude-tasks/internal/version"
)

type tuiSchedulerMode string

const (
	tuiSchedulerAuto tuiSchedulerMode = "auto"
	tuiSchedulerOn   tuiSchedulerMode = "on"
	tuiSchedulerOff  tuiSchedulerMode = "off"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println(version.Info())
			return
		case "upgrade":
			if err := upgrade.Upgrade(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "daemon":
			if err := runDaemon(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "serve":
			if err := runServer(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "doctor":
			if err := runDoctor(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "tui":
			if err := runTUI(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		default:
			if strings.HasPrefix(os.Args[1], "-") {
				if err := runTUI(os.Args[1:]); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				return
			}
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}

	if err := runTUI(nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseTUISchedulerMode(value string) (tuiSchedulerMode, error) {
	switch tuiSchedulerMode(strings.ToLower(strings.TrimSpace(value))) {
	case tuiSchedulerAuto:
		return tuiSchedulerAuto, nil
	case tuiSchedulerOn:
		return tuiSchedulerOn, nil
	case tuiSchedulerOff:
		return tuiSchedulerOff, nil
	default:
		return "", fmt.Errorf("invalid --scheduler value %q (expected auto|on|off)", value)
	}
}

func shouldStartTUIScheduler(mode tuiSchedulerMode, daemonRunning bool) bool {
	switch mode {
	case tuiSchedulerAuto:
		return !daemonRunning
	case tuiSchedulerOn:
		return true
	default:
		return false
	}
}

func runTUI(args []string) error {
	tuiCmd := flag.NewFlagSet("tui", flag.ExitOnError)
	schedulerModeRaw := tuiCmd.String("scheduler", string(tuiSchedulerAuto), "Scheduler mode: auto, on, off")
	_ = tuiCmd.Parse(args)

	schedulerMode, err := parseTUISchedulerMode(*schedulerModeRaw)
	if err != nil {
		return err
	}

	dataDir := os.Getenv("CLAUDE_TASKS_DATA")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".claude-tasks")
	}

	dbPath := filepath.Join(dataDir, "tasks.db")
	pidPath := filepath.Join(dataDir, "daemon.pid")

	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	defer database.Close()

	daemonPID, daemonRunning := isDaemonRunning(pidPath)
	startScheduler := shouldStartTUIScheduler(schedulerMode, daemonRunning)

	var sched *scheduler.Scheduler
	if startScheduler {
		sched = scheduler.New(database, dataDir)
		if err := sched.Start(); err != nil {
			return fmt.Errorf("starting scheduler: %w", err)
		}
		defer sched.Stop()
		fmt.Printf("TUI scheduler mode: %s (leader=%v)\n", schedulerMode, sched.IsLeader())
	} else {
		if daemonRunning {
			fmt.Printf("Daemon running (PID %d), TUI in client mode\n", daemonPID)
		}
		fmt.Printf("TUI scheduler mode: %s (scheduler disabled in this process)\n", schedulerMode)
	}

	daemonMode := sched == nil
	if err := tui.Run(database, sched, daemonMode, dataDir); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}
	return nil
}

func runDoctor() error {
	dataDir := os.Getenv("CLAUDE_TASKS_DATA")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".claude-tasks")
	}

	report := doctor.NewRunner(dataDir).Run()

	fmt.Println("claude-tasks doctor")
	fmt.Printf("Data directory: %s\n", report.DataDir)
	fmt.Printf("Database path: %s\n", report.DBPath)
	for _, result := range report.Results {
		fmt.Printf("[%s] %s: %s\n", result.Status, result.Name, result.Detail)
		if result.Hint != "" {
			fmt.Printf("  hint: %s\n", result.Hint)
		}
	}

	if report.CriticalFailures > 0 {
		return fmt.Errorf("doctor found %d critical issue(s)", report.CriticalFailures)
	}

	fmt.Println("Doctor checks passed")
	return nil
}

func runDaemon() error {
	daemonCmd := flag.NewFlagSet("daemon", flag.ExitOnError)
	schedulerEnabled := daemonCmd.Bool("scheduler", true, "Enable scheduler loop")
	_ = daemonCmd.Parse(os.Args[2:])

	dataDir := os.Getenv("CLAUDE_TASKS_DATA")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".claude-tasks")
	}

	dbPath := filepath.Join(dataDir, "tasks.db")
	pidPath := filepath.Join(dataDir, "daemon.pid")

	// Check if daemon is already running
	if pid, running := isDaemonRunning(pidPath); running {
		return fmt.Errorf("daemon already running (PID %d)", pid)
	}

	// Write PID file
	if err := writePIDFile(pidPath, os.Getpid()); err != nil {
		return fmt.Errorf("writing PID file: %w", err)
	}
	defer os.Remove(pidPath)

	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	defer database.Close()

	var sched *scheduler.Scheduler
	if *schedulerEnabled {
		sched = scheduler.New(database, dataDir)
		if err := sched.Start(); err != nil {
			return fmt.Errorf("starting scheduler: %w", err)
		}
		defer sched.Stop()
		fmt.Printf("Daemon scheduler: enabled (leader=%v)\n", sched.IsLeader())
	} else {
		fmt.Println("Daemon scheduler: disabled")
	}

	fmt.Println("claude-tasks daemon started")
	fmt.Printf("PID: %d\n", os.Getpid())
	fmt.Printf("Database: %s\n", dbPath)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down...")
	return nil
}

func runServer() error {
	// Parse flags for serve command
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	port := serveCmd.Int("port", 8080, "HTTP server port")
	schedulerEnabled := serveCmd.Bool("scheduler", true, "Enable scheduler loop")
	_ = serveCmd.Parse(os.Args[2:])

	dataDir := os.Getenv("CLAUDE_TASKS_DATA")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("getting home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".claude-tasks")
	}

	dbPath := filepath.Join(dataDir, "tasks.db")

	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	defer database.Close()

	var sched *scheduler.Scheduler
	if *schedulerEnabled {
		sched = scheduler.New(database, dataDir)
		if err := sched.Start(); err != nil {
			return fmt.Errorf("starting scheduler: %w", err)
		}
		defer sched.Stop()
		fmt.Printf("Serve scheduler: enabled (leader=%v)\n", sched.IsLeader())
	} else {
		fmt.Println("Serve scheduler: disabled")
	}

	server := api.NewServer(database, sched, dataDir)

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("claude-tasks API server starting on %s\n", addr)
	fmt.Printf("Database: %s\n", dbPath)

	srv := &http.Server{
		Addr:    addr,
		Handler: server.Router(),
	}

	listener, err := createListener(addr)
	if err != nil {
		return fmt.Errorf("starting listener on %s: %w", addr, err)
	}

	serverErrCh := make(chan error, 1)
	go func() {
		if serveErr := srv.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			serverErrCh <- serveErr
		}
		close(serverErrCh)
	}()

	// Wait for shutdown signal or server error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		fmt.Println("\nShutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	case serveErr := <-serverErrCh:
		if serveErr != nil {
			return fmt.Errorf("server error: %w", serveErr)
		}
		return nil
	}
}

func writePIDFile(pidPath string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(pidPath), 0755); err != nil {
		return fmt.Errorf("ensuring pid directory: %w", err)
	}
	return os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func createListener(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

// isDaemonRunning checks if a daemon is running by reading PID file and checking process
func isDaemonRunning(pidPath string) (int, bool) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, false
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}

	// On Unix, FindProcess always succeeds, so send signal 0 to check if alive
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return 0, false
	}

	return pid, true
}

func printHelp() {
	fmt.Println(`claude-tasks - Schedule and run Claude CLI tasks via cron

Usage:
  claude-tasks                              Launch the interactive TUI
  claude-tasks --scheduler=auto|on|off      Launch TUI with explicit scheduler mode
  claude-tasks tui --scheduler=auto|on|off  Launch TUI with explicit scheduler mode
  claude-tasks daemon [--scheduler=true|false]
                                            Run daemon (scheduler optional)
  claude-tasks serve [--port 8080] [--scheduler=true|false]
                                            Run HTTP API server (scheduler optional)
  claude-tasks doctor                       Run environment and runtime diagnostics
  claude-tasks version                      Show version information
  claude-tasks upgrade                      Upgrade to the latest version
  claude-tasks help                         Show this help message

Environment Variables:
  CLAUDE_TASKS_DATA         Override data directory (default: ~/.claude-tasks)

For more information, visit: https://github.com/ASRagab/claude-tasks`)
}
