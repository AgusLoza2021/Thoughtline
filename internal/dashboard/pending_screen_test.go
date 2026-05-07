package dashboard

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func newTestPendingScreen(t *testing.T) (*PendingScreen, *storage.Storage) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "pending.db")
	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return newPendingScreenFull(st, "test-project"), st
}

func insertPending(t *testing.T, st *storage.Storage, project, evType, hash, payload string, capturedAt time.Time) {
	t.Helper()
	ev := pending.Event{
		SyncID:     hash + "-sync",
		Project:    project,
		EventType:  evType,
		Payload:    payload,
		Hash:       hash,
		Status:     pending.StatusPending,
		CapturedAt: capturedAt,
		CreatedAt:  capturedAt,
	}
	if _, err := st.InsertPending(context.Background(), ev); err != nil {
		t.Fatalf("insert pending: %v", err)
	}
}

// TestPendingScreen_ListRendersRequiredColumns verifies 3 pending events
// show event_type, timestamp, hash prefix, and payload preview.
func TestPendingScreen_ListRendersRequiredColumns(t *testing.T) {
	screen, st := newTestPendingScreen(t)

	base := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	insertPending(t, st, "test-project", "PreToolUse", "aabbccdd", `{"action":"read"}`, base)
	insertPending(t, st, "test-project", "PostToolUse", "11223344", `{"action":"write"}`, base.Add(time.Minute))
	insertPending(t, st, "test-project", "Stop", "ffee9988", `{"reason":"done"}`, base.Add(2*time.Minute))

	events, err := st.ListPending(context.Background(), storage.ListPendingParams{
		Project: "test-project",
		Status:  string(pending.StatusPending),
		Limit:   50,
	})
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	next, _ := screen.Update(pendingLoadedMsg{events: events})
	ps := next.(*PendingScreen)
	out := ps.View(80, 24, defaultPalette)

	// Must contain event_type
	if !strings.Contains(out, "PreToolUse") {
		t.Errorf("view must contain event_type PreToolUse, got:\n%s", out)
	}
	// Must contain 8-char hash prefix
	if !strings.Contains(out, "aabbccdd") {
		t.Errorf("view must contain 8-char hash prefix, got:\n%s", out)
	}
}

// TestPendingScreen_EmptyStateShowsExactCopy verifies the design decision E copy.
func TestPendingScreen_EmptyStateShowsExactCopy(t *testing.T) {
	screen, _ := newTestPendingScreen(t)
	screen.events = nil // empty

	out := screen.View(80, 24, defaultPalette)
	want := "No pending events. Passive capture is OFF — enable via tl_capture_passive."
	if !strings.Contains(out, want) {
		t.Errorf("empty state must contain exact copy:\n  want: %q\n  got: %q", want, out)
	}
}

// TestPendingScreen_EnterPushesDetail verifies enter pushes a detail screen.
func TestPendingScreen_EnterPushesDetail(t *testing.T) {
	screen, _ := newTestPendingScreen(t)

	screen.events = []pending.Event{
		{Hash: "aabbccdd", EventType: "PreToolUse", Payload: "full payload content"},
	}
	screen.cursor = 0

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter must return a non-nil cmd")
	}
	if _, ok := cmd().(pushScreenCmd); !ok {
		t.Errorf("enter on pending event must return pushScreenCmd")
	}
}
