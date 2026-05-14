package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// clipboardResultMsg is sent when a CopyToClipboard operation completes.
// err is nil on success, non-nil on failure.
type clipboardResultMsg struct {
	content string
	err     error
}

// copyClipboardCmd issues a CopyToClipboard call as a tea.Cmd.
func copyClipboardCmd(content string) tea.Cmd {
	return func() tea.Msg {
		err := CopyToClipboard(content)
		return clipboardResultMsg{content: content, err: err}
	}
}

// DetailScreen shows the full content of a memory or pending event payload.
// It uses a scrollable viewport for long content.
//
// Changes vs prior version:
//   - Read-only contract: [E] and [D] keys have no effect (Req 18).
//   - [C] copies content to OS clipboard via copyClipboardCmd (Req 19).
//   - Footer shows clipboard status (success = green, failure = yellow).
//   - Content pre-wrapped via wrap() before viewport.SetContent (Req 25).
//   - ↑/↓ scroll one line; PgUp/PgDn scroll half-viewport.
//   - Footer shows scroll percentage (Req 25).
//   - origin tabKey + OriginatingTab() accessor (design C).
type DetailScreen struct {
	content  string
	viewport viewport.Model

	// Clipboard status — set by clipboardResultMsg.
	clipStatus string
	clipOK     bool

	// origin is the tab that pushed this DetailScreen. On esc-pop,
	// flatModel reads OriginatingTab() and restores the correct tab.
	origin tabKey
}

// newDetailScreenFull creates a DetailScreen with a viewport-sized at 80×20
// as the default. The viewport will be resized on the first tea.WindowSizeMsg.
func newDetailScreenFull(content string) *DetailScreen {
	vp := viewport.New(80, 20)
	vp.SetContent(wrap(content, 80))
	return &DetailScreen{
		content:  content,
		viewport: vp,
	}
}

// ─── originator interface ─────────────────────────────────────────────────────

// OriginatingTab returns the tab that launched this DetailScreen. Used by
// flatModel to restore the correct tab on esc-pop.
func (d *DetailScreen) OriginatingTab() tabKey { return d.origin }

// ─── Screen interface ─────────────────────────────────────────────────────────

func (d *DetailScreen) Title() string { return "Detail" }

func (d *DetailScreen) Init() tea.Cmd { return nil }

func (d *DetailScreen) OnFocus() tea.Cmd { return nil }

func (d *DetailScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		// Resize viewport and re-wrap content.
		vpHeight := msg.Height - 6 // leave room for header + footer rows
		if vpHeight < 4 {
			vpHeight = 4
		}
		d.viewport = viewport.New(msg.Width-4, vpHeight)
		d.viewport.SetContent(wrap(d.content, msg.Width-4))
		return d, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return d, func() tea.Msg { return popScreenCmd{} }

		case tea.KeyUp:
			d.viewport.LineUp(1)
			return d, nil

		case tea.KeyDown:
			d.viewport.LineDown(1)
			return d, nil

		case tea.KeyPgUp:
			d.viewport.HalfViewUp()
			return d, nil

		case tea.KeyPgDown:
			d.viewport.HalfViewDown()
			return d, nil
		}

		switch msg.String() {
		case "C": // uppercase C = copy
			return d, copyClipboardCmd(d.content)
		case "q":
			return d, func() tea.Msg { return popScreenCmd{} }
		// [E] and [D] are intentionally NO-OPS — read-only contract (Req 18).
		case "e", "d":
			return d, nil
		}

	case clipboardResultMsg:
		if msg.err == nil {
			d.clipStatus = fmt.Sprintf("Copied to clipboard (%d chars)", len(msg.content))
			d.clipOK = true
		} else {
			d.clipStatus = fmt.Sprintf("Clipboard unavailable: %v", msg.err)
			d.clipOK = false
		}
		return d, nil
	}

	// Forward all other messages (e.g., mouse events) to the viewport.
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

func (d *DetailScreen) View(width, height int, p palette) string {
	var b strings.Builder

	// ── header ────────────────────────────────────────────────────────────────
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(p.Foreground)
	b.WriteString(headerStyle.Render("Detail"))
	b.WriteString("  ")
	b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("(esc back)"))
	b.WriteString("\n\n")

	// ── content (viewport) ────────────────────────────────────────────────────
	// If the viewport was never resized, render content directly for test
	// compatibility (viewport.View() is empty before first use).
	if d.viewport.Width == 0 || d.viewport.Height == 0 {
		b.WriteString(d.content)
	} else {
		b.WriteString(d.viewport.View())
	}
	b.WriteString("\n\n")

	// ── clipboard status line ─────────────────────────────────────────────────
	if d.clipStatus != "" {
		if d.clipOK {
			b.WriteString(lipgloss.NewStyle().Foreground(p.Success).Render(d.clipStatus))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(p.Warning).Render(d.clipStatus))
		}
		b.WriteString("\n")
	}

	// ── footer ────────────────────────────────────────────────────────────────
	scrollPct := ""
	if d.viewport.Height > 0 && d.viewport.TotalLineCount() > 0 {
		pct := int(d.viewport.ScrollPercent() * 100)
		scrollPct = fmt.Sprintf(" %d%%", pct)
	}
	footer := lipgloss.NewStyle().Foreground(p.Muted).
		Render("↑↓ scroll · [C] copy · esc back" + scrollPct)
	b.WriteString(footer)

	return b.String()
}
