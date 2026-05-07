package hooks_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// hooksSchema represents the top-level structure of hooks.json.
type hooksSchema struct {
	Description string                      `json:"description"`
	Hooks       map[string][]hookEventGroup `json:"hooks"`
}

type hookEventGroup struct {
	Matcher string     `json:"matcher,omitempty"`
	Hooks   []hookDef  `json:"hooks"`
}

type hookDef struct {
	Type          string `json:"type"`
	Command       string `json:"command"`
	Timeout       int    `json:"timeout,omitempty"`
	OnFailure     string `json:"on_failure,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

func loadHooksJSON(t *testing.T) hooksSchema {
	t.Helper()
	// Find hooks.json relative to this test file.
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)
	path := filepath.Join(dir, "hooks.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read hooks.json: %v", err)
	}
	var h hooksSchema
	if err := json.Unmarshal(data, &h); err != nil {
		t.Fatalf("parse hooks.json: %v", err)
	}
	return h
}

// TestHooksJSON_ContainsSixPassiveCaptureEntries verifies that all 6 passive-
// capture events are registered in hooks.json.
func TestHooksJSON_ContainsSixPassiveCaptureEntries(t *testing.T) {
	h := loadHooksJSON(t)

	// The 6 events that must have a "thoughtline hook <name>" entry.
	required := map[string]string{
		"SessionStart":      "thoughtline hook session-start",
		"UserPromptSubmit":  "thoughtline hook user-prompt-submit",
		"PreToolUse":        "thoughtline hook pre-tool-use",
		"PostToolUse":       "thoughtline hook post-tool-use",
		"Stop":              "thoughtline hook stop",
		"SessionEnd":        "thoughtline hook session-end",
	}

	for eventName, wantCmd := range required {
		groups, ok := h.Hooks[eventName]
		if !ok {
			t.Errorf("hooks.json missing event %q", eventName)
			continue
		}
		found := false
		for _, group := range groups {
			for _, def := range group.Hooks {
				if def.Command == wantCmd {
					found = true
					// Verify cross-platform: command must be bare binary name (no bash/sh).
					if strings.HasPrefix(def.Command, "bash ") || strings.HasPrefix(def.Command, "/bin/") {
						t.Errorf("event %q: command %q uses shell path — must be bare binary for cross-platform use", eventName, def.Command)
					}
					// Verify on_failure=continue so Claude Code session survives.
					if def.OnFailure != "" && def.OnFailure != "continue" {
						t.Errorf("event %q: on_failure=%q — should be 'continue' to protect the session", eventName, def.OnFailure)
					}
				}
			}
		}
		if !found {
			t.Errorf("event %q: no hook entry with command %q found", eventName, wantCmd)
		}
	}
}

// TestHooksJSON_AllHooksHaveTimeout verifies every passive-capture hook has a
// timeout set (prevents Claude Code from hanging on a slow hook).
func TestHooksJSON_AllHooksHaveTimeout(t *testing.T) {
	h := loadHooksJSON(t)
	passiveCaptureEvents := map[string]bool{
		"UserPromptSubmit": true,
		"PreToolUse":       true,
		"PostToolUse":      true,
		"Stop":             true,
		"SessionEnd":       true,
	}

	for eventName := range passiveCaptureEvents {
		groups, ok := h.Hooks[eventName]
		if !ok {
			continue
		}
		for _, group := range groups {
			for _, def := range group.Hooks {
				if strings.Contains(def.Command, "thoughtline hook") && def.Timeout == 0 {
					t.Errorf("event %q hook %q: missing timeout", eventName, def.Command)
				}
			}
		}
	}
}
