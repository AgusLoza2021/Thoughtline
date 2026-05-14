package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// ─── HomeScreen ─────────────────────────────────────────────────────────────

// HomeScreen is the first tab (Tab 0) in the tui-memory-workspace layout.
// It displays:
//   - Brand header + Quick Actions row
//   - Project Health card (5 rows, global stats)
//   - Latest Memories card (interactive cursor, enter → push DetailScreen)
//   - Recent Activity card (last 5 events across 7-day window)
//
// Empty state (when totalMemories == 0): replaces Latest Memories + Recent
// Activity with the friendly first-run guide showing gamedev examples and
// CLI/MCP hints.
//
// Per Req 24 / I-bug-1: ALL stats are queried globally (no project filter),
// fixing the v0.1.0 bug where the Project Health showed 0 memories because
// the cwd basename did not match any stored project name.
type HomeScreen struct {
	storage *storage.Storage

	// data loaded from storage
	totalMemories  int
	totalSessions  int
	pendingCount   int
	dbPath         string
	diskFreeText   string
	latestMemories []storage.SearchResult
	recentActivity []recentActivityItem
	loaded         bool

	// list cursor for Latest Memories card
	cursor int
}

// recentActivityItem is an entry in the Recent Activity mixed stream.
type recentActivityItem struct {
	Kind    string // "saved" | "session" | "pending"
	Label   string // topic_key or session summary
	RelTime string
}

// homeStatsLoadedMsg carries data from the background loadHomeCmd.
type homeStatsLoadedMsg struct {
	totalMemories  int
	totalSessions  int
	pendingCount   int
	dbPath         string
	diskFreeText   string
	latestMemories []storage.SearchResult
	recentActivity []recentActivityItem
	err            error
}

// NewHomeScreen constructs an unloaded HomeScreen.
// dbPath is the filesystem path to the SQLite file — used for the Storage row
// and for disk-free calculation. It may be empty (tests / :memory: builds).
func NewHomeScreen(st *storage.Storage) *HomeScreen {
	return &HomeScreen{storage: st}
}

// NewHomeScreenWithPath constructs a HomeScreen that knows the DB file path.
func NewHomeScreenWithPath(st *storage.Storage, dbPath string) *HomeScreen {
	return &HomeScreen{storage: st, dbPath: dbPath}
}

// ─── Screen interface ────────────────────────────────────────────────────────

func (h *HomeScreen) Title() string { return "Home" }

func (h *HomeScreen) Init() tea.Cmd {
	return h.loadHomeCmd()
}

func (h *HomeScreen) OnFocus() tea.Cmd {
	return h.loadHomeCmd()
}

func (h *HomeScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return h.handleKey(msg)
	case homeStatsLoadedMsg:
		if msg.err == nil {
			h.totalMemories = msg.totalMemories
			h.totalSessions = msg.totalSessions
			h.pendingCount = msg.pendingCount
			h.dbPath = msg.dbPath
			h.diskFreeText = msg.diskFreeText
			h.latestMemories = msg.latestMemories
			h.recentActivity = msg.recentActivity
		}
		h.loaded = true
		return h, nil
	}
	return h, nil
}

func (h *HomeScreen) handleKey(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		max := len(h.latestMemories)
		if max > 0 && h.cursor < max-1 {
			h.cursor++
		}
		return h, nil
	case "k", "up":
		if h.cursor > 0 {
			h.cursor--
		}
		return h, nil
	case "r":
		return h, h.loadHomeCmd()
	case "enter":
		if len(h.latestMemories) > 0 && h.cursor < len(h.latestMemories) {
			sel := h.latestMemories[h.cursor]
			return h, func() tea.Msg {
				content := fmt.Sprintf("[%s] %s\n\n%s", sel.Type, sel.TopicKey, sel.Snippet)
				d := newDetailScreenFull(content)
				return pushScreenCmd{screen: d}
			}
		}
		return h, nil
	}
	return h, nil
}

// ─── View ────────────────────────────────────────────────────────────────────

func (h *HomeScreen) View(width, height int, p palette) string {
	var b strings.Builder

	// Header: brand + tagline
	b.WriteString(renderBrand(p))
	b.WriteString("\n\n")

	// Quick Actions row
	b.WriteString(h.renderQuickActions(p))
	b.WriteString("\n\n")

	if h.loaded && h.totalMemories == 0 {
		// Empty state
		b.WriteString(h.renderEmptyState(width, p))
	} else {
		// Two-column cards row: Project Health | Latest Memories
		b.WriteString(h.renderCardsRow(width, p))
		b.WriteString("\n")
		// Recent Activity card
		b.WriteString(h.renderRecentActivity(width, p))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).
		Render("↑↓ select · enter open · r refresh · q quit"))

	return b.String()
}

func (h *HomeScreen) renderQuickActions(p palette) string {
	muted := lipgloss.NewStyle().Foreground(p.Muted)
	brand := lipgloss.NewStyle().Foreground(p.Brand)
	return muted.Render("Quick:") + " " +
		brand.Render("[S]") + "ave  " +
		brand.Render("[/]") + "search  " +
		brand.Render("[M]") + "emories  " +
		brand.Render("[I]") + "nbox"
}

