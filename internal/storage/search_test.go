package storage

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// seed inserts a memory scoped to the default "enchanted-inn" brain and
// returns the saved row. It's a thin helper around Save that fails the test
// on error.
func seed(t *testing.T, st *Storage, m memory.Memory) memory.Memory {
	t.Helper()
	brainID, err := st.ResolveOrCreateBrainID(context.Background(), m.Project)
	if err != nil {
		t.Fatalf("seed resolve brain: %v", err)
	}
	saved, _, err := st.Save(context.Background(), brainID, m)
	if err != nil {
		t.Fatalf("seed save: %v", err)
	}
	return saved
}

func TestSearch_FTS5_BasicMatch(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.Title = "Lantern import workflow"
	m.Content = "Bake lantern-base normals before exporting to PlayCanvas."
	seed(t, st, m)

	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Title != "Lantern import workflow" {
		t.Errorf("title mismatch, got %q", r.Title)
	}
	if r.Snippet == "" {
		t.Errorf("snippet must be populated, got empty")
	}
	if r.ID == 0 || r.SyncID == "" {
		t.Errorf("identifiers must be populated, got id=%d sync=%q", r.ID, r.SyncID)
	}
	if r.Project != m.Project {
		t.Errorf("project mismatch, got %q want %q", r.Project, m.Project)
	}
	if r.Type != m.Type {
		t.Errorf("type mismatch, got %q want %q", r.Type, m.Type)
	}
	if r.UpdatedAt.IsZero() {
		t.Errorf("updated_at must be populated")
	}
}

func TestSearch_BM25_OrdersByRelevance(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	// "lantern" appears once in this one — weak match.
	weak := sampleMemory()
	weak.TopicKey = "scene/weak"
	weak.Title = "Mixed scene notes"
	weak.Content = "Various props including a lantern and other items."
	seed(t, st, weak)

	// "lantern" appears multiple times — stronger BM25 match.
	strong := sampleMemory()
	strong.TopicKey = "scene/strong"
	strong.Title = "Lantern lantern lantern setup"
	strong.Content = "lantern bake; lantern export; lantern emissive bloom."
	seed(t, st, strong)

	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// FTS5 BM25: lower (more negative) score = better match. The strong-match
	// row must come first.
	if !strings.Contains(strings.ToLower(results[0].Title), "lantern lantern") {
		t.Errorf("expected strongest BM25 match first, got order: %q, %q",
			results[0].Title, results[1].Title)
	}
	if results[0].Score >= results[1].Score {
		t.Errorf("first result must have lower (better) BM25 score: %v vs %v",
			results[0].Score, results[1].Score)
	}
}

func TestSearch_FilterByType(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	a := sampleMemory()
	a.TopicKey = "scene/a"
	a.Title = "Lantern bake A"
	a.Type = memory.TypeScenePattern
	seed(t, st, a)

	b := sampleMemory()
	b.TopicKey = "perf/b"
	b.Title = "Lantern perf B"
	b.Type = memory.TypePerfGotcha
	seed(t, st, b)

	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{Type: string(memory.TypePerfGotcha)})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result filtered by type, got %d", len(results))
	}
	if results[0].Type != memory.TypePerfGotcha {
		t.Errorf("expected perf-gotcha, got %q", results[0].Type)
	}
}

func TestSearch_FilterByProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	// Two different brains/projects — search is now brain-scoped.
	brainAlpha, err := st.ResolveOrCreateBrainID(ctx, "alpha")
	if err != nil {
		t.Fatalf("brain alpha: %v", err)
	}
	brainBeta, err := st.ResolveOrCreateBrainID(ctx, "beta")
	if err != nil {
		t.Fatalf("brain beta: %v", err)
	}

	a := sampleMemory()
	a.Project = "alpha"
	a.TopicKey = "scene/a"
	a.Title = "Lantern alpha"
	seed(t, st, a)

	b := sampleMemory()
	b.Project = "beta"
	b.TopicKey = "scene/b"
	b.Title = "Lantern beta"
	seed(t, st, b)

	// Search brainBeta — must only return beta rows.
	results, err := st.Search(ctx, brainBeta, "lantern", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for brainBeta, got %d", len(results))
	}
	if results[0].Project != "beta" {
		t.Errorf("expected project beta, got %q", results[0].Project)
	}

	// Search brainAlpha — must only return alpha rows.
	resultsA, err := st.Search(ctx, brainAlpha, "lantern", SearchOptions{})
	if err != nil {
		t.Fatalf("search alpha: %v", err)
	}
	if len(resultsA) != 1 || resultsA[0].Project != "alpha" {
		t.Errorf("expected 1 alpha row, got %v", resultsA)
	}
}

