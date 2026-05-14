package dashboard

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// View renders the active tab. Layout shape:
//
//	┌── header (title + project + db path) ──────────┐
//	│ [Overview] [Browse] [Search] [Sessions] [Tags] │
//	│ ────────────────────────────────────────────── │
//	│ <active tab body>                              │
//	│ ────────────────────────────────────────────── │
//	│ <footer hints>                                 │
//	└────────────────────────────────────────────────┘
//
// View is pure with respect to the Model — Update populates state, View
// only reads it.
// minWidth and minHeight define the smallest terminal Thoughtline supports.
const (
	minWidth  = 80
	minHeight = 24
)

func (m Model) View() string {
	if m.Quitting {
		return ""
	}
	// Small terminal guard (design decision B).
	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		msg := fmt.Sprintf("Terminal too small (%dx%d). Please resize to at least %dx%d.",
			m.width, m.height, minWidth, minHeight)
		return msg + "\n"
	}
	if m.splashActive {
		return m.renderSplash()
	}
	if !m.loaded {
		return "Loading Thoughtline stats…\n"
	}

	var b strings.Builder
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n")
	if m.tab == tabOverview && m.detail == nil {
		b.WriteString(m.renderHero())
		b.WriteString("\n")
	} else {
		b.WriteString(m.renderHeader())
		b.WriteString("\n")
	}
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	if m.detail != nil {
		b.WriteString(m.renderDetail())
	} else {
		switch m.tab {
		case tabOverview:
			b.WriteString(m.renderOverview())
		case tabBrowse:
			b.WriteString(m.renderBrowse())
		case legacyTabSearch:
			b.WriteString(m.renderSearch())
		case tabSessions:
			b.WriteString(m.renderSessions())
		case tabTags:
			b.WriteString(m.renderTags())
		case legacyTabHelp:
			b.WriteString(m.renderHelp())
		}
	}

	b.WriteString("\n")
	if m.err != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("error: %v", m.err)))
		b.WriteString("\n")
	}
	b.WriteString(footerStyle.Render(m.footerHint()))
	return b.String()
}

// renderStatusBar renders the cockpit-style top line:
//
//	◆ THOUGHTLINE ONLINE · MEM 12 · SESSIONS 3 · v0.0.1
//
// When an update is available the right side gets a pill:
//
//	… · v0.0.1   ↑ v0.1.0 available · [u]
//
// One line, full width, surface bg.
func (m Model) renderStatusBar() string {
	brand := titleStyle.Render("◆")
	online := statusOnlineStyle.Render("THOUGHTLINE ONLINE")

	mem := statusMetricStyle.Render("MEM ") +
		statusValueStyle.Render(fmt.Sprintf("%d", m.stats.TotalMemories))
	sess := statusMetricStyle.Render("SESSIONS ") +
		statusValueStyle.Render(fmt.Sprintf("%d", m.stats.OpenSessions+m.stats.ClosedSessions))
	version := statusMetricStyle.Render("v" + m.cfg.Version)

	left := brand + " " + online +
		statusSepStyle.String() + mem +
		statusSepStyle.String() + sess +
		statusSepStyle.String() + version

	if m.updateAvailable && m.latestVersion != "" {
		pill := statusUpdateStyle.Render(fmt.Sprintf(" ↑ %s available ", m.latestVersion))
		hint := statusMetricStyle.Render("[u] open release")
		left += "   " + pill + statusSepStyle.String() + hint
	}

	w := m.contentWidth()
	if w < 40 {
		w = 40
	}
	return statusBarStyle.Width(w).Render(left)
}

// renderHero is the splash-style header shown on the Overview tab.
// Layout:
//
//	┌── cube ──┐  Thoughtline v0.0.1
//	│  /────/  │  ◇ Your project's memory, forever
//	│ /    /│  │  ─────────────────────────────────
//	│ ────  │  │   project: foo  ·  db: …
//	└──────────┘
//
// The cube animates via cubeTickMsg; the right column is composed of
// the brand pill, tagline, and breadcrumb meta. Joined horizontally so
// resize behaves naturally.
func (m Model) renderHero() string {
	cube := m.cube.View()

	scope := m.cfg.Project
	if scope == "" {
		scope = "(all projects)"
	}

	brand := brandPillStyle.Render(fmt.Sprintf(" Thoughtline %s ", m.cfg.Version))
	tagline := subtleStyle.Italic(true).Render("◇ Your project's memory, forever")
	meta := headerMetaStyle.Render("project: ") + headerKeyStyle.Render(scope) +
		breadcrumbSepStyle.String() +
		headerMetaStyle.Render("db: "+truncate(m.cfg.DBPath, 50))

	right := lipgloss.JoinVertical(lipgloss.Left,
		"",
		brand,
		"",
		tagline,
		"",
		meta,
	)

	if cube == "" {
		return right
	}
	gap := lipgloss.NewStyle().Width(3).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Top, cube, gap, right)
}

