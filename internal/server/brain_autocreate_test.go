package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/brain"
	"github.com/AgusLoza2021/Thoughtline/internal/pending"
)

// TestTlSave_ResolvesProject (task 8.1) — tl_save with a known brain creates
// a memory with brain_id set.
func TestTlSave_ResolvesProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	// Pre-create brain so ResolveBrainID finds it.
	ensureBrain(t, st, "test-proj")

	args := saveArgs{
		Title:   "Brain resolution test",
		Content: "Testing that save resolves project to brain_id.",
		Type:    "bugfix",
		Project: "test-proj",
	}
	res, err := doSave(ctx, st, Config{}, args)
	if err != nil {
		t.Fatalf("doSave: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}

	body := textContent(res)
	if !strings.Contains(body, "action=created") {
		t.Errorf("expected action=created, got:\n%s", body)
	}

	// Verify the memory was stored with a valid brain_id via the DB.
	var brainID int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT brain_id FROM memories WHERE project = 'test-proj' LIMIT 1`,
	).Scan(&brainID); err != nil {
		t.Fatalf("check brain_id: %v", err)
	}
	if brainID == 0 {
		t.Errorf("expected non-zero brain_id on saved memory")
	}
}

// TestTlSave_AutoCreatesBrain (task 8.3) — tl_save with a project that has no
// corresponding brain auto-creates it (kind='real') and the memory gets the new brain_id.
func TestTlSave_AutoCreatesBrain(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	const freshProject = "fresh-project-never-seen"

	// Verify brain does NOT exist yet.
	brainStore := brain.New(st.DB())
	_, err := brainStore.GetBySlug(ctx, freshProject)
	if err == nil {
		t.Fatalf("expected ErrBrainNotFound before save, got brain")
	}

	args := saveArgs{
		Title:   "First memory in fresh project",
		Content: "Auto-create brain on first save.",
		Type:    "convention",
		Project: freshProject,
	}
	res, err := doSave(ctx, st, Config{}, args)
	if err != nil {
		t.Fatalf("doSave: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}

	// Assert brain was auto-created with kind='real'.
	b, err := brainStore.GetBySlug(ctx, freshProject)
	if err != nil {
		t.Fatalf("brain not created: %v", err)
	}
	if b.Kind != brain.KindReal {
		t.Errorf("expected kind=real, got %q", b.Kind)
	}

	// Assert the saved memory has the new brain_id.
	var memBrainID int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT brain_id FROM memories WHERE project = ? LIMIT 1`, freshProject,
	).Scan(&memBrainID); err != nil {
		t.Fatalf("check brain_id: %v", err)
	}
	if memBrainID != b.ID {
		t.Errorf("memory brain_id=%d does not match brain.ID=%d", memBrainID, b.ID)
	}

	// Subsequent tl_search returns the saved memory.
	searchRes, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:   "fresh project",
		Project: freshProject,
	})
	if err != nil || searchRes.IsError {
		t.Fatalf("search after auto-create: err=%v body=%s", err, textContent(searchRes))
	}
	if !strings.Contains(textContent(searchRes), "First memory in fresh project") {
		t.Errorf("search should find the saved memory, got:\n%s", textContent(searchRes))
	}

	// tl_search for a different project returns empty — no brain auto-created on read.
	const differentProject = "different-fresh-project"
	emptyRes, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:   "first memory",
		Project: differentProject,
	})
	if err != nil || emptyRes.IsError {
		t.Fatalf("search on different-fresh-project: err=%v body=%s", err, textContent(emptyRes))
	}
	body := strings.ToLower(textContent(emptyRes))
	if !strings.Contains(body, "no") {
		t.Errorf("different project should return empty, got:\n%s", textContent(emptyRes))
	}

	// Assert no brain was created for different-fresh-project (read path must NOT auto-create).
	_, err = brainStore.GetBySlug(ctx, differentProject)
	if err == nil {
		t.Errorf("read-only search must NOT create a brain for %q", differentProject)
	}
}

// TestTlPromote_SetsBrainID (task 8.7) — pending event with matching brain gets
// promoted and the created memory has brain_id set.
func TestTlPromote_SetsBrainID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "my-game"}
	ensureBrain(t, st, "my-game")

	ts := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	seedPendingEvent(t, st, "setbrain-ev1", "setbrain-h1", "my-game", pending.StatusPending, ts)

	var id int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id='setbrain-ev1'`).Scan(&id); err != nil {
		t.Fatalf("get pending id: %v", err)
	}

	req := buildReq("tl_promote", map[string]any{
		"items": []any{
			map[string]any{
				"pending_event_id": float64(id),
				"type":             "convention",
				"title":            "Promote with brain_id",
				"content":          "**What**: Testing brain_id set on promote.",
			},
		},
	})
	result, err := doTLPromote(ctx, st, cfg, req.GetArguments())
	if err != nil {
		t.Fatalf("doTLPromote: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", textContent(result))
	}
	if !strings.Contains(textContent(result), "promoted") {
		t.Errorf("expected promoted in response, got:\n%s", textContent(result))
	}

	// Verify the created memory has a non-zero brain_id.
	var memBrainID int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT brain_id FROM memories WHERE project = 'my-game' LIMIT 1`,
	).Scan(&memBrainID); err != nil {
		t.Fatalf("check brain_id: %v", err)
	}
	if memBrainID == 0 {
		t.Errorf("promoted memory must have non-zero brain_id")
	}
}

// TestTlPromote_FailsIfNoBrain (task 8.9) — pending event with project that has
// no brain returns an error with the expected message.
func TestTlPromote_FailsIfNoBrain(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "orphan"}

	// Do NOT create a brain for "orphan".
	ts := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	seedPendingEvent(t, st, "orphan-ev1", "orphan-h1", "orphan", pending.StatusPending, ts)

	var id int64
	if err := st.DB().QueryRowContext(ctx,
		`SELECT id FROM pending_events WHERE sync_id='orphan-ev1'`).Scan(&id); err != nil {
		t.Fatalf("get pending id: %v", err)
	}

	req := buildReq("tl_promote", map[string]any{
		"items": []any{
			map[string]any{
				"pending_event_id": float64(id),
				"type":             "bugfix",
				"title":            "Orphan fix",
				"content":          "Content.",
			},
		},
	})
	result, err := doTLPromote(ctx, st, cfg, req.GetArguments())
	if err != nil {
		t.Fatalf("doTLPromote: %v", err)
	}

	body := textContent(result)
	if !strings.Contains(body, "error") {
		t.Errorf("expected error in response, got:\n%s", body)
	}
	if !strings.Contains(body, "orphan") {
		t.Errorf("error should mention the project name 'orphan', got:\n%s", body)
	}
}
