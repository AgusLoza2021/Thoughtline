package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// seedMemory persists a Memory directly through storage so tests can prepare
// the search index without going through the tl_save handler.
// It auto-creates the brain for m.Project if it doesn't exist yet.
func seedMemory(t *testing.T, st *storage.Storage, m memory.Memory) memory.Memory {
	t.Helper()
	ctx := context.Background()
	brainID, err := st.ResolveOrCreateBrainID(ctx, m.Project)
	if err != nil {
		t.Fatalf("seed resolve brain: %v", err)
	}
	saved, _, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("seed save: %v", err)
	}
	return saved
}

func validSearchArgs() searchArgs {
	return searchArgs{
		Query:   "lantern",
		Project: "enchanted-inn",
	}
}

func sampleSceneMemory(title, content, topic string) memory.Memory {
	return memory.Memory{
		Project:  "enchanted-inn",
		Scope:    memory.ScopeProject,
		Type:     memory.TypeScenePattern,
		TopicKey: topic,
		Title:    title,
		Content:  content,
	}
}

func TestDoSearch_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	seedMemory(t, st, sampleSceneMemory(
		"Lantern bake workflow",
		"Bake lantern-base normals before exporting to PlayCanvas.",
		"scene/lantern",
	))

	res, err := doSearch(ctx, st, Config{}, validSearchArgs())
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "Lantern bake workflow") {
		t.Errorf("response should include matching title, got:\n%s", body)
	}
	if !strings.Contains(body, "scene/lantern") {
		t.Errorf("response should echo topic_key, got:\n%s", body)
	}
}

func TestDoSearch_EmptyResultMessage(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:   "zxqvbn",
		Project: "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("empty result must not be an error envelope, got: %s", textContent(res))
	}
	body := strings.ToLower(textContent(res))
	if !strings.Contains(body, "no") || !strings.Contains(body, "match") {
		t.Errorf("empty result body should communicate 'no matches', got:\n%s", textContent(res))
	}
}

func TestDoSearch_MissingQuery(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doSearch(ctx, st, Config{}, searchArgs{Query: "  ", Project: "enchanted-inn"})
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	if !res.IsError {
		t.Fatalf("blank query must yield validation error, got success: %s", textContent(res))
	}
	if !strings.Contains(textContent(res), "query") {
		t.Errorf("error message should mention 'query', got:\n%s", textContent(res))
	}
}

func TestDoSearch_DefaultProjectScopesResults(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	a := sampleSceneMemory("Lantern in alpha", "lantern alpha alpha", "scene/a")
	a.Project = "alpha"
	seedMemory(t, st, a)

	b := sampleSceneMemory("Lantern in beta", "lantern beta beta", "scene/b")
	b.Project = "beta"
	seedMemory(t, st, b)

	cfg := Config{DefaultProject: "alpha"}
	res, err := doSearch(ctx, st, cfg, searchArgs{Query: "lantern"}) // no project arg
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "Lantern in alpha") {
		t.Errorf("default project should scope to alpha, got:\n%s", body)
	}
	if strings.Contains(body, "Lantern in beta") {
		t.Errorf("default project must exclude other projects, got:\n%s", body)
	}
}

func TestDoSearch_ExplicitProjectOverridesDefault(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	a := sampleSceneMemory("Lantern in alpha", "lantern alpha alpha", "scene/a")
	a.Project = "alpha"
	seedMemory(t, st, a)

	b := sampleSceneMemory("Lantern in beta", "lantern beta beta", "scene/b")
	b.Project = "beta"
	seedMemory(t, st, b)

	cfg := Config{DefaultProject: "alpha"}
	res, err := doSearch(ctx, st, cfg, searchArgs{Query: "lantern", Project: "beta"})
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	body := textContent(res)
	if strings.Contains(body, "Lantern in alpha") {
		t.Errorf("explicit project=beta must override default=alpha, got:\n%s", body)
	}
	if !strings.Contains(body, "Lantern in beta") {
		t.Errorf("explicit project=beta should include beta hit, got:\n%s", body)
	}
}

func TestDoSearch_TypeFilter(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	scene := sampleSceneMemory("Lantern scene pattern", "lantern lantern", "scene/x")
	seedMemory(t, st, scene)

	perf := sampleSceneMemory("Lantern perf gotcha", "lantern lantern", "perf/y")
	perf.Type = memory.TypePerfGotcha
	seedMemory(t, st, perf)

	res, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:   "lantern",
		Project: "enchanted-inn",
		Type:    string(memory.TypePerfGotcha),
	})
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "Lantern perf gotcha") {
		t.Errorf("type filter should include perf-gotcha hit, got:\n%s", body)
	}
	if strings.Contains(body, "Lantern scene pattern") {
		t.Errorf("type filter should exclude scene-pattern, got:\n%s", body)
	}
}