func (m Model) renderHeader() string {
	scope := m.cfg.Project
	if scope == "" {
		scope = "(all projects)"
	}
	// Keep the literal "Thoughtline <version>" string contiguous so existing
	// view tests can substring-match it — apply the pill style to the whole
	// chunk in one pass.
	brand := brandPillStyle.Render(fmt.Sprintf(" Thoughtline %s ", m.cfg.Version))
	sep := breadcrumbSepStyle.String()
	project := headerMetaStyle.Render("project: ") + headerKeyStyle.Render(scope)
	db := headerMetaStyle.Render("db: ") + headerMetaStyle.Render(truncate(m.cfg.DBPath, 60))
	return brand + sep + project + sep + db
}

func (m Model) renderTabs() string {
	pieces := make([]string, 0, len(allTabs))
	for i, t := range allTabs {
		label := fmt.Sprintf("%d %s", i+1, t.String())
		if t == m.tab {
			pieces = append(pieces, tabActiveStyle.Render(label))
		} else {
			pieces = append(pieces, tabInactiveStyle.Render(label))
		}
	}
	return strings.Join(pieces, tabSepStyle.String())
}

// renderOverview keeps the v1 layout (Stats | Recent activity | Roadmap)
// because it's still the at-a-glance view, and the v1 tests assert on
// substrings from this exact panel.
func (m Model) renderOverview() string {
	stats := m.renderStatsPanel()
	recent := m.renderRecentPanel()
	row := lipgloss.JoinHorizontal(lipgloss.Top, stats, recent)
	return row + "\n" + m.renderRoadmapPanel()
}

func (m Model) renderStatsPanel() string {
	stats := m.stats
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("Stats"))
	b.WriteString("\n\n")

	// Engram-style right-aligned numerals. Each row: [N right-aligned to
	// width 5] [2sp] [label]. Numerals bold in brand color, labels muted.
	statRow := func(n int, label string) {
		num := statsNumberStyle.Render(fmt.Sprintf("%5d", n))
		b.WriteString("  " + num + "  " + statsLabelStyle.Render(label) + "\n")
	}

	statRow(stats.TotalMemories, "memories")
	if stats.DeletedMemories > 0 {
		statRow(stats.DeletedMemories, "deleted")
	}
	statRow(stats.OpenSessions, "open sessions")
	statRow(stats.ClosedSessions, "closed sessions")
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

	width := m.statsPanelWidth()
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
		max := len(m.stats.RecentMemories)
		if max > 8 {
			max = 8
		}
		for _, r := range m.stats.RecentMemories[:max] {
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
		max := len(m.stats.RecentSessions)
		if max > 5 {
			max = 5
		}
		for _, s := range m.stats.RecentSessions[:max] {
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

	width := m.recentPanelWidth()
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
		case "next":
			b.WriteString(nextStyle.Render(line))
		case "in-progress":
			b.WriteString(inProgressStyle.Render(line))
		case "deferred":
			b.WriteString(deferredStyle.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	width := m.roadmapPanelWidth()
	return panelStyle.Width(width).Render(b.String())
}

func (m Model) renderBrowse() string {
	if len(m.browseItems) == 0 && !m.tabsLoaded[tabBrowse] {
		return subtleStyle.Render("Loading memories…")
	}
	if len(m.browseItems) == 0 {
		return subtleStyle.Render("(no memories yet — start saving with tl_save!)")
	}
	return m.browseList.View()
}

func (m Model) renderSearch() string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("Search"))
	b.WriteString("  ")
	b.WriteString(subtleStyle.Render("press / to type a query, enter to run"))
	b.WriteString("\n\n")
	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")
	if m.searchLoading {
		b.WriteString(subtleStyle.Render("searching…"))
		return b.String()
	}
	if m.lastQuery == "" {
		b.WriteString(subtleStyle.Render("(no query yet — press / to start)"))
		return b.String()
	}
	if len(m.resultItems) == 0 {
		b.WriteString(subtleStyle.Render(
			fmt.Sprintf("no results for %q", m.lastQuery)))
		return b.String()
	}
	b.WriteString(m.resultList.View())
	return b.String()
}

func (m Model) renderSessions() string {
	if m.sessionList.Items() == nil || len(m.sessionList.Items()) == 0 {
		return subtleStyle.Render("(no sessions yet — open one with tl_session_start)")
	}
	return m.sessionList.View()
}

func (m Model) renderTags() string {
	if !m.tabsLoaded[tabTags] {
		return subtleStyle.Render("Loading tags…")
	}
	if len(m.tagList.Items()) == 0 {
		return subtleStyle.Render("(no tags yet — pass tags=[\"foo\"] to tl_save)")
	}
	return m.tagList.View()
}