func TestSearch_FilterByScope(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	proj := sampleMemory()
	proj.TopicKey = "scene/proj"
	proj.Title = "Lantern project scope"
	proj.Scope = memory.ScopeProject
	proj.Type = memory.TypeScenePattern
	seed(t, st, proj)

	pers := sampleMemory()
	pers.TopicKey = "preference/lantern"
	pers.Title = "Lantern personal pref"
	pers.Scope = memory.ScopePersonal
	pers.Type = memory.TypePreference
	seed(t, st, pers)

	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{Scope: string(memory.ScopePersonal)})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 personal-scoped result, got %d", len(results))
	}
	if results[0].Scope != memory.ScopePersonal {
		t.Errorf("expected personal scope, got %q", results[0].Scope)
	}
}

func TestSearch_TopicKeyShortcut_ExactMatch(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.TopicKey = "scene/playcanvas/inn-cellar"
	m.Title = "Inn cellar entity hierarchy"
	m.Content = "world/static for chairs; world/interactive for the cellar door."
	seed(t, st, m)

	// A second memory whose CONTENT mentions "scene" and "inn-cellar" but
	// has a different topic_key — must NOT short-circuit on this row.
	noise := sampleMemory()
	noise.TopicKey = "scene/playcanvas/other"
	noise.Title = "Other scene"
	noise.Content = "Mentions scene/playcanvas/inn-cellar in passing."
	seed(t, st, noise)

	// The query string contains "/", so we hit the topic_key shortcut.
	results, err := st.Search(ctx, brainID, "scene/playcanvas/inn-cellar", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("topic_key shortcut must short-circuit FTS, got %d results", len(results))
	}
	if results[0].TopicKey != "scene/playcanvas/inn-cellar" {
		t.Errorf("expected exact topic_key hit, got %q", results[0].TopicKey)
	}
}

func TestSearch_TopicKeyShortcut_GlobWildcard(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	a := sampleMemory()
	a.TopicKey = "design/auth/oauth"
	a.Title = "OAuth flow"
	seed(t, st, a)

	b := sampleMemory()
	b.TopicKey = "design/auth/jwt"
	b.Title = "JWT rotation"
	seed(t, st, b)

	c := sampleMemory()
	c.TopicKey = "scene/lighting"
	c.Title = "Lighting"
	seed(t, st, c)

	results, err := st.Search(ctx, brainID, "design/auth/*", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("glob shortcut must match 2 rows, got %d", len(results))
	}
	for _, r := range results {
		if !strings.HasPrefix(r.TopicKey, "design/auth/") {
			t.Errorf("unexpected topic_key in glob result: %q", r.TopicKey)
		}
	}
}

