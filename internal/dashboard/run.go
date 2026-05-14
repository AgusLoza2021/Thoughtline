package dashboard

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// Run blocks the calling goroutine while the dashboard TUI is on screen.
// It returns nil on a clean quit (q / ctrl+c / esc) and a wrapped error
// otherwise. The caller (cmd/thoughtline) is responsible for opening the
// storage and providing the Config.
func Run(ctx context.Context, st *storage.Storage, cfg Config) error {
	model := newFlatModel(st, cfg)
	prog := tea.NewProgram(model,
		tea.WithContext(ctx),
		tea.WithAltScreen(),
	)
	if _, err := prog.Run(); err != nil {
		return fmt.Errorf("dashboard: %w", err)
	}
	return nil
}

