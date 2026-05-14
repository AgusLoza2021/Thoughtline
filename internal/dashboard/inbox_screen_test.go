package dashboard

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
	ies := NewInboxEditScreen(1, "decision", "proposed title", "proposed body")
	if !ies.InputFocused() {
		t.Error("InboxEditScreen.InputFocused() must always return true")
	}
}

// ─── L5: InboxEditScreen field cycling with tab ──────────────────────────────

func TestInboxEditScreen_L5_FieldCycling(t *testing.T) {
	ies := NewInboxEditScreen(1, "decision", "proposed title", "proposed body")

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

	ies := NewInboxEditScreen(1, "decision", "proposed title", "proposed body")

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

// ─── View smoke ───────────────────────────────────────────────────────────────

func TestInboxScreen_ImplementsScreen(t *testing.T) {
	is, _ := newInboxScreen(t)
	var _ Screen = is
	if is.Title() == "" {
		t.Error("InboxScreen.Title() must not be empty")
	}
}

func TestInboxEditScreen_ImplementsScreen(t *testing.T) {
	ies := NewInboxEditScreen(1, "decision", "title", "body")
	var _ Screen = ies
	if ies.Title() == "" {
		t.Error("InboxEditScreen.Title() must not be empty")
	}
	view := ies.View(100, 30, defaultPalette)
	if view == "" {
		t.Error("InboxEditScreen.View() must not be empty")
	}
}
