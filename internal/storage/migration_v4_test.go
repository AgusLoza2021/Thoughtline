package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// seedV3DB opens a raw SQLite connection at path, applies the current schemaSQL
// (which is idempotent), and inserts the given memories directly via SQL.
// This lets tests set up a "pre-v4" state and then call Open() to trigger
// the v4 migration.
//
// Each entry in rows is (project, scope, type, title, content).
// scope and type default to "project" and "note" if empty.
func seedV3DB(t *testing.T, path string, rows []memRow) {
	t.Helper()
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("seedV3DB open: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		t.Fatalf("seedV3DB schemaSQL: %v", err)
	}
	// Add session_id column if not present (v2 migration step)
	hasSession, err := columnExists(ctx, db, "memories", "session_id")
	if err != nil {
		t.Fatalf("seedV3DB columnExists: %v", err)
	}
	if !hasSession {
		if _, err := db.ExecContext(ctx, `ALTER TABLE memories ADD COLUMN session_id TEXT`); err != nil {
			t.Fatalf("seedV3DB add session_id: %v", err)
		}
	}
	// Mark as v3 so we can see the migration bump to v4 later.
	if _, err := db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_version(version, applied_at) VALUES (3, ?)`,
		time.Now().UnixMilli()); err != nil {
		t.Fatalf("seedV3DB schema_version: %v", err)
	}

	now := time.Now().UnixMilli()
	for i, r := range rows {
		scope := r.scope
		if scope == "" {
			scope = "project"
		}
		typ := r.typ
		if typ == "" {
			typ = "note"
		}
		syncID := fmt.Sprintf("sync-%d-%d", now, i)
		hash := fmt.Sprintf("hash-%d", i)
		if _, err := db.ExecContext(ctx, `
			INSERT INTO memories (sync_id, project, scope, type, title, content, normalized_hash, revision_count, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
			syncID, r.project, scope, typ, r.title, r.content, hash, now, now,
		); err != nil {
			t.Fatalf("seedV3DB insert memory[%d]: %v", i, err)
		}
	}
}

type memRow struct {
	project string
	title   string
	content string
	scope   string
	typ     string
}

