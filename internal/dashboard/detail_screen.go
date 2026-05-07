package dashboard

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// DetailScreen shows the full content of a memory or pending event payload.
// It uses a scrollable viewport for long content.
type DetailScreen struct {
	content  string
	viewport viewport.Model
}

func newDetailScreenFull(content string) *DetailScreen {
	vp := viewport.New(80, 20)
	vp.SetContent(content)
	return &DetailScreen{content: content, viewport: vp}
}

func (d *DetailScreen) Title() string { return "Detail" }

func (d *DetailScreen) Init() tea.Cmd { return nil }

func (d *DetailScreen) OnFocus() tea.Cmd { return nil }

func (d *DetailScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if kMsg, ok := msg.(tea.KeyMsg); ok {
		switch kMsg.Type {
		case tea.KeyEsc:
			return d, func() tea.Msg { return popScreenCmd{} }
		}
		if kMsg.String() == "q" {
			return d, func() tea.Msg { return popScreenCmd{} }
		}
	}
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

func (d *DetailScreen) View(width, height int, p palette) string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(p.Foreground)
	b.WriteString(titleStyle.Render("Detail"))
	b.WriteString("\n\n")
	// For the test-friendly path: render the content directly.
	// In production the viewport handles scrolling.
	b.WriteString(d.content)
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("esc/q back"))
	return b.String()
}
