package server

import (
	"context"
	"strings"
	"testing"
)

func TestDoDelete_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seedMemory(t, st, sampleSceneMemory("Title", "body", "scene/del"))

	res, err := doDelete(ctx, st, deleteArgs{ID: saved.ID})
	if err != nil {
		t.Fatalf("doDelete: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got: %s", textContent(res))
	}
	body := strings.ToLower(textContent(res))
	if !strings.Contains(body, "deleted") {
		t.Errorf("response should confirm deletion, got:\n%s", textContent(res))
	}
}

func TestDoDelete_RemovedFromContextAndSearch(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seedMemory(t, st, sampleSceneMemory("Soon-gone", "lantern lantern lantern", "scene/del2"))

	if _, err := doDelete(ctx, st, deleteArgs{ID: saved.ID}); err != nil {
		t.Fatalf("doDelete: %v", err)
	}

	// Search must not return it.
	res, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:   "lantern",
		Project: "enchanted-inn",
	})
	if err != nil || res.IsError {
		t.Fatalf("search after delete: err=%v body=%s", err, textContent(res))
	}
	if strings.Contains(textContent(res), "Soon-gone") {
		t.Errorf("deleted memory must not appear in search, got:\n%s", textContent(res))
	}

	// Context must not return it either.
	cres, err := doContext(ctx, st, Config{}, contextArgs{Project: "enchanted-inn"})
	if err != nil || cres.IsError {
		t.Fatalf("context after delete: err=%v body=%s", err, textContent(cres))
	}
	if strings.Contains(textContent(cres), "Soon-gone") {
		t.Errorf("deleted memory must not appear in context, got:\n%s", textContent(cres))
	}
}

func TestDoDelete_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doDelete(ctx, st, deleteArgs{ID: 999})
	if err != nil {
		t.Fatalf("doDelete: %v", err)
	}
	if !res.IsError {
		t.Fatalf("missing id should yield error, got success: %s", textContent(res))
	}
}

func TestDoDelete_MissingID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doDelete(ctx, st, deleteArgs{ID: 0})
	if err != nil {
		t.Fatalf("doDelete: %v", err)
	}
	if !res.IsError {
		t.Fatalf("id=0 should yield validation error")
	}
	if !strings.Contains(textContent(res), "id") {
		t.Errorf("error should mention 'id', got:\n%s", textContent(res))
	}
}

func TestDoDelete_DoubleDeleteIsNotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seedMemory(t, st, sampleSceneMemory("Title", "body", "scene/double"))

	if _, err := doDelete(ctx, st, deleteArgs{ID: saved.ID}); err != nil {
		t.Fatalf("first delete: %v", err)
	}

	res, err := doDelete(ctx, st, deleteArgs{ID: saved.ID})
	if err != nil {
		t.Fatalf("doDelete: %v", err)
	}
	if !res.IsError {
		t.Fatalf("repeat delete should be reported as not-found error, got: %s", textContent(res))
	}
}
