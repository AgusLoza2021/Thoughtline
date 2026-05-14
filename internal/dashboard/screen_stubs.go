package dashboard

// screen_stubs.go provides minimal constructor stubs for the drill-in screens
// so the DashboardScreen can compile before Group G implements them fully.
// Each stub will be REPLACED (not extended) when the real screen is created.

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// stubScreen is a placeholder implementation of Screen used until the real
// screen files are created in Group G.
type stubScreen struct{ title string }

func (s *stubScreen) Init() tea.Cmd                            { return nil }
func (s *stubScreen) Update(msg tea.Msg) (Screen, tea.Cmd)    { return s, nil }
func (s *stubScreen) View(w, h int, p palette) string         { return "[ " + s.title + " ]" }
func (s *stubScreen) Title() string                           { return s.title }
func (s *stubScreen) OnFocus() tea.Cmd                        { return nil }

func newSearchScreen(st *storage.Storage, project string) Screen {
	return newSearchScreenFull(st, project)
}

func newRecentScreen(st *storage.Storage, project, filterProject string) Screen {
	return newRecentScreenFull(st, project, filterProject)
}

func newPendingScreen(st *storage.Storage, project string) Screen {
	return newPendingScreenFull(st, project)
}

func newDetailScreen(content string) Screen {
	return newDetailScreenFull(content)
}
