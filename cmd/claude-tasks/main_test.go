package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestIsDaemonRunningReturnsFalseForMissingPidFile(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "missing.pid")
	if pid, ok := isDaemonRunning(pidPath); ok || pid != 0 {
		t.Fatalf("expected daemon not running, got pid=%d ok=%v", pid, ok)
	}
}

func TestIsDaemonRunningReturnsFalseForInvalidPidContent(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "daemon.pid")
	if err := os.WriteFile(pidPath, []byte("not-a-pid"), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	if pid, ok := isDaemonRunning(pidPath); ok || pid != 0 {
		t.Fatalf("expected daemon not running, got pid=%d ok=%v", pid, ok)
	}
}

func TestIsDaemonRunningReturnsTrueForCurrentProcessPid(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "daemon.pid")
	selfPID := os.Getpid()
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(selfPID)), 0o644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	pid, ok := isDaemonRunning(pidPath)
	if !ok {
		t.Fatalf("expected daemon running for current process pid")
	}
	if pid != selfPID {
		t.Fatalf("expected pid %d, got %d", selfPID, pid)
	}
}

func TestWritePIDFileCreatesParentDirectory(t *testing.T) {
	root := t.TempDir()
	pidPath := filepath.Join(root, "nested", "daemon.pid")

	if err := writePIDFile(pidPath, 4242); err != nil {
		t.Fatalf("writePIDFile failed: %v", err)
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("read pid file: %v", err)
	}
	if string(data) != "4242" {
		t.Fatalf("expected pid file content 4242, got %q", string(data))
	}
}

func TestCreateListenerFailsWhenAddressAlreadyInUse(t *testing.T) {
	first, err := createListener("127.0.0.1:0")
	if err != nil {
		t.Fatalf("create first listener: %v", err)
	}
	defer first.Close()

	if second, err := createListener(first.Addr().String()); err == nil {
		_ = second.Close()
		t.Fatalf("expected second listener creation to fail on same address")
	}
}

