package migrate

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// openTestEngramDB creates a fresh SQLite DB in t.TempDir(), applies the Engram
// schema from testdata/engram_schema.sql, and returns an open *sql.DB.
func openTestEngramDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "engram.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open test engram db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile(filepath.Join("testdata", "engram_schema.sql"))
	if err != nil {
		t.Fatalf("read engram schema: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("apply engram schema: %v", err)
	}
	return db
}

// insertTestRow inserts a minimal Engram observation row; callers can
// override individual fields by patching the returned struct after insert.
func insertEngramRow(t *testing.T, db *sql.DB, syncID, typ, title, scope string, deletedAt *string) {
	t.Helper()
	q := `INSERT INTO observations
		(sync_id, type, title, content, project, scope, topic_key,
		 normalized_hash, revision_count, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(q,
		syncID, typ, title,
		"some content", "test-project", scope,
		nil,              // topic_key
		"abc123",         // normalized_hash
		0,                // revision_count
		"2024-01-01T00:00:00Z", // created_at
		"2024-01-01T00:00:00Z", // updated_at
		deletedAt,
	)
	if err != nil {
		t.Fatalf("insert engram row %s: %v", syncID, err)
	}
}

func TestReadObservations_FiltersDeletedRows(t *testing.T) {
	db := openTestEngramDB(t)
	ctx := context.Background()

	// Seed 3 rows: 2 active, 1 soft-deleted.
	insertEngramRow(t, db, "obs-active-1", "bugfix", "Active one", "project", nil)
	insertEngramRow(t, db, "obs-active-2", "decision", "Active two", "project", nil)
	deletedAt := "2024-06-01T12:00:00Z"
	insertEngramRow(t, db, "obs-deleted-1", "pattern", "Deleted one", "project", &deletedAt)

	rows, err := ReadObservations(ctx, db)
	if err != nil {
		t.Fatalf("ReadObservations: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 active observations, got %d: %+v", len(rows), rows)
	}

	// Verify the soft-deleted row is absent.
	for _, r := range rows {
		if r.SyncID == "obs-deleted-1" {
			t.Errorf("soft-deleted row obs-deleted-1 should not appear in results")
		}
	}

	// Spot-check that active rows have their fields mapped correctly.
	found := map[string]bool{}
	for _, r := range rows {
		found[r.SyncID] = true
		if r.SyncID == "obs-active-1" {
			if r.Type != "bugfix" {
				t.Errorf("obs-active-1 type = %q, want %q", r.Type, "bugfix")
			}
			if r.Title != "Active one" {
				t.Errorf("obs-active-1 title = %q, want %q", r.Title, "Active one")
			}
		}
	}
	if !found["obs-active-1"] || !found["obs-active-2"] {
		t.Errorf("missing expected active rows; got %v", found)
	}
}

func TestReadObservations_EmptyDB(t *testing.T) {
	db := openTestEngramDB(t)
	ctx := context.Background()

	rows, err := ReadObservations(ctx, db)
	if err != nil {
		t.Fatalf("ReadObservations on empty DB: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows from empty DB, got %d", len(rows))
	}
}