func TestSearch_TopicKeyShortcut_FallsThroughOnMiss(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	// Topic key contains "/" but no row will match it; FTS path must still run.
	m := sampleMemory()
	m.TopicKey = "scene/cellar"
	m.Title = "Cellar door"
	m.Content = "door pivots on the top hinge"
	seed(t, st, m)

	results, err := st.Search(ctx, brainID, "scene/notfound", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	// Topic key shortcut misses, FTS path searches for "scene" "notfound"
	// (sanitized). "notfound" is not in any content; both tokens AND'd by FTS5
	// produce zero hits. This proves the shortcut miss falls through cleanly.
	if len(results) != 0 {
		t.Errorf("expected 0 results after shortcut miss + FTS miss, got %d", len(results))
	}
}

func TestSearch_SnippetTruncatedTo300Chars(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	long := strings.Repeat("lorem ipsum dolor sit amet ", 200) // ~5400 chars
	long += " marker keyword "
	long += strings.Repeat("consectetur adipiscing elit ", 200)

	m := sampleMemory()
	m.TopicKey = "scene/long"
	m.Title = "Long content"
	m.Content = long
	seed(t, st, m)

	results, err := st.Search(ctx, brainID, "marker", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if got := len(results[0].Snippet); got > 300 {
		t.Errorf("snippet length must be ≤300, got %d chars", got)
	}
	if !strings.Contains(strings.ToLower(results[0].Snippet), "marker") {
		t.Errorf("snippet should contain the matched term: %q", results[0].Snippet)
	}
}

func TestSearch_SanitizesFTSOperators(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.TopicKey = "scene/safe"
	m.Title = "Asset import workflow"
	m.Content = "tag engine:playcanvas notes"
	seed(t, st, m)

	// Raw FTS5 special chars must NOT crash. They get treated as literal terms.
	queries := []string{
		`engine:playcanvas`,
		`"unbalanced`,
		`asset*`,
		`a^b`,
		`(broken`,
	}
	for _, q := range queries {
		_, err := st.Search(ctx, brainID, q, SearchOptions{})
		if err != nil {
			t.Errorf("query %q should not error after sanitization, got: %v", q, err)
		}
	}
}

func TestSearch_LimitDefaultsTo10(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	for i := 0; i < 15; i++ {
		m := sampleMemory()
		m.TopicKey = "scene/" + string(rune('a'+i))
		m.Title = "Lantern entry"
		m.Content = "lantern lantern lantern"
		seed(t, st, m)
	}

	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 10 {
		t.Errorf("default limit must be 10, got %d", len(results))
	}
}

func TestSearch_LimitClampsAt50(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	for i := 0; i < 60; i++ {
		m := sampleMemory()
		m.TopicKey = "scene/" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		m.Title = "Lantern entry"
		m.Content = "lantern lantern lantern"
		seed(t, st, m)
	}

	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{Limit: 999})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 50 {
		t.Errorf("limit must clamp at 50, got %d", len(results))
	}
}

func TestSearch_OffsetSkipsResults(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	for i := 0; i < 5; i++ {
		m := sampleMemory()
		m.TopicKey = "scene/" + string(rune('a'+i))
		m.Title = "Lantern entry"
		m.Content = "lantern lantern"
		seed(t, st, m)
	}

	page1, err := st.Search(ctx, brainID, "lantern", SearchOptions{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("search page1: %v", err)
	}
	page2, err := st.Search(ctx, brainID, "lantern", SearchOptions{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("search page2: %v", err)
	}
	if len(page1) != 2 || len(page2) != 2 {
		t.Fatalf("expected 2 per page, got %d / %d", len(page1), len(page2))
	}
	// Pages must not overlap by id.
	for _, a := range page1 {
		for _, b := range page2 {
			if a.ID == b.ID {
				t.Errorf("pagination overlap: id %d in both pages", a.ID)
			}
		}
	}
}

func TestSearch_EmptyResultSet(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.TopicKey = "scene/x"
	m.Title = "Lantern"
	m.Content = "lantern lantern"
	seed(t, st, m)

	results, err := st.Search(ctx, brainID, "zxqvbn", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestSearch_FilterByTopicKeyGlob(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	a := sampleMemory()
	a.TopicKey = "design/auth/oauth"
	a.Title = "OAuth"
	a.Content = "tokens tokens tokens"
	seed(t, st, a)

	b := sampleMemory()
	b.TopicKey = "scene/lighting"
	b.Title = "Lighting"
	b.Content = "tokens tokens tokens"
	seed(t, st, b)

	// FTS query "tokens" matches both, but the topic_key filter restricts.
	results, err := st.Search(ctx, brainID, "tokens", SearchOptions{TopicKey: "design/*"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result filtered by topic_key glob, got %d", len(results))
	}
	if !strings.HasPrefix(results[0].TopicKey, "design/") {
		t.Errorf("expected topic_key prefix design/, got %q", results[0].TopicKey)
	}
}

func TestSearch_FilterByTags_SingleTag(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	tagged := sampleMemory()
	tagged.TopicKey = "scene/tagged"
	tagged.Title = "Lantern tagged"
	tagged.Content = "lantern lantern lantern"
	tagged.Tags = []string{"engine:unity", "platform:android"}
	seed(t, st, tagged)

	untagged := sampleMemory()
	untagged.TopicKey = "scene/untagged"
	untagged.Title = "Lantern untagged"
	untagged.Content = "lantern lantern lantern"
	seed(t, st, untagged)

	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{Tags: []string{"engine:unity"}})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result with tag engine:unity, got %d", len(results))
	}
	if results[0].TopicKey != "scene/tagged" {
		t.Errorf("expected tagged row, got topic_key=%q", results[0].TopicKey)
	}
}

func TestSearch_FilterByTags_ANDSemantics(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	both := sampleMemory()
	both.TopicKey = "scene/both"
	both.Title = "Lantern both tags"
	both.Content = "lantern lantern lantern"
	both.Tags = []string{"engine:unity", "platform:android"}
	seed(t, st, both)

	oneOnly := sampleMemory()
	oneOnly.TopicKey = "scene/one"
	oneOnly.Title = "Lantern one tag"
	oneOnly.Content = "lantern lantern lantern"
	oneOnly.Tags = []string{"engine:unity"}
	seed(t, st, oneOnly)

	// Require both tags — only "both" should match.
	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{Tags: []string{"engine:unity", "platform:android"}})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("AND semantics: expected 1 result, got %d", len(results))
	}
	if results[0].TopicKey != "scene/both" {
		t.Errorf("expected scene/both, got topic_key=%q", results[0].TopicKey)
	}
}

func TestSearch_FilterByTags_NoMatch(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.TopicKey = "scene/x"
	m.Title = "Lantern"
	m.Content = "lantern lantern"
	m.Tags = []string{"engine:playcanvas"}
	seed(t, st, m)

	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{Tags: []string{"engine:unity"}})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("tag filter with no match should return 0 results, got %d", len(results))
	}
}