func (m Model) renderHelp() string {
	rows := [][2]string{
		{"tab / shift+tab", "cycle tabs"},
		{"1 – 6", "jump to tab"},
		{"j / k or ↑ / ↓", "move list cursor"},
		{"enter", "open detail (Browse / Search) or run query (Search)"},
		{"/", "focus search input (Search tab)"},
		{"esc", "close detail / unfocus search / quit (Overview)"},
		{"r", "refresh active tab"},
		{"t", "cycle theme (brand → zbrush → mono)"},
		{"u", "open latest release in browser (when update available)"},
		{"?", "toggle help"},
		{"q · ctrl+c", "quit"},
	}
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("Keybindings"))
	b.WriteString("\n\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "  %s  %s\n", keyStyle.Render(padRight(r[0], 18)), r[1])
	}
	b.WriteString("\n")
	b.WriteString(subtleStyle.Render(
		"Live refresh fires every 10s on the Overview tab. Other tabs cache until [r] or re-entry."))
	width := m.contentWidth()
	return panelStyle.Width(width).Render(b.String())
}

func (m Model) renderDetail() string {
	if m.detail == nil {
		return ""
	}
	if m.detailLoading {
		return subtleStyle.Render("Loading…")
	}
	return m.detailVP.View()
}

// renderDetailBody composes the body that goes inside the detail viewport.
// Kept as a free function so the unit tests can assert against the rendered
// shape without spinning up a viewport.
func renderDetailBody(env storage.SearchResult, content string) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(env.Title))
	b.WriteString("\n")
	meta := fmt.Sprintf("[%s] · %s · scope=%s · rev=%d · updated %s",
		env.Type, env.Project, env.Scope, env.RevisionCount, relativeTime(env.UpdatedAt))
	b.WriteString(subtleStyle.Render(meta))
	b.WriteString("\n")
	if env.TopicKey != "" {
		b.WriteString(subtleStyle.Render("topic_key: " + env.TopicKey))
		b.WriteString("\n")
	}
	if len(env.Tags) > 0 {
		b.WriteString(subtleStyle.Render("tags: " + strings.Join(env.Tags, ", ")))
		b.WriteString("\n")
	}
	b.WriteString(subtleStyle.Render(fmt.Sprintf("id: %d · sync_id: %s", env.ID, env.SyncID)))
	b.WriteString("\n\n")
	b.WriteString(content)
	return b.String()
}

func (m Model) footerHint() string {
	var chunks []string
	if m.detail != nil {
		chunks = []string{
			footerKeyStyle.Render(" [esc] close "),
			footerKeyStyle.Render(" [↑/↓ pgup/pgdn] scroll "),
		}
	} else if m.tab == legacyTabSearch && m.searchFocused {
		chunks = []string{
			footerKeyStyle.Render(" [enter] run query "),
			footerKeyStyle.Render(" [esc] cancel "),
		}
	} else {
		chunks = []string{
			footerKeyStyle.Render(" [tab] next "),
			footerKeyStyle.Render(" [/] search "),
			footerKeyStyle.Render(" [enter] open "),
			footerKeyStyle.Render(" [r] refresh "),
			footerKeyStyle.Render(" [?] help "),
			footerKeyStyle.Render(" [q] quit "),
		}
	}
	return strings.Join(chunks, footerSepStyle.String()) +
		footerSepStyle.String() +
		footerLabelStyle.Render("thoughtline ui")
}

// applyLayout distributes width/height across the bubbles widgets so they
// fit the current terminal size. Called on every WindowSizeMsg.
func (m *Model) applyLayout() {
	w := m.contentWidth()
	h := m.contentHeight()

	// Lists need a min height of a few rows. The chrome (header + tabs +
	// footer) eats ~6 lines in the worst case.
	listH := h - 8
	if listH < 6 {
		listH = 6
	}
	listW := w - 2
	if listW < 30 {
		listW = 30
	}
	m.browseList.SetSize(listW, listH)
	m.resultList.SetSize(listW, listH-3) // leave room for the textinput
	m.sessionList.SetSize(listW, listH)
	m.tagList.SetSize(listW, listH)

	m.searchInput.Width = listW - 4
	m.detailVP.Width = listW
	m.detailVP.Height = listH
}

// statsPanelWidth, recentPanelWidth, roadmapPanelWidth, contentWidth: width
// budget allocations. The panel widths used to be hard-coded (38/56/96);
// now they scale with the terminal so resize actually does something.
// Max 100 columns per design decision B.
func (m Model) contentWidth() int {
	if m.width <= 0 {
		return 96
	}
	w := m.width - 2
	if w > 100 {
		w = 100
	}
	return w
}

func (m Model) contentHeight() int {
	if m.height <= 0 {
		return 30
	}
	return m.height
}

func (m Model) statsPanelWidth() int {
	w := m.contentWidth()
	// Stats gets ~40% of width on Overview, with a 32-char minimum.
	v := w * 40 / 100
	if v < 32 {
		v = 32
	}
	return v
}

func (m Model) recentPanelWidth() int {
	w := m.contentWidth()
	v := w - m.statsPanelWidth() - 4
	if v < 40 {
		v = 40
	}
	return v
}

func (m Model) roadmapPanelWidth() int {
	w := m.contentWidth()
	if w < 80 {
		return 80
	}
	return w
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

// padRight moved to helpers.go as part of the tui-memory-workspace refactor.

func sortedProjectKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Compile-time check that all our exposed storage symbols are imported
// (silences linters when only some helpers are used).
var _ = storage.SnippetMaxChars
