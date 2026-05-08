package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// newTestStorage opens a fresh on-disk SQLite (in t.TempDir) so the FTS5
// virtual table behaves identically to production. We deliberately avoid
// ":memory:" because some FTS5 edge cases differ in shared-cache memory mode.
func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// defaultBrainID creates (or returns cached) the "enchanted-inn" brain for a
// test storage. All helpers that accept sampleMemory() use this brain.
func defaultBrainID(t *testing.T, st *Storage) int64 {
	t.Helper()
	id, err := st.ResolveOrCreateBrainID(context.Background(), "enchanted-inn")
	if err != nil {
		t.Fatalf("defaultBrainID: %v", err)
	}
	return id
}

func sampleMemory() memory.Memory {
	return memory.Memory{
		Project: "enchanted-inn",
		Scope:   memory.ScopeProject,
		Type:    memory.TypeScenePattern,
		Title:   "Inn cellar entity hierarchy",
		Content: "world/static for chairs and walls; world/interactive for the cellar door.",
	}
}

func TestSave_InsertWithoutTopicKey(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	saved, action, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if action != ActionCreated {
		t.Fatalf("expected ActionCreated, got %s", action)
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero id")
	}
	if saved.SyncID == "" {
		t.Errorf("expected non-empty sync_id")
	}
	if saved.RevisionCount != 0 {
		t.Errorf("first save should have revision_count=0, got %d", saved.RevisionCount)
	}
	if saved.CreatedAt.IsZero() || saved.UpdatedAt.IsZero() {
		t.Errorf("timestamps must be populated")
	}
}

func TestSave_UpsertByTopicKey_FirstSaveIsCreate(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.TopicKey = "scene/playcanvas/inn-cellar"

	saved, action, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if action != ActionCreated {
		t.Fatalf("first save with topic_key should be ActionCreated, got %s", action)
	}
	if saved.RevisionCount != 0 {
		t.Errorf("expected revision_count=0, got %d", saved.RevisionCount)
	}
}

func TestSave_UpsertByTopicKey_DifferentContentIsUpdate(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	first := sampleMemory()
	first.TopicKey = "scene/playcanvas/inn-cellar"
	saved1, _, err := st.Save(ctx, brainID, first)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}

	// Force a non-trivial gap so updated_at != created_at.
	st.SetClock(func() time.Time { return saved1.CreatedAt.Add(2 * time.Second) })

	second := first
	second.Content = first.Content + "\nUpdated: switched lighting batch group."
	saved2, action, err := st.Save(ctx, brainID, second)
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if action != ActionUpdated {
		t.Fatalf("expected ActionUpdated, got %s", action)
	}
	if saved2.ID != saved1.ID {
		t.Errorf("upsert must keep the same id, got %d → %d", saved1.ID, saved2.ID)
	}
	if saved2.SyncID != saved1.SyncID {
		t.Errorf("upsert must keep the same sync_id, got %q → %q", saved1.SyncID, saved2.SyncID)
	}
	if !saved2.CreatedAt.Equal(saved1.CreatedAt) {
		t.Errorf("created_at must be preserved on upsert, got %v → %v", saved1.CreatedAt, saved2.CreatedAt)
	}
	if !saved2.UpdatedAt.After(saved1.UpdatedAt) {
		t.Errorf("updated_at must advance, got %v → %v", saved1.UpdatedAt, saved2.UpdatedAt)
	}
	if saved2.RevisionCount != 1 {
		t.Errorf("revision_count must bump to 1, got %d", saved2.RevisionCount)
	}
}

func TestSave_UpsertByTopicKey_IdenticalContentIsNoop(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.TopicKey = "scene/playcanvas/inn-cellar"
	saved1, _, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}

	saved2, action, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if action != ActionNoop {
		t.Fatalf("expected ActionNoop for identical content, got %s", action)
	}
	if saved2.ID != saved1.ID {
		t.Errorf("noop must return the same id, got %d → %d", saved1.ID, saved2.ID)
	}
	if saved2.RevisionCount != 0 {
		t.Errorf("noop must not bump revision_count, got %d", saved2.RevisionCount)
	}
}

