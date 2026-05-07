package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
)

func TestTLPendingGet_ReturnsFullPayload(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "proj-a"}

	ts := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	seedPendingEvent(t, st, "get-ev1", "get-h1", "proj-a", pending.StatusPending, ts)

	// Get the ID.
	var id int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id='get-ev1'`).Scan(&id); err != nil {
		t.Fatalf("get id: %v", err)
	}

	req := buildReq("tl_pending_get", map[string]any{"id": float64(id)})
	result, err := doTLPendingGet(ctx, st, cfg, req.GetArguments())
	if err != nil {
		t.Fatalf("doTLPendingGet: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	body := textContent(result)
	if !strings.Contains(body, "PreToolUse") {
		t.Errorf("expected event_type in response; got:\n%s", body)
	}
	if !strings.Contains(body, "hook_event_name") {
		t.Errorf("expected payload content in response; got:\n%s", body)
	}
}

func TestTLPendingGet_NotFound_ReturnsError(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "proj-a"}

	req := buildReq("tl_pending_get", map[string]any{"id": float64(9999)})
	result, err := doTLPendingGet(ctx, st, cfg, req.GetArguments())
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.IsError {
		body := textContent(result)
		t.Errorf("expected error result for missing ID; got:\n%s", body)
	}
}
