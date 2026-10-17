package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// withTestDB points the CLI at a throwaway database, never the user's real one.
func withTestDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "knowledge-test.db")
	t.Setenv("THOUGHTLINE_DB", path)
	t.Setenv("THOUGHTLINE_HOME", "")
	return path
}

func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errW bytes.Buffer
	code = runKnowledgeTo(args, &out, &errW)
	return code, out.String(), errW.String()
}

func TestRunKnowledge_UsageErrors(t *testing.T) {
	withTestDB(t)

	cases := []struct {
		name    string
		args    []string
		wantSub string
	}{
		{name: "no action", args: nil, wantSub: "Usage:"},
		{name: "unknown action", args: []string{"explode"}, wantSub: `unknown action "explode"`},
		{name: "sync without directory", args: []string{"sync"}, wantSub: "missing directory"},
		{name: "sync with two directories", args: []string{"sync", "a", "b"}, wantSub: "expected one directory"},
		{name: "sync with unknown flag", args: []string{"sync", "dir", "--nope"}, wantSub: "flag provided but not defined"},
		{name: "list with positional", args: []string{"list", "extra"}, wantSub: "unexpected argument"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := runCLI(t, tc.args...)
			if code != knowledgeExitUsage {
				t.Fatalf("exit = %d, want %d (stderr: %s)", code, knowledgeExitUsage, stderr)
			}
			if !strings.Contains(stderr, tc.wantSub) {
				t.Fatalf("stderr = %q, want it to contain %q", stderr, tc.wantSub)
			}
		})
	}
}

func TestRunKnowledge_SyncTwiceIsIdempotent(t *testing.T) {
	withTestDB(t)
	corpus := filepath.Join("..", "..", "internal", "knowledge", "testdata", "corpus")

	code, stdout, stderr := runCLI(t, "sync", corpus)
	if code != knowledgeExitOK {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	if got, want := strings.TrimSpace(stdout), "scanned=4 created=4 updated=0 unchanged=0 deleted=0 skipped=0"; got != want {
		t.Fatalf("first run printed %q, want %q", got, want)
	}

	// Flags after the directory must still be honoured.
	code, stdout, stderr = runCLI(t, "sync", corpus, "--dry-run")
	if code != knowledgeExitOK {
		t.Fatalf("dry run exit = %d (stderr: %s)", code, stderr)
	}
	if got, want := strings.TrimSpace(stdout), "scanned=4 created=0 updated=0 unchanged=4 deleted=0 skipped=0"; got != want {
		t.Fatalf("second run printed %q, want %q", got, want)
	}
}

func TestRunKnowledge_SyncJSONAndList(t *testing.T) {
	withTestDB(t)
	corpus := filepath.Join("..", "..", "internal", "knowledge", "testdata", "corpus")

	code, stdout, stderr := runCLI(t, "sync", corpus, "--json")
	if code != knowledgeExitOK {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	var report struct {
		Scanned int `json:"scanned"`
		Created int `json:"created"`
		Skipped int `json:"skipped"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode JSON report %q: %v", stdout, err)
	}
	if report.Scanned != 4 || report.Created != 4 || report.Skipped != 0 {
		t.Fatalf("report = %+v", report)
	}

	code, stdout, stderr = runCLI(t, "list")
	if code != knowledgeExitOK {
		t.Fatalf("list exit = %d (stderr: %s)", code, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 4 {
		t.Fatalf("list printed %d lines, want 4:\n%s", len(lines), stdout)
	}
	if !strings.HasPrefix(lines[0], "knowledge/animated-fill-bar\t") {
		t.Fatalf("first line = %q, want the topic_key<TAB>title shape ordered by key", lines[0])
	}
}

func TestRunKnowledge_SyncUnreadableDirectoryFails(t *testing.T) {
	withTestDB(t)

	code, _, stderr := runCLI(t, "sync", filepath.Join(t.TempDir(), "missing"))
	if code != knowledgeExitFatal {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, knowledgeExitFatal, stderr)
	}
	if !strings.Contains(stderr, "read") {
		t.Fatalf("stderr = %q, want the scan error", stderr)
	}
}

func TestRunKnowledge_ListOnEmptyDatabaseIsSilent(t *testing.T) {
	withTestDB(t)

	code, stdout, stderr := runCLI(t, "list")
	if code != knowledgeExitOK {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want nothing", stdout)
	}
}
