package storage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// strPtr / tagsPtr are tiny helpers — UpdatePatch fields are pointers so the
// caller can distinguish "leave unchanged" (nil) from "set to empty" (&"").
func strPtr(s string) *string         { return &s }
func tagsPtr(t []string) *[]string    { return &t }

func TestUpdateByID_TitleOnly_PreservesContent(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seed(t, st, sampleMemory())

	updated, err := st.UpdateByID(ctx, saved.ID, UpdatePatch{
		Title: strPtr("Renamed title"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != "Renamed title" {
		t.Errorf("title should change, got %q", updated.Title)
	}
	if updated.Content != saved.Content {
		t.Errorf("content must be unchanged when only title patched, got %q", updated.Content)
	}
}

func TestUpdateByID_BumpsRevisionPreservesIdentity(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	t0 := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	st.SetClock(func() time.Time { return t0 })

	saved := seed(t, st, sampleMemory())
	if saved.RevisionCount != 0 {
		t.Fatalf("seed memory should start at revision 0")
	}

	st.SetClock(func() time.Time { return t0.Add(2 * time.Second) })

	updated, err := st.UpdateByID(ctx, saved.ID, UpdatePatch{
		Content: strPtr("new body content for the update path"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if updated.ID != saved.ID {
		t.Errorf("ID must be preserved, got %d → %d", saved.ID, updated.ID)
	}
	if updated.SyncID != saved.SyncID {
		t.Errorf("SyncID must be preserved, got %q → %q", saved.SyncID, updated.SyncID)
	}
	if !updated.CreatedAt.Equal(saved.CreatedAt) {
		t.Errorf("created_at must be preserved, got %v → %v", saved.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(saved.UpdatedAt) {
		t.Errorf("updated_at must advance, got %v → %v", saved.UpdatedAt, updated.UpdatedAt)
	}
	if updated.RevisionCount != 1 {
		t.Errorf("revision_count must bump to 1, got %d", updated.RevisionCount)
	}
	// Identity-defining fields must NOT change via UpdateByID.
	if updated.Type != saved.Type {
		t.Errorf("type must not change via UpdateByID, got %q → %q", saved.Type, updated.Type)
	}
	if updated.TopicKey != saved.TopicKey {
		t.Errorf("topic_key must not change via UpdateByID, got %q → %q", saved.TopicKey, updated.TopicKey)
	}
	if updated.Project != saved.Project {
		t.Errorf("project must not change via UpdateByID, got %q → %q", saved.Project, updated.Project)
	}
	if updated.Scope != saved.Scope {
		t.Errorf("scope must not change via UpdateByID, got %q → %q", saved.Scope, updated.Scope)
	}
}

func TestUpdateByID_RefreshesFTSIndex(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	m := sampleMemory()
	m.Title = "Lantern bake"
	m.Content = "old content about lanterns"
	saved := seed(t, st, m)

	if _, err := st.UpdateByID(ctx, saved.ID, UpdatePatch{
		Content: strPtr("new content about kettles only"),
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Old token "lanterns" must no longer be in the FTS index for content.
	hasOld, err := st.FTSContains(ctx, "lanterns")
	if err != nil {
		t.Fatalf("fts contains old: %v", err)
	}
	if hasOld {
		t.Errorf("FTS must not contain old content token after update")
	}

	hasNew, err := st.FTSContains(ctx, "kettles")
	if err != nil {
		t.Fatalf("fts contains new: %v", err)
	}
	if !hasNew {
		t.Errorf("FTS must contain new content token after update")
	}
}

func TestUpdateByID_TagsRoundTrip(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seed(t, st, sampleMemory())

	// Replace tags entirely.
	updated, err := st.UpdateByID(ctx, saved.ID, UpdatePatch{
		Tags: tagsPtr([]string{"engine:playcanvas", "platform:android"}),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(updated.Tags) != 2 {
		t.Errorf("expected 2 tags, got %v", updated.Tags)
	}

	// Empty slice = "remove all tags" (distinct from nil = "no change").
	cleared, err := st.UpdateByID(ctx, saved.ID, UpdatePatch{
		Tags: tagsPtr([]string{}),
	})
	if err != nil {
		t.Fatalf("update clear: %v", err)
	}
	if len(cleared.Tags) != 0 {
		t.Errorf("empty tags slice should clear tags, got %v", cleared.Tags)
	}
}

func TestUpdateByID_NoOpPatchIsNoop(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seed(t, st, sampleMemory())

	updated, err := st.UpdateByID(ctx, saved.ID, UpdatePatch{}) // all nil
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.RevisionCount != saved.RevisionCount {
		t.Errorf("empty patch must NOT bump revision_count, got %d → %d",
			saved.RevisionCount, updated.RevisionCount)
	}
	if !updated.UpdatedAt.Equal(saved.UpdatedAt) {
		t.Errorf("empty patch must NOT bump updated_at, got %v → %v",
			saved.UpdatedAt, updated.UpdatedAt)
	}
}

func TestUpdateByID_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	_, err := st.UpdateByID(ctx, 999, UpdatePatch{Title: strPtr("nope")})
	if err == nil {
		t.Fatalf("expected error for missing id, got nil")
	}
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Errorf("missing id must wrap ErrMemoryNotFound, got %v", err)
	}
}

func TestUpdateByID_RejectsSoftDeleted(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seed(t, st, sampleMemory())
	if err := st.SoftDelete(ctx, saved.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	_, err := st.UpdateByID(ctx, saved.ID, UpdatePatch{Title: strPtr("nope")})
	if err == nil {
		t.Fatalf("expected error updating a soft-deleted row, got nil")
	}
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Errorf("update on soft-deleted should wrap ErrMemoryNotFound, got %v", err)
	}
}

func TestUpdateByID_ValidatesResultingMemory(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seed(t, st, sampleMemory())

	_, err := st.UpdateByID(ctx, saved.ID, UpdatePatch{Title: strPtr("")})
	if err == nil {
		t.Fatalf("setting title to empty must fail validation")
	}
	if !errors.Is(err, memory.ErrEmptyTitle) {
		t.Errorf("empty title must wrap memory.ErrEmptyTitle, got %v", err)
	}

	tooLong := strings.Repeat("x", memory.MaxTitleChars+10)
	_, err = st.UpdateByID(ctx, saved.ID, UpdatePatch{Title: strPtr(tooLong)})
	if err == nil {
		t.Fatalf("oversized title must fail validation")
	}
	if !errors.Is(err, memory.ErrTitleTooLong) {
		t.Errorf("oversized title must wrap memory.ErrTitleTooLong, got %v", err)
	}
}

func TestUpdateByID_HashChangesOnContentEdit(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seed(t, st, sampleMemory())
	originalHash := saved.NormalizedHash

	updated, err := st.UpdateByID(ctx, saved.ID, UpdatePatch{
		Content: strPtr(saved.Content + "\nappended line"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.NormalizedHash == originalHash {
		t.Errorf("content change must produce a new normalized_hash")
	}
}
