// Package pending owns the domain types and helpers for the passive-capture
// queue. Events land in pending_events and stay there until the model promotes
// them to memories via tl_promote, or the worker archives old ones.
package pending

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// ComputeHash returns a stable SHA-256 fingerprint for an event tuple.
//
// Composition:
//
//	sha256( eventType  "\n"
//	        sessionID  "\n"
//	        toolUseID  "\n"
//	        capturedAt-floored-to-second  "\n"
//	        canonicalPayload )
//
// capturedAt is truncated to the second (nanoseconds stripped) so multiple
// observations within the same second from the same tool call hash identically
// — useful for SessionStart/Stop where no tool_use_id distinguishes events.
//
// canonicalPayload is the JSON payload re-encoded with sorted keys and no
// extra whitespace. If the payload is not valid JSON the raw bytes are used
// verbatim (capture still succeeds; dedup is best-effort for malformed input).
func ComputeHash(eventType, sessionID, toolUseID string, capturedAt time.Time, payload []byte) string {
	flooredSec := strconv.FormatInt(capturedAt.Truncate(time.Second).Unix(), 10)
	canonical := canonicalJSON(payload)

	h := sha256.New()
	for _, part := range []string{eventType, sessionID, toolUseID, flooredSec} {
		h.Write([]byte(part))
		h.Write([]byte{'\n'})
	}
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil))
}

// canonicalJSON returns the payload JSON-re-encoded with sorted keys and no
// whitespace. If payload is not valid JSON, it is returned as-is.
func canonicalJSON(payload []byte) []byte {
	var v any
	if err := json.Unmarshal(payload, &v); err != nil {
		// Not valid JSON — use the raw bytes; dedup is best-effort.
		return payload
	}
	out, err := json.Marshal(sortedJSON(v))
	if err != nil {
		return payload
	}
	return out
}

// sortedJSON recursively converts a parsed JSON value so that map keys are
// sorted (by relying on json.Marshal's stable struct encoding — but maps in
// Go serialize in key-sorted order since Go 1.12, so this is already sorted
// after the round-trip through json.Unmarshal + json.Marshal). We just need
// to do the round-trip correctly for nested maps.
func sortedJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = sortedJSON(vv)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			out[i] = sortedJSON(vv)
		}
		return out
	default:
		return val
	}
}

// hashInputDebug is used only in tests that want to inspect the hash input.
// It's unexported so the only way to test it is through ComputeHash.
func hashInputDebug(eventType, sessionID, toolUseID string, capturedAt time.Time, payload []byte) string {
	flooredSec := strconv.FormatInt(capturedAt.Truncate(time.Second).Unix(), 10)
	canonical := canonicalJSON(payload)
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		eventType, sessionID, toolUseID, flooredSec, string(canonical))
}
