package server

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDoSessionStart_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doSessionStart(ctx, st, Config{}, sessionStartArgs{
		Project:    "enchanted-inn",
		AgentLabel: "claude-code",
	})
	if err != nil {
		t.Fatalf("doSessionStart: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "Session ID:") {
		t.Errorf("response should expose the session id, got:\n%s", body)
	}
	if !strings.Contains(body, "claude-code") {
		t.Errorf("response should echo agent_label, got:\n%s", body)
	}
	if !strings.Contains(body, "Started:") {
		t.Errorf("response should report start time, got:\n%s", body)
	}
}

func TestDoSessionStart_DefaultProjectFallback(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	cfg := Config{DefaultProject: "fallback-project"}
	res, err := doSessionStart(ctx, st, cfg, sessionStartArgs{}) // no project
	if err != nil {
		t.Fatalf("doSessionStart: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got: %s", textContent(res))
	}
	if !strings.Contains(textContent(res), "fallback-project") {
		t.Errorf("expected DefaultProject fallback, got:\n%s", textContent(res))
	}
}

func TestDoSessionStart_RequiresProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doSessionStart(ctx, st, Config{}, sessionStartArgs{})
	if err != nil {
		t.Fatalf("doSessionStart: %v", err)
	}
	if !res.IsError {
		t.Fatalf("missing project must yield validation error")
	}
	if !strings.Contains(textContent(res), "project") {
		t.Errorf("error must mention 'project', got:\n%s", textContent(res))
	}
}

func TestDoSessionStart_AgentLabelOptional(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doSessionStart(ctx, st, Config{}, sessionStartArgs{
		Project: "enchanted-inn",
	})
	if err != nil {
		t.Fatalf("doSessionStart: %v", err)
	}
	if res.IsError {
		t.Fatalf("agent label must be optional, got: %s", textContent(res))
	}
}

func TestDoSessionStart_ReturnsValidUUIDv7(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doSessionStart(ctx, st, Config{}, sessionStartArgs{Project: "enchanted-inn"})
	if err != nil || res.IsError {
		t.Fatalf("start: err=%v body=%s", err, textContent(res))
	}

	body := textContent(res)
	// Find the line "Session ID: <uuid>"
	var sid string
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "Session ID: ") {
			sid = strings.TrimSpace(strings.TrimPrefix(line, "Session ID: "))
			break
		}
	}
	if sid == "" {
		t.Fatalf("could not extract session id from:\n%s", body)
	}
	parsed, err := uuid.Parse(sid)
	if err != nil {
		t.Fatalf("session id is not a valid UUID: %v", err)
	}
	if parsed.Version() != 7 {
		t.Errorf("session id must be UUIDv7, got version %d", parsed.Version())
	}
}
