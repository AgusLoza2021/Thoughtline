package dashboard

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// recentLoadedMsg carries results from a RecentAll/Recent call.
type recentLoadedMsg struct {
	results []storage.SearchResult
	err     error
}

// RecentScreen displays up to 20 recently updated memories, optionally scoped
// to a project (filterProject != "" → project-scoped; "" → cross-project).
type RecentScreen struct {
	storage       *storage.Storage
	project       string
	filterProject string // non-empty for Browse→drill-in
	results       []storage.SearchResult
	cursor        int
}

func newRecentScreenFull(st *storage.Storage, project, filterProject string) *RecentScreen {
	return &RecentScreen{storage: st, project: project, filterProject: filterProject}
}

func (r *RecentScreen) Title() string { return "Recent Activity" }

func (r *RecentScreen) Init() tea.Cmd { return r.loadCmd() }

func (r *RecentScreen) OnFocus() tea.Cmd { return r.loadCmd() }

func (r *RecentScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return r, func() tea.Msg { return popScreenCmd{} }
		case tea.KeyEnter:
			if len(r.results) > 0 && r.cursor < len(r.results) {
				selected := r.results[r.cursor]
				return r, func() tea.Msg {
					return pushScreenCmd{screen: newDetailScreen(selected.Title + "\n\n" + selected.Snippet)}
				}
			}
			return r, nil
		}
		switch msg.String() {
		case "q":
			return r, func() tea.Msg { return popScreenCmd{} }
		case "j", "down":
			if r.cursor < len(r.results)-1 {
				r.cursor++
			}
		case "k", "up":
			if r.cursor > 0 {
				r.cursor--
			}
		}
	case recentLoadedMsg:
		if msg.err == nil {
			r.results = msg.results
		}
	}
	return r, nil
}

func (r *RecentScreen) View(width, height int, p palette) string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(p.Foreground)
	b.WriteString(titleStyle.Render("Recent Activity"))
	b.WriteString("\n\n")

	if len(r.results) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("(no memories yet)"))
		return b.String()
	}
	for i, res := range r.results {
		prefix := "  "
		if i == r.cursor {
			prefix = lipgloss.NewStyle().Foreground(p.Cursor).Render("▸") + " "
		}
		b.WriteString(prefix + res.Title + "\n")
	}
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("esc/q back • enter open"))
	return b.String()
}

func (r *RecentScreen) loadCmd() tea.Cmd {
	st := r.storage
	project := r.filterProject
	if project == "" {
		project = r.project
	}
	fp := r.filterProject
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var results []storage.SearchResult
		var err error
		if fp != "" {
			results, err = st.Recent(ctx, fp, 20)
		} else {
			results, err = st.RecentAll(ctx, "", 20)
		}
		return recentLoadedMsg{results: results, err: err}
	}
}
