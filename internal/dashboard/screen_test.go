package dashboard

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// mockScreen is a minimal Screen implementation used to verify that the
// interface compiles and has the exact expected method set.
type mockScreen struct {
	title    string
	viewOut  string
	initCmd  tea.Cmd
	focusCmd tea.Cmd
}

func (m *mockScreen) Init() tea.Cmd                              { return m.initCmd }
func (m *mockScreen) Update(msg tea.Msg) (Screen, tea.Cmd)      { return m, nil }
func (m *mockScreen) View(width, height int, p palette) string   { return m.viewOut }
func (m *mockScreen) Title() string                              { return m.title }
func (m *mockScreen) OnFocus() tea.Cmd                           { return m.focusCmd }

// TestScreenInterface_Compiles verifies that mockScreen satisfies Screen,
// meaning the interface has exactly the right method signatures.
func TestScreenInterface_Compiles(t *testing.T) {
	var s Screen = &mockScreen{title: "test", viewOut: "hello"}
	if s.Title() != "test" {
		t.Errorf("Title() = %q, want %q", s.Title(), "test")
	}
	if s.View(80, 24, defaultPalette) != "hello" {
		t.Errorf("View() = %q, want %q", s.View(80, 24, defaultPalette), "hello")
	}
}

// TestScreenStack_PushPeekPop verifies the screen stack's three operations.
func TestScreenStack_PushPeekPop(t *testing.T) {
	m := newStackModel()

	root := &mockScreen{title: "root"}
	m.Push(root)

	// Peek returns the top screen.
	if m.Peek() != root {
		t.Errorf("Peek after Push: got %v, want root", m.Peek())
	}

	child := &mockScreen{title: "child"}
	m.Push(child)

	if m.Peek() != child {
		t.Errorf("Peek after second Push: got %v, want child", m.Peek())
	}

	m.Pop()

	if m.Peek() != root {
		t.Errorf("Peek after Pop: got %v, want root", m.Peek())
	}
}

// TestScreenStack_PopOnSingleItem verifies that Pop on a one-item stack is a
// no-op (the dashboard is always present).
func TestScreenStack_PopOnSingleItem(t *testing.T) {
	m := newStackModel()
	root := &mockScreen{title: "root"}
	m.Push(root)

	m.Pop() // should be no-op

	if m.Peek() != root {
		t.Errorf("Pop on single item must be no-op; Peek=%v, want root", m.Peek())
	}
}
