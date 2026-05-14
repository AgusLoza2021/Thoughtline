package dashboard

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func newHomeScreen(t *testing.T) (*HomeScreen, *storage.Storage) {
	t.Helper()
	st := newWorkspaceStorage(t)
	hs := NewHomeScreen(st)
	return hs, st
}

func renderHome(t *testing.T, hs *HomeScreen) string {
	t.Helper()
	return hs.View(100, 30, defaultPalette)
}

// loadHome runs the OnFocus command synchronously so the screen has data.
func loadHome(t *testing.T, hs *HomeScreen) *HomeScreen {
	t.Helper()
	cmd := hs.OnFocus()
	if cmd == nil {
		return hs
	}
	msg := cmd()
	s, _ := hs.Update(msg)
	return s.(*HomeScreen)
}

// ─── I1: Header and Quick Actions sections ───────────────────────────────────

func TestHomeScreen_I1_HeaderPresent(t *testing.T) {
	hs, _ := newHomeScreen(t)
	hs = loadHome(t, hs)
	view := renderHome(t, hs)

	// renderBrand output must appear
	if !strings.Contains(view, "Thoughtline") {
		t.Errorf("expected 'Thoughtline' in Home view, got:\n%s", view)
	}
	if !strings.Contains(view, "Local memory for game projects") {
		t.Errorf("expected tagline in Home view, got:\n%s", view)
	}
}

func TestHomeScreen_I1_QuickActionsRow(t *testing.T) {
	hs, _ := newHomeScreen(t)
	hs = loadHome(t, hs)
	view := renderHome(t, hs)

	if !strings.Contains(view, "Quick:") {
		t.Errorf("expected 'Quick:' in Home view, got:\n%s", view)
	}
	// All quick-action hotkey labels must appear
	for _, label := range []string{"[S]", "[M]", "[I]"} {
		if !strings.Contains(view, label) {
			t.Errorf("expected %q in Quick Actions row, got:\n%s", label, view)
		}
	}
}

// ─── I2: Project Health card ─────────────────────────────────────────────────

func TestHomeScreen_I2_ProjectHealthCard(t *testing.T) {
	hs, st := newHomeScreen(t)
	seedMemories(t, st, 5)
	hs = loadHome(t, hs)
	view := renderHome(t, hs)

	// Must contain the card title
	if !strings.Contains(view, "Project Health") {
		t.Errorf("expected 'Project Health' card, got:\n%s", view)
	}

	// Must contain each of the 5 fixed labels in order
	labels := []string{"Memories:", "Sessions:", "Pending:", "Disk free:", "Storage:"}
	for _, label := range labels {
		if !strings.Contains(view, label) {
			t.Errorf("expected label %q in Project Health card, got:\n%s", label, view)
		}
	}
}

// ─── I3: Latest Memories card ────────────────────────────────────────────────

func TestHomeScreen_I3_LatestMemoriesCard(t *testing.T) {
	hs, st := newHomeScreen(t)
	seedMemories(t, st, 5)
	hs = loadHome(t, hs)
	view := renderHome(t, hs)

	if !strings.Contains(view, "Latest Memories") {
		t.Errorf("expected 'Latest Memories' card, got:\n%s", view)
	}
	// At least one seeded memory topic key should appear (format: seed/type/NN)
	if !strings.Contains(view, "seed/") {
		t.Errorf("expected seeded memory topic keys in Latest Memories card, got:\n%s", view)
	}
}

func TestHomeScreen_I3_CursorMovesOnJK(t *testing.T) {
	hs, st := newHomeScreen(t)
	seedMemories(t, st, 5)
	hs = loadHome(t, hs)

	initialCursor := hs.cursor

	// Move cursor down
	s, _ := hs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	hs = s.(*HomeScreen)
	if hs.cursor != initialCursor+1 {
		t.Errorf("cursor after j: got %d, want %d", hs.cursor, initialCursor+1)
	}

	// Move cursor back up
	s, _ = hs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	hs = s.(*HomeScreen)
	if hs.cursor != initialCursor {
		t.Errorf("cursor after k: got %d, want %d", hs.cursor, initialCursor)
	}
}

func TestHomeScreen_I3_EnterPushesDetail(t *testing.T) {
	hs, st := newHomeScreen(t)
	seedMemories(t, st, 3)
	hs = loadHome(t, hs)

	// Enter on a loaded row should push a detail screen
	_, cmd := hs.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on HomeScreen with memories should produce a cmd")
	}
	msg := cmd()
	push, ok := msg.(pushScreenCmd)
	if !ok {
		t.Fatalf("expected pushScreenCmd from enter, got %T", msg)
	}
	if push.screen == nil {
		t.Fatal("pushed screen must not be nil")
	}
}

// ─── I4: Empty state contains required strings ───────────────────────────────

func TestHomeScreen_I4_EmptyState(t *testing.T) {
	hs, _ := newHomeScreen(t) // no memories seeded
	hs = loadHome(t, hs)
	view := renderHome(t, hs)

	required := []string{
		"scene-pattern",
		"perf-gotcha",
		"pipeline-step",
		"tl save",
		"tl_save",
	}
	for _, s := range required {
		if !strings.Contains(view, s) {
			t.Errorf("empty state missing %q in:\n%s", s, view)
		}
	}
}

