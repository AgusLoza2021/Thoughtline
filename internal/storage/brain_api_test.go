package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// setupBrain creates a brain via direct SQL and returns its id.
// Used by Phase 4 tests until ResolveOrCreateBrainID is available.
func setupBrain(t *testing.T, st *Storage, slug string) int64 {
	t.Helper()
	ctx := context.Background()
	now := st.nowMillis().UnixMilli()
	res, err := st.db.ExecContext(ctx, `
		INSERT INTO brains (slug, display_name, kind, description, config_json, created_at, updated_at)
		VALUES (?, ?, 'real', '', '{}', ?, ?)`,
		slug, slug, now, now,
	)
	if err != nil {
		t.Fatalf("setupBrain insert: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("setupBrain lastInsertId: %v", err)
	}
	return id
}

// seedWithBrain inserts a memory scoped to a specific brain.
func seedWithBrain(t *testing.T, st *Storage, brainID int64, m memory.Memory) memory.Memory {
	t.Helper()
	saved, _, err := st.Save(context.Background(), brainID, m)
	if err != nil {
		t.Fatalf("seedWithBrain save: %v", err)
	}
	return saved
}

// ---- Task 4.1 ----

func TestStorage_SaveRequiresBrainID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	m := sampleMemory()
	_, _, err := st.Save(ctx, 0, m)
	if err == nil {
		t.Fatal("expected error when brainID=0, got nil")
	}
	if !errors.Is(err, ErrBrainRequired) {
		t.Errorf("expected ErrBrainRequired, got %v", err)
	}

	// Verify no row was inserted.
	var count int
	_ = st.db.QueryRowContext(ctx, `SELECT count(*) FROM memories`).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 rows inserted on error, got %d", count)
	}
}

// ---- Task 4.4 ----

func TestStorage_SearchScopedToBrain(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	brainA := setupBrain(t, st, "brain-a")
	brainB := setupBrain(t, st, "brain-b")

	mA := memory.Memory{
		Project: "brain-a",
		Scope:   memory.ScopeProject,
		Type:    memory.TypeDecision,
		Title:   "Dragon boss design",
		Content: "The dragon should charge the player when health is low.",
	}
	mB := memory.Memory{
		Project: "brain-b",
		Scope:   memory.ScopeProject,
		Type:    memory.TypeDecision,
		Title:   "Wizard spell design",
		Content: "The wizard casts fireball when player is in range.",
	}
	seedWithBrain(t, st, brainA, mA)
	seedWithBrain(t, st, brainB, mB)

	results, err := st.Search(ctx, brainA, "design", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for brainA, got %d", len(results))
	}
	if results[0].Project != "brain-a" {
		t.Errorf("expected brain-a result, got project=%q", results[0].Project)
	}
}

// ---- Task 4.6 ----

func TestStorage_RecentScopedToBrain(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	brainA := setupBrain(t, st, "proj-a")
	brainB := setupBrain(t, st, "proj-b")

	mA := memory.Memory{
		Project: "proj-a",
		Scope:   memory.ScopeProject,
		Type:    memory.TypeDecision,
		Title:   "Alpha memory",
		Content: "This belongs to brain A.",
	}
	mB := memory.Memory{
		Project: "proj-b",
		Scope:   memory.ScopeProject,
		Type:    memory.TypeDecision,
		Title:   "Beta memory",
		Content: "This belongs to brain B.",
	}
	seedWithBrain(t, st, brainA, mA)
	seedWithBrain(t, st, brainB, mB)

	results, err := st.Recent(ctx, brainA, 10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for brainA, got %d", len(results))
	}
	for _, r := range results {
		if r.Project != "proj-a" {
			t.Errorf("expected only proj-a rows, got project=%q", r.Project)
		}
	}
}

// ---- Task 4.8 ----

func TestStorage_GetByID_CrossBrain(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	brainA := setupBrain(t, st, "scope-a")
	brainB := setupBrain(t, st, "scope-b")

	mB := memory.Memory{
		Project: "scope-b",
		Scope:   memory.ScopeProject,
		Type:    memory.TypeDecision,
		Title:   "BrainB memory",
		Content: "This is brain B data.",
	}
	savedB := seedWithBrain(t, st, brainB, mB)

	// Try to access brainB's memory using brainA's id — must return ErrMemoryNotFound.
	_, err := st.GetByID(ctx, brainA, savedB.ID)
	if err == nil {
		t.Fatal("expected error when accessing cross-brain memory, got nil")
	}
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Errorf("expected ErrMemoryNotFound, got %v", err)
	}
}

// ---- Task 4.11 ----

func TestResolveBrainID_Cache(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	slug := "cache-test"
	_ = setupBrain(t, st, slug)

	// First resolution — should hit DB.
	id1, err := st.ResolveBrainID(ctx, slug)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	if id1 == 0 {
		t.Error("expected non-zero brain id")
	}

	// Second resolution of same slug — should hit cache (same result, no error).
	id2, err := st.ResolveBrainID(ctx, slug)
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if id1 != id2 {
		t.Errorf("cache miss: id changed from %d to %d", id1, id2)
	}

	// Verify cache contains the slug.
	st.brains.mu.RLock()
	cachedID, ok := st.brains.m[slug]
	st.brains.mu.RUnlock()
	if !ok {
		t.Error("expected slug to be in cache after second resolve")
	}
	if cachedID != id1 {
		t.Errorf("cached id %d != resolved id %d", cachedID, id1)
	}
}

// ---- Task 4.12 ----

func TestResolveBrainID_Invalidate(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	slug := "to-archive"
	brainID := setupBrain(t, st, slug)

	// Prime the cache.
	_, err := st.ResolveBrainID(ctx, slug)
	if err != nil {
		t.Fatalf("prime cache: %v", err)
	}

	// Archive the brain.
	now := st.nowMillis().UnixMilli()
	_, err = st.db.ExecContext(ctx,
		`UPDATE brains SET archived_at = ? WHERE id = ?`, now, brainID)
	if err != nil {
		t.Fatalf("archive brain: %v", err)
	}

	// Invalidate cache.
	st.InvalidateBrainCache(slug)

	// Resolve again — archived brain must not be found.
	_, err = st.ResolveBrainID(ctx, slug)
	if err == nil {
		t.Fatal("expected error for archived brain after invalidation, got nil")
	}
}
