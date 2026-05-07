package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

func TestStats_EmptyDatabase(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	stats, err := st.Stats(ctx, StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalMemories != 0 {
		t.Errorf("expected 0 memories, got %d", stats.TotalMemories)
	}
	if stats.DeletedMemories != 0 {
		t.Errorf("expected 0 deleted, got %d", stats.DeletedMemories)
	}
	if stats.OpenSessions != 0 || stats.ClosedSessions != 0 {
		t.Errorf("expected 0 sessions, got open=%d closed=%d", stats.OpenSessions, stats.ClosedSessions)
	}
	if stats.GeneratedAt.IsZero() {
		t.Errorf("GeneratedAt must be populated")
	}
}

func TestStats_TotalMemoriesAndDeletedCount(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	// Seed 3 memories, soft-delete 1.
	a := seed(t, st, sampleMemory())
	_ = seed(t, st, withTopic(sampleMemory(), "scene/b"))
	_ = seed(t, st, withTopic(sampleMemory(), "scene/c"))
	if err := st.SoftDelete(ctx, a.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	stats, err := st.Stats(ctx, StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalMemories != 2 {
		t.Errorf("expected 2 active memories, got %d", stats.TotalMemories)
	}
	if stats.DeletedMemories != 1 {
		t.Errorf("expected 1 deleted memory, got %d", stats.DeletedMemories)
	}
}

func TestStats_GroupsByType(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	a := sampleMemory()
	a.TopicKey = "scene/a"
	a.Type = memory.TypeScenePattern
	seed(t, st, a)

	b := sampleMemory()
	b.TopicKey = "scene/b"
	b.Type = memory.TypeScenePattern
	seed(t, st, b)

	c := sampleMemory()
	c.TopicKey = "perf/c"
	c.Type = memory.TypePerfGotcha
	seed(t, st, c)

	stats, err := st.Stats(ctx, StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.ByType[memory.TypeScenePattern] != 2 {
		t.Errorf("expected 2 scene-pattern, got %d", stats.ByType[memory.TypeScenePattern])
	}
	if stats.ByType[memory.TypePerfGotcha] != 1 {
		t.Errorf("expected 1 perf-gotcha, got %d", stats.ByType[memory.TypePerfGotcha])
	}
	if stats.ByType[memory.TypeBugfix] != 0 {
		t.Errorf("absent type should be 0 (or missing), got %d", stats.ByType[memory.TypeBugfix])
	}
}

func TestStats_GroupsByProject(t *testing.T) {
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

	c := sampleMemory()
	c.Project = "alpha"
	c.TopicKey = "scene/c"
	seed(t, st, c)

	stats, err := st.Stats(ctx, StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.ByProject["alpha"] != 2 {
		t.Errorf("expected 2 in alpha, got %d", stats.ByProject["alpha"])
	}
	if stats.ByProject["beta"] != 1 {
		t.Errorf("expected 1 in beta, got %d", stats.ByProject["beta"])
	}
}

func TestStats_GroupsByScope(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	proj := sampleMemory()
	proj.TopicKey = "scene/proj"
	proj.Scope = memory.ScopeProject
	seed(t, st, proj)

	pers := sampleMemory()
	pers.TopicKey = "preference/something"
	pers.Type = memory.TypePreference
	pers.Scope = memory.ScopePersonal
	seed(t, st, pers)

	stats, err := st.Stats(ctx, StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.ByScope[memory.ScopeProject] != 1 {
		t.Errorf("expected 1 project-scope, got %d", stats.ByScope[memory.ScopeProject])
	}
	if stats.ByScope[memory.ScopePersonal] != 1 {
		t.Errorf("expected 1 personal-scope, got %d", stats.ByScope[memory.ScopePersonal])
	}
}

func TestStats_SessionCounts(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	open1, err := st.StartSession(ctx, "enchanted-inn", "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	open2, err := st.StartSession(ctx, "enchanted-inn", "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	closed, err := st.StartSession(ctx, "enchanted-inn", "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := st.EndSession(ctx, closed.ID, "done"); err != nil {
		t.Fatalf("end: %v", err)
	}

	stats, err := st.Stats(ctx, StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.OpenSessions != 2 {
		t.Errorf("expected 2 open sessions, got %d", stats.OpenSessions)
	}
	if stats.ClosedSessions != 1 {
		t.Errorf("expected 1 closed session, got %d", stats.ClosedSessions)
	}
	_ = open1
	_ = open2
}

func TestStats_RecentMemoriesIncluded(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	t0 := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	st.SetClock(func() time.Time { return t0 })
	seed(t, st, withTopic(sampleMemory(), "scene/oldest"))

	st.SetClock(func() time.Time { return t0.Add(time.Hour) })
	seed(t, st, withTopic(sampleMemory(), "scene/newest"))

	stats, err := st.Stats(ctx, StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats.RecentMemories) != 2 {
		t.Fatalf("expected 2 recent memories, got %d", len(stats.RecentMemories))
	}
	if !strings_contains(stats.RecentMemories[0].TopicKey, "newest") {
		t.Errorf("expected newest first, got %q", stats.RecentMemories[0].TopicKey)
	}
}

func TestStats_RecentMemoriesRespectsLimit(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		seed(t, st, withTopic(sampleMemory(), "scene/n"+string(rune('a'+i))))
	}

	stats, err := st.Stats(ctx, StatsOptions{RecentLimit: 5})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats.RecentMemories) != 5 {
		t.Errorf("expected 5 recent memories with RecentLimit=5, got %d", len(stats.RecentMemories))
	}
}

func TestStats_DefaultRecentLimitIs10(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		seed(t, st, withTopic(sampleMemory(), "scene/n"+string(rune('a'+i))))
	}

	stats, err := st.Stats(ctx, StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats.RecentMemories) != 10 {
		t.Errorf("expected default RecentLimit=10, got %d", len(stats.RecentMemories))
	}
}

func TestStats_RecentSessionsIncluded(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	t0 := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	st.SetClock(func() time.Time { return t0 })
	first, _ := st.StartSession(ctx, "enchanted-inn", "")

	st.SetClock(func() time.Time { return t0.Add(time.Hour) })
	second, _ := st.StartSession(ctx, "enchanted-inn", "")

	stats, err := st.Stats(ctx, StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats.RecentSessions) != 2 {
		t.Fatalf("expected 2 recent sessions, got %d", len(stats.RecentSessions))
	}
	if stats.RecentSessions[0].ID != second.ID {
		t.Errorf("expected newest session first, got %s want %s", stats.RecentSessions[0].ID, second.ID)
	}
	_ = first
}

func TestStats_ProjectFilter(t *testing.T) {
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

	if _, err := st.StartSession(ctx, "alpha", ""); err != nil {
		t.Fatalf("start alpha: %v", err)
	}

	stats, err := st.Stats(ctx, StatsOptions{Project: "alpha"})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalMemories != 1 {
		t.Errorf("project filter must scope memories, got %d", stats.TotalMemories)
	}
	if stats.ByProject["beta"] != 0 {
		t.Errorf("filtered project must not include beta, got %d", stats.ByProject["beta"])
	}
	if stats.OpenSessions != 1 {
		t.Errorf("expected 1 open session for alpha, got %d", stats.OpenSessions)
	}
}

// TestStats_MostRecentProjects verifies that the Stats result includes a
// MostRecentProjects field ordered by MAX(updated_at) DESC, capped at 4.
func TestStats_MostRecentProjects(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	t0 := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	// Seed 5 projects with distinct updated_at (older → newer).
	projects := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	for i, p := range projects {
		st.SetClock(func() time.Time { return t0.Add(time.Duration(i) * time.Hour) })
		m := sampleMemory()
		m.Project = p
		m.TopicKey = fmt.Sprintf("scene/%s", p)
		seed(t, st, m)
	}

	stats, err := st.Stats(ctx, StatsOptions{})
	if err != nil {
		t.Fatalf("stats: %v", err)
	}

	if len(stats.MostRecentProjects) != 4 {
		t.Fatalf("MostRecentProjects len=%d, want 4", len(stats.MostRecentProjects))
	}
	// First should be newest (epsilon), last in top-4 should be beta.
	if stats.MostRecentProjects[0] != "epsilon" {
		t.Errorf("MostRecentProjects[0]=%q, want %q", stats.MostRecentProjects[0], "epsilon")
	}
	if stats.MostRecentProjects[3] != "beta" {
		t.Errorf("MostRecentProjects[3]=%q, want %q", stats.MostRecentProjects[3], "beta")
	}
}

// Helpers used only by stats tests. withTopic returns a copy of m with the
// topic_key swapped (so the seed helper can be reused with distinct keys).
func withTopic(m memory.Memory, topic string) memory.Memory {
	m.TopicKey = topic
	return m
}

func strings_contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
