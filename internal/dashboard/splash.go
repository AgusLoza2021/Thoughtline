package dashboard

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// splashDoneMsg flips Model.splashActive off and lets the dashboard
// render normally. Dispatched by splashTimer after Config.SplashDuration.
type splashDoneMsg struct{}

func splashTimer(d time.Duration) tea.Cmd {
	if d <= 0 {
		d = 1500 * time.Millisecond
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return splashDoneMsg{} })
}

// renderSplash composes the intro screen: cube + wordmark + tagline,
// vertically centered in the available terminal area. Plain ASCII for
// the wordmark so it stays readable on any terminal.
func (m Model) renderSplash() string {
	cube := m.cube.View()

	wordmark := titleStyle.Render(thoughtlineWordmark())
	tagline := subtleStyle.Italic(true).Render("◇ Your project's memory, forever")
	version := subtleStyle.Render("v" + m.cfg.Version)

	stack := lipgloss.JoinVertical(lipgloss.Center,
		cube,
		"",
		wordmark,
		"",
		tagline,
		"",
		version,
	)

	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, stack)
}

// thoughtlineWordmark returns the multi-line ASCII wordmark used by the
// splash screen. Pure ASCII so it renders on Windows CMD without
// requiring a Unicode font.
func thoughtlineWordmark() string {
	return strings.Join([]string{
		"████████ ██   ██  ██████  ██    ██  ██████  ██   ██ ████████ ██      ██ ███    ██ ███████",
		"   ██    ██   ██ ██    ██ ██    ██ ██       ██   ██    ██    ██      ██ ████   ██ ██     ",
		"   ██    ███████ ██    ██ ██    ██ ██   ███ ███████    ██    ██      ██ ██ ██  ██ █████  ",
		"   ██    ██   ██ ██    ██ ██    ██ ██    ██ ██   ██    ██    ██      ██ ██  ██ ██ ██     ",
		"   ██    ██   ██  ██████   ██████   ██████  ██   ██    ██    ███████ ██ ██   ████ ███████",
	}, "\n")
}
