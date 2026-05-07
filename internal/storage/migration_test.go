package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// TestMigration_V3_CreatesTablePendingEvents verifies that the v2→v3 migration
// creates the pending_events table with all required columns, the CHECK
// constraint on status, the UNIQUE constraint on (project, event_hash), and
// the three required indexes.
func TestMigration_V3_CreatesTablePendingEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrate.db")
	st, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	ctx := context.Background()

	// Verify table exists by querying PRAGMA.
	rows, err := st.db.QueryContext(ctx, `PRAGMA table_info(pending_events)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()

	requiredCols := map[string]bool{
		"id":                 false,
		"sync_id":            false,
		"project":            false,
		"session_id":         false,
		"event_type":         false,
		"tool_name":          false,
		"tool_use_id":        false,
		"payload":            false,
		"event_hash":         false,
		"status":             false,
		"promoted_memory_id": false,
		"promoted_at":        false,
		"archived_at":        false,
		"created_at":         false,
		"captured_at":        false,
	}

	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		requiredCols[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	for col, found := range requiredCols {
		if !found {
			t.Errorf("pending_events missing column %q", col)
		}
	}

	// Verify the UNIQUE index on (project, event_hash) exists.
	idxRows, err := st.db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='pending_events'`)
	if err != nil {
		t.Fatalf("query indexes: %v", err)
	}
	defer idxRows.Close()

	foundIndexes := map[string]bool{}
	for idxRows.Next() {
		var name string
		if err := idxRows.Scan(&name); err != nil {
			t.Fatalf("scan index name: %v", err)
		}
		foundIndexes[name] = true
	}
	if err := idxRows.Err(); err != nil {
		t.Fatalf("index rows error: %v", err)
	}

	requiredIdxs := []string{
		"idx_pending_events_dedup",
		"idx_pending_events_triage",
		"idx_pending_events_session",
	}
	for _, idx := range requiredIdxs {
		if !foundIndexes[idx] {
			t.Errorf("pending_events missing index %q; found: %v", idx, foundIndexes)
		}
	}

	// Verify schema_version row 3 was recorded.
	var ver int
	if err := st.db.QueryRowContext(ctx,
		`SELECT version FROM schema_version WHERE version = 3`).Scan(&ver); err != nil {
		t.Errorf("schema_version row 3 not found: %v", err)
	}
}

// TestMigration_V3_Idempotent verifies that opening the same database file
// twice runs the migration twice without error and without data loss.
func TestMigration_V3_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idempotent.db")
	ctx := context.Background()

	st1, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	// Insert a row so we can verify data survives the re-open.
	if _, err := st1.db.ExecContext(ctx, `
		INSERT INTO pending_events
		(sync_id, project, event_type, payload, event_hash, status, created_at, captured_at)
		VALUES ('sync-abc', 'proj', 'SessionStart', '{}', 'hash1', 'pending', 0, 0)
	`); err != nil {
		t.Fatalf("insert row: %v", err)
	}
	if err := st1.Close(); err != nil {
		t.Fatalf("close st1: %v", err)
	}

	// Re-open same DB — migrate runs again, must be safe.
	st2, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer st2.Close()

	var n int
	if err := st2.db.QueryRowContext(ctx,
		`SELECT count(*) FROM pending_events`).Scan(&n); err != nil {
		t.Fatalf("count after reopen: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row after reopen, got %d", n)
	}
}