func TestDoSearch_TopicKeyShortcutEndToEnd(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	seedMemory(t, st, sampleSceneMemory(
		"Inn cellar entity hierarchy",
		"world/static for chairs; world/interactive for the cellar door.",
		"scene/playcanvas/inn-cellar",
	))

	res, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:   "scene/playcanvas/inn-cellar",
		Project: "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "Inn cellar entity hierarchy") {
		t.Errorf("topic_key shortcut should return the matched memory, got:\n%s", body)
	}
}

func TestDoSearch_HintsAtGetObservation(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	long := strings.Repeat("lantern ", 200) + " unique-marker"
	seedMemory(t, st, sampleSceneMemory("Long lantern doc", long, "scene/long"))

	res, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:   "unique-marker",
		Project: "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "tl_get_observation") {
		t.Errorf("when results are truncated previews, response should hint at tl_get_observation, got:\n%s", body)
	}
}

func TestDoSearch_LimitOverride(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		m := sampleSceneMemory("Lantern n", "lantern lantern", "scene/n"+string(rune('a'+i)))
		seedMemory(t, st, m)
	}

	res, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:   "lantern",
		Project: "enchanted-inn",
		Limit:   2,
	})
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	body := textContent(res)
	// Each result block starts with a "Title: " line.
	gotResults := strings.Count(body, "Title: ")
	if gotResults != 2 {
		t.Errorf("limit=2 should produce 2 result blocks, got %d in:\n%s", gotResults, body)
	}
}

func TestDoSearch_TagsFilter(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	tagged := sampleSceneMemory("Lantern unity android", "lantern lantern", "scene/tagged")
	tagged.Tags = []string{"engine:unity", "platform:android"}
	seedMemory(t, st, tagged)

	untagged := sampleSceneMemory("Lantern no tags", "lantern lantern", "scene/untagged")
	seedMemory(t, st, untagged)

	res, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:   "lantern",
		Project: "enchanted-inn",
		Tags:    []string{"engine:unity"},
	})
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "Lantern unity android") {
		t.Errorf("tags filter should include tagged memory, got:\n%s", body)
	}
	if strings.Contains(body, "Lantern no tags") {
		t.Errorf("tags filter should exclude untagged memory, got:\n%s", body)
	}
}

func TestDoSearch_TagsFilter_ANDSemantics(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	both := sampleSceneMemory("Lantern both tags", "lantern lantern", "scene/both")
	both.Tags = []string{"engine:unity", "platform:android"}
	seedMemory(t, st, both)

	oneOnly := sampleSceneMemory("Lantern one tag", "lantern lantern", "scene/one")
	oneOnly.Tags = []string{"engine:unity"}
	seedMemory(t, st, oneOnly)

	res, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:   "lantern",
		Project: "enchanted-inn",
		Tags:    []string{"engine:unity", "platform:android"},
	})
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "Lantern both tags") {
		t.Errorf("AND filter should include memory with both tags, got:\n%s", body)
	}
	if strings.Contains(body, "Lantern one tag") {
		t.Errorf("AND filter should exclude memory missing one tag, got:\n%s", body)
	}
}

func TestDoSearch_RecentFirst(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	older := sampleSceneMemory("Lantern older", "lantern lantern", "scene/older")
	savedOlder := seedMemory(t, st, older)

	st.SetClock(func() time.Time { return savedOlder.UpdatedAt.Add(5 * time.Second) })

	newer := sampleSceneMemory("Lantern newer", "lantern lantern", "scene/newer")
	seedMemory(t, st, newer)

	res, err := doSearch(ctx, st, Config{}, searchArgs{
		Query:       "lantern",
		Project:     "enchanted-inn",
		RecentFirst: true,
	})
	if err != nil {
		t.Fatalf("doSearch: %v", err)
	}
	body := textContent(res)
	newerPos := strings.Index(body, "Lantern newer")
	olderPos := strings.Index(body, "Lantern older")
	if newerPos == -1 || olderPos == -1 {
		t.Fatalf("both memories should appear in results:\n%s", body)
	}
	if newerPos > olderPos {
		t.Errorf("recent_first=true: newer memory should appear before older, got:\n%s", body)
	}
}
