package server

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDoSessionSummary_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, err := st.StartSession(ctx, "enchanted-inn", "claude-code")
	if err != nil {
		t.Fatalf("seed start: %v", err)
	}

	res, err := doSessionSummary(ctx, st, sessionSummaryArgs{
		ID:      sess.ID,
		Summary: "## Goal\nLantern bake.\n## Accomplished\n- baked lanterns",
	})
	if err != nil {
		t.Fatalf("doSessionSummary: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "Session closed") && !strings.Contains(body, "Closed") {
		t.Errorf("response should confirm closure, got:\n%s", body)
	}
	if !strings.Contains(body, "Duration:") {
		t.Errorf("response should include Duration, got:\n%s", body)
	}
}

func TestDoSessionSummary_RequiresID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doSessionSummary(ctx, st, sessionSummaryArgs{ID: "", Summary: "x"})
	if err != nil {
		t.Fatalf("doSessionSummary: %v", err)
	}
	if !res.IsError {
		t.Fatalf("missing id must yield validation error")
	}
}

func TestDoSessionSummary_RequiresSummary(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, _ := st.StartSession(ctx, "enchanted-inn", "")
	res, err := doSessionSummary(ctx, st, sessionSummaryArgs{ID: sess.ID, Summary: "  "})
	if err != nil {
		t.Fatalf("doSessionSummary: %v", err)
	}
	if !res.IsError {
		t.Fatalf("blank summary must yield validation error")
	}
	if !strings.Contains(textContent(res), "summary") {
		t.Errorf("error must mention 'summary', got:\n%s", textContent(res))
	}
}

func TestDoSessionSummary_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	bogus := uuid.Must(uuid.NewV7()).String()
	res, err := doSessionSummary(ctx, st, sessionSummaryArgs{
		ID:      bogus,
		Summary: "irrelevant",
	})
	if err != nil {
		t.Fatalf("doSessionSummary: %v", err)
	}
	if !res.IsError {
		t.Fatalf("unknown id must yield error")
	}
	if !strings.Contains(strings.ToLower(textContent(res)), "not found") {
		t.Errorf("error must say 'not found', got:\n%s", textContent(res))
	}
}

func TestDoSessionSummary_AlreadyClosedRejected(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, _ := st.StartSession(ctx, "enchanted-inn", "")
	if _, err := st.EndSession(ctx, sess.ID, "first close"); err != nil {
		t.Fatalf("first close: %v", err)
	}

	res, err := doSessionSummary(ctx, st, sessionSummaryArgs{
		ID:      sess.ID,
		Summary: "second close",
	})
	if err != nil {
		t.Fatalf("doSessionSummary: %v", err)
	}
	if !res.IsError {
		t.Fatalf("repeat close must yield error")
	}
	if !strings.Contains(strings.ToLower(textContent(res)), "already") {
		t.Errorf("error must explain 'already ended', got:\n%s", textContent(res))
	}
}
