package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateCreatesExpectedColumns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tasks.db")
	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer func() { _ = database.Close() }()

	columns := tableColumns(t, database, "tasks")
	expected := []string{
		"id", "name", "prompt", "cron_expr", "scheduled_at", "working_dir",
		"discord_webhook", "slack_webhook", "model", "permission_mode",
		"enabled", "created_at", "updated_at", "last_run_at", "next_run_at",
	}

	for _, col := range expected {
		if !columns[col] {
			t.Fatalf("expected tasks column %q to exist; got columns: %s", col, strings.Join(sortedKeys(columns), ", "))
		}
	}

	runColumns := tableColumns(t, database, "task_runs")
	if !runColumns["session_id"] {
		t.Fatalf("expected task_runs.session_id column to exist")
	}
}

func TestUsageThresholdDefaultsTo80(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tasks.db")
	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer func() { _ = database.Close() }()

	threshold, err := database.GetUsageThreshold()
	if err != nil {
		t.Fatalf("GetUsageThreshold returned error: %v", err)
	}
	if threshold != 80 {
		t.Fatalf("expected default threshold 80, got %v", threshold)
	}
}

func TestNewFailsWhenMigrationCannotApplyNonDuplicateAlter(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tasks.db")

	bootstrap, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open bootstrap db: %v", err)
	}

	oldSchema := `
	CREATE TABLE tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		prompt TEXT NOT NULL,
		cron_expr TEXT NOT NULL,
		working_dir TEXT NOT NULL DEFAULT '.',
		discord_webhook TEXT DEFAULT '',
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_run_at DATETIME,
		next_run_at DATETIME
	);
	CREATE TABLE task_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id INTEGER NOT NULL,
		started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		ended_at DATETIME,
		status TEXT NOT NULL DEFAULT 'pending',
		output TEXT DEFAULT '',
		error TEXT DEFAULT '',
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
	);
	CREATE INDEX idx_task_runs_task_id ON task_runs(task_id);
	CREATE INDEX idx_task_runs_started_at ON task_runs(started_at);
	CREATE TABLE settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	INSERT INTO settings (key, value) VALUES ('usage_threshold', '80');
	`
	if _, err := bootstrap.Exec(oldSchema); err != nil {
		_ = bootstrap.Close()
		t.Fatalf("create old schema: %v", err)
	}
	if err := bootstrap.Close(); err != nil {
		t.Fatalf("close bootstrap db: %v", err)
	}

	if err := os.Chmod(dbPath, 0o444); err != nil {
		t.Fatalf("chmod readonly db: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dbPath, 0o644)
	})

	database, err := New(dbPath)
	if err == nil {
		_ = database.Close()
		t.Fatalf("expected migration error for readonly db with missing columns")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "migrate") {
		t.Fatalf("expected migration context in error, got: %v", err)
	}
}


func tableColumns(t *testing.T, database *DB, table string) map[string]bool {
	t.Helper()

	rows, err := database.conn.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("query table info: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan table info: %v", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table info: %v", err)
	}
	return columns
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
