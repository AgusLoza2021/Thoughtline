package dashboard

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func newInboxScreen(t *testing.T) (*InboxScreen, *storage.Storage) {
	t.Helper()
	st := newWorkspaceStorage(t)
	is := NewInboxScreen(st)
	return is, st
}

func loadInbox(t *testing.T, is *InboxScreen) *InboxScreen {
	t.Helper()
	cmd := is.OnFocus()
	if cmd == nil {
		return is
	}
	msg := cmd()
	s, _ := is.Update(msg)
	return s.(*InboxScreen)
}

func renderInbox(t *testing.T, is *InboxScreen) string {
	t.Helper()
	return is.View(100, 30, defaultPalette)
}

// ─── L1: InboxScreen lists pending rows with affordances ─────────────────────

func TestInboxScreen_L1_ListsPendingRows(t *testing.T) {
	is, st := newInboxScreen(t)
	seedPending(t, st, 3)
	is = loadInbox(t, is)

	view := renderInbox(t, is)

	// Must show the count summary
	if !strings.Contains(view, "pending") {
		t.Errorf("view must mention 'pending', got:\n%s", view)
	}
	// Must show the affordance keys
	for _, key := range []string{"[A]", "[E]", "[R]"} {
		if !strings.Contains(view, key) {
			t.Errorf("view must contain affordance %q, got:\n%s", key, view)
		}
	}
}

func TestInboxScreen_L1_EmptyState(t *testing.T) {
	is, _ := newInboxScreen(t)
	is = loadInbox(t, is)
	view := renderInbox(t, is)

	if !strings.Contains(view, "No pending") {
		t.Errorf("empty state must contain 'No pending', got:\n%s", view)
	}
}

// ─── L2: [A] accept calls MarkPromoted with unchanged payload ─────────────────

func TestInboxScreen_L2_AKeyAcceptsPassthrough(t *testing.T) {
	is, st := newInboxScreen(t)
	seedPending(t, st, 1)
	is = loadInbox(t, is)

	if len(is.events) == 0 {
		t.Fatal("expected at least 1 pending event after load")
	}

	_, cmd := is.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if cmd == nil {
		t.Fatal("[A] must produce a cmd")
	}

	// Execute the promote command
	msg := cmd()
	// The promote msg may be an internal promote action or a direct storage call.
	// Verify by checking pending count decreased.
	// Feed the msg back into the screen.
	s, _ := is.Update(msg)
	is = s.(*InboxScreen)

	// Reload to see the updated count
	is = loadInbox(t, is)
	ctx := context.Background()
	n, err := st.CountPending(ctx, "test-workspace")
	if err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if n != 0 {
		t.Errorf("after [A] accept, CountPending should be 0, got %d", n)
	}
}

// ─── L3: [E] pushes InboxEditScreen with pre-filled payload ──────────────────

func TestInboxScreen_L3_EKeyPushesEditScreen(t *testing.T) {
	is, st := newInboxScreen(t)
	seedPending(t, st, 1)
	is = loadInbox(t, is)

	_, cmd := is.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'E'}})
	if cmd == nil {
		t.Fatal("[E] must produce a cmd")
	}
	msg := cmd()
	push, ok := msg.(pushScreenCmd)
	if !ok {
		t.Fatalf("expected pushScreenCmd from [E], got %T", msg)
	}
	if push.screen == nil {
		t.Fatal("pushed screen must not be nil")
	}
	// Must be an InboxEditScreen
	if _, ok := push.screen.(*InboxEditScreen); !ok {
		t.Errorf("pushed screen must be *InboxEditScreen, got %T", push.screen)
	}
}

// ─── L4: [R] rejects and removes row from view ───────────────────────────────

func TestInboxScreen_L4_RKeyRejects(t *testing.T) {
	is, st := newInboxScreen(t)
	seedPending(t, st, 2)
	is = loadInbox(t, is)

	initialCount := len(is.events)
	if initialCount == 0 {
		t.Fatal("expected at least 1 pending event")
	}

	_, cmd := is.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	if cmd == nil {
		t.Fatal("[R] must produce a cmd")
	}

	// Execute and feed back
	msg := cmd()
	s, _ := is.Update(msg)
	is = s.(*InboxScreen)

	// Reload
	is = loadInbox(t, is)
	ctx := context.Background()
	n, err := st.CountPending(ctx, "test-workspace")
	if err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if n != initialCount-1 {
		t.Errorf("after [R] reject, CountPending should be %d, got %d", initialCount-1, n)
	}
}

// ─── L8: InboxEditScreen.InputFocused always returns true ────────────────────

