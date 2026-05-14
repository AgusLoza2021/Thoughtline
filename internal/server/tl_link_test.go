package server

import (
	"context"
	"strings"
	"testing"
)

func TestDoLink_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	from := seedMemory(t, st, sampleSceneMemory("Source memory", "from body", "scene/from"))
	to := seedMemory(t, st, sampleSceneMemory("Target memory", "to body", "scene/to"))

	res, err := doLink(ctx, st, Config{}, linkArgs{
		FromID:   from.ID,
		ToID:     to.ID,
		Relation: "related",
		Project:  "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doLink: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "Linked:") {
		t.Errorf("response should start with 'Linked:', got:\n%s", body)
	}
	if !strings.Contains(body, "related") {
		t.Errorf("response should include relation, got:\n%s", body)
	}
	if !strings.Contains(body, "link_id:") {
		t.Errorf("response should include link_id, got:\n%s", body)
	}
}

func TestDoLink_SelfLink(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	m := seedMemory(t, st, sampleSceneMemory("Self memory", "body", "scene/self"))

	res, err := doLink(ctx, st, Config{}, linkArgs{
		FromID:   m.ID,
		ToID:     m.ID,
		Relation: "related",
		Project:  "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doLink: %v", err)
	}
	if !res.IsError {
		t.Fatalf("self-link should be an error, got: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "cannot link to itself") {
		t.Errorf("error should mention self-link, got:\n%s", body)
	}
}

func TestDoLink_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	// Use a valid seed so the brain exists.
	m := seedMemory(t, st, sampleSceneMemory("Real memory", "body", "scene/real"))

	res, err := doLink(ctx, st, Config{}, linkArgs{
		FromID:   m.ID,
		ToID:     99999,
		Relation: "related",
		Project:  "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doLink: %v", err)
	}
	if !res.IsError {
		t.Fatalf("invalid to_id should yield error, got: %s", textContent(res))
	}
}

func TestDoLink_InvalidRelation(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	from := seedMemory(t, st, sampleSceneMemory("From", "body", "scene/f1"))
	to := seedMemory(t, st, sampleSceneMemory("To", "body", "scene/t1"))

	res, err := doLink(ctx, st, Config{}, linkArgs{
		FromID:   from.ID,
		ToID:     to.ID,
		Relation: "invalid-relation",
		Project:  "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doLink: %v", err)
	}
	if !res.IsError {
		t.Fatalf("invalid relation should yield error, got: %s", textContent(res))
	}
}

func TestDoLink_Duplicate(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	from := seedMemory(t, st, sampleSceneMemory("From2", "body", "scene/f2"))
	to := seedMemory(t, st, sampleSceneMemory("To2", "body", "scene/t2"))

	args := linkArgs{
		FromID:   from.ID,
		ToID:     to.ID,
		Relation: "refines",
		Project:  "enchanted-inn",
	}
	if _, err := doLink(ctx, st, Config{}, args); err != nil {
		t.Fatalf("first doLink: %v", err)
	}

	res, err := doLink(ctx, st, Config{}, args)
	if err != nil {
		t.Fatalf("second doLink: %v", err)
	}
	if res.IsError {
		t.Fatalf("duplicate link should be a noop (not error), got: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "already exists") {
		t.Errorf("response should say 'already exists', got:\n%s", body)
	}
}

func TestDoLink_MissingFromID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doLink(ctx, st, Config{}, linkArgs{
		FromID:   0,
		ToID:     1,
		Relation: "related",
	})
	if err != nil {
		t.Fatalf("doLink: %v", err)
	}
	if !res.IsError {
		t.Fatalf("missing from_id should yield error")
	}
}

func TestDoLink_DefaultProjectFromConfig(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	from := seedMemory(t, st, sampleSceneMemory("From3", "body", "scene/f3"))
	to := seedMemory(t, st, sampleSceneMemory("To3", "body", "scene/t3"))

	cfg := Config{DefaultProject: "enchanted-inn"}
	res, err := doLink(ctx, st, cfg, linkArgs{
		FromID:   from.ID,
		ToID:     to.ID,
		Relation: "related",
		// no Project set — falls back to cfg.DefaultProject
	})
	if err != nil {
		t.Fatalf("doLink: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success with default project, got: %s", textContent(res))
	}
}
