package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestDetailScreen_RendersFullContent verifies the detail screen shows the
// full injected content without truncation.
func TestDetailScreen_RendersFullContent(t *testing.T) {
	content := "This is the full memory content with all the details preserved without truncation."
	screen := newDetailScreenFull(content)

	out := screen.View(80, 24, defaultPalette)
	if !strings.Contains(out, content) {
		t.Errorf("detail view must contain full content\ngot:\n%s", out)
	}
}

// TestDetailScreen_EscReturnsPop verifies esc returns a popScreenCmd.
func TestDetailScreen_EscReturnsPop(t *testing.T) {
	screen := newDetailScreenFull("some content")

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc must return a non-nil cmd")
	}
	if _, ok := cmd().(popScreenCmd); !ok {
		t.Errorf("esc must return popScreenCmd")
	}
}

// TestDetailScreen_QReturnsPop verifies q also pops.
func TestDetailScreen_QReturnsPop(t *testing.T) {
	screen := newDetailScreenFull("some content")

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("q must return a non-nil cmd")
	}
	if _, ok := cmd().(popScreenCmd); !ok {
		t.Errorf("q must return popScreenCmd")
	}
}
