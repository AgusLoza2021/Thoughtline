package server

import (
	"context"
	"strings"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

func TestDoStats_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	seedMemory(t, st, sampleSceneMemory("First", "first body", "scene/a"))
	seedMemory(t, st, sampleSceneMemory("Second", "second body", "scene/b"))

	res, err := doStats(ctx, st, Config{}, statsArgs{})
	if err != nil {
		t.Fatalf("doStats: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "Memories: 2") {
		t.Errorf("response should report 2 memories, got:\n%s", body)
	}
	if !strings.Contains(body, "scene-pattern") {
		t.Errorf("response should include type breakdown, got:\n%s", body)
	}
}

func TestDoStats_EmptyDatabase(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doStats(ctx, st, Config{}, statsArgs{})
	if err != nil {
		t.Fatalf("doStats: %v", err)
	}
	if res.IsError {
		t.Fatalf("empty DB should not be an error envelope")
	}
	body := textContent(res)
	if !strings.Contains(body, "Memories: 0") {
		t.Errorf("expected Memories: 0, got:\n%s", body)
	}
}

func TestDoStats_ProjectFilter(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	a := sampleSceneMemory("Alpha", "alpha body", "scene/a")
	a.Project = "alpha"
	seedMemory(t, st, a)

	b := sampleSceneMemory("Beta", "beta body", "scene/b")
	b.Project = "beta"
	seedMemory(t, st, b)

	res, err := doStats(ctx, st, Config{}, statsArgs{Project: "alpha"})
	if err != nil {
		t.Fatalf("doStats: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "Memories: 1") {
		t.Errorf("project filter should yield 1 memory, got:\n%s", body)
	}
	if strings.Contains(body, "beta") {
		t.Errorf("filtered output must not include beta, got:\n%s", body)
	}
}

func TestDoStats_DefaultProjectFromConfig(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	a := sampleSceneMemory("Alpha", "alpha body", "scene/a")
	a.Project = "alpha"
	seedMemory(t, st, a)

	b := sampleSceneMemory("Beta", "beta body", "scene/b")
	b.Project = "beta"
	seedMemory(t, st, b)

	cfg := Config{DefaultProject: "alpha"}
	res, err := doStats(ctx, st, cfg, statsArgs{}) // no project arg
	if err != nil {
		t.Fatalf("doStats: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "Memories: 1") {
		t.Errorf("default project should scope to alpha, got:\n%s", body)
	}
}

func TestDoStats_AllProjectsViaWildcard(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	a := sampleSceneMemory("Alpha", "alpha body", "scene/a")
	a.Project = "alpha"
	seedMemory(t, st, a)

	b := sampleSceneMemory("Beta", "beta body", "scene/b")
	b.Project = "beta"
	seedMemory(t, st, b)

	cfg := Config{DefaultProject: "alpha"}
	// "*" means "ignore default — show all projects".
	res, err := doStats(ctx, st, cfg, statsArgs{Project: "*"})
	if err != nil {
		t.Fatalf("doStats: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "Memories: 2") {
		t.Errorf("'*' should disable project filter, got:\n%s", body)
	}
}

func TestDoStats_IncludesSessions(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	open, _ := st.StartSession(ctx, "enchanted-inn", "claude-code")
	closed, _ := st.StartSession(ctx, "enchanted-inn", "")
	if _, err := st.EndSession(ctx, closed.ID, "done"); err != nil {
		t.Fatalf("end: %v", err)
	}
	_ = open

	res, err := doStats(ctx, st, Config{}, statsArgs{})
	if err != nil {
		t.Fatalf("doStats: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "Sessions") {
		t.Errorf("response should include Sessions section, got:\n%s", body)
	}
	if !strings.Contains(body, "open") || !strings.Contains(body, "closed") {
		t.Errorf("session counts should distinguish open/closed, got:\n%s", body)
	}
}

func TestDoStats_IncludesTypeBreakdown(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	a := sampleSceneMemory("A", "a", "scene/a")
	seedMemory(t, st, a)

	b := sampleSceneMemory("B", "b", "perf/b")
	b.Type = memory.TypePerfGotcha
	seedMemory(t, st, b)

	res, err := doStats(ctx, st, Config{}, statsArgs{})
	if err != nil {
		t.Fatalf("doStats: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "scene-pattern") || !strings.Contains(body, "perf-gotcha") {
		t.Errorf("type breakdown missing, got:\n%s", body)
	}
}

func TestDoStats_IncludesRecentActivity(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	seedMemory(t, st, sampleSceneMemory("Marker title", "marker body", "scene/marker"))

	res, err := doStats(ctx, st, Config{}, statsArgs{})
	if err != nil {
		t.Fatalf("doStats: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "Marker title") {
		t.Errorf("recent activity should list seeded title, got:\n%s", body)
	}
}
