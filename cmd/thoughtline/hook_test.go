package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func openTestDB(t *testing.T) (string, *storage.Storage) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hook_test.db")
	st, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return path, st
}

func countPendingRows(t *testing.T, dbPath string) int {
	t.Helper()
	st, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open for count: %v", err)
	}
	defer st.Close()
	rows, err := storage.DBQueryRowCount(st, "pending_events")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	return rows
}

// TestRunHook_OptInOff_NoWrite verifies that with THOUGHTLINE_PASSIVE_CAPTURE
// unset, no row is inserted.
func TestRunHook_OptInOff_NoWrite(t *testing.T) {
	dbPath, _ := openTestDB(t)
	t.Setenv("THOUGHTLINE_PASSIVE_CAPTURE", "")
	t.Setenv("THOUGHTLINE_DB", dbPath)

	payload := `{"hook_event_name":"PreToolUse","session_id":"sess-1","tool_use_id":"t1","tool_name":"Read"}`
	stdin := strings.NewReader(payload)
	var stderr bytes.Buffer

	err := runHook(context.Background(), []string{"pre-tool-use"}, stdin, &stderr)
	if err != nil {
		t.Fatalf("runHook: %v", err)
	}

	if n := countPendingRows(t, dbPath); n != 0 {
		t.Errorf("expected 0 rows, got %d", n)
	}
}

// TestRunHook_OptInOn_HappyPath verifies that with capture enabled and a valid
// payload, exactly one row is inserted.
func TestRunHook_OptInOn_HappyPath(t *testing.T) {
	dbPath, _ := openTestDB(t)
	t.Setenv("THOUGHTLINE_PASSIVE_CAPTURE", "1")
	t.Setenv("THOUGHTLINE_DB", dbPath)
	t.Setenv("THOUGHTLINE_PROJECT", "test-proj")

	payload := `{"hook_event_name":"PreToolUse","session_id":"sess-1","tool_use_id":"t1","tool_name":"Read"}`
	stdin := strings.NewReader(payload)
	var stderr bytes.Buffer

	err := runHook(context.Background(), []string{"pre-tool-use"}, stdin, &stderr)
	if err != nil {
		t.Fatalf("runHook: %v", err)
	}

	if n := countPendingRows(t, dbPath); n != 1 {
		t.Errorf("expected 1 row, got %d", n)
	}
}

// TestRunHook_MalformedJSON_ExitZeroNoRow verifies that malformed JSON produces
// no DB row and writes an error to stderr.
func TestRunHook_MalformedJSON_ExitZeroNoRow(t *testing.T) {
	dbPath, _ := openTestDB(t)
	t.Setenv("THOUGHTLINE_PASSIVE_CAPTURE", "1")
	t.Setenv("THOUGHTLINE_DB", dbPath)
	t.Setenv("THOUGHTLINE_PROJECT", "test-proj")

	stdin := strings.NewReader("{bad json")
	var stderr bytes.Buffer

	err := runHook(context.Background(), []string{"pre-tool-use"}, stdin, &stderr)
	if err != nil {
		// runHook must not return an error for user-facing failures.
		t.Fatalf("runHook should not error on malformed JSON, got: %v", err)
	}

	stderrStr := stderr.String()
	if stderrStr == "" {
		t.Error("expected error written to stderr for malformed JSON")
	}

	if n := countPendingRows(t, dbPath); n != 0 {
		t.Errorf("expected 0 rows for malformed JSON, got %d", n)
	}
}

// TestRunHook_UnknownEventName_ExitZeroNoRow verifies that an unknown event
// name produces no DB write and logs to stderr.
func TestRunHook_UnknownEventName_ExitZeroNoRow(t *testing.T) {
	dbPath, _ := openTestDB(t)
	t.Setenv("THOUGHTLINE_PASSIVE_CAPTURE", "1")
	t.Setenv("THOUGHTLINE_DB", dbPath)
	t.Setenv("THOUGHTLINE_PROJECT", "test-proj")

	payload := `{"hook_event_name":"UnknownEvent","session_id":"sess-1"}`
	stdin := strings.NewReader(payload)
	var stderr bytes.Buffer

	err := runHook(context.Background(), []string{"unknown-event"}, stdin, &stderr)
	if err != nil {
		t.Fatalf("runHook should not error for unknown event, got: %v", err)
	}

	if stderr.Len() == 0 {
		t.Error("expected error on stderr for unknown event name")
	}
	if n := countPendingRows(t, dbPath); n != 0 {
		t.Errorf("expected 0 rows for unknown event, got %d", n)
	}
}

// TestRunHook_OversizedPayload_ExitZeroNoRow verifies that a payload >1MiB is
// rejected gracefully.
func TestRunHook_OversizedPayload_ExitZeroNoRow(t *testing.T) {
	dbPath, _ := openTestDB(t)
	t.Setenv("THOUGHTLINE_PASSIVE_CAPTURE", "1")
	t.Setenv("THOUGHTLINE_DB", dbPath)
	t.Setenv("THOUGHTLINE_PROJECT", "test-proj")

	// Build a payload just over 1 MiB.
	oversize := make([]byte, 1<<20+1)
	for i := range oversize {
		oversize[i] = 'x'
	}
	stdin := bytes.NewReader(oversize)
	var stderr bytes.Buffer

	err := runHook(context.Background(), []string{"pre-tool-use"}, stdin, &stderr)
	if err != nil {
		t.Fatalf("runHook should not error on oversized payload, got: %v", err)
	}

	if n := countPendingRows(t, dbPath); n != 0 {
		t.Errorf("expected 0 rows for oversized payload, got %d", n)
	}
}

// TestRunHook_Dedup_OnlyOneRow verifies that two identical payloads result in
// only one DB row.
func TestRunHook_Dedup_OnlyOneRow(t *testing.T) {
	dbPath, _ := openTestDB(t)
	t.Setenv("THOUGHTLINE_PASSIVE_CAPTURE", "1")
	t.Setenv("THOUGHTLINE_DB", dbPath)
	t.Setenv("THOUGHTLINE_PROJECT", "test-proj")

	payload := `{"hook_event_name":"PostToolUse","session_id":"sess-1","tool_use_id":"t99","tool_name":"Write"}`
	var stderr bytes.Buffer

	for i := 0; i < 2; i++ {
		stdin := strings.NewReader(payload)
		if err := runHook(context.Background(), []string{"post-tool-use"}, stdin, &stderr); err != nil {
			t.Fatalf("runHook call %d: %v", i+1, err)
		}
	}

	if n := countPendingRows(t, dbPath); n != 1 {
		t.Errorf("expected 1 row after dedup, got %d", n)
	}
}

// Ensure the test helper compiles even if the helper is in the same package.
var _ = os.DevNull
