package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
)

func getPendingIDBySync(t *testing.T, st interface{ DB() interface{ QueryRowContext(context.Context, string, ...any) interface{ Scan(...any) error } } }, syncID string) int64 {
	t.Helper()
	// Use the storage directly via the concrete type.
	return 0
}

// TestTLPromote_SingleEventPromoted verifies the happy path: pending → memory
// created, status=promoted, promoted_memory_id set.
func TestTLPromote_SingleEventPromoted(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "enchanted-inn"}

	ts := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	seedPendingEvent(t, st, "promo-ev1", "promo-h1", "enchanted-inn", pending.StatusPending, ts)

	var id int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id='promo-ev1'`).Scan(&id); err != nil {
		t.Fatalf("get id: %v", err)
	}

	req := buildReq("tl_promote", map[string]any{
		"items": []any{
			map[string]any{
				"pending_event_id": float64(id),
				"type":             "bugfix",
				"title":            "Fixed crash on startup",
				"content":          "**What**: Fixed a nil pointer.\n**Why**: Startup crashed on first run.\n**Where**: main.go",
			},
		},
	})

	result, err := doTLPromote(ctx, st, cfg, req.GetArguments())
	if err != nil {
		t.Fatalf("doTLPromote: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", textContent(result))
	}

	body := textContent(result)
	if !strings.Contains(body, "promoted") {
		t.Errorf("expected 'promoted' in response; got:\n%s", body)
	}

	// Verify DB state: event is promoted.
	var status string
	var memID int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT status, promoted_memory_id FROM pending_events WHERE id=?`, id,
	).Scan(&status, &memID); err != nil {
		t.Fatalf("check DB: %v", err)
	}
	if status != "promoted" {
		t.Errorf("expected status=promoted, got %q", status)
	}
	if memID == 0 {
		t.Errorf("expected non-zero promoted_memory_id")
	}
}

// TestTLPromote_BatchPartialFailure verifies that a valid event is promoted
// even when another event in the batch is missing.
func TestTLPromote_BatchPartialFailure(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "enchanted-inn"}

	ts := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	seedPendingEvent(t, st, "batch-ev1", "batch-h1", "enchanted-inn", pending.StatusPending, ts)

	var id42 int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id='batch-ev1'`).Scan(&id42); err != nil {
		t.Fatalf("get id: %v", err)
	}

	req := buildReq("tl_promote", map[string]any{
		"items": []any{
			map[string]any{
				"pending_event_id": float64(id42),
				"type":             "decision",
				"title":            "Use WAL mode",
				"content":          "**What**: Enabled WAL.\n**Why**: Performance.\n**Where**: storage.go",
			},
			map[string]any{
				"pending_event_id": float64(99999), // does not exist
				"type":             "bugfix",
				"title":            "Ghost fix",
				"content":          "content",
			},
		},
	})

	result, err := doTLPromote(ctx, st, cfg, req.GetArguments())
	if err != nil {
		t.Fatalf("doTLPromote: %v", err)
	}
	// Result is not an error (partial success is still a success response).
	body := textContent(result)

	if !strings.Contains(body, "promoted") {
		t.Errorf("expected first item promoted; got:\n%s", body)
	}
	if !strings.Contains(body, "error") {
		t.Errorf("expected second item to show error; got:\n%s", body)
	}

	// Event id42 must be promoted.
	var status string
	if err := st.DB().QueryRowContext(ctx,
		`SELECT status FROM pending_events WHERE id=?`, id42,
	).Scan(&status); err != nil {
		t.Fatalf("check status: %v", err)
	}
	if status != "promoted" {
		t.Errorf("id42 should be promoted despite batch error, got %q", status)
	}
}

// TestTLPromote_AlreadyPromoted returns error without creating duplicate memory.
func TestTLPromote_AlreadyPromoted(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "enchanted-inn"}

	ts := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	seedPendingEvent(t, st, "dupe-ev1", "dupe-h1", "enchanted-inn", pending.StatusPending, ts)

	var id int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id='dupe-ev1'`).Scan(&id); err != nil {
		t.Fatalf("get id: %v", err)
	}

	item := map[string]any{
		"pending_event_id": float64(id),
		"type":             "convention",
		"title":            "Title",
		"content":          "Content here.",
	}

	// First promote — should succeed.
	req1 := buildReq("tl_promote", map[string]any{"items": []any{item}})
	if _, err := doTLPromote(ctx, st, cfg, req1.GetArguments()); err != nil {
		t.Fatalf("first promote: %v", err)
	}

	// Count memories before second call.
	var beforeCount int
	if err := st.DB().QueryRowContext(ctx, `SELECT count(*) FROM memories`).Scan(&beforeCount); err != nil {
		t.Fatalf("count before: %v", err)
	}

	// Second promote — must return error, no new memory.
	req2 := buildReq("tl_promote", map[string]any{"items": []any{item}})
	result, err := doTLPromote(ctx, st, cfg, req2.GetArguments())
	if err != nil {
		t.Fatalf("second promote: %v", err)
	}
	body := textContent(result)
	if !strings.Contains(strings.ToLower(body), "already promoted") {
		t.Errorf("expected 'already promoted' in response; got:\n%s", body)
	}

	var afterCount int
	if err := st.DB().QueryRowContext(ctx, `SELECT count(*) FROM memories`).Scan(&afterCount); err != nil {
		t.Fatalf("count after: %v", err)
	}
	if afterCount != beforeCount {
		t.Errorf("no new memory should be created on double-promote; before=%d after=%d",
			beforeCount, afterCount)
	}
}

