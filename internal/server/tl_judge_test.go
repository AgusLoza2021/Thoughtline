package server

import (
	"context"
	"strings"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

func TestDoJudge_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seedMemory(t, st, memory.Memory{
		Project:  "enchanted-inn",
		Scope:    memory.ScopeProject,
		Type:     memory.TypeScenePattern,
		TopicKey: "scene/lantern",
		Title:    "Lantern bake workflow",
		Content:  "Bake lantern-base normals before exporting to PlayCanvas.",
	})

	res, err := doJudge(ctx, st, Config{}, judgeArgs{
		ExistingID:      saved.ID,
		IncomingTitle:   "Lantern bake workflow v2",
		IncomingContent: "Updated bake pipeline with LOD support.",
		Relation:        "supersedes",
	})
	if err != nil {
		t.Fatalf("doJudge: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "Lantern bake workflow") {
		t.Errorf("response should include existing title, got:\n%s", body)
	}
	if !strings.Contains(body, "supersedes") {
		t.Errorf("response should include relation, got:\n%s", body)
	}
	if !strings.Contains(body, "save with same topic_key to replace") {
		t.Errorf("response should include recommended action for supersedes, got:\n%s", body)
	}
}

func TestDoJudge_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doJudge(ctx, st, Config{}, judgeArgs{
		ExistingID:      99999,
		IncomingTitle:   "Some title",
		IncomingContent: "Some content",
		Relation:        "not_conflict",
	})
	if err != nil {
		t.Fatalf("doJudge: %v", err)
	}
	if !res.IsError {
		t.Fatalf("non-existent id should yield error, got success: %s", textContent(res))
	}
	body := strings.ToLower(textContent(res))
	if !strings.Contains(body, "no memory found") {
		t.Errorf("error should say no memory found, got:\n%s", textContent(res))
	}
}

func TestDoJudge_InvalidRelation(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seedMemory(t, st, memory.Memory{
		Project:  "enchanted-inn",
		Scope:    memory.ScopeProject,
		Type:     memory.TypeScenePattern,
		TopicKey: "scene/lantern",
		Title:    "Lantern bake workflow",
		Content:  "Some content.",
	})

	res, err := doJudge(ctx, st, Config{}, judgeArgs{
		ExistingID:      saved.ID,
		IncomingTitle:   "New title",
		IncomingContent: "New content",
		Relation:        "totally_wrong",
	})
	if err != nil {
		t.Fatalf("doJudge: %v", err)
	}
	if !res.IsError {
		t.Fatalf("invalid relation should yield error, got success: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "supersedes") || !strings.Contains(body, "compatible") {
		t.Errorf("error should list valid relations, got:\n%s", body)
	}
}

func TestDoJudge_AllRelations(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seedMemory(t, st, memory.Memory{
		Project:  "enchanted-inn",
		Scope:    memory.ScopeProject,
		Type:     memory.TypeScenePattern,
		TopicKey: "scene/lantern",
		Title:    "Lantern bake workflow",
		Content:  "Some content.",
	})

	cases := []struct {
		relation          string
		wantRecommendation string
	}{
		{"supersedes", "save with same topic_key to replace"},
		{"compatible", "save as new memory (different topic_key or no topic_key)"},
		{"conflicts_with", "resolve manually before saving"},
		{"scoped", "save as new memory, both are valid in their scope"},
		{"not_conflict", "safe to save"},
	}

	for _, tc := range cases {
		t.Run(tc.relation, func(t *testing.T) {
			res, err := doJudge(ctx, st, Config{}, judgeArgs{
				ExistingID:      saved.ID,
				IncomingTitle:   "Incoming title",
				IncomingContent: "Incoming content",
				Relation:        tc.relation,
			})
			if err != nil {
				t.Fatalf("doJudge(%s): %v", tc.relation, err)
			}
			if res.IsError {
				t.Fatalf("relation %q should succeed, got error: %s", tc.relation, textContent(res))
			}
			body := textContent(res)
			if !strings.Contains(body, tc.wantRecommendation) {
				t.Errorf("relation %q: expected recommendation %q in:\n%s", tc.relation, tc.wantRecommendation, body)
			}
		})
	}
}
