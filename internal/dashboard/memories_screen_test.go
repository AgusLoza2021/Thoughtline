package dashboard

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func newMemoriesScreen(t *testing.T) (*MemoriesScreen, func(Screen) *MemoriesScreen) {
	t.Helper()
	st := newWorkspaceStorage(t)
	ms := NewMemoriesScreen(st)
	cast := func(s Screen) *MemoriesScreen { return s.(*MemoriesScreen) }
	return ms, cast
}

func loadMemories(t *testing.T, ms *MemoriesScreen) *MemoriesScreen {
	t.Helper()
	cmd := ms.OnFocus()
	if cmd == nil {
		return ms
	}
	msg := cmd()
	s, _ := ms.Update(msg)
	return s.(*MemoriesScreen)
}

func renderMemories(t *testing.T, ms *MemoriesScreen) string {
	t.Helper()
	return ms.View(100, 30, defaultPalette)
}

func sendKey(t *testing.T, s Screen, key string) Screen {
	t.Helper()
	updated, _ := s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated
}

func sendSpecialKey(t *testing.T, s Screen, kt tea.KeyType) Screen {
	t.Helper()
	updated, _ := s.Update(tea.KeyMsg{Type: kt})
	return updated
}

// ─── J1: Pagination — seed 50, page 1 shows rows 0-19 ───────────────────────

func TestMemoriesScreen_J1_Pagination_FirstPage(t *testing.T) {
	ms, _ := newMemoriesScreen(t)
	seedMemories(t, ms.storage, 50)
	ms = loadMemories(t, ms)

	if len(ms.rows) != 20 {
		t.Errorf("page 1 should show 20 rows, got %d", len(ms.rows))
	}
	if ms.page != 0 {
		t.Errorf("initial page should be 0, got %d", ms.page)
	}
	if ms.total != 50 {
		t.Errorf("total should be 50, got %d", ms.total)
	}
}

func TestMemoriesScreen_J1_Pagination_NextPage(t *testing.T) {
	ms, cast := newMemoriesScreen(t)
	seedMemories(t, ms.storage, 50)
	ms = loadMemories(t, ms)

	// Press 'n' to advance to page 2
	s, cmd := ms.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	ms = cast(s)
	if cmd != nil {
		msg := cmd()
		s, _ = ms.Update(msg)
		ms = cast(s)
	}

	if ms.page != 1 {
		t.Errorf("after n, page should be 1, got %d", ms.page)
	}
	if len(ms.rows) != 20 {
		t.Errorf("page 2 should show 20 rows, got %d", len(ms.rows))
	}
}

func TestMemoriesScreen_J1_Pagination_PrevPage(t *testing.T) {
	ms, cast := newMemoriesScreen(t)
	seedMemories(t, ms.storage, 50)
	ms = loadMemories(t, ms)

	// Go to page 2
	s, cmd := ms.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	ms = cast(s)
	if cmd != nil {
		msg := cmd()
		s2, _ := ms.Update(msg)
		ms = cast(s2)
	}

	// Go back with 'p'
	s3, cmd2 := ms.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	ms = cast(s3)
	if cmd2 != nil {
		msg := cmd2()
		s4, _ := ms.Update(msg)
		ms = cast(s4)
	}

	if ms.page != 0 {
		t.Errorf("after p, page should be 0, got %d", ms.page)
	}
}

func TestMemoriesScreen_J1_Pagination_LastPagePartialFill(t *testing.T) {
	ms, cast := newMemoriesScreen(t)
	seedMemories(t, ms.storage, 25) // 25 memories → page 1 has 20, page 2 has 5
	ms = loadMemories(t, ms)

	// Advance to page 2
	s, cmd := ms.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	ms = cast(s)
	if cmd != nil {
		msg := cmd()
		s2, _ := ms.Update(msg)
		ms = cast(s2)
	}

	if len(ms.rows) != 5 {
		t.Errorf("last page with 25 total should have 5 rows, got %d", len(ms.rows))
	}
}

func TestMemoriesScreen_J1_Footer_PageIndicator(t *testing.T) {
	ms, _ := newMemoriesScreen(t)
	seedMemories(t, ms.storage, 50)
	ms = loadMemories(t, ms)
	view := renderMemories(t, ms)

	// Footer must contain page indicator
	if !strings.Contains(view, "Page 1") {
		t.Errorf("footer must contain 'Page 1', got:\n%s", view)
	}
	if !strings.Contains(view, "50 total") {
		t.Errorf("footer must contain '50 total', got:\n%s", view)
	}
}

