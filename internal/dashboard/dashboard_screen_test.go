package dashboard

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// newTestDashboardScreen creates a DashboardScreen bound to a fresh in-memory
// storage. Returns the screen and storage for seeding.
func newTestDashboardScreen(t *testing.T) (*DashboardScreen, *storage.Storage) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "dash.db")
	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	screen := NewDashboardScreen(st, "test-project", "testver")
	return screen, st
}

// keyMsg is a convenience helper that builds a tea.KeyMsg for a single rune.
func keyMsg(r string) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)}
}

// TestDashboardScreen_EmptyStateRendersAllSections verifies that a zero-value
// dashboard renders all 7 mandatory sections.
func TestDashboardScreen_EmptyStateRendersAllSections(t *testing.T) {
	screen, _ := newTestDashboardScreen(t)

	out := screen.View(100, 30, defaultPalette)

	// Section 2: logo rows (at least part of them)
	if !strings.Contains(out, "THOUGHTLINE") && !strings.Contains(out, "▀") && !strings.Contains(out, "_") {
		t.Errorf("view must contain logo content")
	}
	// Section 3: tagline
	if !strings.Contains(out, "thoughtline v") {
		t.Errorf("view must contain tagline with version, got:\n%s", out)
	}
	// Section 5: Top Projects empty state
	if !strings.Contains(out, "No projects yet") {
		t.Errorf("view must contain 'No projects yet' when DB empty, got:\n%s", out)
	}
	// Section 6: cursor glyph
	if !strings.Contains(out, "▸") {
		t.Errorf("view must contain ▸ cursor in action menu, got:\n%s", out)
	}
	// Section 7: footer
	if !strings.Contains(out, "j/k navigate") {
		t.Errorf("view must contain footer 'j/k navigate', got:\n%s", out)
	}
	if !strings.Contains(out, "q quit") {
		t.Errorf("view must contain footer 'q quit', got:\n%s", out)
	}
	// Stat card 0s
	if !strings.Contains(out, "0") {
		t.Errorf("view must show 0 counts in stat card, got:\n%s", out)
	}
}

// TestDashboardScreen_CursorMovesDown verifies j key increments cursor.
func TestDashboardScreen_CursorMovesDown(t *testing.T) {
	screen, _ := newTestDashboardScreen(t)

	// Cursor starts at 0.
	if screen.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", screen.cursor)
	}

	next, _ := screen.Update(keyMsg("j"))
	ds := next.(*DashboardScreen)
	if ds.cursor != 1 {
		t.Errorf("j: cursor = %d, want 1", ds.cursor)
	}
}

// TestDashboardScreen_CursorNoWrapAtBottom verifies j on last item stays.
func TestDashboardScreen_CursorNoWrapAtBottom(t *testing.T) {
	screen, _ := newTestDashboardScreen(t)
	screen.cursor = 4 // last item (Quit)

	next, _ := screen.Update(keyMsg("j"))
	ds := next.(*DashboardScreen)
	if ds.cursor != 4 {
		t.Errorf("j on last item: cursor = %d, want 4 (no wrap)", ds.cursor)
	}
}

// TestDashboardScreen_CursorMovesUp verifies k decrements cursor.
func TestDashboardScreen_CursorMovesUp(t *testing.T) {
	screen, _ := newTestDashboardScreen(t)
	screen.cursor = 3

	next, _ := screen.Update(keyMsg("k"))
	ds := next.(*DashboardScreen)
	if ds.cursor != 2 {
		t.Errorf("k: cursor = %d, want 2", ds.cursor)
	}
}

// TestDashboardScreen_CursorNoWrapAtTop verifies k on first item stays.
func TestDashboardScreen_CursorNoWrapAtTop(t *testing.T) {
	screen, _ := newTestDashboardScreen(t)
	screen.cursor = 0

	next, _ := screen.Update(keyMsg("k"))
	ds := next.(*DashboardScreen)
	if ds.cursor != 0 {
		t.Errorf("k on first item: cursor = %d, want 0 (no wrap)", ds.cursor)
	}
}

// TestDashboardScreen_SShortcutPushesSearch verifies s always pushes SearchScreen.
func TestDashboardScreen_SShortcutPushesSearch(t *testing.T) {
	screen, _ := newTestDashboardScreen(t)
	screen.cursor = 2 // Browse projects — not Search

	_, cmd := screen.Update(keyMsg("s"))
	if cmd == nil {
		t.Fatal("s shortcut must return a non-nil cmd")
	}
	msg := cmd()
	push, ok := msg.(pushScreenCmd)
	if !ok {
		t.Fatalf("s shortcut must return pushScreenCmd, got %T", msg)
	}
	if push.screen == nil {
		t.Errorf("pushed screen must be non-nil")
	}
}

// TestDashboardScreen_RRefreshReturnsCmd verifies r returns a non-nil cmd.
func TestDashboardScreen_RRefreshReturnsCmd(t *testing.T) {
	screen, _ := newTestDashboardScreen(t)
	_, cmd := screen.Update(keyMsg("r"))
	if cmd == nil {
		t.Errorf("r must return a non-nil reload cmd")
	}
}

// TestDashboardScreen_EnterDispatchesScreen verifies that enter on each menu
// item pushes the right screen or quits.
func TestDashboardScreen_EnterDispatchesScreen(t *testing.T) {
	screen, _ := newTestDashboardScreen(t)

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}

	tests := []struct {
		index     int
		wantPush  bool
		wantQuit  bool
	}{
		{0, true, false},  // Search memories
		{1, true, false},  // Recent activity
		{2, true, false},  // Browse projects
		{3, true, false},  // Pending events (always selectable)
		{4, false, true},  // Quit
	}

	for _, tt := range tests {
		screen.cursor = tt.index
		_, cmd := screen.Update(enterKey)
		if cmd == nil {
			t.Errorf("index %d: enter must return non-nil cmd", tt.index)
			continue
		}
		msg := cmd()
		if tt.wantQuit {
			// tea.Quit returns a QuitMsg
			if _, ok := msg.(tea.QuitMsg); !ok {
				t.Errorf("index %d (Quit): expected tea.QuitMsg, got %T", tt.index, msg)
			}
			continue
		}
		if tt.wantPush {
			if _, ok := msg.(pushScreenCmd); !ok {
				t.Errorf("index %d: expected pushScreenCmd, got %T", tt.index, msg)
			}
		}
	}
}

// TestDashboardScreen_InitReturnsTicker verifies Init returns a non-nil cmd
// (the 30s ticker).
func TestDashboardScreen_InitReturnsTicker(t *testing.T) {
	screen, _ := newTestDashboardScreen(t)
	cmd := screen.Init()
	if cmd == nil {
		t.Errorf("DashboardScreen.Init() must return a non-nil cmd (ticker)")
	}
}

// TestDashboardScreen_TickMsgReturnsLoadCmd verifies that a tickMsg results
// in a fresh load command being returned.
func TestDashboardScreen_TickMsgReturnsLoadCmd(t *testing.T) {
	screen, _ := newTestDashboardScreen(t)
	_, cmd := screen.Update(tickMsg{})
	if cmd == nil {
		t.Errorf("tickMsg must return a non-nil cmd")
	}
}
