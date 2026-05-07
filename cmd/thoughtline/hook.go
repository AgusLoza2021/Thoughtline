package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// hookEventNameMap maps the CLI kebab-case argument names that thoughtline
// hook accepts to the PascalCase event type stored in pending_events.
var hookEventNameMap = map[string]string{
	"session-start":       "SessionStart",
	"user-prompt-submit":  "UserPromptSubmit",
	"pre-tool-use":        "PreToolUse",
	"post-tool-use":       "PostToolUse",
	"stop":                "Stop",
	"session-end":         "SessionEnd",
}

// maxHookPayloadBytes is the stdin read cap — 1 MiB, well above any realistic
// Claude Code payload. Payloads larger than this are rejected with an error to
// stderr (exit 0).
const maxHookPayloadBytes = 1 << 20 // 1 MiB

// runHook implements the `thoughtline hook <event-name>` subcommand.
//
// Contract: this function NEVER returns a non-nil error for user-facing error
// conditions (malformed JSON, opt-in OFF, DB errors, oversized payload, unknown
// event name). Errors are written to `errW` and the function returns nil so
// the caller can exit 0.
//
// The function returns a non-nil error only for programmer errors (invalid args
// slice), which map to exit 2 in the caller.
func runHook(ctx context.Context, args []string, stdin io.Reader, errW io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("hook: missing event-name argument")
	}
	eventArg := strings.ToLower(args[0])

	// Fast-path: check opt-in before any DB interaction.
	if os.Getenv("THOUGHTLINE_PASSIVE_CAPTURE") != "1" {
		return nil
	}

	// Validate event name.
	eventType, ok := hookEventNameMap[eventArg]
	if !ok {
		fmt.Fprintf(errW, "thoughtline hook: unknown event %q — skipping\n", eventArg)
		return nil
	}

	// Read stdin with a size cap.
	limited := io.LimitReader(stdin, maxHookPayloadBytes+1) // +1 to detect oversize
	raw, err := io.ReadAll(limited)
	if err != nil {
		fmt.Fprintf(errW, "thoughtline hook: read stdin: %v\n", err)
		return nil
	}
	if len(raw) > maxHookPayloadBytes {
		fmt.Fprintf(errW, "thoughtline hook: payload exceeds 1 MiB limit — discarding\n")
		return nil
	}

	// Validate JSON.
	var jsonMap map[string]any
	if err := json.Unmarshal(raw, &jsonMap); err != nil {
		fmt.Fprintf(errW, "thoughtline hook: invalid JSON payload: %v\n", err)
		return nil
	}

	// Extract fields from the payload.
	sessionID := stringFromMap(jsonMap, "session_id")
	toolName := stringFromMap(jsonMap, "tool_name")
	toolUseID := stringFromMap(jsonMap, "tool_use_id")

	now := time.Now()
	capturedAt := extractTimestamp(jsonMap, now)

	// Compute deterministic event hash.
	hash := pending.ComputeHash(eventType, sessionID, toolUseID, capturedAt, raw)

	// Resolve project.
	project := os.Getenv("THOUGHTLINE_PROJECT")
	if project == "" {
		if cwd, err := os.Getwd(); err == nil {
			project = filpathBase(cwd)
		}
	}
	if project == "" {
		project = "default"
	}

	syncID, err := newSyncIDFromStorage()
	if err != nil {
		fmt.Fprintf(errW, "thoughtline hook: generate sync_id: %v\n", err)
		return nil
	}

	ev := pending.Event{
		SyncID:     syncID,
		Project:    project,
		SessionID:  sessionID,
		EventType:  eventType,
		ToolName:   toolName,
		ToolUseID:  toolUseID,
		Payload:    string(raw),
		Hash:       hash,
		Status:     pending.StatusPending,
		CreatedAt:  now,
		CapturedAt: capturedAt,
	}

	// Open DB and insert.
	dbPath, err := resolveDBPath()
	if err != nil {
		fmt.Fprintf(errW, "thoughtline hook: resolve db path: %v\n", err)
		return nil
	}

	st, err := storage.Open(ctx, dbPath)
	if err != nil {
		fmt.Fprintf(errW, "thoughtline hook: open db: %v\n", err)
		return nil
	}
	defer func() { _ = st.Close() }()

	if _, err := st.InsertPending(ctx, ev); err != nil {
		fmt.Fprintf(errW, "thoughtline hook: insert: %v\n", err)
		return nil
	}

	return nil
}

// stringFromMap extracts a string value from a generic JSON map. Returns ""
// when the key is absent or the value is not a string.
func stringFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// extractTimestamp tries to parse a "timestamp" field from the payload. If
// absent or unparseable, falls back to the provided default.
func extractTimestamp(m map[string]any, fallback time.Time) time.Time {
	ts, _ := m["timestamp"].(string)
	if ts == "" {
		return fallback
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, ts); err == nil {
			return t
		}
	}
	return fallback
}

// filpathBase returns the last element of a path. Mirrors filepath.Base but
// avoids importing "path/filepath" in a file that already imports it via main.
// Actually we do import it via main.go so let's just use a simple approach.
func filpathBase(path string) string {
	// Walk backwards to find the last separator.
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

// newSyncIDFromStorage delegates UUID generation to the storage package helper.
// We need a UUIDv7 and storage already has newSyncID(), but it's unexported.
// We use a small shim here.
func newSyncIDFromStorage() (string, error) {
	// Use the storage package's exported path: open a temp storage and
	// have it generate an ID. Too heavy — instead just use uuid directly.
	// Since we already depend on github.com/google/uuid in go.mod:
	return newHookSyncID()
}
