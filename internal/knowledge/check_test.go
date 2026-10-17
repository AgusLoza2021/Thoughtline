package knowledge

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func newCheckFixture(t *testing.T) (context.Context, *storage.Storage) {
	t.Helper()
	ctx := context.Background()
	st := newSyncStorage(t)
	if _, err := Sync(ctx, st, mustScan(t, "testdata/corpus"), SyncOptions{}); err != nil {
		t.Fatalf("sync testdata corpus: %v", err)
	}
	return ctx, st
}

func TestBuildQuery(t *testing.T) {
	cases := []struct {
		name  string
		task  string
		paths []string
		want  string
	}{
		{
			name: "stopwords dropped",
			task: "add camera shake when the player gets hit",
			want: "camera shake player hit",
		},
		{
			name: "camel case split",
			task: "Fix EnemyHitFlashSystem.cs",
			want: "enemy hit flash system",
		},
		{
			name: "generic code words dropped",
			task: "fix the game state setup for every core view",
			want: "",
		},
		{
			name: "generic code words do not crowd out the subject",
			task: "game state setup for camera shake",
			want: "camera shake",
		},
		{
			name:  "vendor dirs dropped before the camel case split",
			paths: []string{"Assets/GorodiskiGames/Survivalio/Scripts/Core/LoadingGate.cs"},
			want:  "loading gate",
		},
		{
			name:  "vendor dirs dropped on windows paths",
			paths: []string{`A:\Repo\NinjaSurvivor\Assets\GorodiskiGames\Survivalio\GameStartBehaviour.cs`},
			want:  "ninja survivor behaviour",
		},
		{
			name:  "generic dirs and extension dropped",
			paths: []string{"Assets/Scripts/Runtime/EnemyHitFlashSystem.cs"},
			want:  "enemy hit flash system",
		},
		{
			name:  "windows path",
			paths: []string{`A:\Repo\NinjaSurvivor\Assets\EnemyHitFlashSystem.cs`},
			want:  "ninja survivor enemy hit flash system",
		},
		{
			name:  "generic-only path yields nothing",
			paths: []string{"Assets/Scripts/"},
			want:  "",
		},
		{
			name: "dedupe across case",
			task: "camera camera CameraShake",
			want: "camera shake",
		},
		{
			name: "task words come before path words",
			task: "pool VFX particles",
			paths: []string{
				"Assets/Scripts/Vfx/EnemyHitFlashSystem.cs",
			},
			want: "pool vfx particles enemy hit flash system",
		},
		{
			name: "cap at maxQueryTokens",
			task: "word01 word02 word03 word04 word05 word06 word07 word08 word09 word10 " +
				"word11 word12 word13 word14 word15 word16 word17 word18 word19 word20",
			want: "word01 word02 word03 word04 word05 word06 word07 word08 " +
				"word09 word10 word11 word12 word13 word14 word15 word16",
		},
		{
			name: "no usable words",
			task: "the and of for with",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BuildQuery(tc.task, tc.paths); got != tc.want {
				t.Errorf("BuildQuery(%q, %v) = %q, want %q", tc.task, tc.paths, got, tc.want)
			}
		})
	}
}

