package server

import (
	"context"
	"strings"
	"testing"
)

func TestDoRelated_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	center := seedMemory(t, st, sampleSceneMemory("Center", "center body", "scene/center"))
	other := seedMemory(t, st, sampleSceneMemory("Other", "other body", "scene/other"))

	if _, err := doLink(ctx, st, Config{}, linkArgs{
		FromID:   center.ID,
		ToID:     other.ID,
		Relation: "depends_on",
		Note:     "needs the other to work",
		Project:  "enchanted-inn",
	}); err != nil {
		t.Fatalf("doLink: %v", err)
	}

	res, err := doRelated(ctx, st, Config{}, relatedArgs{
		ID:      center.ID,
		Project: "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doRelated: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "depends_on") {
		t.Errorf("response should include relation, got:\n%s", body)
	}
	if !strings.Contains(body, "Other") {
		t.Errorf("response should include linked memory title, got:\n%s", body)
	}
	if !strings.Contains(body, "needs the other to work") {
		t.Errorf("response should include note, got:\n%s", body)
	}
	if !strings.Contains(body, "→") {
		t.Errorf("response should show outgoing direction indicator, got:\n%s", body)
	}
}

func TestDoRelated_EmptyResult(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	m := seedMemory(t, st, sampleSceneMemory("Lonely", "no links", "scene/lonely"))

	res, err := doRelated(ctx, st, Config{}, relatedArgs{
		ID:      m.ID,
		Project: "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doRelated: %v", err)
	}
	if res.IsError {
		t.Fatalf("no links is not an error, got: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "No links found") {
		t.Errorf("response should say 'No links found', got:\n%s", body)
	}
}

func TestDoRelated_DirectionFrom(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	center := seedMemory(t, st, sampleSceneMemory("CenterF", "body", "scene/centerf"))
	outgoing := seedMemory(t, st, sampleSceneMemory("Outgoing", "body", "scene/out"))
	incoming := seedMemory(t, st, sampleSceneMemory("Incoming", "body", "scene/in"))

	// center → outgoing
	if _, err := doLink(ctx, st, Config{}, linkArgs{
		FromID: center.ID, ToID: outgoing.ID, Relation: "references", Project: "enchanted-inn",
	}); err != nil {
		t.Fatalf("doLink out: %v", err)
	}
	// incoming → center
	if _, err := doLink(ctx, st, Config{}, linkArgs{
		FromID: incoming.ID, ToID: center.ID, Relation: "related", Project: "enchanted-inn",
	}); err != nil {
		t.Fatalf("doLink in: %v", err)
	}

	res, err := doRelated(ctx, st, Config{}, relatedArgs{
		ID:        center.ID,
		Direction: "from",
		Project:   "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doRelated: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "1 found") {
		t.Errorf("should show 1 link, got:\n%s", body)
	}
	if !strings.Contains(body, "Outgoing") {
		t.Errorf("should include outgoing memory, got:\n%s", body)
	}
	if strings.Contains(body, "Incoming") {
		t.Errorf("should NOT include incoming memory with direction=from, got:\n%s", body)
	}
}

func TestDoRelated_DirectionTo(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	center := seedMemory(t, st, sampleSceneMemory("CenterT", "body", "scene/centert"))
	outgoing := seedMemory(t, st, sampleSceneMemory("OutgoingT", "body", "scene/outt"))
	incoming := seedMemory(t, st, sampleSceneMemory("IncomingT", "body", "scene/int"))

	// center → outgoing
	if _, err := doLink(ctx, st, Config{}, linkArgs{
		FromID: center.ID, ToID: outgoing.ID, Relation: "references", Project: "enchanted-inn",
	}); err != nil {
		t.Fatalf("doLink out: %v", err)
	}
	// incoming → center
	if _, err := doLink(ctx, st, Config{}, linkArgs{
		FromID: incoming.ID, ToID: center.ID, Relation: "related", Project: "enchanted-inn",
	}); err != nil {
		t.Fatalf("doLink in: %v", err)
	}

	res, err := doRelated(ctx, st, Config{}, relatedArgs{
		ID:        center.ID,
		Direction: "to",
		Project:   "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doRelated: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "1 found") {
		t.Errorf("should show 1 link, got:\n%s", body)
	}
	if !strings.Contains(body, "IncomingT") {
		t.Errorf("should include incoming memory, got:\n%s", body)
	}
	if strings.Contains(body, "OutgoingT") {
		t.Errorf("should NOT include outgoing memory with direction=to, got:\n%s", body)
	}
}

func TestDoRelated_MissingID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doRelated(ctx, st, Config{}, relatedArgs{ID: 0})
	if err != nil {
		t.Fatalf("doRelated: %v", err)
	}
	if !res.IsError {
		t.Fatalf("id=0 should yield error")
	}
}

func TestDoRelated_InvalidDirection(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doRelated(ctx, st, Config{}, relatedArgs{ID: 1, Direction: "sideways"})
	if err != nil {
		t.Fatalf("doRelated: %v", err)
	}
	if !res.IsError {
		t.Fatalf("invalid direction should yield error")
	}
}
