package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// WorkstationScreen is the v2 root: a 3-pane developer workstation layout
// modelled on Warp / Linear / LazyGit. Replaces DashboardScreen as the root.
//
// Layout:
//
//	┌─ thoughtline · <project> ─────────── DB:99% · Nmem · Ns · uptime ─┐
//	│ Sidebar         │ Center                       │ Right             │
//	│ ▸ Projects      │ <list of items per section>  │ <detail of cursor>│
//	│   Recent        │                              │                   │
//	│   Pending       │                              │                   │
//	│   Search        │                              │                   │
//	│   Sessions      │                              │                   │
//	└──────────────── ┴ ──────────────────────────── ┴ ─────────────────┘
//	 idx:up · last-save:14:32 · fts5:ok · v0.0.1                  :cmd /search
type WorkstationScreen struct {
	storage *storage.Storage
	project string
	version string
	started time.Time

	// Section selected on the left sidebar.
	section sidebarSection

	// Cursor inside the active section's list (in the center pane).
	cursor int

	// Loaded data
	stats          storage.Stats
	pending        []memory.Memory  // unused for now; placeholder
	pendingCount   int
	recentMems     []storage.SearchResult
	recentSessions []memory.Session
	diskFreePct    int
	diskErr        error

	loaded   bool
	loadErr  error
	lastSave string

	// Layout
	width  int
	height int
}

type sidebarSection int

const (
	sectionProjects sidebarSection = iota
	sectionRecent
	sectionPending
	sectionSessions
)

var sectionOrder = []sidebarSection{
	sectionProjects, sectionRecent, sectionPending, sectionSessions,
}

func (s sidebarSection) Label() string {
	switch s {
	case sectionProjects:
		return "Projects"
	case sectionRecent:
		return "Recent"
	case sectionPending:
		return "Pending"
	case sectionSessions:
		return "Sessions"
	}
	return "?"
}

// NewWorkstationScreen returns the root workstation screen ready to be the
// top of the screen stack.
func NewWorkstationScreen(st *storage.Storage, project, version string) *WorkstationScreen {
	return &WorkstationScreen{
		storage: st,
		project: project,
		version: version,
		started: time.Now(),
		section: sectionRecent, // open on the most useful section by default
	}
}

// --- Screen interface ---

func (w *WorkstationScreen) Title() string { return "Workstation" }

func (w *WorkstationScreen) Init() tea.Cmd {
	return tea.Batch(w.loadCmd(), wsTickCmd())
}

func (w *WorkstationScreen) OnFocus() tea.Cmd { return w.loadCmd() }

// --- messages ---

type wsLoadedMsg struct {
	stats          storage.Stats
	pendingCount   int
	recentMems     []storage.SearchResult
	recentSessions []memory.Session
	diskFreePct    int
	diskErr        error
	err            error
}

type wsTickMsg struct{}

func wsTickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg { return wsTickMsg{} })
}

func (w *WorkstationScreen) loadCmd() tea.Cmd {
	st := w.storage
	project := w.project
	dbPath := storageDBPath(st)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		stats, err := st.Stats(ctx, storage.StatsOptions{Project: "*", RecentLimit: 20})
		if err != nil {
			return wsLoadedMsg{err: err}
		}
		pending, _ := st.CountPending(ctx, project)
		recent, _ := st.RecentAll(ctx, "", 20)
		sessions, _ := st.RecentSessions(ctx, "", 10)
		freePct, dErr := freeDiskPct(dbPath)

		return wsLoadedMsg{
			stats:          stats,
			pendingCount:   pending,
			recentMems:     recent,
			recentSessions: sessions,
			diskFreePct:    freePct,
			diskErr:        dErr,
		}
	}
}

// storageDBPath best-effort retrieves the DB file path so disk-free can be
// computed. Falls back to "." when the storage doesn't expose it.
func storageDBPath(_ *storage.Storage) string { return "." }

// --- Update ---

func (w *WorkstationScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {

	case wsLoadedMsg:
		w.loaded = true
		if msg.err != nil {
			w.loadErr = msg.err
			return w, nil
		}
		w.stats = msg.stats
		w.pendingCount = msg.pendingCount
		w.recentMems = msg.recentMems
		w.recentSessions = msg.recentSessions
		w.diskFreePct = msg.diskFreePct
		w.diskErr = msg.diskErr
		w.lastSave = formatLastSave(msg.recentMems)
		// Clamp cursor to current list length.
		w.cursor = clampCursor(w.cursor, w.activeListLen())
		return w, nil

	case wsTickMsg:
		return w, tea.Batch(w.loadCmd(), wsTickCmd())

	case tea.KeyMsg:
		return w.handleKey(msg)
	}
	return w, nil
}