// TestTLPromote_PromotedMemoryFindable verifies the created memory is
// searchable via tl_search.
func TestTLPromote_PromotedMemoryFindable(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "enchanted-inn"}

	ts := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	seedPendingEvent(t, st, "findable-ev1", "findable-h1", "enchanted-inn", pending.StatusPending, ts)

	var id int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id='findable-ev1'`).Scan(&id); err != nil {
		t.Fatalf("get id: %v", err)
	}

	req := buildReq("tl_promote", map[string]any{
		"items": []any{
			map[string]any{
				"pending_event_id": float64(id),
				"type":             "decision",
				"title":            "Use WAL mode for performance",
				"content":          "**What**: Enabled WAL.\n**Why**: Reduces lock contention.\n**Where**: storage.go",
			},
		},
	})
	if _, err := doTLPromote(ctx, st, cfg, req.GetArguments()); err != nil {
		t.Fatalf("promote: %v", err)
	}

	// Search for the promoted memory.
	searchReq := buildReq("tl_search", map[string]any{
		"query":   "WAL mode",
		"project": "enchanted-inn",
	})
	searchResult, err := doSearch(ctx, st, cfg, decodeSearchArgs(searchReq))
	if err != nil {
		t.Fatalf("tl_search: %v", err)
	}
	body := textContent(searchResult)
	if !strings.Contains(body, "WAL mode") {
		t.Errorf("promoted memory should be findable via tl_search; got:\n%s", body)
	}
}

// promoteResultItem is used to parse the JSON output of doTLPromote.
type promoteResultItem struct {
	PendingEventID int64  `json:"pending_event_id"`
	MemoryID       int64  `json:"memory_id,omitempty"`
	SyncID         string `json:"sync_id,omitempty"`
	Status         string `json:"status"`
	Error          string `json:"error,omitempty"`
}

func parsePromoteResults(t *testing.T, body string) []promoteResultItem {
	t.Helper()
	// Find the JSON array in the response.
	start := strings.Index(body, "[")
	end := strings.LastIndex(body, "]")
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("no JSON array found in promote response:\n%s", body)
	}
	jsonStr := body[start : end+1]
	var items []promoteResultItem
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		t.Fatalf("parse promote results JSON: %v\nbody was:\n%s", err, body)
	}
	return items
}
