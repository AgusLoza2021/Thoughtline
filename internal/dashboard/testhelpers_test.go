package dashboard

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/pending"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// newWorkspaceStorage opens a fresh on-disk SQLite database in a temp dir for
// the test. Mirrors the shape of the other per-screen newTest* helpers but is
// generic enough to be shared across the new workspace tests.
//
// We use t.TempDir() (not :memory:) because FTS5 semantics differ between
// memory and file-backed SQLite, and several of the screens we exercise rely
// on FTS5.
func newWorkspaceStorage(t *testing.T) *storage.Storage {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "workspace.db")
	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// seedMemories inserts n memories into s with deterministic Type, TopicKey,
// Title, Content, and UpdatedAt values. Memories alternate between several
// gamedev types so filter tests can group them.
//
// Returns nothing; callers re-query the storage to read what they need.
// Implements task A6 from the tui-memory-workspace tasks.
func seedMemories(t *testing.T, s *storage.Storage, n int) {
	t.Helper()
	ctx := context.Background()
	project := "test-workspace"
	brainID, err := s.ResolveOrCreateBrainID(ctx, project)
	if err != nil {
		t.Fatalf("resolve brain: %v", err)
	}

	types := []memory.Type{
		memory.TypeScenePattern,
		memory.TypeBugfix,
		memory.TypePerfGotcha,
	}

	// Anchor "now" deterministically so relative-time renderers produce a
	// stable string in goldens.
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	for i := 0; i < n; i++ {
		typ := types[i%len(types)]
		m := memory.Memory{
			Project:   project,
			Scope:     memory.ScopeProject,
			Type:      typ,
			Title:     fmt.Sprintf("seed-%02d %s", i, typ),
			Content:   fmt.Sprintf("seeded content for memory #%d (type=%s)", i, typ),
			TopicKey:  fmt.Sprintf("seed/%s/%02d", typ, i),
			UpdatedAt: base.Add(time.Duration(i) * time.Minute),
			CreatedAt: base.Add(time.Duration(i) * time.Minute),
		}
		if _, _, err := s.Save(ctx, brainID, m); err != nil {
			t.Fatalf("seed memory %d: %v", i, err)
		}
	}
}

// seedPending inserts n pending capture events with distinct proposed types
// into s. Used by Inbox-tab integration tests.
func seedPending(t *testing.T, s *storage.Storage, n int) {
	t.Helper()
	ctx := context.Background()
	project := "test-workspace"
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	for i := 0; i < n; i++ {
		ev := pending.Event{
			SyncID:     fmt.Sprintf("pending-sync-%04d", i),
			Project:    project,
			EventType:  fmt.Sprintf("Proposed-%d", i),
			Payload:    fmt.Sprintf(`{"index":%d,"note":"seeded pending row"}`, i),
			Hash:       fmt.Sprintf("hash%04d", i),
			Status:     pending.StatusPending,
			CapturedAt: base.Add(time.Duration(i) * time.Minute),
			CreatedAt:  base.Add(time.Duration(i) * time.Minute),
		}
		if _, err := s.InsertPending(ctx, ev); err != nil {
			t.Fatalf("seed pending %d: %v", i, err)
		}
	}
}

// TestSeedHelpers_Smoke is the helper-of-helpers test. It proves seedMemories
// and seedPending each leave the expected row count behind, so later tests can
// rely on the helpers without re-asserting the contract.
func TestSeedHelpers_Smoke(t *testing.T) {
	t.Run("seedMemories writes n rows", func(t *testing.T) {
		st := newWorkspaceStorage(t)
		seedMemories(t, st, 5)
		stats, err := st.Stats(context.Background(), storage.StatsOptions{Project: "test-workspace"})
		if err != nil {
			t.Fatalf("stats: %v", err)
		}
		if stats.TotalMemories != 5 {
			t.Errorf("after seedMemories(5), TotalMemories = %d, want 5", stats.TotalMemories)
		}
	})

	t.Run("seedPending writes n pending rows", func(t *testing.T) {
		st := newWorkspaceStorage(t)
		seedPending(t, st, 3)
		n, err := st.CountPending(context.Background(), "test-workspace")
		if err != nil {
			t.Fatalf("count pending: %v", err)
		}
		if n != 3 {
			t.Errorf("after seedPending(3), CountPending = %d, want 3", n)
		}
	})
}
