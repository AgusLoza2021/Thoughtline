package migrate

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// buildEngramFixture creates a fresh engram.db in t.TempDir() and seeds it
// with the rows described in the test strategy. Returns the DB path.
//
// Rows seeded:
//   - 1 per Engram type: bugfix, preference, decision, architecture,
//     pattern, config, discovery, manual (= 8 active rows)
//   - 1 soft-deleted row (type=bugfix, sync_id=obs-deleted)
//   - 1 row with oversize content (> 64 KiB) (sync_id=obs-oversize-content)
//   - 1 row with title > 200 runes (sync_id=obs-long-title)  [still migrated]
//
// Total seeded: 11 rows (10 active + 1 soft-deleted)
// Reader filters soft-deleted at SQL level, so 10 rows reach the mapper/writer.
// Expected after migration:
//   - Total processed:  10 (= 11 seeded - 1 soft-deleted)
//   - Created:           9 (= 10 - 1 oversize-content error)
//   - Errors:            1 (the oversize row)
//   - Plus 1 truncation (the long-title row, still migrated)
func buildEngramFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "engram.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	defer func() { _ = db.Close() }()

	schema, err := os.ReadFile(filepath.Join("testdata", "engram_schema.sql"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	ins := `INSERT INTO observations
		(sync_id, type, title, content, project, scope, topic_key,
		 normalized_hash, revision_count, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	ts := "2024-03-15T10:30:00Z"

	rows := []struct {
		syncID    string
		typ       string
		title     string
		content   string
		topicKey  interface{}
		deletedAt interface{}
	}{
		{"obs-bugfix-1", "bugfix", "Fix startup crash", "Root cause: nil pointer.", "bug/startup", nil},
		{"obs-preference-1", "preference", "Prefer 2-space indent", "Editor setting.", nil, nil},
		{"obs-decision-1", "decision", "Use WAL journal mode", "Why: concurrent readers.", "decision/wal", nil},
		{"obs-architecture-1", "architecture", "Storage owns SQL", "Packages: internal/storage.", "architecture/storage", nil},
		{"obs-pattern-1", "pattern", "Component pooling pattern", "How to pool entities.", nil, nil},
		{"obs-config-1", "config", "Blender export config", "Export settings.", nil, nil},
		{"obs-discovery-1", "discovery", "FTS5 quirk in shared cache", "Shared cache breaks FTS5.", nil, nil},
		{"obs-manual-1", "manual", "Manual checkpoint note", "Checkpoint reminder.", nil, nil},
		{"obs-deleted-1", "bugfix", "Deleted row", "Should not appear.", nil, "2024-06-01T00:00:00Z"},
		// Oversized content: > 64 KiB — should produce an error, not be migrated.
		{"obs-oversize-content", "bugfix", "Oversize content row",
			strings.Repeat("x", memory.MaxContentBytes+1), nil, nil},
		// Long title: > 200 runes — migrated but title truncated.
		{"obs-long-title", "bugfix", strings.Repeat("T", 201), "Short content.", nil, nil},
	}

	for _, r := range rows {
		_, err := db.Exec(ins,
			r.syncID, r.typ, r.title, r.content,
			"test-project", "project", r.topicKey,
			"hash123", 0, ts, ts,
			r.deletedAt,
		)
		if err != nil {
			t.Fatalf("insert fixture row %s: %v", r.syncID, err)
		}
	}

	return dbPath
}

// openTLDB opens a fresh Thoughtline storage at path for integration tests.
func openTLDB(t *testing.T) (*storage.Storage, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "thoughtline.db")
	ctx := context.Background()
	st, err := storage.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open tl storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st, dbPath
}

func TestMigrate_EndToEnd(t *testing.T) {
	engramPath := buildEngramFixture(t)
	_, tlPath := openTLDB(t)

	cfg := Config{
		Source:  engramPath,
		Dest:    tlPath,
		DryRun:  false,
		Verbose: true,
	}

	ctx := context.Background()
	summary, err := Run(ctx, cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 11 rows total in the fixture excluding the soft-deleted one.
	// Reader should see 11 (it returns only active rows; soft-deleted counted
	// separately in SkippedDeleted).
	// Actually: Total = rows processed by the mapper/writer loop.
	// SkippedDeleted is handled at the reader level, so those rows never enter
	// the loop. We seed 11 active + 1 deleted = 12 rows; reader returns 11.
	if summary.Total != 10 {
		t.Errorf("Total = %d, want 10", summary.Total)
	}
	// 10 created: 11 active - 1 oversized content (error).
	// The long-title row IS created (truncated), so it counts as created.
	if summary.Created != 9 {
		t.Errorf("Created = %d, want 9", summary.Created)
	}
	// 1 error: the oversized content row.
	if summary.Errors != 1 {
		t.Errorf("Errors = %d, want 1", summary.Errors)
	}
	// 1 truncation: the long-title row.
	if summary.Truncations != 1 {
		t.Errorf("Truncations = %d, want 1", summary.Truncations)
	}

	// Open destination for spot-checks.
	rawDB, err := sql.Open("sqlite", tlPath)
	if err != nil {
		t.Fatalf("open raw dest db: %v", err)
	}
	defer func() { _ = rawDB.Close() }()

	// sync_id preserved: obs-bugfix-1 must appear verbatim.
	var storedSyncID string
	if err := rawDB.QueryRowContext(ctx,
		"SELECT sync_id FROM memories WHERE sync_id = ?", "obs-bugfix-1",
	).Scan(&storedSyncID); err != nil {
		t.Fatalf("lookup obs-bugfix-1 by sync_id: %v", err)
	}
	if storedSyncID != "obs-bugfix-1" {
		t.Errorf("sync_id not preserved: got %q", storedSyncID)
	}

	// Type mapping: obs-pattern-1 must be type=convention with origin-type:pattern tag.
	var (
		storedType string
		storedTags string
	)
	if err := rawDB.QueryRowContext(ctx,
		"SELECT type, COALESCE(tags, '') FROM memories WHERE sync_id = ?", "obs-pattern-1",
	).Scan(&storedType, &storedTags); err != nil {
		t.Fatalf("lookup obs-pattern-1: %v", err)
	}
	if storedType != "convention" {
		t.Errorf("pattern row type = %q, want convention", storedType)
	}
	if !strings.Contains(storedTags, "origin-type:pattern") {
		t.Errorf("pattern row tags = %q, want to contain origin-type:pattern", storedTags)
	}

	// Soft-deleted row must be absent from destination.
	var cnt int
	if err := rawDB.QueryRowContext(ctx,
		"SELECT count(*) FROM memories WHERE sync_id = ?", "obs-deleted-1",
	).Scan(&cnt); err != nil {
		t.Fatalf("count deleted row: %v", err)
	}
	if cnt != 0 {
		t.Errorf("soft-deleted row obs-deleted-1 appears in destination (count=%d)", cnt)
	}
}

func TestMigrate_DryRun(t *testing.T) {
	engramPath := buildEngramFixture(t)
	_, tlPath := openTLDB(t)

	cfg := Config{
		Source:  engramPath,
		Dest:    tlPath,
		DryRun:  true,
		Verbose: false,
	}

	ctx := context.Background()
	summary, err := Run(ctx, cfg)
	if err != nil {
		t.Fatalf("Run dry-run: %v", err)
	}

	// No writes in dry-run mode.
	if summary.Created != 0 {
		t.Errorf("dry-run Created = %d, want 0", summary.Created)
	}

	// But rows must have been counted.
	if summary.Total == 0 {
		t.Errorf("dry-run Total = 0, expected > 0")
	}

	// Destination DB must be empty.
	rawDB, err := sql.Open("sqlite", tlPath)
	if err != nil {
		t.Fatalf("open raw dest: %v", err)
	}
	defer func() { _ = rawDB.Close() }()

	var cnt int
	if err := rawDB.QueryRowContext(context.Background(),
		"SELECT count(*) FROM memories",
	).Scan(&cnt); err != nil {
		t.Fatalf("count in dry-run dest: %v", err)
	}
	if cnt != 0 {
		t.Errorf("dry-run wrote %d rows, expected 0", cnt)
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	engramPath := buildEngramFixture(t)
	_, tlPath := openTLDB(t)

	cfg := Config{
		Source:  engramPath,
		Dest:    tlPath,
		DryRun:  false,
		Verbose: false,
	}

	ctx := context.Background()

	// First run.
	s1, err := Run(ctx, cfg)
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}
	firstCreated := s1.Created

	// Second run — same source, same dest.
	s2, err := Run(ctx, cfg)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if s2.Created != 0 {
		t.Errorf("second run Created = %d, want 0 (all already migrated)", s2.Created)
	}
	if s2.SkippedDuplicate != firstCreated {
		t.Errorf("second run SkippedDuplicate = %d, want %d (matching first run Created)",
			s2.SkippedDuplicate, firstCreated)
	}
	// The oversized content row still errors on the second run (it's never
	// inserted, so it can't be "skipped-duplicate" either).
	if s2.Errors != 1 {
		t.Errorf("second run Errors = %d, want 1 (oversized row always errors)", s2.Errors)
	}
}