func TestCheck_CameraShakeFirstAndProjectRuleWins(t *testing.T) {
	ctx, st := newCheckFixture(t)
	seedProjectConvention(t, ctx, st, "ninjasurvivor", "Camera feel lives in the FeelConfig asset",
		"All camera shake values are authored in the FeelConfig asset, never in code.")

	res, err := Check(ctx, st, CheckRequest{
		Task:    "add camera shake when the player gets hit",
		Project: "ninjasurvivor",
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.Query != "camera shake player hit" {
		t.Errorf("query = %q", res.Query)
	}
	if len(res.Hits) < 2 {
		t.Fatalf("expected the project rule plus the camera-shake entry, got %d hits", len(res.Hits))
	}
	if res.Hits[0].Source != SourceProjectRule {
		t.Errorf("first hit source = %q, want %q (project rules win)", res.Hits[0].Source, SourceProjectRule)
	}
	if !strings.Contains(strings.ToLower(res.Hits[0].Title), "feelconfig") {
		t.Errorf("first hit should be the project convention, got %q", res.Hits[0].Title)
	}
	if got := res.Hits[1].TopicKey; got != "knowledge/camera-shake" {
		t.Errorf("second hit topic_key = %q, want knowledge/camera-shake", got)
	}

	shake := res.Hits[1]
	if !strings.Contains(shake.AppliesWhen, "impact") {
		t.Errorf("Applies when not extracted: %q", shake.AppliesWhen)
	}
	if !strings.Contains(shake.Validate, "pan") {
		t.Errorf("Validate not extracted: %q", shake.Validate)
	}
	if !strings.Contains(shake.Limits, "hit-stop") {
		t.Errorf("Limits not extracted: %q", shake.Limits)
	}
	// Coverage is what the absolute gate reads: title + tags + Applies when.
	// The task names three of its terms, so this entry must report all three.
	if shake.Coverage != 3 {
		t.Errorf("camera-shake coverage = %d, want 3 (%v)", shake.Coverage, shake.Matched)
	}
	if got := strings.Join(shake.Matched, ","); got != "camera,shake,hit" {
		t.Errorf("matched = %q, want camera,shake,hit", got)
	}
	if res.Hits[0].Coverage < 2 {
		t.Errorf("project rule coverage = %d, want >= 2", res.Hits[0].Coverage)
	}
	for _, h := range res.Hits {
		if h.Score <= 0 || h.Score > 1 {
			t.Errorf("score out of (0,1]: %v (%s)", h.Score, h.Title)
		}
	}
	if strings.Contains(shake.Limits, "_knowledge:") {
		t.Errorf("provenance footer leaked into a section: %q", shake.Limits)
	}
}

func TestCheck_RecipeSectionsComeFromItsTemplate(t *testing.T) {
	ctx, st := newCheckFixture(t)

	res, err := Check(ctx, st, CheckRequest{Paths: []string{"Assets/UI/AnimatedFillBar.cs"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d: %+v", len(res.Hits), res.Hits)
	}
	hit := res.Hits[0]
	if hit.TopicKey != "knowledge/animated-fill-bar" {
		t.Fatalf("topic_key = %q", hit.TopicKey)
	}
	// A recipe has no "Applies when": Context is the gate, Example is the proof.
	if !strings.Contains(hit.AppliesWhen, "fillAmount") {
		t.Errorf("Context should fill Applies when, got %q", hit.AppliesWhen)
	}
	if !strings.Contains(hit.Validate, "RectTransform") {
		t.Errorf("Example should fill Validate, got %q", hit.Validate)
	}
	if !strings.Contains(hit.Limits, "BOM") {
		t.Errorf("Notes should fill Limits, got %q", hit.Limits)
	}
}

func TestCheck_ExcludesForeignStack(t *testing.T) {
	ctx, st := newCheckFixture(t)

	res, err := Check(ctx, st, CheckRequest{
		Task:          "svn externals audit before migration",
		ExcludeStacks: []string{"svn"},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.Excluded != 1 {
		t.Fatalf("Excluded = %d, want 1 (the svn entry)", res.Excluded)
	}
	for _, h := range res.Hits {
		for _, s := range h.Stacks {
			if s == "svn" {
				t.Errorf("hit %q carries the excluded stack", h.Title)
			}
		}
	}

	// Accepting the tag form keeps one value usable as both env item and tag.
	res2, err := Check(ctx, st, CheckRequest{
		Task:          "svn externals audit before migration",
		ExcludeStacks: []string{"stack:svn"},
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res2.Excluded != 1 {
		t.Errorf("stack: prefix not normalized, Excluded = %d", res2.Excluded)
	}
}

func TestCheck_MinScoreGatesWeakHits(t *testing.T) {
	ctx, st := newCheckFixture(t)
	brainID, err := st.ResolveOrCreateBrainID(ctx, DefaultSyncProject)
	if err != nil {
		t.Fatalf("resolve knowledge brain: %v", err)
	}
	// A short entry that mentions both terms once: it matches, but ranks below
	// the real recipe, which is exactly the tail MinScore exists to cut.
	medium := memory.Memory{
		Project:  DefaultSyncProject,
		Scope:    memory.ScopeProject,
		Type:     memory.TypeConvention,
		TopicKey: "knowledge/medium-camera-note",
		Title:    "Camera shake notes",
		Content:  "One camera shake per hit is enough.",
		Tags:     []string{KindTag, "corpus:unity", "template:procedural"},
	}
	if _, _, err := st.Save(ctx, brainID, medium); err != nil {
		t.Fatalf("save medium entry: %v", err)
	}

	all, err := Check(ctx, st, CheckRequest{Task: "camera shake", MinScore: 0.0001, MinWeight: 0.0001})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(all.Hits) < 2 {
		t.Fatalf("expected at least 2 hits before gating, got %d", len(all.Hits))
	}

	gated, err := Check(ctx, st, CheckRequest{Task: "camera shake", MinScore: 0.9})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(gated.Hits) >= len(all.Hits) {
		t.Fatalf("MinScore 0.9 kept %d hits, want fewer than %d", len(gated.Hits), len(all.Hits))
	}
	for _, h := range gated.Hits {
		if h.Score < 0.9 {
			t.Errorf("gated hit %q has score %.3f", h.Title, h.Score)
		}
	}

	// The default gate keeps the strong hit and drops the footer-only noise:
	// a footer is provenance, not what the entry is about, so only the anchor
	// (title, tags, gate section) may earn coverage.
	def, err := Check(ctx, st, CheckRequest{Task: "camera shake"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	for _, h := range def.Hits {
		if h.Score < DefaultMinScore {
			t.Errorf("default check returned hit %q below DefaultMinScore: %.4f", h.Title, h.Score)
		}
	}
}

// TestCheck_IDFWeightsRareTokensOverCommonOnes is the reason weighting exists:
// coverage alone ranks an entry that claims "panel layout" as high as one that
// claims "hologram nebula", so a tangent built out of words half the corpus
// uses used to pass everything the real match passed.
func TestCheck_IDFWeightsRareTokensOverCommonOnes(t *testing.T) {
	ctx, st := newCheckFixture(t)
	brainID, err := st.ResolveOrCreateBrainID(ctx, DefaultSyncProject)
	if err != nil {
		t.Fatalf("resolve knowledge brain: %v", err)
	}

	seedKnowledgeRow(t, ctx, st, brainID, "knowledge/hologram-nebula",
		"Hologram nebula bake", "A hologram nebula needs its own bake pass.")
	for i := 0; i < 4; i++ {
		seedKnowledgeRow(t, ctx, st, brainID, "knowledge/panel-layout-notes-"+strconv.Itoa(i),
			"Panel layout notes", "A panel layout is laid out once in the prefab.")
	}

	const task = "hologram nebula panel layout"

	// Both entries cover two of the four terms, so only the weight tells them
	// apart: the rare pair sums to more than the common pair.
	ranked, err := Check(ctx, st, CheckRequest{Task: task, MinScore: 0.0001, MinWeight: 0.0001})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(ranked.Hits) < 2 {
		t.Fatalf("expected both entries before any weight gate, got %+v", ranked.Hits)
	}
	rare, common := ranked.Hits[0], ranked.Hits[1]
	if rare.TopicKey != "knowledge/hologram-nebula" {
		t.Fatalf("first hit = %q, want the rare-token entry", rare.TopicKey)
	}
	if !strings.HasPrefix(common.TopicKey, "knowledge/panel-layout-notes-") {
		t.Fatalf("second hit = %q, want a common-token entry", common.TopicKey)
	}
	if rare.Coverage != common.Coverage {
		t.Fatalf("coverage differs (%d vs %d): the test would not prove the weight", rare.Coverage, common.Coverage)
	}
	if rare.Weight <= common.Weight {
		t.Errorf("rare weight %.2f <= common weight %.2f", rare.Weight, common.Weight)
	}

	// A gate between the two weights keeps the rare match and drops the entry
	// whose only matches are tokens four rows share.
	gated, err := Check(ctx, st, CheckRequest{Task: task, MinWeight: common.Weight + 0.01})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(gated.Hits) != 1 || gated.Hits[0].TopicKey != "knowledge/hologram-nebula" {
		t.Fatalf("weight gate kept %+v, want only the rare-token entry", gated.Hits)
	}
}

func seedKnowledgeRow(t *testing.T, ctx context.Context, st *storage.Storage, brainID int64, topicKey, title, prose string) {
	t.Helper()
	m := memory.Memory{
		Project:  DefaultSyncProject,
		Scope:    memory.ScopeProject,
		Type:     memory.TypeConvention,
		TopicKey: topicKey,
		Title:    title,
		Content:  "## Applies when\n\n" + prose,
		Tags:     []string{KindTag, "corpus:unity", "template:procedural"},
	}
	if _, _, err := st.Save(ctx, brainID, m); err != nil {
		t.Fatalf("save knowledge row %q: %v", topicKey, err)
	}
}

// TestCheck_DefaultMinWeightScalesWithTheCorpus pins that the default gate is
// derived, not pinned: on the 4-row fixture three unique terms clear it, while
// "camera" (a group tag on two of the four entries) is common enough that the
// default no longer trusts a two-term match.
func TestCheck_DefaultMinWeightScalesWithTheCorpus(t *testing.T) {
	ctx, st := newCheckFixture(t)

	fill, err := Check(ctx, st, CheckRequest{Paths: []string{"Assets/UI/AnimatedFillBar.cs"}})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(fill.Hits) != 1 || fill.Hits[0].TopicKey != "knowledge/animated-fill-bar" {
		t.Fatalf("unique-token hit was gated away: %+v", fill.Hits)
	}

	res, err := Check(ctx, st, CheckRequest{Task: "camera shake"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(res.Hits) != 0 {
		t.Fatalf("corpus-scaled weight default kept %+v", res.Hits)
	}
	if res.Best == nil || res.Best.Coverage != 2 {
		t.Errorf("Best = %+v, want the camera-shake entry at coverage 2", res.Best)
	}
}

func TestCoverageOf(t *testing.T) {
	cases := []struct {
		name   string
		tokens []string
		anchor string
		want   int
		match  string
	}{
		{
			name:   "distinct terms only",
			tokens: []string{"camera", "shake", "camera"},
			anchor: "Camera Shake as an offset",
			want:   2,
			match:  "camera,shake",
		},
		{
			name:   "whole words, not stems",
			tokens: []string{"hit"},
			anchor: "the player hits the wall",
			want:   0,
		},
		{
			name:   "camel case is split like the query",
			tokens: []string{"shake"},
			anchor: "CameraShake",
			want:   1,
			match:  "shake",
		},
		{
			name:   "tags lose their prefix and their hyphens",
			tokens: []string{"camera", "feel", "group"},
			anchor: tagWords([]string{"group:camera-and-feel", "stack:svn"}),
			want:   2,
			match:  "camera,feel",
		},
		{
			name:   "no anchor, no coverage",
			tokens: []string{"camera"},
			want:   0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, matched := coverageOf(tc.tokens, tc.anchor)
			if got != tc.want {
				t.Errorf("coverageOf(%v, %q) = %d, want %d", tc.tokens, tc.anchor, got, tc.want)
			}
			if strings.Join(matched, ",") != tc.match {
				t.Errorf("matched = %v, want %q", matched, tc.match)
			}
		})
	}
}

// TestCheck_CoverageGateDropsTheTopHit is the regression the absolute gate
// exists for: the bm25 best hit of a source always normalises to score 1.00, so
// only coverage can reject it.
func TestCheck_CoverageGateDropsTheTopHit(t *testing.T) {
	ctx, st := newCheckFixture(t)
	brainID, err := st.ResolveOrCreateBrainID(ctx, DefaultSyncProject)
	if err != nil {
		t.Fatalf("resolve knowledge brain: %v", err)
	}
	// One query term in the anchor, the other only in the body: FTS matches it
	// (it is the sole match, so it scores 1.00) but it does not claim to be about
	// the task.
	noise := memory.Memory{
		Project:  DefaultSyncProject,
		Scope:    memory.ScopeProject,
		Type:     memory.TypeConvention,
		TopicKey: "knowledge/shake-notes",
		Title:    "Shake notes",
		Content:  "zoom zoom zoom zoom zoom zoom",
		Tags:     []string{KindTag, "corpus:unity", "template:procedural"},
	}
	if _, _, err := st.Save(ctx, brainID, noise); err != nil {
		t.Fatalf("save noise entry: %v", err)
	}

	// With the gate at 1 the noise entry is visible and it is the only hit, so
	// it scores 1.00: the relative gate cannot be what removes it. MinWeight is
	// disabled here on purpose: this test is about the coverage gate, and the
	// corpus-scaled weight default (see the IDF tests) would also drop it.
	loose, err := Check(ctx, st, CheckRequest{Task: "zoom shake", MinCoverage: 1, MinScore: 0.0001, MinWeight: 0.0001})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(loose.Hits) == 0 || loose.Hits[0].TopicKey != "knowledge/shake-notes" {
		t.Fatalf("loose check hits = %+v, want the noise entry first", loose.Hits)
	}
	if loose.Hits[0].Score != 1 {
		t.Errorf("noise hit score = %.3f, want 1.0 (bm25 best)", loose.Hits[0].Score)
	}

	strict, err := Check(ctx, st, CheckRequest{Task: "zoom shake", MinWeight: 0.0001})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(strict.Hits) != 0 {
		t.Fatalf("single-term match survived the default gate: %+v", strict.Hits)
	}
	if strict.Best == nil || strict.Best.Coverage != 1 || strict.Best.Title != "Shake notes" {
		t.Errorf("Best = %+v, want the noise entry at coverage 1", strict.Best)
	}
}

func TestCheck_EmptyQueryAndMissingBrains(t *testing.T) {
	ctx, st := newCheckFixture(t)

	// Stopwords only: nothing to search for, no hits, no error.
	res, err := Check(ctx, st, CheckRequest{Task: "the and of"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(res.Hits) != 0 || res.Query != "" {
		t.Errorf("stopword-only task produced query=%q hits=%d", res.Query, len(res.Hits))
	}

	// A project brain that does not exist is silent, not fatal: checks run on
	// every edit and must never fail because nothing was saved yet. MinWeight is
	// disabled because the 4-entry fixture makes "camera" half of the corpus,
	// which the corpus-scaled default treats as too common to count.
	res, err = Check(ctx, st, CheckRequest{Task: "camera shake", Project: "never-used-project", MinWeight: 0.0001})
	if err != nil {
		t.Fatalf("Check with missing project brain: %v", err)
	}
	if len(res.Hits) == 0 {
		t.Error("knowledge hits should still be returned without a project brain")
	}

	// Same for the corpus brain: an empty database answers "nothing applies".
	fresh := newSyncStorage(t)
	res, err = Check(ctx, fresh, CheckRequest{Task: "camera shake"})
	if err != nil {
		t.Fatalf("Check on empty db: %v", err)
	}
	if len(res.Hits) != 0 {
		t.Errorf("empty db returned %d hits", len(res.Hits))
	}

	// A nil storage is answered as empty rather than panicking.
	if res, err = Check(ctx, nil, CheckRequest{Task: "camera shake"}); err != nil || len(res.Hits) != 0 {
		t.Errorf("nil storage: hits=%d err=%v", len(res.Hits), err)
	}
}

func TestCheck_LimitIsCapped(t *testing.T) {
	ctx, st := newCheckFixture(t)
	res, err := Check(ctx, st, CheckRequest{Task: "camera shake fill bar svn audit", Limit: 100})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(res.Hits) > MaxCheckLimit {
		t.Errorf("Limit 100 returned %d hits, cap is %d", len(res.Hits), MaxCheckLimit)
	}
}

func seedProjectConvention(t *testing.T, ctx context.Context, st *storage.Storage, project, title, content string) {
	t.Helper()
	brainID, err := st.ResolveOrCreateBrainID(ctx, project)
	if err != nil {
		t.Fatalf("resolve project brain: %v", err)
	}
	m := memory.Memory{
		Project:  project,
		Scope:    memory.ScopeProject,
		Type:     memory.TypeConvention,
		TopicKey: "convention/camera-feel",
		Title:    title,
		Content:  content,
	}
	if _, _, err := st.Save(ctx, brainID, m); err != nil {
		t.Fatalf("save project convention: %v", err)
	}
}