func TestSave_DifferentTopicKeysCreateSeparateRows(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	a := sampleMemory()
	a.TopicKey = "scene/playcanvas/inn-cellar"
	b := sampleMemory()
	b.TopicKey = "scene/playcanvas/inn-courtyard"
	b.Content = "different content for the courtyard"

	savedA, actA, err := st.Save(ctx, brainID, a)
	if err != nil {
		t.Fatalf("save a: %v", err)
	}
	savedB, actB, err := st.Save(ctx, brainID, b)
	if err != nil {
		t.Fatalf("save b: %v", err)
	}

	if actA != ActionCreated || actB != ActionCreated {
		t.Fatalf("both should be created; got %s, %s", actA, actB)
	}
	if savedA.ID == savedB.ID {
		t.Errorf("different topic_keys must yield different rows")
	}
	if savedA.SyncID == savedB.SyncID {
		t.Errorf("different topic_keys must yield different sync_ids")
	}
}

func TestSave_DifferentProjectsAllowSameTopicKey(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	brainA, err := st.ResolveOrCreateBrainID(ctx, "project-alpha")
	if err != nil {
		t.Fatalf("brain-alpha: %v", err)
	}
	brainB, err := st.ResolveOrCreateBrainID(ctx, "project-beta")
	if err != nil {
		t.Fatalf("brain-beta: %v", err)
	}

	a := sampleMemory()
	a.Project = "project-alpha"
	a.TopicKey = "convention/asset-naming"

	b := sampleMemory()
	b.Project = "project-beta"
	b.TopicKey = "convention/asset-naming"

	if _, _, err := st.Save(ctx, brainA, a); err != nil {
		t.Fatalf("save a: %v", err)
	}
	if _, _, err := st.Save(ctx, brainB, b); err != nil {
		t.Fatalf("save b: %v", err)
	}

	// Both should have been saved without conflict.
	row, err := st.db.QueryContext(ctx,
		`SELECT count(*) FROM memories WHERE topic_key = 'convention/asset-naming'`)
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	defer row.Close()
	if !row.Next() {
		t.Fatalf("no rows returned by count")
	}
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 rows across projects, got %d", n)
	}
}

func TestSave_FTSStaysInSync(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	// 1. Insert — FTS row count should grow.
	if n, err := st.FTSCount(ctx); err != nil || n != 0 {
		t.Fatalf("expected empty FTS, got %d err %v", n, err)
	}

	first := sampleMemory()
	first.TopicKey = "scene/playcanvas/inn-cellar"
	first.Content = "lantern-base texture, bloom-safe import"
	if _, _, err := st.Save(ctx, brainID, first); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := st.FTSContains(ctx, "lantern")
	if err != nil {
		t.Fatalf("fts contains: %v", err)
	}
	if !got {
		t.Errorf("FTS should contain 'lantern' after insert")
	}

	// 2. Update — old content must be evicted, new content searchable.
	updated := first
	updated.Content = "tea kettle prop, alpha mask import"
	if _, _, err := st.Save(ctx, brainID, updated); err != nil {
		t.Fatalf("save update: %v", err)
	}

	gotOld, err := st.FTSContains(ctx, "lantern")
	if err != nil {
		t.Fatalf("fts contains old: %v", err)
	}
	if gotOld {
		t.Errorf("FTS should NOT contain 'lantern' after update")
	}
	gotNew, err := st.FTSContains(ctx, "kettle")
	if err != nil {
		t.Fatalf("fts contains new: %v", err)
	}
	if !gotNew {
		t.Errorf("FTS should contain 'kettle' after update")
	}
}

func TestSave_PersistsTags(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.Tags = []string{"engine:playcanvas", "platform:android"}
	saved, _, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := st.GetByID(ctx, brainID, saved.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "engine:playcanvas" || got.Tags[1] != "platform:android" {
		t.Errorf("tags not persisted/round-tripped correctly: got %v", got.Tags)
	}
}

func TestOpen_PersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "persist.db")
	ctx := context.Background()

	st1, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("open 1: %v", err)
	}
	brainID, err := st1.ResolveOrCreateBrainID(ctx, "enchanted-inn")
	if err != nil {
		t.Fatalf("brain: %v", err)
	}
	saved, _, err := st1.Save(ctx, brainID, sampleMemory())
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := st1.Close(); err != nil {
		t.Fatalf("close 1: %v", err)
	}

	st2, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("open 2: %v", err)
	}
	defer st2.Close()
	got, err := st2.GetByID(ctx, brainID, saved.ID)
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if got.SyncID != saved.SyncID {
		t.Errorf("sync_id must persist across reopen, got %q vs %q", got.SyncID, saved.SyncID)
	}
}
