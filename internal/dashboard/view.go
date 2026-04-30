package dashboard

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// Lipgloss styles. Colors picked for readability on both dark and light
// terminal themes — staying within the 16-color ANSI safe set so we don't
// surprise users with truecolor-only schemes.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")) // bright blue

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // bright black / gray

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11")) // bright yellow

	doneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")) // bright green

	deferredStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")) // bright red

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginTop(1)
)

// View renders the dashboard. The layout is:
//
//	┌── header ───────────────────────────┐
//	│ Stats                │ Recent       │
//	│                      │ Activity     │
//	├──────────────────────┴──────────────┤
//	│ Roadmap                              │
//	└──── footer (key bindings) ──────────┘
//
// View() is pure with respect to the Model — it never reads the database.
// All data was already loaded into m.stats by Update via statsLoadedMsg.
func (m Model) View() string {
	if m.Quitting {
		return ""
	}
	if !m.loaded {
		return "Loading Thoughtline stats…\n"
	}

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("Error loading stats: %v", m.err)))
		b.WriteString("\n")
		b.WriteString(footerStyle.Render(footerHint()))
		return b.String()
	}

	stats := m.renderStatsPanel()
	recent := m.renderRecentPanel()

	// Place stats and recent side-by-side. Lipgloss handles different
	// heights gracefully via JoinHorizontal.
	row := lipgloss.JoinHorizontal(lipgloss.Top, stats, recent)
	b.WriteString(row)
	b.WriteString("\n")
	b.WriteString(m.renderRoadmapPanel())
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(footerHint()))
	return b.String()
}

func (m Model) renderHeader() string {
	scope := m.cfg.Project
	if scope == "" {
		scope = "(all projects)"
	}
	title := titleStyle.Render(fmt.Sprintf("Thoughtline %s", m.cfg.Version))
	meta := subtleStyle.Render(fmt.Sprintf("project: %s · db: %s", scope, m.cfg.DBPath))
	return title + "\n" + meta
}

func (m Model) renderStatsPanel() string {
	stats := m.stats
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("Stats"))
	b.WriteString("\n\n")

	fmt.Fprintf(&b, "Memories: %d active", stats.TotalMemories)
	if stats.DeletedMemories > 0 {
		fmt.Fprintf(&b, " (%d deleted)", stats.DeletedMemories)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "Sessions: %d open · %d closed\n",
		stats.OpenSessions, stats.ClosedSessions)
	b.WriteString("\n")

	if len(stats.ByType) > 0 {
		b.WriteString(subtleStyle.Render("by type"))
		b.WriteString("\n")
		for _, t := range memory.AllTypes() {
			n, ok := stats.ByType[t]
			if !ok || n == 0 {
				continue
			}
			fmt.Fprintf(&b, "  %-22s %3d\n", string(t), n)
		}
		b.WriteString("\n")
	}

	if m.cfg.Project == "" && len(stats.ByProject) > 0 {
		b.WriteString(subtleStyle.Render("by project"))
		b.WriteString("\n")
		projects := sortedProjectKeys(stats.ByProject)
		for _, p := range projects {
			fmt.Fprintf(&b, "  %-22s %3d\n", p, stats.ByProject[p])
		}
	}

	width := 38
	return panelStyle.Width(width).Render(b.String())
}

func (m Model) renderRecentPanel() string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("Recent activity"))
	b.WriteString("\n\n")

	if len(m.stats.RecentMemories) == 0 && len(m.stats.RecentSessions) == 0 {
		b.WriteString(subtleStyle.Render("(no memories or sessions yet — start saving!)"))
	}

	if len(m.stats.RecentMemories) > 0 {
		b.WriteString(subtleStyle.Render("memories"))
		b.WriteString("\n")
		for _, r := range m.stats.RecentMemories {
			fmt.Fprintf(&b, "  · %s\n", truncate(r.Title, 50))
			suffix := fmt.Sprintf("    [%s] %s", r.Type, r.Project)
			if r.TopicKey != "" {
				suffix += " · " + r.TopicKey
			}
			b.WriteString(subtleStyle.Render(suffix))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(m.stats.RecentSessions) > 0 {
		b.WriteString(subtleStyle.Render("sessions"))
		b.WriteString("\n")
		for _, s := range m.stats.RecentSessions {
			state := "open"
			if s.EndedAt != nil {
				state = "closed"
			}
			label := s.AgentLabel
			if label == "" {
				label = "—"
			}
			fmt.Fprintf(&b, "  · %s · %s\n", state, label)
			fmt.Fprintf(&b, "    %s\n", subtleStyle.Render(s.ID))
		}
	}

	width := 56
	return panelStyle.Width(width).Render(b.String())
}

func (m Model) renderRoadmapPanel() string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("Roadmap"))
	b.WriteString("\n\n")

	for _, ms := range Roadmap() {
		glyph := StatusGlyph(ms.Status)
		line := fmt.Sprintf("  %s %s — %s", glyph, ms.ID, ms.Name)
		switch ms.Status {
		case "done":
			b.WriteString(doneStyle.Render(line))
		case "deferred":
			b.WriteString(deferredStyle.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	width := panelStyle.GetWidth()
	if width <= 0 {
		width = 96
	}
	return panelStyle.Width(96).Render(b.String())
}

func footerHint() string {
	return "[r] refresh   [q] quit   ·   thoughtline ui"
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

func sortedProjectKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Compile-time check that all our exposed storage symbols are imported (silences linters in single-package builds).
var _ = storage.SnippetMaxChars