// TestMigrateV4_Backfill verifies that the v4 migration creates exactly one
// brain row per distinct memories.project value (kind='real'), and that every
// memory has a non-null brain_id pointing to the correct brain.
func TestMigrateV4_Backfill(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backfill.db")
	seedV3DB(t, path, []memRow{
		{project: "alpha", title: "a1", content: "c"},
		{project: "alpha", title: "a2", content: "c"},
		{project: "beta", title: "b1", content: "c"},
		{project: "gamma", title: "g1", content: "c"},
		{project: "gamma", title: "g2", content: "c"},
	})

	ctx := context.Background()
	st, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	// Exactly 3 brains must exist.
	var brainCount int
	if err := st.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM brains`).Scan(&brainCount); err != nil {
		t.Fatalf("count brains: %v", err)
	}
	if brainCount != 3 {
		t.Errorf("expected 3 brains, got %d", brainCount)
	}

	// All brains must have kind='real'.
	var realCount int
	if err := st.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM brains WHERE kind = 'real'`).Scan(&realCount); err != nil {
		t.Fatalf("count real brains: %v", err)
	}
	if realCount != 3 {
		t.Errorf("expected 3 real brains, got %d", realCount)
	}

	// All memories must have a non-null brain_id.
	var nullCount int
	if err := st.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM memories WHERE brain_id IS NULL`).Scan(&nullCount); err != nil {
		t.Fatalf("count null brain_id: %v", err)
	}
	if nullCount != 0 {
		t.Errorf("expected 0 memories with null brain_id, got %d", nullCount)
	}

	// Each memory's brain_id must match its project slug.
	rows, err := st.db.QueryContext(ctx, `
		SELECT m.project, b.slug
		FROM memories m
		JOIN brains b ON b.id = m.brain_id`)
	if err != nil {
		t.Fatalf("query project/slug match: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var project, slug string
		if err := rows.Scan(&project, &slug); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if project != slug {
			t.Errorf("memory.project=%q but brain.slug=%q", project, slug)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}
}

// TestMigrateV4_Idempotent verifies that running Open twice on the same
// database does not create duplicate brains and does not return an error.
// Memory brain_id values must be unchanged after the second open.
func TestMigrateV4_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idempotent.db")
	seedV3DB(t, path, []memRow{
		{project: "alpha", title: "a1", content: "c"},
		{project: "beta", title: "b1", content: "c"},
	})

	ctx := context.Background()

	// First open — runs v4 migration.
	st1, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	// Capture brain IDs after first migration.
	type brainInfo struct {
		id   int64
		slug string
	}
	var firstBrains []brainInfo
	rows, err := st1.db.QueryContext(ctx, `SELECT id, slug FROM brains ORDER BY slug`)
	if err != nil {
		t.Fatalf("query brains first open: %v", err)
	}
	for rows.Next() {
		var b brainInfo
		if err := rows.Scan(&b.id, &b.slug); err != nil {
			t.Fatalf("scan: %v", err)
		}
		firstBrains = append(firstBrains, b)
	}
	_ = rows.Close()

	// Capture memory brain_ids after first migration.
	type memBrain struct {
		id      int64
		brainID sql.NullInt64
	}
	var firstMemBrains []memBrain
	mrows, err := st1.db.QueryContext(ctx, `SELECT id, brain_id FROM memories ORDER BY id`)
	if err != nil {
		t.Fatalf("query memories first open: %v", err)
	}
	for mrows.Next() {
		var mb memBrain
		if err := mrows.Scan(&mb.id, &mb.brainID); err != nil {
			t.Fatalf("scan memory: %v", err)
		}
		firstMemBrains = append(firstMemBrains, mb)
	}
	_ = mrows.Close()
	st1.Close()

	// Second open — must be idempotent.
	st2, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer st2.Close()

	// Brain count must be unchanged.
	var brainCount int
	if err := st2.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM brains`).Scan(&brainCount); err != nil {
		t.Fatalf("count brains: %v", err)
	}
	if brainCount != 2 {
		t.Errorf("expected 2 brains after second open, got %d", brainCount)
	}

	// Brain IDs and slugs must match first open.
	rows2, err := st2.db.QueryContext(ctx, `SELECT id, slug FROM brains ORDER BY slug`)
	if err != nil {
		t.Fatalf("query brains second open: %v", err)
	}
	var secondBrains []brainInfo
	for rows2.Next() {
		var b brainInfo
		if err := rows2.Scan(&b.id, &b.slug); err != nil {
			t.Fatalf("scan: %v", err)
		}
		secondBrains = append(secondBrains, b)
	}
	_ = rows2.Close()

	if len(firstBrains) != len(secondBrains) {
		t.Fatalf("brain count mismatch: first=%d second=%d", len(firstBrains), len(secondBrains))
	}
	for i := range firstBrains {
		if firstBrains[i].id != secondBrains[i].id {
			t.Errorf("brain[%d] id changed: %d → %d", i, firstBrains[i].id, secondBrains[i].id)
		}
		if firstBrains[i].slug != secondBrains[i].slug {
			t.Errorf("brain[%d] slug changed: %q → %q", i, firstBrains[i].slug, secondBrains[i].slug)
		}
	}

	// Memory brain_ids must be unchanged.
	mrows2, err := st2.db.QueryContext(ctx, `SELECT id, brain_id FROM memories ORDER BY id`)
	if err != nil {
		t.Fatalf("query memories second open: %v", err)
	}
	var secondMemBrains []memBrain
	for mrows2.Next() {
		var mb memBrain
		if err := mrows2.Scan(&mb.id, &mb.brainID); err != nil {
			t.Fatalf("scan memory: %v", err)
		}
		secondMemBrains = append(secondMemBrains, mb)
	}
	_ = mrows2.Close()

	if len(firstMemBrains) != len(secondMemBrains) {
		t.Fatalf("memory count mismatch: first=%d second=%d", len(firstMemBrains), len(secondMemBrains))
	}
	for i := range firstMemBrains {
		if firstMemBrains[i].brainID != secondMemBrains[i].brainID {
			t.Errorf("memory[%d] brain_id changed: %v → %v", i,
				firstMemBrains[i].brainID, secondMemBrains[i].brainID)
		}
	}
}

