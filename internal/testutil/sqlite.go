package testutil

import (
	"testing"

	"github.com/ASRagab/claude-tasks/internal/db"
)

func NewTestDB(t *testing.T) (*db.DB, string) {
	t.Helper()

	dataDir := TempDataDir(t)
	database, err := db.New(dataDir + "/tasks.db")
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}

	t.Cleanup(func() {
		_ = database.Close()
	})

	return database, dataDir
}
