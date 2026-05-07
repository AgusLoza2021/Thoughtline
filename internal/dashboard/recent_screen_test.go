package dashboard

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func newTestRecentScreen(t *testing.T) (*RecentScreen, *storage.Storage) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "recent.db")
	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return newRecentScreenFull(st, "test-project", ""), st
}

func seedMemory(t *testing.T, st *storage.Storage, project, title string, updatedAt time.Time) storage.SearchResult {
	t.Helper()
	st.SetClock(func() time.Time { return updatedAt })
	m := memory.Memory{
		Project: project,
		Scope:   memory.ScopeProject,
		Type:    memory.TypeScenePattern,
		Title:   title,
		Content: "content for " + title,
		TopicKey: "scene/" + title,
	}
	saved, _, err := st.Save(context.Background(), m)
	if err != nil {
		t.Fatalf("seed memory %q: %v", title, err)
	}
	return storage.SearchResult{ID: saved.ID, Title: saved.Title, UpdatedAt: updatedAt}
}

// TestRecentScreen_MemoriesOrderedByRecency verifies the most-recent item is first.
func TestRecentScreen_MemoriesOrderedByRecency(t *testing.T) {
	screen, st := newTestRecentScreen(t)

	t0 := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	oldest := seedMemory(t, st, "test-project", "oldest", t0)
	middle := seedMemory(t, st, "test-project", "middle", t0.Add(time.Hour))
	newest := seedMemory(t, st, "test-project", "newest", t0.Add(2*time.Hour))
	_ = oldest
	_ = middle

	// Load results into screen.
	next, _ := screen.Update(recentLoadedMsg{results: []storage.SearchResult{
		{ID: newest.ID, Title: "newest", UpdatedAt: t0.Add(2 * time.Hour)},
		{ID: middle.ID, Title: "middle", UpdatedAt: t0.Add(time.Hour)},
		{ID: oldest.ID, Title: "oldest", UpdatedAt: t0},
	}})
	rs := next.(*RecentScreen)
	out := rs.View(80, 24, defaultPalette)

	// The first occurrence of a title in the output indicates rendering order.
	newestPos := strings.Index(out, "newest")
	middlePos := strings.Index(out, "middle")
	oldestPos := strings.Index(out, "oldest")

	if newestPos < 0 || middlePos < 0 || oldestPos < 0 {
		t.Fatalf("view missing expected items, got:\n%s", out)
	}
	if newestPos > middlePos || middlePos > oldestPos {
		t.Errorf("items not in descending recency order: newest=%d middle=%d oldest=%d",
			newestPos, middlePos, oldestPos)
	}
}

// TestRecentScreen_EnterPushesDetailScreen verifies enter on a highlighted item
// pushes the detail screen.
func TestRecentScreen_EnterPushesDetailScreen(t *testing.T) {
	screen, _ := newTestRecentScreen(t)

	// Inject one result so there's something to select.
	screen.results = []storage.SearchResult{{ID: 1, Title: "test memory"}}
	screen.cursor = 0

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter must return a non-nil cmd")
	}
	if _, ok := cmd().(pushScreenCmd); !ok {
		t.Errorf("enter on result must return pushScreenCmd")
	}
}
