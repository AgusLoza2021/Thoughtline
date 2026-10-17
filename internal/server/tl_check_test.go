package server

import (
	"context"
	"strings"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/knowledge"
	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// seedKnowledgeCorpus imports the testdata recipe corpus into the given brain,
// which is the state a real deployment has after `thoughtline knowledge sync`.
func seedKnowledgeCorpus(t *testing.T, st *storage.Storage, project string) {
	t.Helper()
	ctx := context.Background()
	corpus, err := knowledge.Scan("../../internal/knowledge/testdata/corpus")
	if err != nil {
		t.Fatalf("scan corpus: %v", err)
	}
	if _, err := knowledge.Sync(ctx, st, corpus, knowledge.SyncOptions{Project: project}); err != nil {
		t.Fatalf("sync corpus: %v", err)
	}
}

func TestDecodeCheckArgs(t *testing.T) {
	req := buildReq("tl_check", map[string]any{
		"task":              " add camera shake ",
		"paths":             []any{"Assets/A.cs", "Assets/B.cs"},
		"project":           " ninjasurvivor ",
		"knowledge_project": "knowledge-unity",
		"exclude_stacks":    []any{"svn", "fiber"},
		"limit":             float64(3),
	})

	args := decodeCheckArgs(req)
	if args.Task != "add camera shake" {
		t.Errorf("Task = %q", args.Task)
	}
	if len(args.Paths) != 2 || args.Paths[1] != "Assets/B.cs" {
		t.Errorf("Paths = %v", args.Paths)
	}
	if args.Project != "ninjasurvivor" {
		t.Errorf("Project = %q", args.Project)
	}
	if args.KnowledgeProject != "knowledge-unity" {
		t.Errorf("KnowledgeProject = %q", args.KnowledgeProject)
	}
	if len(args.ExcludeStacks) != 2 || args.ExcludeStacks[0] != "svn" {
		t.Errorf("ExcludeStacks = %v", args.ExcludeStacks)
	}
	if args.Limit != 3 {
		t.Errorf("Limit = %d", args.Limit)
	}
}

func TestDoCheck_RequiresTaskOrPaths(t *testing.T) {
	st := newTestStorage(t)
	res, err := doCheck(context.Background(), st, Config{}, checkArgs{})
	if err != nil {
		t.Fatalf("doCheck: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected an error envelope when neither task nor paths is given")
	}
	if !strings.Contains(textContent(res), "task") {
		t.Errorf("error should name the missing arguments, got %q", textContent(res))
	}
}

func TestDoCheck_RendersMarkdown(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	seedKnowledgeCorpus(t, st, knowledge.DefaultSyncProject)

	// A project rule: it must be listed before the recipes.
	seedMemory(t, st, memory.Memory{
		Project:  "ninjasurvivor",
		Scope:    memory.ScopeProject,
		Type:     memory.TypeConvention,
		TopicKey: "convention/camera-feel",
		Title:    "Camera feel lives in the FeelConfig asset",
		Content:  "All camera shake values are authored in the FeelConfig asset.",
	})

	res, err := doCheck(ctx, st, Config{DefaultProject: "ninjasurvivor"}, checkArgs{
		Task: "add camera shake when the player gets hit",
	})
	if err != nil {
		t.Fatalf("doCheck: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error envelope: %s", textContent(res))
	}
	body := textContent(res)

	for _, want := range []string{
		"### [project-rule] Camera feel lives in the FeelConfig asset (convention/camera-feel, score 1.00)",
		"### [knowledge] Camera Shake as an Additive Local Position Offset (knowledge/camera-shake",
		"— coverage 2: camera, shake",
		"— coverage 3: camera, shake, hit",
		"**Applies when**",
		"**Validate**",
		"**Limits**",
		"excluded 0 foreign-stack entries",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q:\n%s", want, body)
		}
	}
	if strings.Index(body, "[project-rule]") > strings.Index(body, "[knowledge]") {
		t.Errorf("project rules must be listed first:\n%s", body)
	}
}

func TestDoCheck_ExcludeStacksFromConfig(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	seedKnowledgeCorpus(t, st, knowledge.DefaultSyncProject)

	cfg := Config{KnowledgeExclude: []string{"svn"}}
	res, err := doCheck(ctx, st, cfg, checkArgs{Task: "svn externals audit before migration"})
	if err != nil {
		t.Fatalf("doCheck: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "excluded 1 foreign-stack entries") {
		t.Errorf("config exclude list not applied:\n%s", body)
	}
	if strings.Contains(body, "knowledge/legacy-svn-audit") {
		t.Errorf("excluded entry leaked into the answer:\n%s", body)
	}
}

func TestDoCheck_KnowledgeProjectFromConfig(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	seedKnowledgeCorpus(t, st, "custom-corpus")

	// The default project holds nothing: only the configured brain can answer.
	empty, err := doCheck(ctx, st, Config{}, checkArgs{Task: "camera shake"})
	if err != nil {
		t.Fatalf("doCheck: %v", err)
	}
	if strings.Contains(textContent(empty), "knowledge/camera-shake") {
		t.Fatalf("expected no hits from the default corpus brain:\n%s", textContent(empty))
	}

	res, err := doCheck(ctx, st, Config{KnowledgeProject: "custom-corpus", KnowledgeMinWeight: 0.0001}, checkArgs{Task: "camera shake"})
	if err != nil {
		t.Fatalf("doCheck: %v", err)
	}
	if !strings.Contains(textContent(res), "knowledge/camera-shake") {
		t.Errorf("configured knowledge project not used:\n%s", textContent(res))
	}
}

func TestDoCheck_NoHitsNamesTheClosestCandidate(t *testing.T) {
	st := newTestStorage(t)
	seedKnowledgeCorpus(t, st, knowledge.DefaultSyncProject)

	// "camera" is the only shared term: below the default gate of two, so the
	// answer must say the corpus was consulted and how close it came. Without
	// that line an agent cannot tell "nothing applies" from "nothing was read".
	res, err := doCheck(context.Background(), st, Config{}, checkArgs{Task: "camera kubernetes ingress"})
	if err != nil {
		t.Fatalf("doCheck: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "No applicable knowledge for this task (best candidate had coverage 1: Camera Shake") {
		t.Errorf("empty answer should name the closest candidate, got:\n%s", body)
	}
}

func TestDoCheck_MinCoverageFromConfig(t *testing.T) {
	st := newTestStorage(t)
	seedKnowledgeCorpus(t, st, knowledge.DefaultSyncProject)

	res, err := doCheck(context.Background(), st, Config{KnowledgeMinCoverage: 3}, checkArgs{Task: "camera shake"})
	if err != nil {
		t.Fatalf("doCheck: %v", err)
	}
	if !strings.Contains(textContent(res), "No applicable knowledge") {
		t.Errorf("MinCoverage 3 should reject a two-term match, got:\n%s", textContent(res))
	}
}

func TestDoCheck_NoHits(t *testing.T) {
	st := newTestStorage(t)
	seedKnowledgeCorpus(t, st, knowledge.DefaultSyncProject)

	res, err := doCheck(context.Background(), st, Config{}, checkArgs{Task: "kubernetes ingress controller"})
	if err != nil {
		t.Fatalf("doCheck: %v", err)
	}
	body := textContent(res)
	if !strings.Contains(body, "No applicable knowledge") {
		t.Errorf("empty answer should say so, got:\n%s", body)
	}
	if !strings.Contains(body, "excluded 0 foreign-stack entries") {
		t.Errorf("excluded line must always be present, got:\n%s", body)
	}
}
