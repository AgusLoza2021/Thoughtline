package dashboard

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func newTestSearchScreen(t *testing.T) (*SearchScreen, *storage.Storage) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "search.db")
	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return newSearchScreenFull(st, "test-project"), st
}

var enterKey = tea.KeyMsg{Type: tea.KeyEnter}
var escKey = tea.KeyMsg{Type: tea.KeyEsc}
var qKey = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}

// TestSearchScreen_EmptyQueryIsNoOp verifies that enter on empty input does nothing.
func TestSearchScreen_EmptyQueryIsNoOp(t *testing.T) {
	screen, _ := newTestSearchScreen(t)

	next, cmd := screen.Update(enterKey)
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(pushScreenCmd); ok {
			t.Errorf("empty enter must not push a screen")
		}
	}
	ss := next.(*SearchScreen)
	if len(ss.results) != 0 {
		t.Errorf("empty enter: results must remain empty, got %d", len(ss.results))
	}
}

// TestSearchScreen_NonEmptyQueryExecutesSearch verifies that enter with text
// returns a load cmd, and that after receiving results the View shows them.
func TestSearchScreen_NonEmptyQueryExecutesSearch(t *testing.T) {
	screen, _ := newTestSearchScreen(t)

	// Type into input and press enter.
	screen.input.SetValue("spawning bug")
	_, cmd := screen.Update(enterKey)
	if cmd == nil {
		t.Fatal("non-empty enter must return a search cmd")
	}

	// Simulate receiving results.
	results := []storage.SearchResult{{Title: "spawning bug fix"}}
	next, _ := screen.Update(searchResultsMsg{results: results})
	ss := next.(*SearchScreen)

	out := ss.View(80, 24, defaultPalette)
	if !strings.Contains(out, "spawning bug fix") {
		t.Errorf("view must show result titles after results received, got:\n%s", out)
	}
}

// TestSearchScreen_EscReturnsToDashboard verifies esc returns a popScreenCmd.
// TestSearchScreen_EscBlursInputInsteadOfPopping verifies the bugfix where
// esc on the Search TAB (not a stack-pushed screen) was emitting popScreenCmd
// which is a no-op against an empty stack, leaving the user trapped with the
// textinput focused. The fix: esc blurs the input so the user can navigate
// tabs with 1-6 / tab / Quick Actions afterward. A second esc (with input
// already blurred) is a no-op (still no popScreenCmd, since tabs aren't
// stack-poppable).
func TestSearchScreen_EscBlursInputInsteadOfPopping(t *testing.T) {
	screen, _ := newTestSearchScreen(t)

	// Precondition: input is focused on entry (newSearchScreenFull calls
	// ti.Focus()).
	if !screen.input.Focused() {
		t.Fatal("precondition: input should be focused on entry")
	}

	// First esc: input must blur, no popScreenCmd emitted.
	_, cmd := screen.Update(escKey)
	if screen.input.Focused() {
		t.Errorf("after first esc: input must be blurred")
	}
	if cmd != nil {
		if msg, ok := cmd().(popScreenCmd); ok {
			t.Errorf("first esc must NOT emit popScreenCmd (got %T)", msg)
		}
	}

	// Second esc (input already blurred): no-op, still no popScreenCmd.
	_, cmd2 := screen.Update(escKey)
	if screen.input.Focused() {
		t.Errorf("after second esc: input must remain blurred")
	}
	if cmd2 != nil {
		if msg, ok := cmd2().(popScreenCmd); ok {
			t.Errorf("second esc must NOT emit popScreenCmd (got %T)", msg)
		}
	}
}

// TestSearchScreen_QTypesIntoInput verifies q is a regular character inside
// the search input — it must NOT pop the screen, otherwise queries
// containing the letter q (e.g. "query") would be impossible. esc is the
// universal way to back out.
func TestSearchScreen_QTypesIntoInput(t *testing.T) {
	screen, _ := newTestSearchScreen(t)

	_, _ = screen.Update(qKey)

	if _, ok := any(screen).(*SearchScreen); !ok {
		t.Fatalf("update must return a SearchScreen, got %T", screen)
	}
	got := screen.input.Value()
	if got != "q" {
		t.Errorf("after typing q, input value = %q, want %q", got, "q")
	}
}
