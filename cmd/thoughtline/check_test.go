package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/knowledge"
)

// checkCorpus is the recipe fixture every CLI test syncs into its temp DB.
var checkCorpus = filepath.Join("..", "..", "internal", "knowledge", "testdata", "corpus")

// withCheckDB points the CLI at a throwaway database holding the fixture
// corpus, and keeps the hook state out of the real temp dir.
func withCheckDB(t *testing.T) {
	t.Helper()
	withTestDB(t)
	t.Setenv("THOUGHTLINE_KNOWLEDGE_PROJECT", "")
	t.Setenv("THOUGHTLINE_KNOWLEDGE_EXCLUDE", "")
	t.Setenv("THOUGHTLINE_PROJECT", "ninjasurvivor")

	stateDir := filepath.Join(t.TempDir(), "hook-state")
	old := hookStateDir
	hookStateDir = func() string { return stateDir }
	t.Cleanup(func() { hookStateDir = old })

	if code, _, stderr := runCLI(t, "sync", checkCorpus); code != knowledgeExitOK {
		t.Fatalf("seed sync failed (exit %d): %s", code, stderr)
	}
}

func runCheckCLI(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errW bytes.Buffer
	code = runCheck(t.Context(), args, strings.NewReader(stdin), &out, &errW)
	return code, out.String(), errW.String()
}

func hookPayload(session, tool, file string) string {
	raw, _ := json.Marshal(map[string]any{
		"session_id": session,
		"tool_name":  tool,
		"tool_input": map[string]any{"file_path": file},
	})
	return string(raw)
}

type decodedHookOutput struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

func TestRunCheck_HookEmitsContextOncePerFilePerSession(t *testing.T) {
	withCheckDB(t)
	const file = "Assets/Scripts/AnimatedFillBar.cs"
	payload := hookPayload("session-1", "Edit", file)

	code, stdout, stderr := runCheckCLI(t, payload, "--hook")
	if code != checkExitOK {
		t.Fatalf("hook exit = %d, stderr = %s", code, stderr)
	}
	if stdout == "" {
		t.Fatal("expected hook output for a .cs file with matching knowledge")
	}

	var out decodedHookOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("hook output is not valid JSON: %v\n%s", err, stdout)
	}
	if out.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("hookEventName = %q", out.HookSpecificOutput.HookEventName)
	}
	ctx := out.HookSpecificOutput.AdditionalContext
	if !strings.HasPrefix(ctx, "Thoughtline knowledge check for "+file+":") {
		t.Errorf("context should name the file, got:\n%s", ctx)
	}
	if !strings.Contains(ctx, "knowledge/animated-fill-bar") {
		t.Errorf("context should carry the topic key, got:\n%s", ctx)
	}
	// Only the Validate block travels: the hook has no room for the rest.
	if !strings.Contains(ctx, "Validate:") || strings.Contains(ctx, "Applies when") {
		t.Errorf("context should carry only the Validate block, got:\n%s", ctx)
	}
	if strings.Contains(stdout, `"decision"`) {
		t.Errorf("hook must never decide, got:\n%s", stdout)
	}
	if len([]rune(ctx)) > hookMaxChars {
		t.Errorf("additionalContext is %d runes, cap is %d", len([]rune(ctx)), hookMaxChars)
	}

	// Same session, same file: silent.
	code, stdout, _ = runCheckCLI(t, payload, "--hook")
	if code != checkExitOK || stdout != "" {
		t.Fatalf("second hook call printed %q (exit %d), want silence", stdout, code)
	}

	// Same file in a new session: allowed again.
	code, stdout, _ = runCheckCLI(t, hookPayload("session-2", "Write", file), "--hook")
	if code != checkExitOK || stdout == "" {
		t.Fatalf("new session should get its own hint, got %q", stdout)
	}
}

