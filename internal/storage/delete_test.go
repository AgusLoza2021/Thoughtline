package storage

import (
	"context"
	"errors"
	"testing"
)

func TestSoftDelete_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seed(t, st, sampleMemory())

	if err := st.SoftDelete(ctx, saved.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	got, err := st.GetByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if got.DeletedAt == nil {
		t.Errorf("deleted_at must be populated after SoftDelete")
	}
}

func TestSoftDelete_RemovedFromSearch(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	m := sampleMemory()
	m.Title = "Lantern bake"
	m.Content = "lantern lantern lantern"
	saved := seed(t, st, m)

	// Sanity: search finds it before delete.
	pre, err := st.Search(ctx, "lantern", SearchOptions{})
	if err != nil {
		t.Fatalf("pre search: %v", err)
	}
	if len(pre) != 1 {
		t.Fatalf("expected 1 hit pre-delete, got %d", len(pre))
	}

	if err := st.SoftDelete(ctx, saved.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	post, err := st.Search(ctx, "lantern", SearchOptions{})
	if err != nil {
		t.Fatalf("post search: %v", err)
	}
	if len(post) != 0 {
		t.Errorf("soft-deleted row must not appear in search, got %d", len(post))
	}
}

func TestSoftDelete_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	err := st.SoftDelete(ctx, 999)
	if err == nil {
		t.Fatalf("expected error for unknown id, got nil")
	}
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Errorf("missing id must wrap ErrMemoryNotFound, got %v", err)
	}
}

func TestSoftDelete_SecondCallIsNotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seed(t, st, sampleMemory())

	if err := st.SoftDelete(ctx, saved.ID); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	// Second delete on an already-deleted row: row is no longer "active",
	// so we treat it as not found rather than silently succeeding. This
	// keeps the contract symmetric with UpdateByID on soft-deleted rows.
	err := st.SoftDelete(ctx, saved.ID)
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Errorf("repeat delete must wrap ErrMemoryNotFound, got %v", err)
	}
}

func TestSoftDelete_FreesTopicKeyForNewSave(t *testing.T) {
	// The unique index on (project, topic_key) is partial: WHERE topic_key
	// IS NOT NULL AND deleted_at IS NULL. So once a row is soft-deleted,
	// the same topic_key should be reusable for a fresh insert.
	st := newTestStorage(t)
	ctx := context.Background()

	first := sampleMemory()
	first.TopicKey = "scene/reusable"
	first.Title = "Original"
	saved := seed(t, st, first)

	if err := st.SoftDelete(ctx, saved.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	second := sampleMemory()
	second.TopicKey = "scene/reusable"
	second.Title = "Replacement"
	second.Content = "different content for the new row"
	replacement, action, err := st.Save(ctx, second)
	if err != nil {
		t.Fatalf("re-save after soft delete: %v", err)
	}
	if action != ActionCreated {
		t.Errorf("re-save after soft delete should be ActionCreated, got %s", action)
	}
	if replacement.ID == saved.ID {
		t.Errorf("re-save must produce a new row, got same ID %d", replacement.ID)
	}
}