func (h *HomeScreen) renderCardsRow(width int, p palette) string {
	// Left card: Project Health (~38 cols); Right card: Latest Memories (~56 cols)
	leftWidth := 38
	if width < 80 {
		leftWidth = width / 2
	}
	rightWidth := width - leftWidth - 6 // account for borders + gap
	if rightWidth < 20 {
		rightWidth = 20
	}

	left := h.renderProjectHealth(leftWidth, p)
	right := h.renderLatestMemories(rightWidth, p)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func (h *HomeScreen) renderProjectHealth(width int, p palette) string {
	var rows []string

	numStyle := lipgloss.NewStyle().Foreground(p.Brand).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(p.Foreground)
	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)
	warnStyle := lipgloss.NewStyle().Foreground(p.Warning)

	row := func(label string, val string, warn bool) string {
		valStyle := numStyle
		if warn {
			valStyle = warnStyle
		}
		_ = labelStyle
		return fmt.Sprintf("%-12s %s", label, valStyle.Render(val))
	}

	rows = append(rows, row("Memories:", fmt.Sprintf("%d", h.totalMemories), false))
	rows = append(rows, row("Sessions:", fmt.Sprintf("%d", h.totalSessions), false))
	pendingWarn := h.pendingCount > 0
	rows = append(rows, row("Pending:", fmt.Sprintf("%d", h.pendingCount), pendingWarn))
	diskText := h.diskFreeText
	if diskText == "" {
		diskText = "—"
	}
	rows = append(rows, row("Disk free:", diskText, false))

	dbPath := h.dbPath
	if dbPath == "" {
		dbPath = "—"
	}
	pathTrunc := truncateLeft(dbPath, width-16)
	rows = append(rows, mutedStyle.Render(fmt.Sprintf("%-12s %s", "Storage:", pathTrunc)))

	body := strings.Join(rows, "\n")
	titleStyle := lipgloss.NewStyle().Foreground(p.Foreground).Bold(true)
	header := titleStyle.Render("Project Health")
	inner := header + "\n" + body

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(p.Border).
		Width(width - 2).
		Padding(0, 1)
	return cardStyle.Render(inner)
}

func (h *HomeScreen) renderLatestMemories(width int, p palette) string {
	titleStyle := lipgloss.NewStyle().Foreground(p.Foreground).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)
	brandStyle := lipgloss.NewStyle().Foreground(p.Brand)
	cursorStyle := lipgloss.NewStyle().Foreground(p.NavActive)
	focusedBg := lipgloss.NewStyle().Background(p.FocusedBg)

	var lines []string
	lines = append(lines, titleStyle.Render("Latest Memories"))

	if len(h.latestMemories) == 0 {
		lines = append(lines, mutedStyle.Render("(no memories yet)"))
	} else {
		maxRows := 5
		for i, m := range h.latestMemories {
			if i >= maxRows {
				break
			}
			prefix := "  "
			if i == h.cursor {
				prefix = cursorStyle.Render("▸") + " "
			}
			typeTag := truncate(string(m.Type), 8)
			badge := "[" + brandStyle.Render(typeTag) + "]"
			topicKey := truncate(m.TopicKey, 28)
			timeStr := mutedStyle.Render(relTime(m.UpdatedAt))
			scope := mutedStyle.Render(string(m.Scope))
			line := fmt.Sprintf("%s%s %-28s  %s  %s", prefix, badge, topicKey, timeStr, scope)
			if i == h.cursor {
				line = focusedBg.Render(line)
			}
			lines = append(lines, line)
		}
	}

	body := strings.Join(lines, "\n")
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(p.Border).
		Width(width - 2).
		Padding(0, 1)
	return cardStyle.Render(body)
}

func (h *HomeScreen) renderRecentActivity(width int, p palette) string {
	titleStyle := lipgloss.NewStyle().Foreground(p.Foreground).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)
	savedStyle := lipgloss.NewStyle().Foreground(p.Success)
	sessionStyle := lipgloss.NewStyle().Foreground(p.Nav)
	pendingStyle := lipgloss.NewStyle().Foreground(p.Warning)

	var lines []string
	lines = append(lines, titleStyle.Render("Recent Activity"))

	if len(h.recentActivity) == 0 {
		lines = append(lines, mutedStyle.Render("(no recent activity)"))
	} else {
		for _, item := range h.recentActivity {
			var kindRendered string
			switch item.Kind {
			case "saved":
				kindRendered = savedStyle.Render(fmt.Sprintf("%-7s", "saved"))
			case "session":
				kindRendered = sessionStyle.Render(fmt.Sprintf("%-7s", "session"))
			case "pending":
				kindRendered = pendingStyle.Render(fmt.Sprintf("%-7s", "pending"))
			default:
				kindRendered = mutedStyle.Render(fmt.Sprintf("%-7s", item.Kind))
			}
			label := truncate(item.Label, 40)
			line := fmt.Sprintf("%s  %-40s  %s", kindRendered, label, mutedStyle.Render(item.RelTime))
			lines = append(lines, line)
		}
	}

	body := strings.Join(lines, "\n")
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(p.Border).
		Width(width - 4).
		Padding(0, 1)
	return cardStyle.Render(body)
}

