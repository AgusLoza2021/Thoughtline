package dashboard

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// M1 — empty state.
//
// Sessions tab with no sessions in storage renders a friendly empty-state
// message rather than a blank panel or a "0 sessions" header. The friendly
// copy must mention how sessions are created (via tl_session_start MCP).
func TestSessionsScreen_EmptyState(t *testing.T) {
	st := newWorkspaceStorage(t)
	s := NewSessionsScreen(st)

	// Drive Init → load → consume the resulting message synchronously.
	cmd := s.Init()
	if cmd == nil {
		t.Fatal("Init returned nil cmd")
	}
	msg := cmd()
	updated, _ := s.Update(msg)

	out := updated.View(100, 30, defaultPalette)
	if !strings.Contains(out, "No sessions yet") {
		t.Errorf("expected empty-state copy 'No sessions yet'; got:\n%s", out)
	}
	if strings.Contains(out, "0 sessions") {
		t.Errorf("empty state should NOT render a count header; got:\n%s", out)
	}
}

// M2 — list rendering with seeded sessions.
//
// Verifies:
//   - header shows N sessions
//   - both projects render
//   - open status renders ('open' word present), closed status renders too
//   - cursor moves on j/k
func TestSessionsScreen_ListAndCursor(t *testing.T) {
	st := newWorkspaceStorage(t)
	ctx := context.Background()

	// Seed: one open session and one closed session in two projects.
	sess1, err := st.StartSession(ctx, "alpha", "claude-code")
	if err != nil {
		t.Fatalf("start alpha: %v", err)
	}
	sess2, err := st.StartSession(ctx, "bravo", "")
	if err != nil {
		t.Fatalf("start bravo: %v", err)
	}
	if _, err := st.EndSession(ctx, sess2.ID, "wrap-up"); err != nil {
		t.Fatalf("end bravo: %v", err)
	}
	_ = sess1

	s := NewSessionsScreen(st)
	msg := s.Init()()
	updated, _ := s.Update(msg)
	sc := updated.(*SessionsScreen)

	out := sc.View(100, 30, defaultPalette)

	if !strings.Contains(out, "2 sessions") {
		t.Errorf("expected header '2 sessions' in:\n%s", out)
	}
	if !strings.Contains(out, "alpha") {
		t.Errorf("expected project 'alpha' in:\n%s", out)
	}
	if !strings.Contains(out, "bravo") {
		t.Errorf("expected project 'bravo' in:\n%s", out)
	}
	if !strings.Contains(out, "open") {
		t.Errorf("expected status 'open' rendered:\n%s", out)
	}
	if !strings.Contains(out, "closed") {
		t.Errorf("expected status 'closed' rendered:\n%s", out)
	}

	// Cursor starts at 0.
	if sc.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", sc.cursor)
	}
	// j advances cursor to 1.
	upd, _ := sc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	sc = upd.(*SessionsScreen)
	if sc.cursor != 1 {
		t.Errorf("after 'j', cursor = %d, want 1", sc.cursor)
	}
	// j again — clamps at len-1 (=1).
	upd, _ = sc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	sc = upd.(*SessionsScreen)
	if sc.cursor != 1 {
		t.Errorf("after second 'j' (clamp), cursor = %d, want 1", sc.cursor)
	}
	// k goes back to 0.
	upd, _ = sc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	sc = upd.(*SessionsScreen)
	if sc.cursor != 0 {
		t.Errorf("after 'k', cursor = %d, want 0", sc.cursor)
	}
}
