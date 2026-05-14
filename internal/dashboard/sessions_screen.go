package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// ─── SessionsScreen ─────────────────────────────────────────────────────────
//
// SessionsScreen is tabs[TabSessions] in the flat workspace TUI. v1 scope is
// restyle-only: a single chronological list of recent sessions with project,
// relative start time, and open/closed status. Open/closed grouping and
// session search are deferred to a follow-up change per design decision #6.
//
// Empty state renders a friendly "No sessions yet" message.
//
// Keys:
//   - ↑/↓ or k/j — move cursor
//   - enter      — no-op for v1 (full session detail is out of scope)
//   - r          — refresh

// sessionsLoadedMsg carries the recent sessions list from storage.
type sessionsLoadedMsg struct {
	sessions []memory.Session
	err      error
}

// SessionsScreen lists recent sessions across all projects.
type SessionsScreen struct {
	storage  *storage.Storage
	sessions []memory.Session
	cursor   int
	loaded   bool
	err      error
}

// NewSessionsScreen constructs an unloaded SessionsScreen.
func NewSessionsScreen(st *storage.Storage) *SessionsScreen {
	return &SessionsScreen{storage: st}
}

func (s *SessionsScreen) Title() string  { return "Sessions" }
func (s *SessionsScreen) Init() tea.Cmd  { return s.loadCmd() }
func (s *SessionsScreen) OnFocus() tea.Cmd { return s.loadCmd() }

func (s *SessionsScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case sessionsLoadedMsg:
		s.loaded = true
		s.err = msg.err
		if msg.err == nil {
			s.sessions = msg.sessions
			if s.cursor >= len(s.sessions) {
				s.cursor = clampCursor(s.cursor, len(s.sessions))
			}
		}
		return s, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if s.cursor < len(s.sessions)-1 {
				s.cursor++
			}
		case "k", "up":
			if s.cursor > 0 {
				s.cursor--
			}
		case "r":
			return s, s.loadCmd()
		}
	}
	return s, nil
}

func (s *SessionsScreen) View(width, height int, p palette) string {
	var b strings.Builder

	b.WriteString(renderBrand(p))
	b.WriteString("\n\n")

	titleStyle := lipgloss.NewStyle().Foreground(p.Foreground).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)

	if !s.loaded {
		b.WriteString(mutedStyle.Render("Loading sessions…"))
		return b.String()
	}
	if s.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(p.Error)
		b.WriteString(errStyle.Render("error loading sessions: " + s.err.Error()))
		return b.String()
	}
	if len(s.sessions) == 0 {
		b.WriteString(titleStyle.Render("Sessions"))
		b.WriteString("\n\n")
		b.WriteString(mutedStyle.Render("No sessions yet — sessions are created via tl_session_start (MCP)."))
		return b.String()
	}

	header := fmt.Sprintf("%d sessions · most recent first", len(s.sessions))
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n\n")

	cursorStyle := lipgloss.NewStyle().Foreground(p.NavActive)
	openStyle := lipgloss.NewStyle().Foreground(p.Success).Bold(true)
	closedStyle := lipgloss.NewStyle().Foreground(p.Muted)
	projStyle := lipgloss.NewStyle().Foreground(p.Brand)

	for i, sess := range s.sessions {
		prefix := "  "
		if i == s.cursor {
			prefix = cursorStyle.Render("▸") + " "
		}
		proj := sess.Project
		if sess.AgentLabel != "" {
			proj = sess.Project + " / " + sess.AgentLabel
		}
		var statusRendered string
		if sess.EndedAt == nil {
			statusRendered = openStyle.Render("open")
		} else {
			statusRendered = closedStyle.Render("closed")
		}
		rel := relTime(sess.StartedAt)
		// width-aware truncation of project line keeps row to 1 line
		maxProj := width - 32
		if maxProj < 10 {
			maxProj = 10
		}
		proj = truncate(proj, maxProj)
		line := fmt.Sprintf("%s%s  %s  %s",
			prefix,
			projStyle.Render(padRight(proj, maxProj)),
			mutedStyle.Render(padRight(rel, 10)),
			statusRendered,
		)
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("↑↓ select · r refresh · q quit"))

	// Avoid unused param when height==0 in tests; height is reserved for future paging.
	_ = height
	return b.String()
}

// loadCmd queries Stats with no project scope to get cross-project recent
// sessions. Reuses Stats.RecentSessions (cross-project when Project is empty).
func (s *SessionsScreen) loadCmd() tea.Cmd {
	st := s.storage
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stats, err := st.Stats(ctx, storage.StatsOptions{Project: "", RecentLimit: 50})
		if err != nil {
			return sessionsLoadedMsg{err: err}
		}
		return sessionsLoadedMsg{sessions: stats.RecentSessions}
	}
}
