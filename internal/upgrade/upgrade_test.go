package upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceExecutableReplacesBinaryAndCleansBackup(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "claude-tasks")
	newPath := filepath.Join(dir, "claude-tasks-new")

	if err := os.WriteFile(oldPath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("write old binary: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("new-binary"), 0o755); err != nil {
		t.Fatalf("write new binary: %v", err)
	}

	if err := replaceExecutable(oldPath, newPath); err != nil {
		t.Fatalf("replaceExecutable failed: %v", err)
	}

	data, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatalf("read replaced binary: %v", err)
	}
	if string(data) != "new-binary" {
		t.Fatalf("expected replaced binary content %q, got %q", "new-binary", string(data))
	}

	if _, err := os.Stat(oldPath + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("expected backup to be removed, stat err=%v", err)
	}
}

func TestReplaceExecutableKeepsOriginalWhenStagingFails(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "claude-tasks")
	missingNewPath := filepath.Join(dir, "does-not-exist")

	if err := os.WriteFile(oldPath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("write old binary: %v", err)
	}

	err := replaceExecutable(oldPath, missingNewPath)
	if err == nil {
		t.Fatalf("expected replaceExecutable to fail for missing new binary")
	}

	data, readErr := os.ReadFile(oldPath)
	if readErr != nil {
		t.Fatalf("read original binary: %v", readErr)
	}
	if string(data) != "old-binary" {
		t.Fatalf("expected original binary content %q, got %q", "old-binary", string(data))
	}
}
