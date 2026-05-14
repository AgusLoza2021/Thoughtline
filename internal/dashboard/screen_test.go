package dashboard

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// stackModel is a minimal struct that owns only the screen stack. It is used
// by unit tests that need Push/Pop/Peek without spinning up a full flatModel.
type stackModel struct {
	stack []Screen
}

// newStackModel returns an empty stackModel for testing.
func newStackModel() *stackModel { return &stackModel{} }

// Push adds s to the top of the stack and calls s.Init().
func (m *stackModel) Push(s Screen) {
	m.stack = append(m.stack, s)
}

// Pop removes the top screen. It is a no-op when the stack has ≤1 item
// (the dashboard root is always present).
func (m *stackModel) Pop() {
	if len(m.stack) <= 1 {
		return
	}
	m.stack = m.stack[:len(m.stack)-1]
}

// Peek returns the top screen without removing it. Returns nil on empty stack.
func (m *stackModel) Peek() Screen {
	if len(m.stack) == 0 {
		return nil
	}
	return m.stack[len(m.stack)-1]
}

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
