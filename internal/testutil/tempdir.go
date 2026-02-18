package testutil

import (
	"path/filepath"
	"testing"
)

func TempDataDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func TempDBPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(TempDataDir(t), "tasks.db")
}