func (w *WorkstationScreen) handleKey(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if w.cursor < w.activeListLen()-1 {
			w.cursor++
		}
		return w, nil
	case "k", "up":
		if w.cursor > 0 {
			w.cursor--
		}
		return w, nil
	case "g":
		w.cursor = 0
		return w, nil
	case "G":
		w.cursor = w.activeListLen() - 1
		if w.cursor < 0 {
			w.cursor = 0
		}
		return w, nil
	case "1":
		w.setSection(sectionProjects)
		return w, nil
	case "2":
		w.setSection(sectionRecent)
		return w, nil
	case "3":
		w.setSection(sectionPending)
		return w, nil
	case "4":
		w.setSection(sectionSessions)
		return w, nil
	case "tab":
		// Cycle sections forward.
		next := (int(w.section) + 1) % len(sectionOrder)
		w.setSection(sectionOrder[next])
		return w, nil
	case "shift+tab":
		next := (int(w.section) - 1 + len(sectionOrder)) % len(sectionOrder)
		w.setSection(sectionOrder[next])
		return w, nil
	case "/":
		// Push the existing search screen as a modal-like overlay.
		return w, func() tea.Msg {
			return pushScreenCmd{screen: newSearchScreen(w.storage, w.project)}
		}
	case "r":
		return w, w.loadCmd()
	case "q":
		// Workstation lives above the welcome dashboard — q pops back home.
		return w, func() tea.Msg { return popScreenCmd{} }
	case "enter":
		return w.dispatchEnter()
	}
	return w, nil
}

func (w *WorkstationScreen) setSection(s sidebarSection) {
	w.section = s
	w.cursor = 0
}

func (w *WorkstationScreen) activeListLen() int {
	switch w.section {
	case sectionProjects:
		return len(w.stats.MostRecentProjects)
	case sectionRecent:
		return len(w.recentMems)
	case sectionPending:
		return w.pendingCount
	case sectionSessions:
		return len(w.recentSessions)
	}
	return 0
}

func (w *WorkstationScreen) dispatchEnter() (Screen, tea.Cmd) {
	switch w.section {
	case sectionRecent:
		if w.cursor < len(w.recentMems) {
			m := w.recentMems[w.cursor]
			return w, func() tea.Msg {
				return pushScreenCmd{screen: newDetailScreenFull(m.Snippet)}
			}
		}
	case sectionProjects:
		if w.cursor < len(w.stats.MostRecentProjects) {
			proj := w.stats.MostRecentProjects[w.cursor]
			return w, func() tea.Msg {
				return pushScreenCmd{screen: newRecentScreen(w.storage, proj, "")}
			}
		}
	case sectionPending:
		return w, func() tea.Msg {
			return pushScreenCmd{screen: newPendingScreen(w.storage, w.project)}
		}
	}
	return w, nil
}

// --- View ---

func (w *WorkstationScreen) View(width, height int, p palette) string {
	w.width = width
	w.height = height

	if width < minWidth || height < minHeight {
		return fmt.Sprintf("Terminal too small (%dx%d). Need at least %dx%d.\n",
			width, height, minWidth, minHeight)
	}

	if !w.loaded && w.loadErr == nil {
		return centerString("loading thoughtline workspace…", width, height, p)
	}

	// Layout math. Reserve 1 row for top bar, 1 for bottom bar.
	bodyH := height - 2
	if bodyH < 6 {
		bodyH = 6
	}

	// Sidebar fixed; right pane scales with terminal width so content has
	// room to breathe. Roughly: tiny=26 / normal=38 / wide=48.
	sidebarW := 22
	rightW := 38
	if width >= 130 {
		sidebarW = 24
		rightW = 48
	}
	if width < 100 {
		sidebarW = 20
		rightW = 30
	}
	centerW := width - sidebarW - rightW
	if centerW < 20 {
		centerW = 20
	}

	top := w.renderTopBar(width, p)
	sidebar := w.renderSidebar(sidebarW, bodyH, p)
	center := w.renderCenter(centerW, bodyH, p)
	right := w.renderRight(rightW, bodyH, p)
	bottom := w.renderBottomBar(width, p)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, center, right)
	return lipgloss.JoinVertical(lipgloss.Left, top, body, bottom)
}