// TestMigrateV4_GlobalConfigPreserved verifies that a pre-existing global_config
// row (e.g. from a previous migration) is not overwritten on re-migration.
func TestMigrateV4_GlobalConfigPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "global_config.db")
	seedV3DB(t, path, []memRow{
		{project: "proj", title: "t", content: "c"},
	})

	ctx := context.Background()

	// First open runs the migration and seeds global_config with "{}".
	st1, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	// Overwrite global_config with a custom value.
	customJSON := `{"ranking":{"bm25":0.99}}`
	if _, err := st1.db.ExecContext(ctx,
		`UPDATE global_config SET config_json = ? WHERE id = 1`, customJSON); err != nil {
		t.Fatalf("update global_config: %v", err)
	}
	st1.Close()

	// Second open must not overwrite the custom config.
	st2, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer st2.Close()

	var stored string
	if err := st2.db.QueryRowContext(ctx,
		`SELECT config_json FROM global_config WHERE id = 1`).Scan(&stored); err != nil {
		t.Fatalf("read global_config: %v", err)
	}
	if stored != customJSON {
		t.Errorf("global_config overwritten: got %q, want %q", stored, customJSON)
	}
}

// TestMigrateV4_GlobalConfigSeededWithDefaults verifies that a freshly
// migrated database has the global_config row seeded with config.DefaultGlobalJSON()
// (not the old hardcoded "{}").
func TestMigrateV4_GlobalConfigSeededWithDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "globalcfg_defaults.db")
	seedV3DB(t, path, []memRow{
		{project: "alpha", title: "t1", content: "c1"},
	})

	ctx := context.Background()
	st, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	var stored string
	if err := st.db.QueryRowContext(ctx,
		`SELECT config_json FROM global_config WHERE id = 1`).Scan(&stored); err != nil {
		t.Fatalf("read global_config: %v", err)
	}

	want := configDefaultGlobalJSON()
	if stored == "{}" {
		t.Error("global_config seeded with bare '{}' — expected full DefaultGlobalJSON()")
	}
	if stored != want {
		t.Errorf("global_config mismatch:\ngot:  %s\nwant: %s", stored, want)
	}
}

// TestMigrateV4_EmptyProject verifies that when the DB contains a memory with
// project="" (empty string), the v4 migration aborts and returns an error
// that references the offending memory ID. No partial write is committed.
func TestMigrateV4_EmptyProject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty_project.db")
	// Seed a v3 DB with one memory that has project="".
	seedV3DB(t, path, []memRow{
		{project: "", title: "orphan memory", content: "no project"},
	})

	ctx := context.Background()
	_, err := Open(ctx, path)
	if err == nil {
		t.Fatal("expected Open to return an error for empty-project memory, got nil")
	}

	errMsg := err.Error()
	// Must mention the count or the offending ID.
	if !strings.Contains(errMsg, "brain_id") && !strings.Contains(errMsg, "project") && !strings.Contains(errMsg, "backfill") {
		t.Errorf("error message should reference backfill/brain_id/project, got: %q", errMsg)
	}

	// Verify no partial write: brains table must still be empty.
	// Re-open raw to check (without triggering migrate again via Open).
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err2 := sql.Open("sqlite", dsn)
	if err2 != nil {
		t.Fatalf("raw re-open: %v", err2)
	}
	defer db.Close()

	// The brains table may not even exist yet (migration was rolled back).
	// If it exists, it must be empty.
	var tableExists int
	_ = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='brains'`,
	).Scan(&tableExists)

	if tableExists == 1 {
		var n int
		if err3 := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM brains`).Scan(&n); err3 == nil && n > 0 {
			t.Errorf("partial write: brains table has %d rows after failed migration", n)
		}
	}
}
