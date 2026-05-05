package migrate

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// openTestStorage opens a fresh Thoughtline *storage.Storage in t.TempDir()
// and also returns a raw *sql.DB to the same file for post-mutation queries.
func openTestStorage(t *testing.T) (*storage.Storage, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "thoughtline.db")

	ctx := context.Background()
	st, err := storage.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open test storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// Open a second handle for raw queries (the post-UPDATE path in writer.go
	// also needs this dual-handle approach).
	rawDB, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })

	return st, rawDB
}

// validMigrateMemory returns a memory.Memory suitable for writeRow tests.
func validMigrateMemory() memory.Memory {
	return memory.Memory{
		Project:   "test-project",
		Scope:     memory.ScopeProject,
		Type:      memory.TypeBugfix,
		Title:     "Test memory for migration",
		Content:   "**What**: test.\n**Why**: migration test.\n",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestWriteRow_HappyPath(t *testing.T) {
	st, rawDB := openTestStorage(t)
	ctx := context.Background()

	m := validMigrateMemory()
	engramSyncID := "obs-test-happy-path-001"

	result, err := writeRow(ctx, st, rawDB, m, engramSyncID)
	if err != nil {
		t.Fatalf("writeRow: %v", err)
	}
	if result.Action != "created" {
		t.Errorf("expected action=created, got %q", result.Action)
	}

	// The Engram sync_id must appear verbatim in the destination.
	var storedSyncID string
	err = rawDB.QueryRowContext(ctx,
		"SELECT sync_id FROM memories WHERE sync_id = ?", engramSyncID,
	).Scan(&storedSyncID)
	if err != nil {
		t.Fatalf("query for Engram sync_id %q: %v", engramSyncID, err)
	}
	if storedSyncID != engramSyncID {
		t.Errorf("stored sync_id = %q, want %q", storedSyncID, engramSyncID)
	}
}

func TestWriteRow_IdempotentDuplicateSyncID(t *testing.T) {
	st, rawDB := openTestStorage(t)
	ctx := context.Background()

	m := validMigrateMemory()
	engramSyncID := "obs-idempotent-001"

	// First write.
	r1, err := writeRow(ctx, st, rawDB, m, engramSyncID)
	if err != nil {
		t.Fatalf("first writeRow: %v", err)
	}
	if r1.Action != "created" {
		t.Fatalf("first write expected created, got %q", r1.Action)
	}

	// Second write with same sync_id — must be skipped.
	r2, err := writeRow(ctx, st, rawDB, m, engramSyncID)
	if err != nil {
		t.Fatalf("second writeRow: %v", err)
	}
	if r2.Action != "skipped-duplicate" {
		t.Errorf("second write expected skipped-duplicate, got %q", r2.Action)
	}

	// Exactly one row must exist.
	var count int
	if err := rawDB.QueryRowContext(ctx,
		"SELECT count(*) FROM memories WHERE deleted_at IS NULL",
	).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row after idempotent run, got %d", count)
	}
}

func TestWriteRow_TopicKeyCollision(t *testing.T) {
	st, rawDB := openTestStorage(t)
	ctx := context.Background()

	// Pre-insert a Thoughtline-native row with a topic_key.
	existing := validMigrateMemory()
	existing.TopicKey = "architecture/storage"
	existing.Title = "Pre-existing Thoughtline row"
	existing.Content = "This row lives in Thoughtline already."
	if _, _, err := st.Save(ctx, existing); err != nil {
		t.Fatalf("pre-insert existing row: %v", err)
	}

	// Attempt to migrate an Engram row with the same (project, topic_key).
	migrating := validMigrateMemory()
	migrating.TopicKey = "architecture/storage"
	migrating.Title = "Engram row that collides"
	migrating.Content = "Different content from Engram."
	engramSyncID := "obs-collision-001"

	result, err := writeRow(ctx, st, rawDB, migrating, engramSyncID)
	if err != nil {
		t.Fatalf("writeRow on collision: %v", err)
	}
	if result.Action != "skipped-topic-collision" {
		t.Errorf("expected skipped-topic-collision, got %q", result.Action)
	}

	// The pre-existing row must be unchanged.
	var title string
	err = rawDB.QueryRowContext(ctx,
		"SELECT title FROM memories WHERE project = ? AND topic_key = ? AND deleted_at IS NULL",
		"test-project", "architecture/storage",
	).Scan(&title)
	if err != nil {
		t.Fatalf("query pre-existing row: %v", err)
	}
	if title != "Pre-existing Thoughtline row" {
		t.Errorf("pre-existing row title changed to %q — data loss!", title)
	}
}