// ─── J2: Filter cycling with f ───────────────────────────────────────────────

func TestMemoriesScreen_J2_FilterCycling(t *testing.T) {
	ms, _ := newMemoriesScreen(t)

	// Initial focus is on the row cursor (filterFocus == 0)
	if ms.filterFocus != 0 {
		t.Errorf("initial filterFocus should be 0, got %d", ms.filterFocus)
	}

	// Press f cycles focus: 0 → 1 → 2 → 3 → 4 → 0
	for i := 1; i <= 4; i++ {
		s := sendKey(t, ms, "f")
		ms = s.(*MemoriesScreen)
		if ms.filterFocus != i {
			t.Errorf("after %d 'f' presses, filterFocus = %d, want %d", i, ms.filterFocus, i)
		}
	}
	// One more 'f' wraps back to 0
	s := sendKey(t, ms, "f")
	ms = s.(*MemoriesScreen)
	if ms.filterFocus != 0 {
		t.Errorf("filterFocus should wrap to 0, got %d", ms.filterFocus)
	}
}

// ─── J3: Filter clear with c ─────────────────────────────────────────────────

func TestMemoriesScreen_J3_FilterClear(t *testing.T) {
	ms, _ := newMemoriesScreen(t)

	// Set some filter values
	ms.filter = MemoriesFilter{Type: "decision", Tag: "platform:windows", Sort: "created_desc"}

	// Press c
	s := sendKey(t, ms, "c")
	ms = s.(*MemoriesScreen)

	if ms.filter.Type != "" {
		t.Errorf("after c, filter.Type should be empty, got %q", ms.filter.Type)
	}
	if ms.filter.Tag != "" {
		t.Errorf("after c, filter.Tag should be empty, got %q", ms.filter.Tag)
	}
	if ms.filter.Sort != "" {
		t.Errorf("after c, filter.Sort should be empty, got %q", ms.filter.Sort)
	}
}

// ─── J4: Filter persistence across tab switches ───────────────────────────────

func TestMemoriesScreen_J4_FilterPersistence(t *testing.T) {
	ms, _ := newMemoriesScreen(t)

	// Set a filter
	ms.filter = MemoriesFilter{Type: "decision"}

	// Simulate a tab switch away and back (OnFocus called again).
	// The filter must still be set — it's an in-memory field on the struct.
	cmd := ms.OnFocus()
	if cmd != nil {
		msg := cmd()
		s, _ := ms.Update(msg)
		ms = s.(*MemoriesScreen)
	}

	if ms.filter.Type != "decision" {
		t.Errorf("filter.Type should persist across tab switch, got %q", ms.filter.Type)
	}
}

// ─── J5: Cursor preservation on detail push/pop ──────────────────────────────

func TestMemoriesScreen_J5_CursorPreservation(t *testing.T) {
	ms, cast := newMemoriesScreen(t)
	seedMemories(t, ms.storage, 25)
	ms = loadMemories(t, ms)

	// Move cursor to row 5
	for i := 0; i < 5; i++ {
		s := sendKey(t, ms, "j")
		ms = cast(s)
	}
	if ms.cursor != 5 {
		t.Errorf("cursor should be at 5, got %d", ms.cursor)
	}

	// Push detail via enter
	_, cmd := ms.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		cmd() // ignore the push message in this unit test
	}

	// OnFocus called when returning from Detail (simulates esc-pop)
	ms = loadMemories(t, ms)

	// Cursor must still be at 5
	if ms.cursor != 5 {
		t.Errorf("cursor should be preserved at 5 after returning from detail, got %d", ms.cursor)
	}
}

// ─── J6: Default sort updated_at DESC ────────────────────────────────────────

func TestMemoriesScreen_J6_DefaultSort(t *testing.T) {
	ms, _ := newMemoriesScreen(t)
	seedMemories(t, ms.storage, 5)
	ms = loadMemories(t, ms)

	// Rows should be in reverse-chronological order (seed-04 first, since it has latest UpdatedAt)
	if len(ms.rows) < 2 {
		t.Fatal("expected at least 2 rows")
	}
	// First row should have a later UpdatedAt than second row
	if ms.rows[0].UpdatedAt.Before(ms.rows[1].UpdatedAt) {
		t.Errorf("rows not sorted by updated_at DESC: row[0]=%v row[1]=%v",
			ms.rows[0].UpdatedAt, ms.rows[1].UpdatedAt)
	}
}