func TestHomeScreen_I4_EmptyStateNotShownWhenMemoriesExist(t *testing.T) {
	hs, st := newHomeScreen(t)
	seedMemories(t, st, 1)
	hs = loadHome(t, hs)
	view := renderHome(t, hs)

	// When memories exist, the empty state headline must not appear
	if strings.Contains(view, "Welcome — no memories yet") {
		t.Error("empty state should not appear when memories exist")
	}
}

// ─── I5: Project Health 5 rows in fixed order ────────────────────────────────

func TestHomeScreen_I5_ProjectHealthFiveRowsInOrder(t *testing.T) {
	hs, st := newHomeScreen(t)
	seedMemories(t, st, 3)
	hs = loadHome(t, hs)
	view := renderHome(t, hs)

	// All 5 label rows must appear
	orderedLabels := []string{"Memories:", "Sessions:", "Pending:", "Disk free:", "Storage:"}
	prev := 0
	for _, label := range orderedLabels {
		idx := strings.Index(view[prev:], label)
		if idx < 0 {
			t.Errorf("missing or out-of-order label %q in:\n%s", label, view)
			return
		}
		prev += idx + len(label)
	}
}

// ─── I6: Recent Activity card ────────────────────────────────────────────────

func TestHomeScreen_I6_RecentActivityCard(t *testing.T) {
	hs, st := newHomeScreen(t)
	seedMemories(t, st, 3)
	hs = loadHome(t, hs)
	view := renderHome(t, hs)

	if !strings.Contains(view, "Recent Activity") {
		t.Errorf("expected 'Recent Activity' card, got:\n%s", view)
	}
}

// ─── I7: r refreshes, OnFocus refreshes ──────────────────────────────────────

func TestHomeScreen_I7_RKeyRefreshes(t *testing.T) {
	hs, st := newHomeScreen(t)
	hs = loadHome(t, hs)

	// Now seed memories after first load
	seedMemories(t, st, 5)

	// Press r — should return a non-nil cmd (load command)
	_, cmd := hs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("pressing r should issue a load command")
	}

	// Execute cmd and update — stats should now reflect the new memories
	msg := cmd()
	s, _ := hs.Update(msg)
	hs = s.(*HomeScreen)

	if hs.totalMemories != 5 {
		t.Errorf("after r+load, totalMemories = %d, want 5", hs.totalMemories)
	}
}

func TestHomeScreen_I7_OnFocusRefreshes(t *testing.T) {
	st := newWorkspaceStorage(t)
	hs2 := NewHomeScreen(st)
	seedMemories(t, st, 7)

	cmd := hs2.OnFocus()
	if cmd == nil {
		t.Fatal("OnFocus must return a load cmd")
	}
	msg := cmd()
	s, _ := hs2.Update(msg)
	hs2 = s.(*HomeScreen)

	if hs2.totalMemories != 7 {
		t.Errorf("after OnFocus+load, totalMemories = %d, want 7", hs2.totalMemories)
	}
}

// ─── I8: Integration smoke — screen implements Screen interface ───────────────

func TestHomeScreen_I8_ImplementsScreen(t *testing.T) {
	hs, _ := newHomeScreen(t)
	var _ Screen = hs
	// Init must return a cmd or nil
	cmd := hs.Init()
	// Init may return a load cmd
	_ = cmd
	// Title must not be empty
	if hs.Title() == "" {
		t.Error("HomeScreen.Title() must not be empty")
	}
}

// ─── I-bug-1: Project Health uses global stats, not project-scoped ────────────

func TestHomeScreen_IBug1_GlobalScoping(t *testing.T) {
	// Seed 27 memories across 4 different projects.
	// Use a HomeScreen whose cfg.Project does NOT match any of those projects.
	// The Project Health card must still show "Memories: 27" (global count).
	st := newWorkspaceStorage(t)
	ctx := context.Background()

	// Seed exactly 27 memories: 4+7+8+8 = 27 across 4 projects.
	projectCounts := map[string]int{"alpha": 4, "beta": 7, "gamma": 8, "delta": 8}
	total := 0
	for proj, count := range projectCounts {
		brainID, err := st.ResolveOrCreateBrainID(ctx, proj)
		if err != nil {
			t.Fatalf("resolve brain for %s: %v", proj, err)
		}
		for j := 0; j < count; j++ {
			m := memory.Memory{
				Project:  proj,
				Scope:    memory.ScopeProject,
				Type:     memory.TypeDecision,
				Title:    fmt.Sprintf("%s-mem-%d", proj, j),
				Content:  "content",
				TopicKey: fmt.Sprintf("%s/key/%d", proj, j),
			}
			if _, _, err := st.Save(ctx, brainID, m); err != nil {
				t.Fatalf("save %s#%d: %v", proj, j, err)
			}
		}
		total += count
	}
	if total != 27 {
		t.Fatalf("test setup error: expected 27 memories, seeded %d", total)
	}

	// HomeScreen with a project that has 0 memories of its own.
	hs := NewHomeScreen(st)
	hs = loadHome(t, hs)
	view := renderHome(t, hs)

	// Must show global total: "Memories:" followed by "27" somewhere on the same line.
	// The actual rendering pads to column width, so we strip spaces and check.
	stripped := strings.ReplaceAll(view, " ", "")
	if !strings.Contains(stripped, "Memories:27") {
		t.Errorf("Project Health must show global total 'Memories: 27', got:\n%s", view)
	}
	// Must NOT show Memories: 0 (the project-scoping bug).
	if strings.Contains(stripped, "Memories:0") {
		t.Errorf("Project Health must not show 'Memories: 0' when global count is 27:\n%s", view)
	}
}
