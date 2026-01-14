package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kylemclaren/claude-tasks/internal/db"
	"github.com/kylemclaren/claude-tasks/internal/scheduler"
	"github.com/kylemclaren/claude-tasks/internal/tui"
	"github.com/kylemclaren/claude-tasks/internal/upgrade"
	"github.com/kylemclaren/claude-tasks/internal/version"
)

func main() {
	// Handle CLI commands
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
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}

	// Determine database path
	dataDir := os.Getenv("CLAUDE_TASKS_DATA")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}
		dataDir = filepath.Join(homeDir, ".claude-tasks")
	}

	dbPath := filepath.Join(dataDir, "tasks.db")

	// Initialize database
	database, err := db.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Initialize scheduler
	sched := scheduler.New(database)
	if err := sched.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting scheduler: %v\n", err)
		os.Exit(1)
	}
	defer sched.Stop()

	// Run TUI
	if err := tui.Run(database, sched); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`claude-tasks - Schedule and run Claude CLI tasks via cron

Usage:
  claude-tasks              Launch the interactive TUI
  claude-tasks version      Show version information
  claude-tasks upgrade      Upgrade to the latest version
  claude-tasks help         Show this help message

Environment Variables:
  CLAUDE_TASKS_DATA         Override data directory (default: ~/.claude-tasks)

For more information, visit: https://github.com/kylemclaren/claude-tasks`)
}