// ─── J7: No-wrap cursor at edges ─────────────────────────────────────────────

func TestMemoriesScreen_J7_CursorNoWrapAtEdges(t *testing.T) {
	ms, cast := newMemoriesScreen(t)
	seedMemories(t, ms.storage, 3)
	ms = loadMemories(t, ms)

	// Press k at top — cursor stays at 0
	s := sendKey(t, ms, "k")
	ms = cast(s)
	if ms.cursor != 0 {
		t.Errorf("cursor at top after k should stay 0, got %d", ms.cursor)
	}

	// Move to last row
	for i := 0; i < len(ms.rows)-1; i++ {
		ms = cast(sendKey(t, ms, "j"))
	}
	lastRow := ms.cursor

	// Press j at last row — cursor stays
	ms = cast(sendKey(t, ms, "j"))
	if ms.cursor != lastRow {
		t.Errorf("cursor at last row after j should stay %d, got %d", lastRow, ms.cursor)
	}
}

// ─── J8: Enter pushes DetailScreen with OriginatingTab = TabMemories ─────────

func TestMemoriesScreen_J8_EnterPushesDetail(t *testing.T) {
	ms, _ := newMemoriesScreen(t)
	seedMemories(t, ms.storage, 3)
	ms = loadMemories(t, ms)

	_, cmd := ms.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on MemoriesScreen with rows should produce a cmd")
	}
	msg := cmd()
	push, ok := msg.(pushScreenCmd)
	if !ok {
		t.Fatalf("expected pushScreenCmd, got %T", msg)
	}
	// The pushed detail must have OriginatingTab() == TabMemories
	orig, ok := push.screen.(originator)
	if !ok {
		t.Fatal("pushed DetailScreen must implement originator interface")
	}
	if orig.OriginatingTab() != TabMemories {
		t.Errorf("OriginatingTab should be TabMemories (%d), got %d", TabMemories, orig.OriginatingTab())
	}
}

// ─── J9: Filter bar visible in View ──────────────────────────────────────────

func TestMemoriesScreen_J9_FilterBarVisible(t *testing.T) {
	ms, _ := newMemoriesScreen(t)
	ms = loadMemories(t, ms)
	view := renderMemories(t, ms)

	// Filter bar must show all 4 labels
	for _, label := range []string{"Type:", "Tag:", "Scope:", "Sort:"} {
		if !strings.Contains(view, label) {
			t.Errorf("filter bar missing %q in:\n%s", label, view)
		}
	}
	// Help text
	if !strings.Contains(view, "f cycle") {
		t.Errorf("filter bar missing '(f cycle, c clear)' in:\n%s", view)
	}
}

// ─── View smoke: screen renders non-empty ────────────────────────────────────

func TestMemoriesScreen_View_Smoke(t *testing.T) {
	ms, _ := newMemoriesScreen(t)
	view := renderMemories(t, ms)
	if view == "" {
		t.Error("MemoriesScreen.View must not be empty")
	}
	// Must contain "Memories" tab identity
	if !strings.Contains(view, "Memories") {
		t.Errorf("view must contain 'Memories', got:\n%s", view)
	}
}

// ─── No-op pagination at boundaries ──────────────────────────────────────────

func TestMemoriesScreen_NoPrevOnFirstPage(t *testing.T) {
	ms, cast := newMemoriesScreen(t)
	seedMemories(t, ms.storage, 5)
	ms = loadMemories(t, ms)

	// p on page 0 is a no-op
	s, cmd := ms.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	ms = cast(s)
	_ = cmd
	if ms.page != 0 {
		t.Errorf("p on page 0 should not change page, got %d", ms.page)
	}
}

func TestMemoriesScreen_ImplementsScreen(t *testing.T) {
	ms, _ := newMemoriesScreen(t)
	var _ Screen = ms
	if ms.Title() == "" {
		t.Error("MemoriesScreen.Title() must not be empty")
	}
}

// helpers used by tests below
func (ms *MemoriesScreen) seedAndLoad(t *testing.T, n int) *MemoriesScreen {
	t.Helper()
	seedMemories(t, ms.storage, n)
	return loadMemories(t, ms)
}

// Expose storage for test helpers that need it
func (ms *MemoriesScreen) getStorage() interface{} { return ms.storage }

// formatPageIndicator builds the expected footer page text.
func formatPageIndicator(page, totalPages, total int) string {
	return fmt.Sprintf("Page %d of %d · %d total", page+1, totalPages, total)
}
