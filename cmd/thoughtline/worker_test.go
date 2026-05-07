package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func insertOldPending(t *testing.T, st *storage.Storage, syncID, hash string, capturedAt time.Time) {
	t.Helper()
	ev := pending.Event{
		SyncID:     syncID,
		Project:    "test-proj",
		SessionID:  "sess-1",
		EventType:  "PreToolUse",
		ToolName:   "Read",
		ToolUseID:  "tu-" + syncID,
		Payload:    `{"hook_event_name":"PreToolUse"}`,
		Hash:       hash,
		Status:     pending.StatusPending,
		CreatedAt:  capturedAt,
		CapturedAt: capturedAt,
	}
	if _, err := st.InsertPending(context.Background(), ev); err != nil {
		t.Fatalf("insertOldPending(%s): %v", syncID, err)
	}
}

// TestRunWorker_EmptyQueue exits 0 with no errors.
func TestRunWorker_EmptyQueue(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "worker.db")
	t.Setenv("THOUGHTLINE_DB", dbPath)

	var stderr bytes.Buffer
	if err := runWorker(context.Background(), []string{}, &stderr); err != nil {
		t.Fatalf("runWorker on empty queue: %v", err)
	}
}

// TestRunWorker_ArchivesOldPendingRows verifies that pending rows older than
// the retention window are archived and promoted rows are left intact.
func TestRunWorker_ArchivesOldPendingRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "worker.db")
	t.Setenv("THOUGHTLINE_DB", dbPath)
	t.Setenv("THOUGHTLINE_PROJECT", "test-proj")

	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	now := time.Now()
	old := now.Add(-8 * 24 * time.Hour)
	recent := now.Add(-1 * 24 * time.Hour)

	insertOldPending(t, st, "old-row", "hash-old", old)
	insertOldPending(t, st, "recent-row", "hash-recent", recent)

	// Mark one as promoted so worker must not touch it.
	var promotedID int64
	if err := st.DB().QueryRowContext(context.Background(),
		`SELECT id FROM pending_events WHERE sync_id='old-row'`).Scan(&promotedID); err != nil {
		t.Logf("old-row not found for promoted test, skipping promoted check")
	}

	// Insert a separate promoted row.
	insertOldPending(t, st, "promoted-row", "hash-promoted", old)
	var pid int64
	if err := st.DB().QueryRowContext(context.Background(),
		`SELECT id FROM pending_events WHERE sync_id='promoted-row'`).Scan(&pid); err != nil {
		t.Fatalf("get promoted id: %v", err)
	}
	if err := st.MarkPromoted(context.Background(), pid, 999); err != nil {
		t.Fatalf("mark promoted: %v", err)
	}

	_ = st.Close()

	var stderr bytes.Buffer
	if err := runWorker(context.Background(), []string{}, &stderr); err != nil {
		t.Fatalf("runWorker: %v", err)
	}

	// Re-open to inspect.
	st2, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()

	ctx := context.Background()
	rows, err := st2.DB().QueryContext(ctx, `SELECT sync_id, status FROM pending_events`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	statuses := map[string]string{}
	for rows.Next() {
		var s, st string
		if err := rows.Scan(&s, &st); err != nil {
			t.Fatalf("scan: %v", err)
		}
		statuses[s] = st
	}

	if statuses["old-row"] != "archived" {
		t.Errorf("old-row: want archived, got %q", statuses["old-row"])
	}
	if statuses["recent-row"] != "pending" {
		t.Errorf("recent-row: want pending, got %q", statuses["recent-row"])
	}
	if statuses["promoted-row"] != "promoted" {
		t.Errorf("promoted-row: want promoted, got %q", statuses["promoted-row"])
	}
}