func TestInboxEditScreen_L8_InputFocusedAlwaysTrue(t *testing.T) {
	ies := NewInboxEditScreen(pending.Event{
		ID:        1,
		Project:   "test-workspace",
		EventType: "PostToolUse",
		Payload: `{"proposed_type":"decision","proposed_title":"proposed title","proposed_content":"proposed body"}`,
	})
	if !ies.InputFocused() {
		t.Error("InboxEditScreen.InputFocused() must always return true")
	}
}

// ─── L5: InboxEditScreen field cycling with tab ──────────────────────────────

func TestInboxEditScreen_L5_FieldCycling(t *testing.T) {
	ies := NewInboxEditScreen(pending.Event{
		ID:        1,
		Project:   "test-workspace",
		EventType: "PostToolUse",
		Payload: `{"proposed_type":"decision","proposed_title":"proposed title","proposed_content":"proposed body"}`,
	})

	if ies.focus != 0 {
		t.Errorf("initial focus should be 0 (type field), got %d", ies.focus)
	}

	// Tab cycles: 0 → 1 → 2 → 0
	for i := 1; i <= 2; i++ {
		s, _ := ies.Update(tea.KeyMsg{Type: tea.KeyTab})
		ies = s.(*InboxEditScreen)
		if ies.focus != i {
			t.Errorf("after %d tabs, focus = %d, want %d", i, ies.focus, i)
		}
	}
	// One more tab wraps back to 0
	s, _ := ies.Update(tea.KeyMsg{Type: tea.KeyTab})
	ies = s.(*InboxEditScreen)
	if ies.focus != 0 {
		t.Errorf("focus should wrap to 0, got %d", ies.focus)
	}
}

// ─── L7: InboxEditScreen esc cancels — no DB changes ─────────────────────────

func TestInboxEditScreen_L7_EscCancels(t *testing.T) {
	st := newWorkspaceStorage(t)
	seedPending(t, st, 1)

	ctx := context.Background()
	before, err := st.CountPending(ctx, "test-workspace")
	if err != nil {
		t.Fatalf("count before: %v", err)
	}

	ies := NewInboxEditScreen(pending.Event{
		ID:        1,
		Project:   "test-workspace",
		EventType: "PostToolUse",
		Payload: `{"proposed_type":"decision","proposed_title":"proposed title","proposed_content":"proposed body"}`,
	})

	_, cmd := ies.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc must produce a pop cmd")
	}
	msg := cmd()
	if _, ok := msg.(popScreenCmd); !ok {
		t.Fatalf("esc must produce popScreenCmd, got %T", msg)
	}

	// Count must be unchanged
	after, err := st.CountPending(ctx, "test-workspace")
	if err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != before {
		t.Errorf("esc cancel must not change pending count: before=%d after=%d", before, after)
	}
}

// ─── L6: Inbox promotion fidelity ────────────────────────────────────────────
//
// L6 closes the gap surfaced by sdd-verify on the tui-memory-workspace change:
// every other Inbox test seeded pending events with the literal project
// "test-workspace" and the production paths happened to hardcode the same
// literal — so the bugs (lost ev.Project, raw EventType used as memory.Type,
// dropped SessionID/CapturedAt, hardcoded loadCmd filter) were invisible.
//
// These two tests use DISTINCT project names ("my-game-x" and "another-game")
// so any hardcoded "test-workspace" anywhere in the promotion path will leak
// into the assertions.

func TestInboxScreen_L6_AcceptPreservesProjectAndContent(t *testing.T) {
	st := newWorkspaceStorage(t)
	ctx := context.Background()

	// Seed a pending event with a project that is NOT "test-workspace".
	ev := pending.Event{
		Project:    "my-game-x",
		SessionID:  "sess-l6-accept",
		EventType:  "PostToolUse",
		Payload:    `{"proposed_type":"decision","proposed_title":"Use WAL","proposed_content":"Enable WAL via DSN"}`,
		Hash:       "test-hash-l6-accept",
		SyncID:     "sync-l6-accept",
		Status:     pending.StatusPending,
		CapturedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		CreatedAt:  time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}
	if _, err := st.InsertPending(ctx, ev); err != nil {
		t.Fatalf("InsertPending: %v", err)
	}

	// Reload to get the assigned ID via the unscoped list.
	events, err := st.ListPending(ctx, storage.ListPendingParams{Status: "pending", Limit: 50})
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 pending event after seed, got %d", len(events))
	}
	loaded := events[0]
	if loaded.Project != "my-game-x" {
		t.Fatalf("seeded pending should remain in my-game-x, got %s", loaded.Project)
	}

	// Drive Accept directly via the command (the cmd is what carries the work).
	is := NewInboxScreen(st)
	is.events = events
	is.cursor = 0
	cmd := is.acceptCmd(loaded)
	if cmd == nil {
		t.Fatal("acceptCmd returned nil")
	}
	msg := cmd()
	if action, ok := msg.(inboxActionMsg); ok && action.err != nil {
		t.Fatalf("acceptCmd: %v", action.err)
	}

	// Load saved memories from the originating project and assert every field.
	results, err := st.RecentAll(ctx, "my-game-x", 10)
	if err != nil {
		t.Fatalf("RecentAll(my-game-x): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 saved memory in my-game-x, got %d", len(results))
	}
	saved := results[0]
	if saved.Project != "my-game-x" {
		t.Errorf("Project: want my-game-x, got %s", saved.Project)
	}
	if saved.Type != "decision" {
		t.Errorf("Type: want decision, got %s", saved.Type)
	}
	if saved.Title != "Use WAL" {
		t.Errorf("Title: want 'Use WAL', got %s", saved.Title)
	}
	if !strings.Contains(saved.Snippet, "Enable WAL via DSN") {
		t.Errorf("Content snippet: want to contain 'Enable WAL via DSN', got %q", saved.Snippet)
	}

	// Nothing should have leaked into "test-workspace".
	leaked, err := st.RecentAll(ctx, "test-workspace", 10)
	if err != nil {
		t.Fatalf("RecentAll(test-workspace): %v", err)
	}
	if len(leaked) != 0 {
		t.Errorf("no memories should land in test-workspace, got %d (project leak)", len(leaked))
	}
}

