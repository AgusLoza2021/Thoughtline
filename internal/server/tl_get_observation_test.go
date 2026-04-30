package server

import (
	"context"
	"strings"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

func TestDoGetObservation_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	saved := seedMemory(t, st, memory.Memory{
		Project:  "enchanted-inn",
		Scope:    memory.ScopeProject,
		Type:     memory.TypeScenePattern,
		TopicKey: "scene/lantern",
		Title:    "Lantern bake workflow",
		Content:  "Bake lantern-base normals before exporting to PlayCanvas. " + strings.Repeat("more details. ", 50),
		Tags:     []string{"engine:playcanvas", "platform:android"},
	})

	res, err := doGetObservation(ctx, st, getObservationArgs{ID: saved.ID})
	if err != nil {
		t.Fatalf("doGetObservation: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	// Untruncated content must come back in full — the full saved.Content.
	if !strings.Contains(body, saved.Content) {
		t.Errorf("response should contain the full untruncated content, got:\n%s", body)
	}
	if !strings.Contains(body, "Lantern bake workflow") {
		t.Errorf("response should include title, got:\n%s", body)
	}
	if !strings.Contains(body, "scene/lantern") {
		t.Errorf("response should include topic_key, got:\n%s", body)
	}
	if !strings.Contains(body, "engine:playcanvas") {
		t.Errorf("response should include tags, got:\n%s", body)
	}
}

func TestDoGetObservation_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doGetObservation(ctx, st, getObservationArgs{ID: 999})
	if err != nil {
		t.Fatalf("doGetObservation: %v", err)
	}
	if !res.IsError {
		t.Fatalf("missing id should yield error result, got success: %s", textContent(res))
	}
	body := strings.ToLower(textContent(res))
	if !strings.Contains(body, "not found") && !strings.Contains(body, "no memory") {
		t.Errorf("error message should clearly state not found, got:\n%s", textContent(res))
	}
}

func TestDoGetObservation_MissingID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doGetObservation(ctx, st, getObservationArgs{ID: 0})
	if err != nil {
		t.Fatalf("doGetObservation: %v", err)
	}
	if !res.IsError {
		t.Fatalf("id=0 should yield validation error, got success: %s", textContent(res))
	}
	if !strings.Contains(textContent(res), "id") {
		t.Errorf("error message should mention 'id', got:\n%s", textContent(res))
	}
}
