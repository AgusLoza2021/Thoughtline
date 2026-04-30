package server

import (
	"context"
	"strings"
	"testing"
)

func TestDoUpdate_HappyPath_TitleOnly(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seedMemory(t, st, sampleSceneMemory("Original title", "original body", "scene/x"))

	args := updateArgs{
		ID:       saved.ID,
		Title:    "Renamed title",
		HasTitle: true,
	}
	res, err := doUpdate(ctx, st, args)
	if err != nil {
		t.Fatalf("doUpdate: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "action=updated") {
		t.Errorf("response should mention action=updated, got:\n%s", body)
	}
	if !strings.Contains(body, "Renamed title") {
		t.Errorf("response should echo new title, got:\n%s", body)
	}
	if !strings.Contains(body, "Revision: 1") {
		t.Errorf("revision should bump to 1, got:\n%s", body)
	}
}

func TestDoUpdate_TagsCleared(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	m := sampleSceneMemory("With tags", "body", "scene/tags")
	m.Tags = []string{"engine:playcanvas"}
	saved := seedMemory(t, st, m)

	res, err := doUpdate(ctx, st, updateArgs{
		ID:      saved.ID,
		Tags:    []string{},
		HasTags: true, // explicitly setting to empty = clear
	})
	if err != nil {
		t.Fatalf("doUpdate: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got: %s", textContent(res))
	}
	if strings.Contains(textContent(res), "engine:playcanvas") {
		t.Errorf("tags should be cleared, got:\n%s", textContent(res))
	}
}

func TestDoUpdate_NoOp(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seedMemory(t, st, sampleSceneMemory("Title", "body", "scene/noop"))

	res, err := doUpdate(ctx, st, updateArgs{ID: saved.ID}) // no Has* flags set
	if err != nil {
		t.Fatalf("doUpdate: %v", err)
	}
	if res.IsError {
		t.Fatalf("noop must succeed, got: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "action=noop") {
		t.Errorf("empty patch should be reported as action=noop, got:\n%s", body)
	}
	if !strings.Contains(body, "Revision: 0") {
		t.Errorf("noop must not bump revision, got:\n%s", body)
	}
}

func TestDoUpdate_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doUpdate(ctx, st, updateArgs{
		ID:       999,
		Title:    "anything",
		HasTitle: true,
	})
	if err != nil {
		t.Fatalf("doUpdate: %v", err)
	}
	if !res.IsError {
		t.Fatalf("missing id should yield error result, got success: %s", textContent(res))
	}
	body := strings.ToLower(textContent(res))
	if !strings.Contains(body, "not found") && !strings.Contains(body, "no memory") {
		t.Errorf("error should clearly say not found, got:\n%s", textContent(res))
	}
}

func TestDoUpdate_MissingID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doUpdate(ctx, st, updateArgs{ID: 0, Title: "x", HasTitle: true})
	if err != nil {
		t.Fatalf("doUpdate: %v", err)
	}
	if !res.IsError {
		t.Fatalf("id=0 should yield validation error")
	}
	if !strings.Contains(textContent(res), "id") {
		t.Errorf("error should mention 'id', got:\n%s", textContent(res))
	}
}

func TestDoUpdate_ValidationErrorPropagates(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seedMemory(t, st, sampleSceneMemory("Title", "body", "scene/val"))

	// Setting title to empty must surface as validation error, not crash.
	res, err := doUpdate(ctx, st, updateArgs{
		ID:       saved.ID,
		Title:    "",
		HasTitle: true,
	})
	if err != nil {
		t.Fatalf("doUpdate: %v", err)
	}
	if !res.IsError {
		t.Fatalf("empty title must yield validation error, got success")
	}
	if !strings.Contains(textContent(res), "title") {
		t.Errorf("error should mention title, got:\n%s", textContent(res))
	}
}
