package dashboard

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// P-series — golden snapshot tests for every tab/screen at the canonical
// 100×30 viewport. Goldens live in testdata/<name>.golden and are
// normalized via normalizeForGolden (ANSI stripped, relative-time tokens
// replaced with "<reltime>") so they are deterministic across machines
// and time zones.
//
// Regenerate after intentional style/layout changes:
//   go test ./internal/dashboard/ -run "Golden" -update

const goldenW, goldenH = 100, 30

// seedMemoriesDeterministic seeds n memories with a 2ms sleep between Save
// calls so SQLite's millisecond-precision UpdatedAt yields a strict
// chronological order. Used only by the golden tests where row order is
// part of the snapshot.
func seedMemoriesDeterministic(t *testing.T, st *storage.Storage, n int) {
	t.Helper()
	ctx := context.Background()
	project := "test-workspace"
	brainID, err := st.ResolveOrCreateBrainID(ctx, project)
	if err != nil {
		t.Fatalf("resolve brain: %v", err)
	}
	types := []memory.Type{
		memory.TypeScenePattern,
		memory.TypeBugfix,
		memory.TypePerfGotcha,
	}
	for i := 0; i < n; i++ {
		typ := types[i%len(types)]
		m := memory.Memory{
			Project:  project,
			Scope:    memory.ScopeProject,
			Type:     typ,
			Title:    "seed-" + intToStr(i),
			Content:  "seeded content for memory #" + intToStr(i),
			TopicKey: "seed/" + string(typ) + "/" + intToStr(i),
		}
		if _, _, err := st.Save(ctx, brainID, m); err != nil {
			t.Fatalf("seed memory %d: %v", i, err)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func intToStr(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "00"
	}
	var buf [16]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}
	if len(buf)-pos == 1 {
		pos--
		buf[pos] = '0'
	}
	return string(buf[pos:])
}

// drainCmd resolves a tea.Cmd to its first message and feeds it back to the
// screen so async data loads complete synchronously for the golden capture.
func drainCmd(t *testing.T, s Screen, cmd tea.Cmd) Screen {
	t.Helper()
	if cmd == nil {
		return s
	}
	msg := cmd()
	if msg == nil {
		return s
	}
	updated, _ := s.Update(msg)
	return updated
}

// P10 — brand line.
func TestGolden_Brand(t *testing.T) {
	out := renderBrand(defaultPalette)
	assertGolden(t, "brand", out)
}

// P2 — empty Home tab (no memories).
func TestGolden_HomeEmpty(t *testing.T) {
	st := newWorkspaceStorage(t)
	s := NewHomeScreen(st)
	s = drainCmd(t, s, s.Init()).(*HomeScreen)
	assertGolden(t, "home_empty", s.View(goldenW, goldenH, defaultPalette))
}

// P1 — Home tab with seeded data.
func TestGolden_HomeDefault(t *testing.T) {
	st := newWorkspaceStorage(t)
	seedMemoriesDeterministic(t, st, 10)
	s := NewHomeScreen(st)
	s = drainCmd(t, s, s.Init()).(*HomeScreen)
	assertGolden(t, "home_default", s.View(goldenW, goldenH, defaultPalette))
}

// P3 — Memories tab page 1, no filters, with seeded rows.
func TestGolden_MemoriesPage1(t *testing.T) {
	st := newWorkspaceStorage(t)
	seedMemoriesDeterministic(t, st, 20)
	s := NewMemoriesScreen(st)
	s = drainCmd(t, s, s.Init()).(*MemoriesScreen)
	assertGolden(t, "memories_page1", s.View(goldenW, goldenH, defaultPalette))
}

// P4 — Memories tab with type filter applied.
func TestGolden_MemoriesFiltered(t *testing.T) {
	st := newWorkspaceStorage(t)
	seedMemoriesDeterministic(t, st, 20)
	s := NewMemoriesScreen(st)
	s = drainCmd(t, s, s.Init()).(*MemoriesScreen)
	// Apply a hard-coded filter via direct struct mutation so the golden
	// has a deterministic filter row independent of UI key choreography.
	s.filter.Type = "perf-gotcha"
	s = drainCmd(t, s, s.loadCmd()).(*MemoriesScreen)
	assertGolden(t, "memories_filtered", s.View(goldenW, goldenH, defaultPalette))
}

// P5 — Inbox tab with three pending captures.
func TestGolden_InboxThree(t *testing.T) {
	st := newWorkspaceStorage(t)
	seedPending(t, st, 3)
	s := NewInboxScreen(st)
	s = drainCmd(t, s, s.Init()).(*InboxScreen)
	assertGolden(t, "inbox_three", s.View(goldenW, goldenH, defaultPalette))
}

// P6 — InboxEditScreen pre-filled with type, title, body.
func TestGolden_InboxEditForm(t *testing.T) {
	s := NewInboxEditScreen(42, "decision", "Choose WAL for SQLite",
		"We pick WAL journal mode because concurrent MCP readers + writer\nneed shared access without lock errors.")
	assertGolden(t, "inbox_edit_form", s.View(goldenW, goldenH, defaultPalette))
}

// P7 — Memory Detail view with multi-paragraph body.
func TestGolden_Detail(t *testing.T) {
	body := "[decision] decision/sqlite-wal\n\n" +
		"**What**: Enable WAL via DSN pragma.\n" +
		"**Why**: Concurrent MCP tool calls need readers + writer simultaneously.\n" +
		"**Where**: internal/storage/storage.go\n" +
		"**Learned**: busy_timeout=5000 prevents lock errors under load."
	s := newDetailScreenFull(body)
	assertGolden(t, "detail", s.View(goldenW, goldenH, defaultPalette))
}

// P8 — Help tab.
func TestGolden_Help(t *testing.T) {
	s := NewHelpScreen()
	assertGolden(t, "help", s.View(goldenW, goldenH, defaultPalette))
}

// P9 — Sessions tab with mixed open + closed sessions.
func TestGolden_Sessions(t *testing.T) {
	st := newWorkspaceStorage(t)
	ctx := context.Background()
	if _, err := st.StartSession(ctx, "alpha", "claude-code"); err != nil {
		t.Fatalf("start alpha: %v", err)
	}
	// Sleep to ensure StartedAt differs between sessions so the chronological
	// "most recent first" order is deterministic in the golden.
	time.Sleep(5 * time.Millisecond)
	sess2, err := st.StartSession(ctx, "bravo", "")
	if err != nil {
		t.Fatalf("start bravo: %v", err)
	}
	if _, err := st.EndSession(ctx, sess2.ID, "wrap up"); err != nil {
		t.Fatalf("end bravo: %v", err)
	}
	s := NewSessionsScreen(st)
	s = drainCmd(t, s, s.Init()).(*SessionsScreen)
	assertGolden(t, "sessions", s.View(goldenW, goldenH, defaultPalette))
}
