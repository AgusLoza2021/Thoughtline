package dashboard

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles every Bubbletea message the dashboard cares about. Side
// effects (re-fetching stats) are returned as commands so the function
// stays pure and testable.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case statsLoadedMsg:
		m.loading = false
		m.loaded = true
		m.err = msg.err
		if msg.err == nil {
			m.stats = msg.stats
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.Quitting = true
			return m, tea.Quit
		case "r":
			// Reload stats. Set loading=true immediately so the View() can
			// render a "refreshing..." hint before the new stats arrive.
			m.loading = true
			return m, loadStatsCmd(m.storage, m.cfg.Project)
		}
	}
	return m, nil
}
