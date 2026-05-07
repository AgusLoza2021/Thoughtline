package dashboard

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// searchResultsMsg carries results from a storage.Search call.
type searchResultsMsg struct {
	results []storage.SearchResult
	err     error
}

// SearchScreen shows a text-input field and a list of search results.
type SearchScreen struct {
	storage *storage.Storage
	project string
	input   textinput.Model
	results []storage.SearchResult
	cursor  int
}

// newSearchScreenFull creates a fully initialised SearchScreen (used by tests).
func newSearchScreenFull(st *storage.Storage, project string) *SearchScreen {
	ti := textinput.New()
	ti.Placeholder = "search memories…"
	ti.CharLimit = 200
	ti.Prompt = "› "
	ti.Focus()
	return &SearchScreen{storage: st, project: project, input: ti}
}

func (s *SearchScreen) Title() string { return "Search" }

func (s *SearchScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (s *SearchScreen) OnFocus() tea.Cmd { return textinput.Blink }

func (s *SearchScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return s, func() tea.Msg { return popScreenCmd{} }
		case tea.KeyEnter:
			q := strings.TrimSpace(s.input.Value())
			if q == "" {
				return s, nil
			}
			return s, s.runSearch(q)
		}
		if msg.String() == "q" {
			return s, func() tea.Msg { return popScreenCmd{} }
		}
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		return s, cmd
	case searchResultsMsg:
		if msg.err == nil {
			s.results = msg.results
			s.cursor = 0
		}
		return s, nil
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s *SearchScreen) View(width, height int, p palette) string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(p.Foreground)
	b.WriteString(titleStyle.Render("Search memories"))
	b.WriteString("\n\n")
	b.WriteString(s.input.View())
	b.WriteString("\n\n")

	if len(s.results) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("(no results yet — type a query and press enter)"))
		return b.String()
	}
	for i, r := range s.results {
		prefix := "  "
		if i == s.cursor {
			prefix = lipgloss.NewStyle().Foreground(p.Cursor).Render("▸") + " "
		}
		b.WriteString(prefix + r.Title + "\n")
	}
	return b.String()
}

func (s *SearchScreen) runSearch(query string) tea.Cmd {
	st := s.storage
	project := s.project
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		results, err := st.Search(ctx, query, storage.SearchOptions{Project: project, Limit: 50})
		return searchResultsMsg{results: results, err: err}
	}
}
