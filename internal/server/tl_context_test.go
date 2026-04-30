package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

func TestDoContext_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	t0 := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	st.SetClock(func() time.Time { return t0 })
	seedMemory(t, st, sampleSceneMemory("Older entry", "older content", "scene/old"))

	st.SetClock(func() time.Time { return t0.Add(1 * time.Hour) })
	seedMemory(t, st, sampleSceneMemory("Newer entry", "newer content", "scene/new"))

	res, err := doContext(ctx, st, Config{}, contextArgs{Project: "enchanted-inn"})
	if err != nil {
		t.Fatalf("doContext: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "Newer entry") || !strings.Contains(body, "Older entry") {
		t.Errorf("response should contain both entries, got:\n%s", body)
	}
	// Newer comes before Older in the body.
	if strings.Index(body, "Newer entry") > strings.Index(body, "Older entry") {
		t.Errorf("recent ordering broken: newer should appear first in:\n%s", body)
	}
}

func TestDoContext_DefaultProjectFromConfig(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	a := sampleSceneMemory("Alpha entry", "alpha content", "scene/a")
	a.Project = "alpha"
	seedMemory(t, st, a)

	b := sampleSceneMemory("Beta entry", "beta content", "scene/b")
	b.Project = "beta"
	seedMemory(t, st, b)

	cfg := Config{DefaultProject: "alpha"}
	res, err := doContext(ctx, st, cfg, contextArgs{}) // no project arg
	if err != nil {
		t.Fatalf("doContext: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "Alpha entry") {
		t.Errorf("default project should scope to alpha, got:\n%s", body)
	}
	if strings.Contains(body, "Beta entry") {
		t.Errorf("default project must exclude beta, got:\n%s", body)
	}
}

func TestDoContext_RequiresProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doContext(ctx, st, Config{}, contextArgs{}) // no project, no default
	if err != nil {
		t.Fatalf("doContext: %v", err)
	}
	if !res.IsError {
		t.Fatalf("missing project (with no default) must yield validation error, got: %s", textContent(res))
	}
	if !strings.Contains(textContent(res), "project") {
		t.Errorf("error should mention 'project', got:\n%s", textContent(res))
	}
}

func TestDoContext_EmptyProjectMessage(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doContext(ctx, st, Config{}, contextArgs{Project: "empty-project"})
	if err != nil {
		t.Fatalf("doContext: %v", err)
	}
	if res.IsError {
		t.Fatalf("empty result must not be an error envelope, got: %s", textContent(res))
	}
	body := strings.ToLower(textContent(res))
	if !strings.Contains(body, "no") {
		t.Errorf("empty result should communicate emptiness, got:\n%s", textContent(res))
	}
}

func TestDoContext_LimitOverride(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		seedMemory(t, st, sampleSceneMemory("Entry "+string(rune('a'+i)), "content", "scene/n"+string(rune('a'+i))))
	}

	res, err := doContext(ctx, st, Config{}, contextArgs{
		Project: "enchanted-inn",
		Limit:   2,
	})
	if err != nil {
		t.Fatalf("doContext: %v", err)
	}
	body := textContent(res)
	got := strings.Count(body, "Title: ")
	if got != 2 {
		t.Errorf("limit=2 should yield 2 result blocks, got %d in:\n%s", got, body)
	}
}

// Verifies that the type filter (proxied to storage.Recent via SearchOptions
// equivalence) is NOT exposed by tl_context — recency-only is the M3 contract.
// If we need filters later, this test will need updating; for now it pins the
// "tl_context returns whatever Recent returns, scoped only by project+limit".
func TestDoContext_DoesNotAcceptTypeFilter(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	scene := sampleSceneMemory("Scene entry", "scene scene", "scene/x")
	seedMemory(t, st, scene)

	bug := sampleSceneMemory("Bug entry", "bug bug", "bug/y")
	bug.Type = memory.TypeBugfix
	seedMemory(t, st, bug)

	res, err := doContext(ctx, st, Config{}, contextArgs{Project: "enchanted-inn"})
	if err != nil {
		t.Fatalf("doContext: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "Scene entry") || !strings.Contains(body, "Bug entry") {
		t.Errorf("tl_context must return all types, got:\n%s", body)
	}
}
