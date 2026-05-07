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
//
// Uses the new flatModel (engram-style screen stack). The legacy Model
// in model.go is preserved for its test suite but no longer drives the
// binary. To boot the legacy UI for comparison, call RunLegacy instead.
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

// RunLegacy boots the v1 tab-based dashboard. Kept as an escape hatch in case
// the new flat dashboard regresses on a specific terminal — invoke it from
// cmd/thoughtline behind a hidden flag if needed. Not exposed by default.
func RunLegacy(ctx context.Context, st *storage.Storage, cfg Config) error {
	model := New(st, cfg)
	prog := tea.NewProgram(model,
		tea.WithContext(ctx),
		tea.WithAltScreen(),
	)
	if _, err := prog.Run(); err != nil {
		return fmt.Errorf("dashboard (legacy): %w", err)
	}
	return nil
}
