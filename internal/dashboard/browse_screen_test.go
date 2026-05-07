package dashboard

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func newTestBrowseScreen(t *testing.T, projects ...string) (*BrowseProjectsScreen, *storage.Storage) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "browse.db")
	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	for _, p := range projects {
		m := memory.Memory{
			Project:  p,
			Scope:    memory.ScopeProject,
			Type:     memory.TypeScenePattern,
			Title:    "mem for " + p,
			Content:  "content",
			TopicKey: "scene/" + p,
		}
		if _, _, err := st.Save(context.Background(), m); err != nil {
			t.Fatalf("seed project %q: %v", p, err)
		}
	}

	stats, err := st.Stats(context.Background(), storage.StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	return newBrowseProjectsScreenFull(st, stats), st
}

// TestBrowseProjectsScreen_AlphabeticalOrder verifies projects sort alphabetically.
func TestBrowseProjectsScreen_AlphabeticalOrder(t *testing.T) {
	screen, _ := newTestBrowseScreen(t, "zombie-game", "alpha-studio", "pipeline-tools")

	out := screen.View(80, 24, defaultPalette)
	aPos := strings.Index(out, "alpha-studio")
	pPos := strings.Index(out, "pipeline-tools")
	zPos := strings.Index(out, "zombie-game")

	if aPos < 0 || pPos < 0 || zPos < 0 {
		t.Fatalf("all projects must appear in view, got:\n%s", out)
	}
	if !(aPos < pPos && pPos < zPos) {
		t.Errorf("projects not in alpha order: alpha=%d pipeline=%d zombie=%d", aPos, pPos, zPos)
	}
}

// TestBrowseProjectsScreen_EnterPushesRecentScreen verifies enter pushes a
// project-filtered RecentScreen.
func TestBrowseProjectsScreen_EnterPushesRecentScreen(t *testing.T) {
	screen, _ := newTestBrowseScreen(t, "zombie-game", "alpha-studio", "pipeline-tools")
	screen.cursor = 0 // alpha-studio (sorted first)

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter must return a non-nil cmd")
	}
	msg := cmd()
	push, ok := msg.(pushScreenCmd)
	if !ok {
		t.Fatalf("enter must return pushScreenCmd, got %T", msg)
	}
	if push.screen == nil {
		t.Errorf("pushed screen must be non-nil")
	}
}