func TestRunCheck_HookStaysSilentOnEverythingElse(t *testing.T) {
	withCheckDB(t)

	cases := []struct {
		name    string
		stdin   string
		args    []string
		wantOut bool
	}{
		{name: "non-C# path", stdin: hookPayload("s", "Edit", "Assets/README.md")},
		{name: "custom extension", stdin: hookPayload("s", "Edit", "Assets/AnimatedFillBar.shader"), args: []string{"--ext", ".shader"}, wantOut: true},
		{name: "unknown tool", stdin: hookPayload("s", "Bash", "Assets/Scripts/AnimatedFillBar.cs")},
		{name: "garbage stdin", stdin: "not json at all"},
		{name: "empty stdin", stdin: ""},
		{name: "json without file", stdin: `{"session_id":"s","tool_name":"Edit"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--hook"}, tc.args...)
			code, stdout, stderr := runCheckCLI(t, tc.stdin, args...)
			if code != checkExitOK {
				t.Fatalf("hook must always exit 0, got %d", code)
			}
			if tc.wantOut != (stdout != "") {
				t.Fatalf("stdout = %q, wantOut = %v (stderr %q)", stdout, tc.wantOut, stderr)
			}
			if stderr != "" {
				t.Errorf("hook must not write to stderr, got %q", stderr)
			}
		})
	}
}

func TestRunCheck_HookWithoutDatabaseIsSilent(t *testing.T) {
	// No sync: the DB file does not exist yet. A hook fired on a machine with
	// no memories must not create one and must not print anything.
	withTestDB(t)
	t.Setenv("THOUGHTLINE_PROJECT", "ninjasurvivor")

	code, stdout, _ := runCheckCLI(t,
		hookPayload("s", "Edit", "Assets/Scripts/AnimatedFillBar.cs"), "--hook")
	if code != checkExitOK || stdout != "" {
		t.Fatalf("hook without db printed %q (exit %d)", stdout, code)
	}
}

func TestRunCheck_PlainAndJSONOutput(t *testing.T) {
	withCheckDB(t)

	code, stdout, stderr := runCheckCLI(t, "", "--task", "add camera shake when the player gets hit", "--limit", "3")
	if code != checkExitOK {
		t.Fatalf("exit = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(stdout, `query: camera shake player hit`) {
		t.Errorf("plain output should echo the query, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "knowledge/camera-shake") {
		t.Errorf("plain output should name the matching entry, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "score=") || !strings.Contains(stdout, "excluded 0 foreign-stack entries") {
		t.Errorf("plain output should carry score and excluded lines, got:\n%s", stdout)
	}

	code, stdout, stderr = runCheckCLI(t, "", "--task", "add camera shake when the player gets hit", "--json")
	if code != checkExitOK {
		t.Fatalf("json exit = %d, stderr = %s", code, stderr)
	}
	var res knowledge.CheckResult
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("--json output is not a CheckResult: %v\n%s", err, stdout)
	}
	if len(res.Hits) == 0 || res.Hits[0].TopicKey != "knowledge/camera-shake" {
		t.Fatalf("json hits = %+v", res.Hits)
	}
	if res.Hits[0].Validate == "" {
		t.Error("json hit should carry the Validate section")
	}
}

func TestRunCheck_UsageErrors(t *testing.T) {
	withCheckDB(t)

	for _, args := range [][]string{
		{},                       // neither task nor path
		{"--task", "x", "extra"}, // positional argument
		{"--nope"},               // unknown flag
	} {
		code, stdout, stderr := runCheckCLI(t, "", args...)
		if code != checkExitUsage {
			t.Errorf("args %v: exit = %d, want %d", args, code, checkExitUsage)
		}
		if stdout != "" {
			t.Errorf("args %v: usage errors must not print to stdout, got %q", args, stdout)
		}
		if !strings.Contains(stderr, "Usage:") && !strings.Contains(stderr, "required") {
			t.Errorf("args %v: stderr should explain the problem, got %q", args, stderr)
		}
	}
}

func TestCheckHookHelpers(t *testing.T) {
	if !hasExtension("A:\\Repo\\X.cs", ".cs") || !hasExtension("X.CS", "cs") {
		t.Error("extension check should be case-insensitive and accept a bare extension")
	}
	if hasExtension("X.cs", ".shader") {
		t.Error("extension check should reject a different extension")
	}
	if got := clampRunes(strings.Repeat("a", 20), 5); len([]rune(got)) != 5 || !strings.HasSuffix(got, "…") {
		t.Errorf("clampRunes = %q", got)
	}
	if got := clampRunes("short", 50); got != "short" {
		t.Errorf("clampRunes should not touch short strings, got %q", got)
	}
	if _, ok := parsePreToolUse([]byte(`{"tool_name":"Edit"}`)); ok {
		t.Error("payload without file_path must be rejected")
	}
	if p, ok := parsePreToolUse([]byte(hookPayload("s3", "MultiEdit", "A/B.cs"))); !ok || p.ToolName != "MultiEdit" || p.FilePath != "A/B.cs" {
		t.Errorf("parsePreToolUse = %+v ok=%v", p, ok)
	}
	if got := sanitizeSessionID("../../etc/passwd"); strings.ContainsAny(got, "/\\") {
		t.Errorf("session id was not sanitized: %q", got)
	}
}
