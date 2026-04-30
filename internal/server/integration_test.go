package server

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// This file holds an end-to-end "smoke" scenario that exercises every tl_*
// MCP tool the server registers, in the realistic order an AI agent would
// call them across a session:
//
//   tl_save → tl_search → tl_get_observation → tl_context →
//   tl_update → tl_search (post-update) → tl_delete → tl_context (post-delete)
//
// Each step builds an actual mcp.CallToolRequest with a JSON-shaped arguments
// map, runs it through the same decode + handler pair the registered tool
// would, and asserts on the formatted text response. This covers the full
// in-process pipeline (decoding, validation, storage, formatting) — the only
// thing it does NOT cover is the stdio JSON-RPC transport, which is owned
// entirely by the mcp-go library.
//
// We also call New(storage, cfg) to confirm the wiring path doesn't panic.

// buildReq constructs the kind of CallToolRequest mcp-go would hand a tool
// handler when an MCP client calls the tool with a JSON arguments object.
func buildReq(toolName string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}
}

// extractID parses the "ID: 123" line from a tool response. The format is
// shared across tl_save, tl_search, tl_context, and tl_update outputs.
func extractID(t *testing.T, body string) int64 {
	t.Helper()
	re := regexp.MustCompile(`(?m)^ID: (\d+)$`)
	m := re.FindStringSubmatch(body)
	if len(m) != 2 {
		t.Fatalf("could not extract ID from response:\n%s", body)
	}
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		t.Fatalf("parse id %q: %v", m[1], err)
	}
	return id
}

