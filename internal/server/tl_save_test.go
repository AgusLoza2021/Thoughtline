package server

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func newTestStorage(t *testing.T) *storage.Storage {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func validArgs() saveArgs {
	return saveArgs{
		Title:   "Lock player loot UI to a 4x6 grid",
		Content: "**What**: chose grid\n**Why**: cognitive load on mobile\n**Where**: ui/loot/grid.js\n",
		Type:    string(memory.TypeGameDesignDecision),
		Project: "enchanted-inn",
	}
}

func TestDoSave_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	res, err := doSave(ctx, st, Config{Version: "test"}, validArgs())
	if err != nil {
		t.Fatalf("doSave returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected success result, got error: %s", textContent(res))
	}
	body := textContent(res)
	if !strings.Contains(body, "action=created") {
		t.Errorf("response should mention action=created, got:\n%s", body)
	}
	if !strings.Contains(body, "Sync ID:") {
		t.Errorf("response should include Sync ID, got:\n%s", body)
	}
	if !strings.Contains(body, "Project: enchanted-inn") {
		t.Errorf("response should echo project, got:\n%s", body)
	}
}

func TestDoSave_ScopeAutoDefaults(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	t.Run("non-preference defaults to project scope", func(t *testing.T) {
		args := validArgs()
		args.Scope = "" // unset
		res, err := doSave(ctx, st, Config{}, args)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected error: %s", textContent(res))
		}
		if !strings.Contains(textContent(res), "Scope: project") {
			t.Errorf("expected default scope=project, got:\n%s", textContent(res))
		}
	})

	t.Run("preference auto-defaults to personal scope", func(t *testing.T) {
		args := validArgs()
		args.Type = string(memory.TypePreference)
		args.Scope = ""
		args.TopicKey = "preference/keybindings"
		res, err := doSave(ctx, st, Config{}, args)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected error: %s", textContent(res))
		}
		if !strings.Contains(textContent(res), "Scope: personal") {
			t.Errorf("preference should default scope=personal, got:\n%s", textContent(res))
		}
	})
}

func TestDoSave_DefaultProjectFromConfig(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	args := validArgs()
	args.Project = "" // unset

	cfg := Config{DefaultProject: "fallback-project"}
	res, err := doSave(ctx, st, cfg, args)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", textContent(res))
	}
	if !strings.Contains(textContent(res), "Project: fallback-project") {
		t.Errorf("expected DefaultProject fallback, got:\n%s", textContent(res))
	}
}

func TestDoSave_ValidationErrors(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		mutate    func(*saveArgs)
		wantSubstr string
	}{
		{"missing title", func(a *saveArgs) { a.Title = "" }, "'title' is required"},
		{"missing content", func(a *saveArgs) { a.Content = "" }, "'content' is required"},
		{"bogus type", func(a *saveArgs) { a.Type = "made-up" }, "invalid 'type'"},
		{"bogus scope", func(a *saveArgs) { a.Scope = "team" }, "invalid 'scope'"},
		{"preference with project scope", func(a *saveArgs) {
			a.Type = string(memory.TypePreference)
			a.Scope = string(memory.ScopeProject)
		}, "preference"},
		{"missing project (no fallback)", func(a *saveArgs) { a.Project = "" }, "'project' is required"},
		{"invalid topic_key", func(a *saveArgs) { a.TopicKey = "Inv@lid" }, "topic_key"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := validArgs()
			tc.mutate(&args)
			res, err := doSave(ctx, st, Config{}, args)
			if err != nil {
				t.Fatalf("doSave returned err: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected validation error result; body:\n%s", textContent(res))
			}
			if !strings.Contains(textContent(res), tc.wantSubstr) {
				t.Errorf("error message should contain %q, got:\n%s", tc.wantSubstr, textContent(res))
			}
		})
	}
}

func TestDoSave_TopicKeyUpsertEndToEnd(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{}

	args := validArgs()
	args.TopicKey = "design/inventory/grid-vs-list"

	// First save → created.
	res1, err := doSave(ctx, st, cfg, args)
	if err != nil || res1.IsError {
		t.Fatalf("first save failed: err=%v body=%s", err, textContent(res1))
	}
	if !strings.Contains(textContent(res1), "action=created") {
		t.Errorf("first save should be created, got:\n%s", textContent(res1))
	}
	if !strings.Contains(textContent(res1), "Revision: 0") {
		t.Errorf("first save should be revision 0, got:\n%s", textContent(res1))
	}

	// Identical re-save → noop.
	res2, err := doSave(ctx, st, cfg, args)
	if err != nil || res2.IsError {
		t.Fatalf("second save failed: err=%v body=%s", err, textContent(res2))
	}
	if !strings.Contains(textContent(res2), "action=noop") {
		t.Errorf("identical re-save should be noop, got:\n%s", textContent(res2))
	}

	// Changed content → updated, revision bumped.
	args.Content = args.Content + "\n2026-04-29: revisited after playtests."
	res3, err := doSave(ctx, st, cfg, args)
	if err != nil || res3.IsError {
		t.Fatalf("third save failed: err=%v body=%s", err, textContent(res3))
	}
	if !strings.Contains(textContent(res3), "action=updated") {
		t.Errorf("changed re-save should be updated, got:\n%s", textContent(res3))
	}
	if !strings.Contains(textContent(res3), "Revision: 1") {
		t.Errorf("revision should bump to 1, got:\n%s", textContent(res3))
	}
}

func TestNew_RegistersTLSaveTool(t *testing.T) {
	st := newTestStorage(t)
	srv := New(st, Config{Version: "test"})
	if srv == nil {
		t.Fatalf("New returned nil server")
	}
	// Smoke: can't easily introspect tools without parsing internals, but
	// a panic-free build here is the main contract for now. The Save handler
	// itself is exercised by the doSave-level tests above.
}

// textContent extracts the concatenated text from a CallToolResult.
// mcp-go represents both success and error responses via the same
// `Content []mcp.Content` slice; we walk it and join any TextContent items.
func textContent(res *mcp.CallToolResult) string {
	if res == nil {
		return ""
	}
	var parts []string
	for _, c := range res.Content {
		if t, ok := mcp.AsTextContent(c); ok {
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, "")
}
