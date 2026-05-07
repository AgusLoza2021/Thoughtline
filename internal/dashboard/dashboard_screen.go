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

// menuItem labels for the 5-item action menu.
var menuItems = []string{
	"Search memories",
	"Recent activity",
	"Browse projects",
	"Pending events (%d)",
	"Quit",
}

const dashMenuLen = 5

// DashboardScreen is the root screen of the Thoughtline TUI. It shows the
// ASCII logo, a 4-cell stat card, top projects, and a 5-item action menu.
type DashboardScreen struct {
	storage      *storage.Storage
	project      string
	version      string
	cursor       int
	stats        storage.Stats
	pendingCount int
	diskFreePct  int
	diskErr      error
	statsErr     error
	loaded       bool
}

// NewDashboardScreen creates a new DashboardScreen bound to the given storage.
func NewDashboardScreen(st *storage.Storage, project, version string) *DashboardScreen {
	return &DashboardScreen{
		storage: st,
		project: project,
		version: version,
	}
}

// --- Screen interface ---

func (d *DashboardScreen) Title() string { return "Dashboard" }

// Init starts the 30-second ticker and kicks off the initial data load.
func (d *DashboardScreen) Init() tea.Cmd {
	return tea.Batch(d.loadStatsCmd(), dashTickCmd())
}

// OnFocus is called when the dashboard returns to the top of the stack.
// Re-issues data-load so the view is fresh on re-entry.
func (d *DashboardScreen) OnFocus() tea.Cmd {
	return d.loadStatsCmd()
}

// Update handles key events and async data messages.
func (d *DashboardScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return d.handleKey(msg)
	case dashStatsLoadedMsg:
		d.loaded = true
		d.statsErr = msg.err
		if msg.err == nil {
			d.stats = msg.stats
			d.pendingCount = msg.pendingCount
			d.diskFreePct = msg.diskFreePct
			d.diskErr = msg.diskErr
		}
		return d, nil
	case tickMsg:
		return d, d.loadStatsCmd()
	}
	return d, nil
}

func (d *DashboardScreen) handleKey(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if d.cursor < dashMenuLen-1 {
			d.cursor++
		}
		return d, nil
	case "k", "up":
		if d.cursor > 0 {
			d.cursor--
		}
		return d, nil
	case "s":
		return d, func() tea.Msg {
			return pushScreenCmd{screen: newSearchScreen(d.storage, d.project)}
		}
	case "r":
		return d, d.loadStatsCmd()
	case "enter":
		return d.dispatchEnter()
	}
	return d, nil
}

func (d *DashboardScreen) dispatchEnter() (Screen, tea.Cmd) {
	switch d.cursor {
	case 0: // Search memories
		return d, func() tea.Msg {
			return pushScreenCmd{screen: newSearchScreen(d.storage, d.project)}
		}
	case 1: // Recent activity
		return d, func() tea.Msg {
			return pushScreenCmd{screen: newRecentScreen(d.storage, d.project, "")}
		}
	case 2: // Browse projects
		return d, func() tea.Msg {
			return pushScreenCmd{screen: newBrowseProjectsScreen(d.storage, d.stats)}
		}
	case 3: // Pending events
		return d, func() tea.Msg {
			return pushScreenCmd{screen: newPendingScreen(d.storage, d.project)}
		}
	case 4: // Quit
		return d, tea.Quit
	}
	return d, nil
}

// View renders all 7 sections of the dashboard.
func (d *DashboardScreen) View(width, height int, p palette) string {
	var b strings.Builder

	// Section 1: header card with status indicator
	b.WriteString(d.renderHeader(width, p))
	b.WriteString("\n")

	// Section 2: logo
	b.WriteString(renderLogo(p, width))
	b.WriteString("\n")

	// Section 3: tagline
	ver := d.version
	if ver == "" {
		ver = "dev"
	}
	tagline := lipgloss.NewStyle().Foreground(p.Muted).
		Render(fmt.Sprintf("> thoughtline v%s — game-dev memory that survives", ver))
	b.WriteString(tagline)
	b.WriteString("\n\n")

	// Section 4: stat card (double-border)
	b.WriteString(d.renderStatCard(p))
	b.WriteString("\n")

	// Section 5: top projects
	b.WriteString(d.renderTopProjects(p))
	b.WriteString("\n")

	// Section 6: action menu
	b.WriteString(d.renderMenu(p))
	b.WriteString("\n")

	// Section 7: footer
	footer := lipgloss.NewStyle().Foreground(p.Muted).
		Render("j/k navigate • enter select • s search • q quit")
	b.WriteString(footer)

	return b.String()
}

