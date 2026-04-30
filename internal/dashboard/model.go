package dashboard

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// Config carries the wiring the dashboard needs from main(). DBPath and
// Version come from the binary's runtime; Project is the auto-detected
// default ("" means "show all projects").
type Config struct {
	Version string
	DBPath  string
	Project string
}

// Model is the Bubbletea Model for the dashboard. Tested directly via
// model.Update() (see SKILL.md pattern 2) — no global state, no goroutines
// outside Bubbletea's command pipeline.
type Model struct {
	storage *storage.Storage
	cfg     Config

	stats   storage.Stats
	loaded  bool
	loading bool
	err     error

	// Last rendered dimensions; refreshed on tea.WindowSizeMsg.
	width  int
	height int

	// Quitting=true means the Model is on its way out — used by tests to
	// assert the q/ctrl+c key path.
	Quitting bool
}

// New constructs a Model bound to the given storage. The Init() command
// triggers the first stats load.
func New(st *storage.Storage, cfg Config) Model {
	return Model{
		storage: st,
		cfg:     cfg,
	}
}

// Init kicks off the first stats fetch.
func (m Model) Init() tea.Cmd {
	return loadStatsCmd(m.storage, m.cfg.Project)
}

// statsLoadedMsg is delivered by loadStatsCmd. The Model in Update() reads
// it to populate stats and clear loading state.
type statsLoadedMsg struct {
	stats storage.Stats
	err   error
}

// loadStatsCmd returns a Bubbletea command that runs storage.Stats off the
// main goroutine and posts the result back as a statsLoadedMsg.
func loadStatsCmd(st *storage.Storage, project string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		stats, err := st.Stats(ctx, storage.StatsOptions{Project: project})
		return statsLoadedMsg{stats: stats, err: err}
	}
}
