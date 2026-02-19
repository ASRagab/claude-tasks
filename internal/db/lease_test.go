package db_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ASRagab/claude-tasks/internal/db"
)

func newLeaseTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.New(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	return database
}

func TestTryAcquireSchedulerLease(t *testing.T) {
	database := newLeaseTestDB(t)

	acquired, lease, err := database.TryAcquireSchedulerLease("holder-a", 2*time.Second)
	if err != nil {
		t.Fatalf("acquire lease: %v", err)
	}
	if !acquired {
		t.Fatalf("expected holder-a to acquire lease")
	}
	if lease == nil || lease.HolderID != "holder-a" {
		t.Fatalf("expected holder-a lease, got %#v", lease)
	}
}

func TestTryAcquireSchedulerLeaseRejectsContenderBeforeExpiry(t *testing.T) {
	database := newLeaseTestDB(t)

	acquiredA, _, err := database.TryAcquireSchedulerLease("holder-a", 2*time.Second)
	if err != nil {
		t.Fatalf("acquire holder-a lease: %v", err)
	}
	if !acquiredA {
		t.Fatalf("expected holder-a to acquire lease")
	}

	acquiredB, lease, err := database.TryAcquireSchedulerLease("holder-b", 2*time.Second)
	if err != nil {
		t.Fatalf("acquire holder-b lease: %v", err)
	}
	if acquiredB {
		t.Fatalf("expected holder-b to fail acquiring active lease")
	}
	if lease == nil || lease.HolderID != "holder-a" {
		t.Fatalf("expected holder-a to remain lease holder, got %#v", lease)
	}
}

func TestTryAcquireSchedulerLeaseAllowsTakeoverAfterExpiry(t *testing.T) {
	database := newLeaseTestDB(t)

	acquiredA, _, err := database.TryAcquireSchedulerLease("holder-a", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire holder-a lease: %v", err)
	}
	if !acquiredA {
		t.Fatalf("expected holder-a to acquire lease")
	}

	time.Sleep(80 * time.Millisecond)

	acquiredB, lease, err := database.TryAcquireSchedulerLease("holder-b", 2*time.Second)
	if err != nil {
		t.Fatalf("acquire holder-b lease: %v", err)
	}
	if !acquiredB {
		t.Fatalf("expected holder-b to acquire expired lease")
	}
	if lease == nil || lease.HolderID != "holder-b" {
		t.Fatalf("expected holder-b to hold lease, got %#v", lease)
	}
}

func TestReleaseSchedulerLease(t *testing.T) {
	database := newLeaseTestDB(t)

	acquired, _, err := database.TryAcquireSchedulerLease("holder-a", 2*time.Second)
	if err != nil {
		t.Fatalf("acquire holder-a lease: %v", err)
	}
	if !acquired {
		t.Fatalf("expected holder-a to acquire lease")
	}

	if err := database.ReleaseSchedulerLease("holder-a"); err != nil {
		t.Fatalf("release lease: %v", err)
	}

	acquiredB, lease, err := database.TryAcquireSchedulerLease("holder-b", 2*time.Second)
	if err != nil {
		t.Fatalf("acquire holder-b lease: %v", err)
	}
	if !acquiredB {
		t.Fatalf("expected holder-b to acquire released lease")
	}
	if lease == nil || lease.HolderID != "holder-b" {
		t.Fatalf("expected holder-b to hold lease, got %#v", lease)
	}
}
