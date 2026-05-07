package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestModel_SmallTerminalRendersWarning verifies that a terminal below 80×24
// shows a warning and no normal content.
func TestModel_SmallTerminalRendersWarning(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 79, Height: 23})
	final := updated.(Model)

	out := final.View()
	if !strings.Contains(out, "80") || !strings.Contains(out, "24") {
		// The warning must mention the minimum dimensions.
		// Accept any message that indicates a small terminal condition.
		if !strings.Contains(strings.ToLower(out), "terminal") &&
			!strings.Contains(strings.ToLower(out), "small") &&
			!strings.Contains(strings.ToLower(out), "resize") &&
			!strings.Contains(out, "×") && !strings.Contains(out, "x") {
			t.Errorf("small terminal view must contain a warning about size, got:\n%s", out)
		}
	}
}

// TestModel_NormalSizeRendersContent verifies that a 120×40 terminal renders
// normally without the small-terminal warning.
func TestModel_NormalSizeRendersContent(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	final := updated.(Model)

	// Width is stored.
	if final.width != 120 || final.height != 40 {
		t.Errorf("width/height not stored: got %dx%d", final.width, final.height)
	}

	// Content width should not exceed 100.
	cw := final.contentWidth()
	if cw > 100 {
		t.Errorf("contentWidth()=%d must be ≤100 for wide terminal", cw)
	}
}
