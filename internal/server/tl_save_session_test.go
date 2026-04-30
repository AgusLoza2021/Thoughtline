package server

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestDoSave_AttachesSessionID — regression for the M4 wiring: tl_save must
// accept an optional session_id arg, persist it on memories.session_id, and
// echo it back in the response so the AI can confirm the linkage.
func TestDoSave_AttachesSessionID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, err := st.StartSession(ctx, "enchanted-inn", "claude-code")
	if err != nil {
		t.Fatalf("seed start: %v", err)
	}

	args := validArgs()
	args.SessionID = sess.ID

	res, err := doSave(ctx, st, Config{}, args)
	if err != nil {
		t.Fatalf("doSave: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "Session: "+sess.ID) {
		t.Errorf("response should echo session id, got:\n%s", body)
	}
}

// TestDoSave_RejectsCrossProjectSession — server layer should surface the
// storage-level cross-project mismatch as a clean error result.
func TestDoSave_RejectsCrossProjectSession(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, err := st.StartSession(ctx, "alpha", "")
	if err != nil {
		t.Fatalf("seed start: %v", err)
	}

	args := validArgs()
	args.Project = "beta"
	args.SessionID = sess.ID

	res, err := doSave(ctx, st, Config{}, args)
	if err != nil {
		t.Fatalf("doSave: %v", err)
	}
	if !res.IsError {
		t.Fatalf("cross-project session_id must yield error, got success: %s", textContent(res))
	}
	body := strings.ToLower(textContent(res))
	if !strings.Contains(body, "project") || !strings.Contains(body, "session") {
		t.Errorf("error must mention project/session mismatch, got:\n%s", textContent(res))
	}
}

// TestDoSave_RejectsUnknownSessionID — referencing a non-existent session
// must fail cleanly.
func TestDoSave_RejectsUnknownSessionID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	args := validArgs()
	args.SessionID = uuid.Must(uuid.NewV7()).String()

	res, err := doSave(ctx, st, Config{}, args)
	if err != nil {
		t.Fatalf("doSave: %v", err)
	}
	if !res.IsError {
		t.Fatalf("unknown session_id must yield error, got success: %s", textContent(res))
	}
	if !strings.Contains(strings.ToLower(textContent(res)), "session") {
		t.Errorf("error must mention session, got:\n%s", textContent(res))
	}
}

// TestDoSave_RejectsMalformedSessionID — domain validation rejects non-UUIDv7.
func TestDoSave_RejectsMalformedSessionID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	args := validArgs()
	args.SessionID = "not-a-uuid"

	res, err := doSave(ctx, st, Config{}, args)
	if err != nil {
		t.Fatalf("doSave: %v", err)
	}
	if !res.IsError {
		t.Fatalf("malformed session_id must yield validation error")
	}
}
