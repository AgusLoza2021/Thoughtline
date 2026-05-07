package server

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// TestE2E_PassiveCapture_FullFlow exercises the complete happy path:
// seed a pending event (simulating what thoughtline hook would insert) →
// tl_pending_list finds it → tl_pending_get returns full payload →
// tl_promote converts it → tl_search finds the promoted memory.
func TestE2E_PassiveCapture_FullFlow(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "enchanted-inn"}

	ts := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	seedPendingEvent(t, st, "e2e-ev1", "e2e-h1", "enchanted-inn", pending.StatusPending, ts)

	// Step 1: tl_pending_list finds the event.
	listResult, err := doTLPendingList(ctx, st, cfg, map[string]any{
		"project": "enchanted-inn",
		"status":  "pending",
	})
	if err != nil || listResult.IsError {
		t.Fatalf("tl_pending_list failed: err=%v isError=%v body=%s",
			err, listResult.IsError, textContent(listResult))
	}
	listBody := textContent(listResult)
	if !strings.Contains(listBody, "PreToolUse") {
		t.Errorf("tl_pending_list should show PreToolUse event; got:\n%s", listBody)
	}

	// Get the actual row ID.
	var id int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id='e2e-ev1'`).Scan(&id); err != nil {
		t.Fatalf("get id: %v", err)
	}

	// Step 2: tl_pending_get returns full payload.
	getResult, err := doTLPendingGet(ctx, st, cfg, map[string]any{"id": float64(id)})
	if err != nil || getResult.IsError {
		t.Fatalf("tl_pending_get failed: err=%v isError=%v body=%s",
			err, getResult.IsError, textContent(getResult))
	}
	getBody := textContent(getResult)
	if !strings.Contains(getBody, "hook_event_name") {
		t.Errorf("tl_pending_get should include raw payload; got:\n%s", getBody)
	}

	// Step 3: tl_promote converts the event to a memory.
	promoteResult, err := doTLPromote(ctx, st, cfg, map[string]any{
		"items": []any{
			map[string]any{
				"pending_event_id": float64(id),
				"type":             "bugfix",
				"title":            "Fixed the cellar door lock",
				"content":          "**What**: Fixed lock.\n**Why**: Door was jammed.\n**Where**: dungeon/cellar.go",
			},
		},
	})
	if err != nil {
		t.Fatalf("tl_promote err: %v", err)
	}
	promoteBody := textContent(promoteResult)
	if !strings.Contains(promoteBody, "promoted") {
		t.Errorf("tl_promote should report promoted; got:\n%s", promoteBody)
	}

	// Step 4: tl_search finds the promoted memory.
	searchResult, err := doSearch(ctx, st, cfg, decodeSearchArgs(
		buildReq("tl_search", map[string]any{
			"query":   "cellar door lock",
			"project": "enchanted-inn",
		}),
	))
	if err != nil {
		t.Fatalf("tl_search err: %v", err)
	}
	searchBody := textContent(searchResult)
	if !strings.Contains(searchBody, "cellar door") {
		t.Errorf("promoted memory should be findable via tl_search; got:\n%s", searchBody)
	}

	// Step 5: promoted event no longer appears in pending list.
	listResult2, err := doTLPendingList(ctx, st, cfg, map[string]any{
		"project": "enchanted-inn",
		"status":  "pending",
	})
	if err != nil || listResult2.IsError {
		t.Fatalf("second tl_pending_list failed")
	}
	listBody2 := textContent(listResult2)
	// The promoted event's sync_id should not appear in the pending-filtered list.
	if strings.Contains(listBody2, "e2e-h1") {
		t.Errorf("promoted event should not appear in pending-status list; got:\n%s", listBody2)
	}
}

// TestDBFilePermissions verifies the SQLite file is not world-readable on Unix.
// On Windows, NTFS ACLs handle access control and os.Stat mode bits are always
// 0666, so the test is skipped there.
func TestDBFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not enforced on Windows (NTFS ACLs apply)")
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "perms_test.db")

	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer st.Close()

	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat db: %v", err)
	}

	mode := info.Mode().Perm()
	// World-readable bit (0o004) must NOT be set.
	if mode&0o004 != 0 {
		t.Errorf("DB file %s is world-readable (mode %04o) — privacy risk", dbPath, mode)
	}
}