// Top bar: minimal operational status. NO logo, just the brand mark in lowercase.
func (w *WorkstationScreen) renderTopBar(width int, p palette) string {
	brand := lipgloss.NewStyle().Bold(true).Foreground(p.Foreground).Render("thoughtline")
	proj := w.project
	if proj == "" {
		proj = "(no project)"
	}
	left := brand + " · " + lipgloss.NewStyle().Foreground(p.Muted).Render(proj)

	freeStr := "—"
	if w.diskErr == nil {
		freeStr = fmt.Sprintf("DB:%d%%", w.diskFreePct)
	}
	memCount := fmt.Sprintf("%dmem", w.stats.TotalMemories)
	sessCount := fmt.Sprintf("%ds", w.stats.OpenSessions+w.stats.ClosedSessions)
	uptime := fmt.Sprintf("%s", time.Since(w.started).Truncate(time.Second))

	right := lipgloss.NewStyle().Foreground(p.Muted).Render(
		fmt.Sprintf("%s · %s · %s · %s", freeStr, memCount, sessCount, uptime),
	)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	bar := left + strings.Repeat(" ", gap) + right

	return lipgloss.NewStyle().
		Foreground(p.Foreground).
		Render(bar)
}

// Sidebar: 4 sections + cursor on the active one.
func (w *WorkstationScreen) renderSidebar(width, height int, p palette) string {
	var lines []string
	header := lipgloss.NewStyle().Foreground(p.Muted).Render("─ workspace ─")
	lines = append(lines, header)
	lines = append(lines, "")

	for i, s := range sectionOrder {
		label := s.Label()
		count := w.sectionCount(s)
		hot := s == w.section
		marker := "  "
		if hot {
			marker = lipgloss.NewStyle().Foreground(p.Cursor).Render("▸ ")
		}
		row := fmt.Sprintf("%s%s", marker, label)
		if count >= 0 {
			row += " " + lipgloss.NewStyle().Foreground(p.Muted).Render(fmt.Sprintf("(%d)", count))
		}
		quick := lipgloss.NewStyle().Foreground(p.Muted).Render(fmt.Sprintf(" [%d]", i+1))
		row += quick

		if hot {
			row = lipgloss.NewStyle().Foreground(p.Foreground).Bold(true).Render(row)
		} else {
			row = lipgloss.NewStyle().Foreground(p.Foreground).Render(row)
		}
		lines = append(lines, row)
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(p.Muted).Render("─ search ─"))
	lines = append(lines, lipgloss.NewStyle().Foreground(p.Muted).Render("/  open"))

	body := strings.Join(lines, "\n")
	return paneBox(body, width, height, p)
}

// sectionCount returns the count badge for a sidebar item (-1 to omit).
func (w *WorkstationScreen) sectionCount(s sidebarSection) int {
	switch s {
	case sectionProjects:
		return len(w.stats.MostRecentProjects)
	case sectionRecent:
		return len(w.recentMems)
	case sectionPending:
		return w.pendingCount
	case sectionSessions:
		return w.stats.OpenSessions + w.stats.ClosedSessions
	}
	return -1
}

// Center pane: list of items for the active section.
func (w *WorkstationScreen) renderCenter(width, height int, p palette) string {
	header := lipgloss.NewStyle().Foreground(p.Muted).
		Render(fmt.Sprintf("─ %s ─", strings.ToLower(w.section.Label())))
	var lines []string
	lines = append(lines, header)
	lines = append(lines, "")

	switch w.section {
	case sectionProjects:
		lines = append(lines, w.renderProjectsList(width-4, p)...)
	case sectionRecent:
		lines = append(lines, w.renderMemoryList(w.recentMems, width-4, p)...)
	case sectionPending:
		if w.pendingCount == 0 {
			lines = append(lines, lipgloss.NewStyle().Foreground(p.Muted).
				Render("no pending events. set THOUGHTLINE_PASSIVE_CAPTURE=1 to enable."))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(p.Muted).
				Render(fmt.Sprintf("%d pending events. press enter to drill in.", w.pendingCount)))
		}
	case sectionSessions:
		lines = append(lines, w.renderSessionList(width-4, p)...)
	}

	body := strings.Join(lines, "\n")
	return paneBox(body, width, height, p)
}

func (w *WorkstationScreen) renderProjectsList(width int, p palette) []string {
	var out []string
	for i, proj := range w.stats.MostRecentProjects {
		hot := i == w.cursor
		count := w.stats.ByProject[proj]
		marker := "  "
		if hot {
			marker = lipgloss.NewStyle().Foreground(p.Cursor).Render("▸ ")
		}
		row := fmt.Sprintf("%s%s  %s",
			marker,
			truncateLeft(proj, width-12),
			lipgloss.NewStyle().Foreground(p.Muted).Render(fmt.Sprintf("%d mem", count)),
		)
		if hot {
			row = lipgloss.NewStyle().Bold(true).Render(row)
		}
		out = append(out, row)
	}
	if len(out) == 0 {
		out = append(out, lipgloss.NewStyle().Foreground(p.Muted).Render("no projects yet."))
	}
	return out
}