func TestIntegration_FullScenario(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "enchanted-inn"}

	// Wiring sanity — building the server must not panic and every tool must
	// be registerable side-by-side. The returned *MCPServer is what stdio
	// would attach to.
	if srv := New(st, cfg); srv == nil {
		t.Fatalf("New returned nil server")
	}

	// 1. tl_save — create a fresh memory with a stable topic_key.
	saveReq := buildReq("tl_save", map[string]any{
		"title":     "Lantern bake workflow",
		"content":   "Bake lantern-base normals before exporting to PlayCanvas. The bloom-safe import settings live in scene-tools/lantern-import.md.",
		"type":      "scene-pattern",
		"topic_key": "scene/lantern",
		"tags":      []any{"engine:playcanvas"},
	})
	saveRes, err := doSave(ctx, st, cfg, decodeSaveArgs(saveReq))
	if err != nil || saveRes.IsError {
		t.Fatalf("tl_save failed: err=%v body=%s", err, textContent(saveRes))
	}
	saveBody := textContent(saveRes)
	if !strings.Contains(saveBody, "action=created") {
		t.Fatalf("save: expected action=created, got:\n%s", saveBody)
	}
	id := extractID(t, saveBody)

	// 2. tl_search — keyword query finds the memory we just saved.
	searchReq := buildReq("tl_search", map[string]any{
		"query": "lantern bloom",
	})
	searchRes, err := doSearch(ctx, st, cfg, decodeSearchArgs(searchReq))
	if err != nil || searchRes.IsError {
		t.Fatalf("tl_search failed: err=%v body=%s", err, textContent(searchRes))
	}
	searchBody := textContent(searchRes)
	if !strings.Contains(searchBody, "Lantern bake workflow") {
		t.Fatalf("search: expected to find seeded title, got:\n%s", searchBody)
	}
	if !strings.Contains(searchBody, "tl_get_observation") {
		t.Fatalf("search: expected hint at tl_get_observation, got:\n%s", searchBody)
	}

	// 2b. tl_search — topic-key shortcut path (query containing "/").
	shortcutReq := buildReq("tl_search", map[string]any{
		"query": "scene/lantern",
	})
	shortcutRes, err := doSearch(ctx, st, cfg, decodeSearchArgs(shortcutReq))
	if err != nil || shortcutRes.IsError {
		t.Fatalf("topic-key shortcut failed: err=%v body=%s", err, textContent(shortcutRes))
	}
	if !strings.Contains(textContent(shortcutRes), "Lantern bake workflow") {
		t.Fatalf("shortcut: expected hit, got:\n%s", textContent(shortcutRes))
	}

	// 3. tl_get_observation — pull the full untruncated content for that id.
	getReq := buildReq("tl_get_observation", map[string]any{
		"id": float64(id), // JSON numbers decode to float64 — match that here
	})
	getRes, err := doGetObservation(ctx, st, decodeGetObservationArgs(getReq))
	if err != nil || getRes.IsError {
		t.Fatalf("tl_get_observation failed: err=%v body=%s", err, textContent(getRes))
	}
	getBody := textContent(getRes)
	if !strings.Contains(getBody, "Bake lantern-base normals") {
		t.Fatalf("get_observation: expected full content, got:\n%s", getBody)
	}

	// 4. tl_context — recent memories for the active project (default).
	contextReq := buildReq("tl_context", map[string]any{})
	contextRes, err := doContext(ctx, st, cfg, decodeContextArgs(contextReq))
	if err != nil || contextRes.IsError {
		t.Fatalf("tl_context failed: err=%v body=%s", err, textContent(contextRes))
	}
	if !strings.Contains(textContent(contextRes), "Lantern bake workflow") {
		t.Fatalf("context: expected to include the seeded memory, got:\n%s", textContent(contextRes))
	}

	// 5. tl_update — patch the title only; revision must bump, content stays.
	updateReq := buildReq("tl_update", map[string]any{
		"id":    float64(id),
		"title": "Lantern bake workflow (revised)",
	})
	updateRes, err := doUpdate(ctx, st, decodeUpdateArgs(updateReq))
	if err != nil || updateRes.IsError {
		t.Fatalf("tl_update failed: err=%v body=%s", err, textContent(updateRes))
	}
	updateBody := textContent(updateRes)
	if !strings.Contains(updateBody, "action=updated") {
		t.Fatalf("update: expected action=updated, got:\n%s", updateBody)
	}
	if !strings.Contains(updateBody, "Revision: 1") {
		t.Fatalf("update: expected Revision: 1, got:\n%s", updateBody)
	}
	if !strings.Contains(updateBody, "Lantern bake workflow (revised)") {
		t.Fatalf("update: expected new title in response, got:\n%s", updateBody)
	}

	// 5b. tl_search — post-update, search by NEW title token must hit.
	postSearchReq := buildReq("tl_search", map[string]any{
		"query": "revised",
	})
	postSearchRes, err := doSearch(ctx, st, cfg, decodeSearchArgs(postSearchReq))
	if err != nil || postSearchRes.IsError {
		t.Fatalf("post-update search failed: err=%v body=%s", err, textContent(postSearchRes))
	}
	if !strings.Contains(textContent(postSearchRes), "(revised)") {
		t.Fatalf("post-update search: expected new title token, got:\n%s", textContent(postSearchRes))
	}

	// 6. tl_delete — soft-delete the row.
	deleteReq := buildReq("tl_delete", map[string]any{
		"id": float64(id),
	})
	deleteRes, err := doDelete(ctx, st, decodeDeleteArgs(deleteReq))
	if err != nil || deleteRes.IsError {
		t.Fatalf("tl_delete failed: err=%v body=%s", err, textContent(deleteRes))
	}
	if !strings.Contains(textContent(deleteRes), "Deleted memory id=") {
		t.Fatalf("delete: expected confirmation, got:\n%s", textContent(deleteRes))
	}

	// 7. tl_search — post-delete must return empty (soft-deleted is hidden).
	postDeleteSearchRes, err := doSearch(ctx, st, cfg, decodeSearchArgs(buildReq("tl_search", map[string]any{
		"query": "lantern",
	})))
	if err != nil || postDeleteSearchRes.IsError {
		t.Fatalf("post-delete search failed: err=%v body=%s", err, textContent(postDeleteSearchRes))
	}
	if !strings.Contains(textContent(postDeleteSearchRes), "No matches") {
		t.Fatalf("post-delete search: expected no matches, got:\n%s", textContent(postDeleteSearchRes))
	}

	// 8. tl_context — post-delete must omit the deleted row too.
	postDeleteContextRes, err := doContext(ctx, st, cfg, decodeContextArgs(buildReq("tl_context", map[string]any{})))
	if err != nil || postDeleteContextRes.IsError {
		t.Fatalf("post-delete context failed: err=%v body=%s", err, textContent(postDeleteContextRes))
	}
	if strings.Contains(textContent(postDeleteContextRes), "Lantern") {
		t.Fatalf("post-delete context: deleted memory must not appear, got:\n%s", textContent(postDeleteContextRes))
	}

	// 9. tl_get_observation on the soft-deleted id must report not found.
	postDeleteGetRes, err := doGetObservation(ctx, st, decodeGetObservationArgs(buildReq("tl_get_observation", map[string]any{
		"id": float64(id),
	})))
	if err != nil {
		t.Fatalf("post-delete get_observation: %v", err)
	}
	if !postDeleteGetRes.IsError {
		t.Fatalf("post-delete get_observation: expected not-found error, got success: %s", textContent(postDeleteGetRes))
	}

	// 10. tl_update on the soft-deleted id must also report not found.
	postDeleteUpdateRes, err := doUpdate(ctx, st, decodeUpdateArgs(buildReq("tl_update", map[string]any{
		"id":    float64(id),
		"title": "should not work",
	})))
	if err != nil {
		t.Fatalf("post-delete update: %v", err)
	}
	if !postDeleteUpdateRes.IsError {
		t.Fatalf("post-delete update: expected not-found error, got success: %s", textContent(postDeleteUpdateRes))
	}

	// 11. tl_save with the same topic_key — soft-delete must have freed it.
	resaveRes, err := doSave(ctx, st, cfg, decodeSaveArgs(buildReq("tl_save", map[string]any{
		"title":     "Lantern bake workflow (replacement)",
		"content":   "fresh notes for the lantern bake after the prior row was deleted",
		"type":      "scene-pattern",
		"topic_key": "scene/lantern",
	})))
	if err != nil || resaveRes.IsError {
		t.Fatalf("re-save after delete: err=%v body=%s", err, textContent(resaveRes))
	}
	if !strings.Contains(textContent(resaveRes), "action=created") {
		t.Fatalf("re-save: expected action=created (topic_key freed by soft delete), got:\n%s", textContent(resaveRes))
	}
	newID := extractID(t, textContent(resaveRes))
	if newID == id {
		t.Errorf("re-save must produce a new row id, got same id %d", id)
	}
}
