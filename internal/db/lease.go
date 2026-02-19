package db

import (
	"database/sql"
	"fmt"
	"time"
)

// SchedulerLease describes the active scheduler lease holder.
type SchedulerLease struct {
	HolderID       string
	LeaseExpiresAt time.Time
	UpdatedAt      time.Time
}

// TryAcquireSchedulerLease acquires or renews the scheduler lease for holderID.
// Returns true when holderID is now the active lease holder.
func (db *DB) TryAcquireSchedulerLease(holderID string, ttl time.Duration) (bool, *SchedulerLease, error) {
	if holderID == "" {
		return false, nil, fmt.Errorf("holder id is required")
	}
	if ttl <= 0 {
		return false, nil, fmt.Errorf("lease ttl must be positive")
	}

	now := time.Now()
	nowMS := now.UnixMilli()
	expiresMS := now.Add(ttl).UnixMilli()

	tx, err := db.conn.Begin()
	if err != nil {
		return false, nil, fmt.Errorf("begin lease transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO scheduler_leases (id, holder_id, lease_expires_at, updated_at)
		VALUES (1, ?, ?, ?)
	`, holderID, expiresMS, nowMS); err != nil {
		return false, nil, fmt.Errorf("insert scheduler lease: %w", err)
	}

	if _, err := tx.Exec(`
		UPDATE scheduler_leases
		SET holder_id = ?, lease_expires_at = ?, updated_at = ?
		WHERE id = 1 AND (holder_id = ? OR lease_expires_at <= ?)
	`, holderID, expiresMS, nowMS, holderID, nowMS); err != nil {
		return false, nil, fmt.Errorf("update scheduler lease: %w", err)
	}

	lease, err := readSchedulerLeaseTx(tx)
	if err != nil {
		return false, nil, err
	}

	if err := tx.Commit(); err != nil {
		return false, nil, fmt.Errorf("commit lease transaction: %w", err)
	}

	if lease == nil {
		return false, nil, nil
	}
	return lease.HolderID == holderID && lease.LeaseExpiresAt.After(now), lease, nil
}

// GetSchedulerLease returns the current scheduler lease row, if present.
func (db *DB) GetSchedulerLease() (*SchedulerLease, error) {
	return readSchedulerLeaseQuery(db.conn.QueryRow(`
		SELECT holder_id, lease_expires_at, updated_at
		FROM scheduler_leases
		WHERE id = 1
	`))
}

// ReleaseSchedulerLease releases the lease when held by holderID.
func (db *DB) ReleaseSchedulerLease(holderID string) error {
	if holderID == "" {
		return fmt.Errorf("holder id is required")
	}

	nowMS := time.Now().UnixMilli()
	_, err := db.conn.Exec(`
		UPDATE scheduler_leases
		SET lease_expires_at = ?, updated_at = ?
		WHERE id = 1 AND holder_id = ?
	`, nowMS, nowMS, holderID)
	if err != nil {
		return fmt.Errorf("release scheduler lease: %w", err)
	}
	return nil
}

func readSchedulerLeaseTx(tx *sql.Tx) (*SchedulerLease, error) {
	return readSchedulerLeaseQuery(tx.QueryRow(`
		SELECT holder_id, lease_expires_at, updated_at
		FROM scheduler_leases
		WHERE id = 1
	`))
}

func readSchedulerLeaseQuery(row *sql.Row) (*SchedulerLease, error) {
	var holderID string
	var leaseExpiresMS int64
	var updatedMS int64
	if err := row.Scan(&holderID, &leaseExpiresMS, &updatedMS); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("read scheduler lease: %w", err)
	}

	return &SchedulerLease{
		HolderID:       holderID,
		LeaseExpiresAt: time.UnixMilli(leaseExpiresMS),
		UpdatedAt:      time.UnixMilli(updatedMS),
	}, nil
}