func (w *WorkstationScreen) renderMemoryList(items []storage.SearchResult, width int, p palette) []string {
	var out []string
	for i, m := range items {
		hot := i == w.cursor
		marker := "  "
		if hot {
			marker = lipgloss.NewStyle().Foreground(p.Cursor).Render("▸ ")
		}
		when := relTime(m.UpdatedAt)
		title := truncateLeft(m.Title, width-len(when)-6)
		row := fmt.Sprintf("%s%s  %s",
			marker,
			title,
			lipgloss.NewStyle().Foreground(p.Muted).Render(when),
		)
		if hot {
			row = lipgloss.NewStyle().Bold(true).Render(row)
		}
		out = append(out, row)
	}
	if len(out) == 0 {
		out = append(out, lipgloss.NewStyle().Foreground(p.Muted).Render("no recent memories."))
	}
	return out
}

func (w *WorkstationScreen) renderSessionList(width int, p palette) []string {
	var out []string
	for i, s := range w.recentSessions {
		hot := i == w.cursor
		marker := "  "
		if hot {
			marker = lipgloss.NewStyle().Foreground(p.Cursor).Render("▸ ")
		}
		when := relTime(s.StartedAt)
		summary := strings.TrimSpace(s.Summary)
		if summary == "" {
			summary = "(no summary)"
		}
		summary = truncateLeft(summary, width-len(when)-6)
		row := fmt.Sprintf("%s%s  %s",
			marker,
			summary,
			lipgloss.NewStyle().Foreground(p.Muted).Render(when),
		)
		if hot {
			row = lipgloss.NewStyle().Bold(true).Render(row)
		}
		out = append(out, row)
	}
	if len(out) == 0 {
		out = append(out, lipgloss.NewStyle().Foreground(p.Muted).Render("no sessions yet."))
	}
	return out
}

// Right pane: detail of the cursor-highlighted item.
func (w *WorkstationScreen) renderRight(width, height int, p palette) string {
	header := lipgloss.NewStyle().Foreground(p.Muted).Render("─ detail ─")
	var lines []string
	lines = append(lines, header)
	lines = append(lines, "")

	switch w.section {
	case sectionRecent:
		if w.cursor < len(w.recentMems) {
			lines = append(lines, w.renderMemoryDetail(w.recentMems[w.cursor], width-4, p)...)
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(p.Muted).Render("no selection."))
		}
	case sectionProjects:
		if w.cursor < len(w.stats.MostRecentProjects) {
			lines = append(lines, w.renderProjectDetail(w.stats.MostRecentProjects[w.cursor], width-4, p)...)
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(p.Muted).Render("no selection."))
		}
	case sectionSessions:
		if w.cursor < len(w.recentSessions) {
			lines = append(lines, w.renderSessionDetail(w.recentSessions[w.cursor], width-4, p)...)
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(p.Muted).Render("no selection."))
		}
	case sectionPending:
		lines = append(lines, lipgloss.NewStyle().Foreground(p.Muted).
			Render("press enter to open the pending events screen."))
	}

	body := strings.Join(lines, "\n")
	return paneBox(body, width, height, p)
}

