package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// ─── HelpScreen ─────────────────────────────────────────────────────────────
//
// HelpScreen is tabs[TabHelp]. It renders a two-column layout:
//   - Left:  Keybindings (from the Keybindings registry in keybindings.go)
//   - Right: Roadmap (from roadmap.yaml via Roadmap())
//
// HelpScreen is purely presentational — no async data load, no key handling
// beyond the cursor-style no-ops that fall through the active-tab dispatcher.

// HelpScreen has no state of its own; it renders the package-level
// Keybindings registry and the embedded roadmap.
type HelpScreen struct{}

// NewHelpScreen constructs a HelpScreen ready to render.
func NewHelpScreen() *HelpScreen { return &HelpScreen{} }

func (h *HelpScreen) Title() string  { return "Help" }
func (h *HelpScreen) Init() tea.Cmd  { return nil }
func (h *HelpScreen) OnFocus() tea.Cmd { return nil }

func (h *HelpScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	_ = msg
	return h, nil
}

func (h *HelpScreen) View(width, height int, p palette) string {
	var b strings.Builder
	b.WriteString(renderBrand(p))
	b.WriteString("\n\n")

	// Allocate ~half-and-half for the two columns, less borders + gap.
	if width < 80 {
		width = 80
	}
	colWidth := (width - 6) / 2

	left := h.renderKeybindings(colWidth, p)
	right := h.renderRoadmap(colWidth, p)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right))

	_ = height
	return b.String()
}

func (h *HelpScreen) renderKeybindings(width int, p palette) string {
	titleStyle := lipgloss.NewStyle().Foreground(p.Foreground).Bold(true)
	groupTitleStyle := lipgloss.NewStyle().Foreground(p.Brand).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(p.Foreground).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(p.Muted)

	var lines []string
	lines = append(lines, titleStyle.Render("Keybindings"))
	lines = append(lines, "")

	for i, g := range Keybindings {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, groupTitleStyle.Render(g.Title))
		for _, kb := range g.Bindings {
			line := fmt.Sprintf("  %s  %s",
				keyStyle.Render(padRight(kb.Keys, 18)),
				descStyle.Render(kb.Desc),
			)
			lines = append(lines, line)
		}
	}

	body := strings.Join(lines, "\n")
	return paneBox(body, width, len(lines)+4, p)
}

func (h *HelpScreen) renderRoadmap(width int, p palette) string {
	titleStyle := lipgloss.NewStyle().Foreground(p.Foreground).Bold(true)
	doneSt := lipgloss.NewStyle().Foreground(p.Success)
	inProgSt := lipgloss.NewStyle().Foreground(p.Warning)
	deferredSt := lipgloss.NewStyle().Foreground(p.Muted)
	nextSt := lipgloss.NewStyle().Foreground(p.Nav).Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render("Roadmap"))
	lines = append(lines, "")

	for _, m := range Roadmap() {
		glyph := StatusGlyph(m.Status)
		var st lipgloss.Style
		switch m.Status {
		case "done":
			st = doneSt
		case "in-progress":
			st = inProgSt
		case "deferred":
			st = deferredSt
		case "next":
			st = nextSt
		default:
			st = deferredSt
		}
		line := fmt.Sprintf("  %s %s  %s",
			st.Render(glyph),
			st.Render(m.ID),
			st.Render(m.Name),
		)
		lines = append(lines, line)
	}

	body := strings.Join(lines, "\n")
	return paneBox(body, width, len(lines)+4, p)
}
