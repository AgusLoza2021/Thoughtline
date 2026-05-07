package pending_test

import (
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
)

func TestComputeHash_Deterministic(t *testing.T) {
	ts := time.Date(2026, 5, 7, 12, 30, 45, 0, time.UTC)
	payload := []byte(`{"session_id":"abc","tool_name":"Read"}`)

	h1 := pending.ComputeHash("PreToolUse", "sess-1", "tool-1", ts, payload)
	h2 := pending.ComputeHash("PreToolUse", "sess-1", "tool-1", ts, payload)

	if h1 != h2 {
		t.Errorf("same inputs produced different hashes: %q vs %q", h1, h2)
	}
	if h1 == "" {
		t.Error("hash must not be empty")
	}
	// SHA-256 hex is always 64 chars
	if len(h1) != 64 {
		t.Errorf("expected 64-char hex hash, got len=%d: %q", len(h1), h1)
	}
}

func TestComputeHash_DifferentInputsDifferentHashes(t *testing.T) {
	ts := time.Date(2026, 5, 7, 12, 30, 45, 0, time.UTC)
	payload1 := []byte(`{"tool_name":"Read","tool_input":{"file":"/a"}}`)
	payload2 := []byte(`{"tool_name":"Read","tool_input":{"file":"/b"}}`)

	h1 := pending.ComputeHash("PreToolUse", "sess-1", "tool-1", ts, payload1)
	h2 := pending.ComputeHash("PreToolUse", "sess-1", "tool-1", ts, payload2)

	if h1 == h2 {
		t.Error("different payloads should produce different hashes")
	}
}

func TestComputeHash_DifferentEventTypesDifferentHashes(t *testing.T) {
	ts := time.Date(2026, 5, 7, 12, 30, 45, 0, time.UTC)
	payload := []byte(`{}`)

	h1 := pending.ComputeHash("SessionStart", "sess-1", "", ts, payload)
	h2 := pending.ComputeHash("SessionEnd", "sess-1", "", ts, payload)

	if h1 == h2 {
		t.Error("different event types should produce different hashes")
	}
}

func TestComputeHash_SubsecondTimestampsFoldToSameHash(t *testing.T) {
	// Two timestamps in the same second should yield the same hash (captured_at
	// is floored to the second).
	base := time.Date(2026, 5, 7, 12, 30, 45, 0, time.UTC)
	withNanos := time.Date(2026, 5, 7, 12, 30, 45, 999999999, time.UTC)
	payload := []byte(`{"session_id":"abc"}`)

	h1 := pending.ComputeHash("Stop", "sess-1", "", base, payload)
	h2 := pending.ComputeHash("Stop", "sess-1", "", withNanos, payload)

	if h1 != h2 {
		t.Errorf("hashes should be equal when floored to same second: %q vs %q", h1, h2)
	}
}

func TestComputeHash_PayloadWhitespaceNormalized(t *testing.T) {
	// Semantically identical JSON with different whitespace should produce the
	// same hash (keys are sorted, whitespace is stripped in canonical form).
	ts := time.Date(2026, 5, 7, 12, 30, 45, 0, time.UTC)
	compact := []byte(`{"a":1,"b":2}`)
	spaced := []byte(`{ "b" : 2 , "a" : 1 }`)

	h1 := pending.ComputeHash("Stop", "sess-1", "", ts, compact)
	h2 := pending.ComputeHash("Stop", "sess-1", "", ts, spaced)

	if h1 != h2 {
		t.Errorf("semantically identical JSON should hash the same: %q vs %q", h1, h2)
	}
}