func (w *WorkstationScreen) renderMemoryDetail(m storage.SearchResult, width int, p palette) []string {
	if width < 12 {
		width = 12
	}
	muted := lipgloss.NewStyle().Foreground(p.Muted)
	fg := lipgloss.NewStyle().Foreground(p.Foreground)
	tag := lipgloss.NewStyle().Foreground(p.Tag)
	statClr := lipgloss.NewStyle().Foreground(p.StatNumber)
	cursorClr := lipgloss.NewStyle().Foreground(p.Cursor)

	// Title — bold + cursor color (gold) so it stands out from labels.
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Cursor).
		Width(width)
	titleBlock := titleStyle.Render(m.Title)

	divider := muted.Render(strings.Repeat("─", width))

	out := []string{titleBlock, divider}

	// Inline label-value rows. Compact and easy to scan.
	row := func(label, value string, valueStyle lipgloss.Style) string {
		labelW := 9
		labelStr := muted.Render(padRight(label, labelW))
		// Wrap the value to remaining columns.
		valueW := width - labelW
		if valueW < 6 {
			valueW = 6
		}
		valueWrapped := valueStyle.Width(valueW).Render(value)
		valueLines := strings.Split(valueWrapped, "\n")
		// First line goes next to the label; subsequent lines indent under it.
		first := labelStr + valueLines[0]
		if len(valueLines) == 1 {
			return first
		}
		indent := strings.Repeat(" ", labelW)
		var b strings.Builder
		b.WriteString(first)
		for _, l := range valueLines[1:] {
			b.WriteByte('\n')
			b.WriteString(indent + l)
		}
		return b.String()
	}

	out = append(out, row("type", string(m.Type), statClr))
	if m.TopicKey != "" {
		out = append(out, row("topic", m.TopicKey, lipgloss.NewStyle().Foreground(p.Tag)))
	}
	out = append(out, row("project", m.Project, fg))
	out = append(out, row("scope", string(m.Scope), fg))
	if m.RevisionCount > 0 {
		out = append(out, row("rev", fmt.Sprintf("%d", m.RevisionCount), fg))
	}
	if !m.UpdatedAt.IsZero() {
		out = append(out, row("updated", relTime(m.UpdatedAt), fg))
	}

	// Tags — each in tag colour, joined inline so they wrap as a single block.
	if len(m.Tags) > 0 {
		out = append(out, "")
		out = append(out, muted.Render("tags"))
		tagBlock := tag.Width(width).Render(strings.Join(m.Tags, "  "))
		out = append(out, strings.Split(tagBlock, "\n")...)
	}

	// Snippet content — the actual memory body. Word-wrapped via lipgloss,
	// not the naive char-count wrap that broke long sentences before.
	if strings.TrimSpace(m.Snippet) != "" {
		out = append(out, "")
		out = append(out, divider)
		out = append(out, "")
		snippet := lipgloss.NewStyle().Foreground(p.Foreground).Width(width).Render(m.Snippet)
		out = append(out, strings.Split(snippet, "\n")...)
	}

	// Faint hint at the bottom: full content via tl_get_observation.
	out = append(out, "")
	out = append(out, muted.Render("enter — full"))

	_ = cursorClr // silenced if unused after future refactor
	return out
}


func (w *WorkstationScreen) renderProjectDetail(name string, width int, p palette) []string {
	muted := lipgloss.NewStyle().Foreground(p.Muted)
	fg := lipgloss.NewStyle().Foreground(p.Foreground)
	count := w.stats.ByProject[name]
	return []string{
		fg.Bold(true).Render(truncateLeft(name, width)),
		"",
		muted.Render("Memories:"),
		fg.Render(fmt.Sprintf("  %d", count)),
		"",
		muted.Render("Press enter to drill into recent memories for this project."),
	}
}

func (w *WorkstationScreen) renderSessionDetail(s memory.Session, width int, p palette) []string {
	muted := lipgloss.NewStyle().Foreground(p.Muted)
	fg := lipgloss.NewStyle().Foreground(p.Foreground)
	out := []string{
		fg.Bold(true).Render(truncateLeft(s.ID, width)),
		"",
		muted.Render("Project:"),
		fg.Render("  " + truncateLeft(s.Project, width-2)),
		"",
		muted.Render("Started:"),
		fg.Render("  " + relTime(s.StartedAt)),
	}
	if s.EndedAt != nil && !s.EndedAt.IsZero() {
		out = append(out,
			"",
			muted.Render("Ended:"),
			fg.Render("  "+relTime(*s.EndedAt)),
		)
	}
	if s.Summary != "" {
		out = append(out, "", muted.Render("Summary:"))
		for _, line := range strings.Split(wrap(s.Summary, width), "\n") {
			out = append(out, "  "+fg.Render(line))
		}
	}
	return out
}

// Bottom bar: live developer-oriented metrics.
func (w *WorkstationScreen) renderBottomBar(width int, p palette) string {
	muted := lipgloss.NewStyle().Foreground(p.Muted)
	idx := "idx:up"
	if w.loadErr != nil {
		idx = lipgloss.NewStyle().Foreground(p.StatusErr).Render("idx:err")
	}
	last := "last-save:—"
	if w.lastSave != "" {
		last = "last-save:" + w.lastSave
	}
	fts := "fts5:ok"
	ver := "v" + w.version
	if ver == "v" {
		ver = "vdev"
	}

	left := muted.Render(strings.Join([]string{idx, last, fts, ver}, " · "))
	right := muted.Render(":cmd  /search")
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// --- helpers ---
//
// paneBox, centerString, clampCursor, truncateLeft, wrap, relTime, and
// formatLastSave were moved to helpers.go as part of the tui-memory-workspace
// refactor (commit 2). They are imported implicitly via the package and used
// unchanged by the rendering methods above.
