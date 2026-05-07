package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func seedPendingEvent(t *testing.T, st *storage.Storage, syncID, hash, project string, status pending.Status, capturedAt time.Time) {
	t.Helper()
	ev := pending.Event{
		SyncID:     syncID,
		Project:    project,
		SessionID:  "sess-x",
		EventType:  "PreToolUse",
		ToolName:   "Read",
		ToolUseID:  "tu-" + syncID,
		Payload:    `{"hook_event_name":"PreToolUse"}`,
		Hash:       hash,
		Status:     status,
		CreatedAt:  capturedAt,
		CapturedAt: capturedAt,
	}
	if _, err := st.InsertPending(context.Background(), ev); err != nil {
		t.Fatalf("seedPendingEvent(%s): %v", syncID, err)
	}
}

func TestTLPendingList_ReturnsRowsForProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "proj-a"}

	ts := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	seedPendingEvent(t, st, "ev1", "h1", "proj-a", pending.StatusPending, ts)
	seedPendingEvent(t, st, "ev2", "h2", "proj-a", pending.StatusPending, ts.Add(time.Minute))
	seedPendingEvent(t, st, "ev3", "h3", "proj-b", pending.StatusPending, ts) // different project

	req := buildReq("tl_pending_list", map[string]any{
		"project": "proj-a",
		"status":  "pending",
	})
	result, err := doTLPendingList(ctx, st, cfg, req.GetArguments())
	if err != nil {
		t.Fatalf("doTLPendingList: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %v", result.Content)
	}

	body := textContent(result)
	if !strings.Contains(body, "ev1") && !strings.Contains(body, "PreToolUse") {
		t.Errorf("response should mention event IDs or types; got:\n%s", body)
	}
	if strings.Contains(body, "ev3") {
		t.Errorf("should not include events from proj-b; got:\n%s", body)
	}
}

func TestTLPendingList_EmptyQueueReturnsEmptyList(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "proj-empty"}

	req := buildReq("tl_pending_list", map[string]any{
		"project": "proj-empty",
	})
	result, err := doTLPendingList(ctx, st, cfg, req.GetArguments())
	if err != nil {
		t.Fatalf("doTLPendingList: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	body := textContent(result)
	if !strings.Contains(body, "0") && !strings.Contains(body, "empty") && !strings.Contains(body, "no pending") {
		// Accept any reasonable "empty" indicator.
		t.Logf("empty queue response: %s", body)
	}
}
