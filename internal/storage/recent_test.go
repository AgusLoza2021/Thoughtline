package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRecent_OrdersByUpdatedAtDesc(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	// Lock the clock so we control updated_at deterministically.
	t0 := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	st.SetClock(func() time.Time { return t0 })

	a := sampleMemory()
	a.TopicKey = "scene/oldest"
	a.Title = "Oldest"
	seed(t, st, a)

	st.SetClock(func() time.Time { return t0.Add(1 * time.Hour) })
	b := sampleMemory()
	b.TopicKey = "scene/middle"
	b.Title = "Middle"
	seed(t, st, b)

	st.SetClock(func() time.Time { return t0.Add(2 * time.Hour) })
	c := sampleMemory()
	c.TopicKey = "scene/newest"
	c.Title = "Newest"
	seed(t, st, c)

	results, err := st.Recent(ctx, a.Project, 0) // 0 = use default limit
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Title != "Newest" || results[1].Title != "Middle" || results[2].Title != "Oldest" {
		t.Errorf("expected Newest, Middle, Oldest order; got %q, %q, %q",
			results[0].Title, results[1].Title, results[2].Title)
	}
}

func TestRecent_FiltersByProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	a := sampleMemory()
	a.Project = "alpha"
	a.TopicKey = "scene/a"
	seed(t, st, a)

	b := sampleMemory()
	b.Project = "beta"
	b.TopicKey = "scene/b"
	seed(t, st, b)

	results, err := st.Recent(ctx, "alpha", 0)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(results) != 1 || results[0].Project != "alpha" {
		t.Errorf("project filter must return only alpha rows, got %+v", results)
	}
}

func TestRecent_RequiresProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	if _, err := st.Recent(ctx, "", 0); err == nil {
		t.Errorf("empty project must return an error to prevent cross-project leak")
	}
}

func TestRecent_ExcludesSoftDeleted(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	keep := sampleMemory()
	keep.TopicKey = "scene/keep"
	keep.Title = "Keep"
	seed(t, st, keep)

	gone := sampleMemory()
	gone.TopicKey = "scene/gone"
	gone.Title = "Gone"
	saved := seed(t, st, gone)

	if err := st.SoftDelete(ctx, saved.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	results, err := st.Recent(ctx, keep.Project, 0)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("soft-deleted row must be excluded, got %d results", len(results))
	}
	if results[0].Title != "Keep" {
		t.Errorf("expected Keep, got %q", results[0].Title)
	}
}

func TestRecent_LimitDefaultAndClamp(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	for i := 0; i < 60; i++ {
		m := sampleMemory()
		m.TopicKey = "scene/" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		seed(t, st, m)
	}

	// Default limit (limit=0) → 10.
	def, err := st.Recent(ctx, "enchanted-inn", 0)
	if err != nil {
		t.Fatalf("recent default: %v", err)
	}
	if len(def) != 10 {
		t.Errorf("default limit must be 10, got %d", len(def))
	}

	// Oversized limit (999) → clamped to 50.
	clamped, err := st.Recent(ctx, "enchanted-inn", 999)
	if err != nil {
		t.Fatalf("recent clamped: %v", err)
	}
	if len(clamped) != 50 {
		t.Errorf("limit must clamp at 50, got %d", len(clamped))
	}
}

func TestRecent_SnippetAndMetadataPopulated(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	long := "first part of the body. " + // non-trivial content so snippet is meaningful
		"middle part of the body. last part of the body."
	m := sampleMemory()
	m.TopicKey = "scene/snip"
	m.Title = "Snippet check"
	m.Content = long
	m.Tags = []string{"engine:playcanvas"}
	seed(t, st, m)

	results, err := st.Recent(ctx, m.Project, 0)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Snippet == "" {
		t.Errorf("snippet must be populated from content")
	}
	if len(r.Tags) != 1 || r.Tags[0] != "engine:playcanvas" {
		t.Errorf("tags must round-trip, got %v", r.Tags)
	}
	if r.SyncID == "" || r.UpdatedAt.IsZero() {
		t.Errorf("metadata must be populated, got sync_id=%q updated=%v", r.SyncID, r.UpdatedAt)
	}
}

func TestSearch_ExcludesSoftDeletedRegression(t *testing.T) {
	// Regression guard: M2's Search must also honor deleted_at IS NULL.
	st := newTestStorage(t)
	ctx := context.Background()

	m := sampleMemory()
	m.TopicKey = "scene/lantern"
	m.Title = "Lantern"
	m.Content = "lantern lantern lantern"
	saved := seed(t, st, m)

	if err := st.SoftDelete(ctx, saved.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	// FTS path must also exclude.
	res, err := st.Search(ctx, "lantern", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("FTS search must exclude soft-deleted rows, got %d", len(res))
	}

	// Topic-key shortcut path must also exclude.
	res2, err := st.Search(ctx, "scene/lantern", SearchOptions{})
	if err != nil {
		t.Fatalf("search shortcut: %v", err)
	}
	if len(res2) != 0 {
		t.Errorf("topic-key shortcut must exclude soft-deleted rows, got %d", len(res2))
	}
}

func TestRecent_ReturnsEmptyForUnknownProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	seed(t, st, sampleMemory())

	results, err := st.Recent(ctx, "nonexistent-project", 0)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("unknown project should return empty, got %d", len(results))
	}
}

// Sentinel guard: ErrMemoryNotFound must exist and be a stable, exported value
// so callers (and the server layer) can errors.Is against it.
func TestErrMemoryNotFound_IsExported(t *testing.T) {
	if ErrMemoryNotFound == nil {
		t.Fatalf("ErrMemoryNotFound must be a non-nil sentinel")
	}
	if !errors.Is(ErrMemoryNotFound, ErrMemoryNotFound) {
		t.Errorf("ErrMemoryNotFound must satisfy errors.Is against itself")
	}
}
