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

// ─── MemoriesFilter ───────────────────────────────────────────────────────────

// MemoriesFilter holds the current filter state for the Memories tab.
// All fields default to "" which means "no filter for this column".
type MemoriesFilter struct {
	Type  string // "" = all
	Tag   string // "" = none
	Scope string // "" = all
	Sort  string // "" = "updated_desc" | "created_desc"
}

// ─── MemoriesScreen ───────────────────────────────────────────────────────────

const memoriesPerPage = 20

// memoriesLoadedMsg carries the paged result from a loadMemoriesCmd call.
type memoriesLoadedMsg struct {
	rows  []storage.SearchResult
	total int
	page  int
	err   error
}

// MemoriesScreen is tabs[TabMemories]. It shows a paginated list of memories
// with an inline filter bar at the top. The cursor position and filter state
// are preserved across tab switches (held on the struct, not re-initialised
// on OnFocus). Per design decision 5, filters DO NOT persist across process
// restarts.
type MemoriesScreen struct {
	storage *storage.Storage

	// pagination state
	rows    []storage.SearchResult
	total   int
	page    int // 0-indexed
	perPage int

	// cursor and filter state
	cursor      int
	filter      MemoriesFilter
	filterFocus int // 0=row cursor, 1=type, 2=tag, 3=scope, 4=sort
}

// NewMemoriesScreen constructs an empty MemoriesScreen.
func NewMemoriesScreen(st *storage.Storage) *MemoriesScreen {
	return &MemoriesScreen{
		storage: st,
		perPage: memoriesPerPage,
	}
}

// ─── Screen interface ─────────────────────────────────────────────────────────

func (ms *MemoriesScreen) Title() string { return "Memories" }

func (ms *MemoriesScreen) Init() tea.Cmd {
	return ms.loadCmd()
}

// OnFocus re-fetches the current page. Filter and cursor are preserved.
func (ms *MemoriesScreen) OnFocus() tea.Cmd {
	return ms.loadCmd()
}

// InputFocused returns true when the tag filter input is actively focused
// (filterFocus == 2, the Tag slot). This prevents global hotkeys from
// firing while the user is typing a tag filter.
func (ms *MemoriesScreen) InputFocused() bool {
	return ms.filterFocus == 2
}

func (ms *MemoriesScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return ms.handleKey(msg)
	case memoriesLoadedMsg:
		if msg.err == nil && msg.page == ms.page {
			ms.rows = msg.rows
			ms.total = msg.total
		}
		return ms, nil
	}
	return ms, nil
}

func (ms *MemoriesScreen) handleKey(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if ms.filterFocus == 0 && ms.cursor < len(ms.rows)-1 {
			ms.cursor++
		}
		return ms, nil

	case "k", "up":
		if ms.filterFocus == 0 && ms.cursor > 0 {
			ms.cursor--
		}
		return ms, nil

	case "n":
		// Next page — only if there are more pages
		totalPages := ms.totalPages()
		if ms.page < totalPages-1 {
			ms.page++
			ms.cursor = 0
			return ms, ms.loadCmd()
		}
		return ms, nil

	case "p":
		// Prev page
		if ms.page > 0 {
			ms.page--
			ms.cursor = 0
			return ms, ms.loadCmd()
		}
		return ms, nil

	case "f":
		// Cycle filter focus: 0 → 1 → 2 → 3 → 4 → 0
		ms.filterFocus = (ms.filterFocus + 1) % 5
		return ms, nil

	case "c":
		// Clear all filters, reset to page 0
		ms.filter = MemoriesFilter{}
		ms.page = 0
		ms.cursor = 0
		return ms, ms.loadCmd()

	case "enter":
		if ms.filterFocus == 0 && len(ms.rows) > 0 && ms.cursor < len(ms.rows) {
			sel := ms.rows[ms.cursor]
			return ms, func() tea.Msg {
				d := newDetailScreenWithTabOrigin(sel, TabMemories)
				return pushScreenCmd{screen: d}
			}
		}
		return ms, nil
	}
	return ms, nil
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (ms *MemoriesScreen) View(width, height int, p palette) string {
	var b strings.Builder
	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)
	brandStyle := lipgloss.NewStyle().Foreground(p.Brand)
	cursorStyle := lipgloss.NewStyle().Foreground(p.NavActive)
	focusedBg := lipgloss.NewStyle().Background(p.FocusedBg)

	// Title
	b.WriteString(lipgloss.NewStyle().Foreground(p.Foreground).Bold(true).Render("Memories"))
	b.WriteString("\n")

	// Filter bar
	b.WriteString(ms.renderFilterBar(width, p))
	b.WriteString("\n")

	// Content rows
	if len(ms.rows) == 0 {
		b.WriteString(mutedStyle.Render("(no memories)"))
	} else {
		for i, row := range ms.rows {
			prefix := "  "
			if i == ms.cursor && ms.filterFocus == 0 {
				prefix = cursorStyle.Render("▸") + " "
			}
			typeTag := truncate(string(row.Type), 10)
			badge := "[" + brandStyle.Render(typeTag) + "]"
			topicKey := truncate(row.TopicKey, 36)
			timeStr := mutedStyle.Render(relTime(row.UpdatedAt))
			scope := mutedStyle.Render(string(row.Scope))
			line := fmt.Sprintf("%s%-12s %-36s  %s  %s",
				prefix, badge, topicKey, timeStr, scope)
			if i == ms.cursor && ms.filterFocus == 0 {
				line = focusedBg.Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")

	// Footer
	b.WriteString(ms.renderFooter(p))
	return b.String()
}

func (ms *MemoriesScreen) renderFilterBar(width int, p palette) string {
	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)
	brandStyle := lipgloss.NewStyle().Foreground(p.Brand)
	focusedStyle := lipgloss.NewStyle().Foreground(p.NavActive).Bold(true)

	typeVal := ms.filter.Type
	if typeVal == "" {
		typeVal = "all"
	}
	tagVal := ms.filter.Tag
	if tagVal == "" {
		tagVal = "-"
	}
	scopeVal := ms.filter.Scope
	if scopeVal == "" {
		scopeVal = "all"
	}
	sortVal := ms.filter.Sort
	if sortVal == "" || sortVal == "updated_desc" {
		sortVal = "updated↓"
	}

	renderLabel := func(label string, focusIdx int, val string) string {
		var labelRend string
		if ms.filterFocus == focusIdx {
			labelRend = focusedStyle.Render(label)
		} else {
			labelRend = brandStyle.Render(label)
		}
		return labelRend + " " + val
	}

	parts := []string{
		"Filters:",
		renderLabel("Type:", 1, typeVal),
		renderLabel("Tag:", 2, tagVal),
		renderLabel("Scope:", 3, scopeVal),
		renderLabel("Sort:", 4, sortVal),
		mutedStyle.Render("(f cycle, c clear)"),
	}
	return mutedStyle.Render("Filters:") + "  " + strings.Join(parts[1:], "  ")
}

func (ms *MemoriesScreen) renderFooter(p palette) string {
	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)
	totalPages := ms.totalPages()
	if totalPages == 0 {
		totalPages = 1
	}
	pageStr := fmt.Sprintf("Page %d of %d · %d total", ms.page+1, totalPages, ms.total)
	nav := "↑↓ row · n next · p prev · enter open · f filter · q quit"
	return mutedStyle.Render(pageStr + " · " + nav)
}