func (h *HomeScreen) renderEmptyState(width int, p palette) string {
	brandStyle := lipgloss.NewStyle().Foreground(p.Brand)
	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)

	headline := brandStyle.Render("Welcome — no memories yet")
	centered := lipgloss.PlaceHorizontal(width-4, lipgloss.Center, headline)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(centered)
	b.WriteString("\n\n")
	b.WriteString("        Thoughtline remembers the gamedev decisions you'd otherwise forget. Try:\n\n")
	b.WriteString("        " + brandStyle.Render("scene-pattern") + "   — how you wired a Unity / UE5 / PlayCanvas scene\n")
	b.WriteString("        " + brandStyle.Render("perf-gotcha") + "     — the GPU stall that took you a day to find\n")
	b.WriteString("        " + brandStyle.Render("pipeline-step") + "   — the FBX → glTF → engine import you almost wrote down\n")
	b.WriteString("\n")
	b.WriteString("        Save your first one from the CLI:\n")
	b.WriteString("          " + mutedStyle.Render(`$ tl save --type perf-gotcha --title "dx11 instancing" --content "…"`) + "\n")
	b.WriteString("\n")
	b.WriteString("        Or from Claude (MCP):\n")
	b.WriteString("          " + mutedStyle.Render("> Save this to Thoughtline as a scene-pattern") + "\n")
	b.WriteString("          " + mutedStyle.Render("  (calls tl_save automatically)") + "\n")
	b.WriteString("\n")
	b.WriteString("        Then press " + brandStyle.Render("r") + " here to refresh.\n\n")
	return b.String()
}

// ─── Data loading ────────────────────────────────────────────────────────────

func (h *HomeScreen) loadHomeCmd() tea.Cmd {
	st := h.storage
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Global stats — pass empty project so the WHERE clause is "1=1",
		// returning counts across ALL projects. This is the Req 24 / I-bug-1 fix:
		// the old DashboardScreen scoped to the cwd basename, causing 0 counts.
		stats, err := st.Stats(ctx, storage.StatsOptions{Project: "", RecentLimit: 20})
		if err != nil {
			return homeStatsLoadedMsg{err: err}
		}

		// CountPending globally: pass empty project string.
		pending, _ := st.CountPending(ctx, "")

		// Latest memories: global query, up to 7 rows.
		latest, _ := st.RecentAll(ctx, "", 7)

		// Recent activity: derive from RecentMemories in stats.
		activity := buildRecentActivity(stats)

		// DB path is stored on the screen struct (passed at construction time).
		// It may be empty for in-memory / test builds.
		dbPath := h.dbPath

		// Disk free text
		diskFreeText := formatDiskFree(dbPath)

		totalSessions := stats.OpenSessions + stats.ClosedSessions

		return homeStatsLoadedMsg{
			totalMemories:  stats.TotalMemories,
			totalSessions:  totalSessions,
			pendingCount:   pending,
			dbPath:         dbPath,
			diskFreeText:   diskFreeText,
			latestMemories: latest,
			recentActivity: activity,
		}
	}
}

// buildRecentActivity constructs the mixed stream from Stats data.
// It takes up to 5 entries from RecentMemories (labeled "saved") and
// up to 5 entries from RecentSessions (labeled "session"), merges them,
// sorts by UpdatedAt / StartedAt desc, and returns the top 5.
func buildRecentActivity(stats storage.Stats) []recentActivityItem {
	type rawItem struct {
		kind string
		label string
		t    time.Time
	}
	var items []rawItem

	for _, m := range stats.RecentMemories {
		items = append(items, rawItem{"saved", m.TopicKey, m.UpdatedAt})
	}
	for _, s := range stats.RecentSessions {
		label := s.Project + " / " + s.Summary
		items = append(items, rawItem{"session", label, s.StartedAt})
	}

	// Simple sort: bubble-sort by time desc (small N, ≤ 50 entries)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].t.After(items[i].t) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Take top 5
	limit := 5
	if len(items) < limit {
		limit = len(items)
	}
	out := make([]recentActivityItem, limit)
	for i := 0; i < limit; i++ {
		out[i] = recentActivityItem{
			Kind:    items[i].kind,
			Label:   items[i].label,
			RelTime: relTime(items[i].t),
		}
	}
	return out
}

// formatDiskFree returns a human-readable disk free percentage string for the
// directory containing the given path. On error or empty path returns "—".
func formatDiskFree(path string) string {
	if path == "" {
		return "—"
	}
	pct, err := freeDiskPct(path)
	if err != nil {
		return "—"
	}
	return fmt.Sprintf("%d%% free", pct)
}