func (d *DashboardScreen) renderHeader(width int, p palette) string {
	statusLine := formatStatusLine(d.diskFreePct, d.diskErr, p)
	title := lipgloss.NewStyle().Bold(true).Foreground(p.Foreground).Render("THOUGHTLINE ONLINE")
	inner := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", statusLine)
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(p.Border).
		Padding(0, 1).
		Width(min(width-4, 96))
	return borderStyle.Render(inner)
}

func (d *DashboardScreen) renderStatCard(p palette) string {
	numStyle := lipgloss.NewStyle().Bold(true).Foreground(p.StatNumber)
	labelStyle := lipgloss.NewStyle().Foreground(p.Muted)

	cell := func(n int, label string) string {
		num := numStyle.Render(fmt.Sprintf("%d", n))
		return num + " " + labelStyle.Render(label)
	}

	// Show "—" on error per design decision K.
	mem := fmt.Sprintf("%d", d.stats.TotalMemories)
	sessions := fmt.Sprintf("%d", d.stats.OpenSessions+d.stats.ClosedSessions)
	projects := fmt.Sprintf("%d", len(d.stats.ByProject))
	thisWeek := "0"
	if d.statsErr != nil {
		mem, sessions, projects, thisWeek = "—", "—", "—", "—"
	}

	row1 := lipgloss.JoinHorizontal(lipgloss.Top,
		numStyle.Render(mem)+" "+labelStyle.Render("memories"),
		"    ",
		numStyle.Render(sessions)+" "+labelStyle.Render("sessions"),
	)
	row2 := lipgloss.JoinHorizontal(lipgloss.Top,
		numStyle.Render(projects)+" "+labelStyle.Render("projects"),
		"    ",
		cell(0, "this week"),
	)
	_ = thisWeek // used via row2
	inner := lipgloss.JoinVertical(lipgloss.Left, row1, row2)
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(p.Border).
		Padding(0, 1)
	return borderStyle.Render(inner)
}

func (d *DashboardScreen) renderTopProjects(p palette) string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(p.Foreground).Bold(true)
	b.WriteString(titleStyle.Render("Top Projects"))
	b.WriteString("\n")

	projects := d.stats.MostRecentProjects
	if len(projects) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  No projects yet"))
		return b.String()
	}

	max := 3
	if len(projects) < max {
		max = len(projects)
	}
	for i := 0; i < max; i++ {
		b.WriteString("  • " + projects[i] + "\n")
	}
	if len(projects) > 3 {
		b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).
			Render(fmt.Sprintf("  ...and %d more", len(projects)-3)))
	}
	return b.String()
}

func (d *DashboardScreen) renderMenu(p palette) string {
	cursor := lipgloss.NewStyle().Foreground(p.Cursor).Render("▸")
	space := "  "
	selectedBg := lipgloss.NewStyle().Background(p.MenuSelectedBg)
	normalStyle := lipgloss.NewStyle().Foreground(p.Foreground)
	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)

	var b strings.Builder
	for i := 0; i < dashMenuLen; i++ {
		label := menuItems[i]
		if i == 3 {
			label = fmt.Sprintf("Pending events (%d)", d.pendingCount)
		}

		var line string
		if i == d.cursor {
			line = cursor + " " + selectedBg.Render(label)
		} else {
			line = space + normalStyle.Render(label)
		}
		_ = mutedStyle
		b.WriteString(line + "\n")
	}
	return b.String()
}

// --- async commands ---

// dashStatsLoadedMsg carries the result of a combined stats + pending + disk load.
type dashStatsLoadedMsg struct {
	stats        storage.Stats
	pendingCount int
	diskFreePct  int
	diskErr      error
	err          error
}

// dashTickMsg is a ticker signal for the 30s auto-refresh.
type dashTickMsg time.Time

// dashTickCmd fires every 30 seconds.
func dashTickCmd() tea.Cmd {
	return tea.Every(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// loadStatsCmd issues the combined storage + disk query as a single tea.Cmd.
func (d *DashboardScreen) loadStatsCmd() tea.Cmd {
	st := d.storage
	project := d.project
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stats, statsErr := st.Stats(ctx, storage.StatsOptions{Project: project, RecentLimit: 20})
		pending, _ := st.CountPending(ctx, project)

			return dashStatsLoadedMsg{
			stats:        stats,
			pendingCount: pending,
			err:          statsErr,
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