func TestSearch_RecentFirst_OrdersByUpdatedAt(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	older := sampleMemory()
	older.TopicKey = "scene/older"
	older.Title = "Lantern older"
	older.Content = "lantern lantern"
	savedOlder := seed(t, st, older)

	// Advance the clock so the second memory has a later updated_at.
	st.SetClock(func() time.Time { return savedOlder.UpdatedAt.Add(5 * time.Second) })

	newer := sampleMemory()
	newer.TopicKey = "scene/newer"
	newer.Title = "Lantern newer"
	newer.Content = "lantern lantern"
	seed(t, st, newer)

	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{RecentFirst: true})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].TopicKey != "scene/newer" {
		t.Errorf("recent_first: expected scene/newer first, got %q", results[0].TopicKey)
	}
	if results[0].Score != 0.0 {
		t.Errorf("recent_first: score must be 0.0, got %v", results[0].Score)
	}
}

func TestSearch_RecentFirst_FalseKeepsBM25Order(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	// Weak BM25 match but inserted later — BM25 should still win without recent_first.
	weak := sampleMemory()
	weak.TopicKey = "scene/weak"
	weak.Title = "Lantern weak"
	weak.Content = "lantern once"
	savedWeak := seed(t, st, weak)

	st.SetClock(func() time.Time { return savedWeak.UpdatedAt.Add(5 * time.Second) })

	strong := sampleMemory()
	strong.TopicKey = "scene/strong"
	strong.Title = "Lantern strong"
	strong.Content = "lantern lantern lantern lantern lantern"
	seed(t, st, strong)

	// Without recent_first, BM25 should rank the strong match first.
	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{RecentFirst: false})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].TopicKey != "scene/strong" {
		t.Errorf("BM25 order: expected scene/strong first, got %q", results[0].TopicKey)
	}
}

func TestSearch_RevisionCountIncluded(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.TopicKey = "scene/rev"
	m.Title = "Lantern bake v1"
	m.Content = "first version of the bake notes lantern"
	saved1 := seed(t, st, m)

	m.Content = "second version of the bake notes lantern"
	if _, _, err := st.Save(ctx, brainID, m); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	results, err := st.Search(ctx, brainID, "lantern", SearchOptions{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != saved1.ID {
		t.Errorf("upsert must keep id, got %d → %d", saved1.ID, results[0].ID)
	}
	if results[0].RevisionCount != 1 {
		t.Errorf("expected revision_count=1 after one upsert, got %d", results[0].RevisionCount)
	}
}
