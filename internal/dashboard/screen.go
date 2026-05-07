package dashboard

import tea "github.com/charmbracelet/bubbletea"

// Screen is the interface every drill-in screen must implement.
// The root Model dispatches keyboard input to the top of the stack
// and calls View on the same screen each frame.
type Screen interface {
	// Init is called once when the screen is pushed onto the stack.
	// Return a tea.Cmd to kick off any async work needed before first render.
	Init() tea.Cmd

	// Update handles a single message. Return the (possibly new) Screen and
	// any follow-up command. The screen may return a pushScreenCmd or
	// popScreenCmd to navigate.
	Update(msg tea.Msg) (Screen, tea.Cmd)

	// View renders the screen into a string. width and height are the
	// available terminal dimensions; p is the active palette.
	View(width, height int, p palette) string

	// Title returns a short human-readable name shown in breadcrumbs or logs.
	Title() string

	// OnFocus is called when this screen returns to the top of the stack
	// after a child screen is popped. Use it to re-issue data-load commands
	// so the view is always fresh on re-entry.
	OnFocus() tea.Cmd
}

// pushScreenCmd wraps a Screen to signal the root Model that it should push
// the given screen onto the navigation stack.
type pushScreenCmd struct{ screen Screen }

// popScreenCmd signals the root Model to pop the top screen off the stack.
type popScreenCmd struct{}