func TestInboxEditScreen_L6_SubmitPreservesProjectAndSession(t *testing.T) {
	st := newWorkspaceStorage(t)
	ctx := context.Background()

	// Seed a pending event in a DIFFERENT project to defeat fixture coincidence.
	ev := pending.Event{
		Project:    "another-game",
		SessionID:  "sess-l6-edit",
		EventType:  "UserPromptSubmit",
		Payload:    `{"proposed_type":"convention","proposed_title":"original title","proposed_content":"original body"}`,
		Hash:       "test-hash-l6-edit",
		SyncID:     "sync-l6-edit",
		Status:     pending.StatusPending,
		CapturedAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		CreatedAt:  time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
	}
	if _, err := st.InsertPending(ctx, ev); err != nil {
		t.Fatalf("InsertPending: %v", err)
	}
	events, err := st.ListPending(ctx, storage.ListPendingParams{Status: "pending", Limit: 50})
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 pending event, got %d", len(events))
	}
	loaded := events[0]

	// Build the edit screen with the full event and override the fields with
	// edited values to simulate the user editing before pressing ctrl+s.
	ies := NewInboxEditScreen(loaded)
	ies.storage = st
	ies.typeField.SetValue("bugfix")
	ies.titleField.SetValue("edited title")
	ies.bodyField.SetValue("edited body")

	cmd := ies.promoteCmd()
	if cmd == nil {
		t.Fatal("promoteCmd returned nil")
	}
	msg := cmd()
	if submit, ok := msg.(inboxEditSubmitMsg); ok && submit.err != nil {
		t.Fatalf("promoteCmd: %v", submit.err)
	}

	results, err := st.RecentAll(ctx, "another-game", 10)
	if err != nil {
		t.Fatalf("RecentAll(another-game): %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 saved memory in another-game, got %d", len(results))
	}
	saved := results[0]
	if saved.Project != "another-game" {
		t.Errorf("Project: want another-game, got %s", saved.Project)
	}
	if saved.Type != "bugfix" {
		t.Errorf("Type: want bugfix, got %s", saved.Type)
	}
	if saved.Title != "edited title" {
		t.Errorf("Title: want 'edited title', got %s", saved.Title)
	}
	if !strings.Contains(saved.Snippet, "edited body") {
		t.Errorf("Content snippet: want to contain 'edited body', got %q", saved.Snippet)
	}

	leaked, err := st.RecentAll(ctx, "test-workspace", 10)
	if err != nil {
		t.Fatalf("RecentAll(test-workspace): %v", err)
	}
	if len(leaked) != 0 {
		t.Errorf("no memories should land in test-workspace, got %d (project leak)", len(leaked))
	}
}

// ─── View smoke ───────────────────────────────────────────────────────────────

func TestInboxScreen_ImplementsScreen(t *testing.T) {
	is, _ := newInboxScreen(t)
	var _ Screen = is
	if is.Title() == "" {
		t.Error("InboxScreen.Title() must not be empty")
	}
}

func TestInboxEditScreen_ImplementsScreen(t *testing.T) {
	ies := NewInboxEditScreen(pending.Event{
		ID:        1,
		Project:   "test-workspace",
		EventType: "PostToolUse",
		Payload:   `{"proposed_type":"decision","proposed_title":"title","proposed_content":"body"}`,
	})
	var _ Screen = ies
	if ies.Title() == "" {
		t.Error("InboxEditScreen.Title() must not be empty")
	}
	view := ies.View(100, 30, defaultPalette)
	if view == "" {
		t.Error("InboxEditScreen.View() must not be empty")
	}
}
