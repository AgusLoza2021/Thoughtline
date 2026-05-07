package dashboard

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestTeatestSmoke verifies that the teatest dependency is wired and functional.
// It creates a trivial model, sends 'q', and asserts the program exits cleanly.
func TestTeatestSmoke(t *testing.T) {
	m, _ := newTestModel(t)

	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 30),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