func (ms *MemoriesScreen) totalPages() int {
	if ms.total == 0 {
		return 1
	}
	pages := ms.total / ms.perPage
	if ms.total%ms.perPage != 0 {
		pages++
	}
	return pages
}

// ─── Data loading ─────────────────────────────────────────────────────────────

func (ms *MemoriesScreen) loadCmd() tea.Cmd {
	st := ms.storage
	page := ms.page
	perPage := ms.perPage
	if perPage == 0 {
		perPage = memoriesPerPage
	}
	filter := ms.filter
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		offset := page * perPage
		rows, err := st.RecentAll(ctx, "", perPage+offset) // fetch enough for offset
		if err != nil {
			return memoriesLoadedMsg{err: err, page: page}
		}

		// Apply in-memory filtering
		rows = applyMemoriesFilter(rows, filter)

		// Get total count from unfiltered global query for accurate paging.
		totalRows, _ := st.RecentAll(ctx, "", 1000)
		totalRows = applyMemoriesFilter(totalRows, filter)
		total := len(totalRows)

		// Page slice
		start := page * perPage
		if start > len(rows) {
			start = len(rows)
		}
		end := start + perPage
		if end > len(rows) {
			end = len(rows)
		}
		pageRows := rows[start:end]

		// Sort
		if filter.Sort == "created_desc" {
			sortSearchResultsByCreatedDesc(pageRows)
		} else {
			// Default: updated_desc (already the order from RecentAll)
		}

		return memoriesLoadedMsg{rows: pageRows, total: total, page: page}
	}
}

// applyMemoriesFilter applies in-memory type/tag/scope filters.
func applyMemoriesFilter(rows []storage.SearchResult, f MemoriesFilter) []storage.SearchResult {
	if f.Type == "" && f.Tag == "" && f.Scope == "" {
		return rows
	}
	out := rows[:0:0]
	for _, r := range rows {
		if f.Type != "" && string(r.Type) != f.Type {
			continue
		}
		if f.Scope != "" && string(r.Scope) != f.Scope {
			continue
		}
		if f.Tag != "" {
			found := false
			for _, tag := range r.Tags {
				if tag == f.Tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// sortSearchResultsByCreatedDesc sorts results by UpdatedAt ASC to simulate
// a "created_desc" alternative view. (We don't have CreatedAt on SearchResult,
// so we approximate with UpdatedAt ascending as a proxy for oldest-first.)
func sortSearchResultsByCreatedDesc(rows []storage.SearchResult) {
	// Bubble sort for small N
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].UpdatedAt.After(rows[i].UpdatedAt) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}

// ─── DetailScreen with origin ─────────────────────────────────────────────────

// detailScreenWithOrigin wraps DetailScreen to implement the originator
// interface. Used by HomeScreen and MemoriesScreen to ensure esc-pop returns
// to the correct tab.
type detailScreenWithOrigin struct {
	*DetailScreen
	origin tabKey
}

func newDetailScreenWithTabOrigin(sel storage.SearchResult, origin tabKey) Screen {
	content := fmt.Sprintf("[%s] %s\n\n%s", sel.Type, sel.TopicKey, sel.Snippet)
	d := newDetailScreenFull(content)
	return &detailScreenWithOrigin{DetailScreen: d, origin: origin}
}

func (d *detailScreenWithOrigin) OriginatingTab() tabKey { return d.origin }

func (d *detailScreenWithOrigin) Update(msg tea.Msg) (Screen, tea.Cmd) {
	s, cmd := d.DetailScreen.Update(msg)
	// Re-wrap so we don't lose the origin on Update.
	if ds, ok := s.(*DetailScreen); ok {
		return &detailScreenWithOrigin{DetailScreen: ds, origin: d.origin}, cmd
	}
	return s, cmd
}
