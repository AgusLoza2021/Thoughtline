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

// newTestModel sets up a Model bound to a fresh in-memory storage. Returns
// the Model and the Storage so tests can seed data before calling Init().
func newTestModel(t *testing.T) (Model, *storage.Storage) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "dash.db")
	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	cfg := Config{
		Version: "test",
		DBPath:  dbPath,
		Project: "enchanted-inn",
	}
	return New(st, cfg), st
}

// drainInit synchronously executes Init()'s returned command and feeds the
// resulting message back into Update — useful for unit tests that need the
// "stats loaded" state without spinning up a full tea.Program.
func drainInit(t *testing.T, m Model) Model {
	t.Helper()
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("Init returned nil cmd")
	}
	msg := cmd()
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func TestModel_InitLoadsStats(t *testing.T) {
	m, st := newTestModel(t)

	// Seed one memory so stats has something to count.
	ctx := context.Background()
	brainID, err := st.ResolveOrCreateBrainID(ctx, "enchanted-inn")
	if err != nil {
		t.Fatalf("resolve brain: %v", err)
	}
	m_input := memory.Memory{
		Project: "enchanted-inn",
		Scope:   memory.ScopeProject,
		Type:    memory.TypeScenePattern,
		Title:   "Test",
		Content: "test body",
	}
	if _, _, err := st.Save(ctx, brainID, m_input); err != nil {
		t.Fatalf("seed: %v", err)
	}

	loaded := drainInit(t, m)
	if !loaded.loaded {
		t.Errorf("Model should be loaded after Init drains")
	}
	if loaded.err != nil {
		t.Errorf("unexpected err: %v", loaded.err)
	}
	if loaded.stats.TotalMemories != 1 {
		t.Errorf("expected 1 memory in stats, got %d", loaded.stats.TotalMemories)
	}
}

func TestModel_QuitOnQ(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	final := updated.(Model)
	if !final.Quitting {
		t.Errorf("q should set Quitting=true")
	}
	if cmd == nil {
		t.Errorf("q should return a non-nil command (tea.Quit)")
	}
}

func TestModel_QuitOnCtrlC(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	final := updated.(Model)
	if !final.Quitting {
		t.Errorf("ctrl+c should set Quitting=true")
	}
	if cmd == nil {
		t.Errorf("ctrl+c should return tea.Quit")
	}
}

func TestModel_QuitOnEsc(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !updated.(Model).Quitting {
		t.Errorf("esc should set Quitting=true")
	}
}

func TestModel_RefreshOnR(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	final := updated.(Model)
	if !final.loading {
		t.Errorf("r should set loading=true while reload runs")
	}
	if cmd == nil {
		t.Errorf("r must return a reload command")
	}
}

func TestModel_WindowResizeStored(t *testing.T) {
	m, _ := newTestModel(t)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	final := updated.(Model)
	if final.width != 120 || final.height != 40 {
		t.Errorf("window size not stored, got %dx%d", final.width, final.height)
	}
}

func TestModel_StatsLoadedClearsLoading(t *testing.T) {
	m, _ := newTestModel(t)
	m.loading = true

	updated, _ := m.Update(statsLoadedMsg{stats: storage.Stats{TotalMemories: 7}})
	final := updated.(Model)
	if final.loading {
		t.Errorf("loading must be cleared after statsLoadedMsg")
	}
	if !final.loaded {
		t.Errorf("loaded must be set after first stats load")
	}
	if final.stats.TotalMemories != 7 {
		t.Errorf("stats not stored, got %d", final.stats.TotalMemories)
	}
}

func TestModel_StatsLoadedWithErrPropagates(t *testing.T) {
	m, _ := newTestModel(t)
	updated, _ := m.Update(statsLoadedMsg{err: errSentinel{}})
	final := updated.(Model)
	if final.err == nil {
		t.Errorf("err must propagate from statsLoadedMsg")
	}
}

func TestRoadmap_HasAllExpectedMilestones(t *testing.T) {
	got := Roadmap()
	want := []string{"M0", "M1", "M2", "M3", "M4", "M5", "M6"}
	if len(got) != len(want) {
		t.Fatalf("expected %d milestones, got %d", len(want), len(got))
	}
	for i, m := range got {
		if m.ID != want[i] {
			t.Errorf("milestone %d: got %s want %s", i, m.ID, want[i])
		}
	}
	// M0–M5 are done at v0.0.1.
	for _, m := range got[:6] {
		if m.Status != "done" {
			t.Errorf("milestone %s should be done, got %s", m.ID, m.Status)
		}
	}
	// M6 is deferred.
	if got[6].Status != "deferred" {
		t.Errorf("M6 should be deferred, got %s", got[6].Status)
	}
}

func TestStatusGlyph_KnownStatuses(t *testing.T) {
	tests := []struct {
		status string
		glyph  string
	}{
		{"done", "✓"},
		{"next", "→"},
		{"deferred", "·"},
		{"in-progress", "*"},
		{"unknown", "?"},
	}
	for _, tt := range tests {
		if got := StatusGlyph(tt.status); got != tt.glyph {
			t.Errorf("StatusGlyph(%q) = %q, want %q", tt.status, got, tt.glyph)
		}
	}
}

func TestView_RendersHeaderAndPanels(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)
	m.width = 120
	m.height = 40

	out := m.View()
	for _, want := range []string{
		"Thoughtline test",
		"enchanted-inn",
		"Stats",
		"Recent activity",
		"Roadmap",
		"M0",
		"M5",
		"M6",
		"[r] refresh",
		"[q] quit",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("view should contain %q\nFull view:\n%s", want, out)
		}
	}
}

func TestView_QuittingReturnsEmpty(t *testing.T) {
	m, _ := newTestModel(t)
	m.Quitting = true
	if got := m.View(); got != "" {
		t.Errorf("Quitting view should be empty, got %q", got)
	}
}

func TestView_LoadingShowsHint(t *testing.T) {
	m, _ := newTestModel(t)
	// Before drainInit, loaded=false and View should show a loading hint.
	if !strings.Contains(m.View(), "Loading") {
		t.Errorf("pre-load view should mention 'Loading'")
	}
}

// errSentinel implements error for testing the err path of statsLoadedMsg.
type errSentinel struct{}

func (errSentinel) Error() string { return "test error" }
