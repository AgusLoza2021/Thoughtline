package server

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

// extractSessionID parses the "Session ID: <uuid>" line emitted by
// tl_session_start / tl_session_summary responses.
func extractSessionID(t *testing.T, body string) string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^Session ID: ([0-9a-f-]+)$`)
	m := re.FindStringSubmatch(body)
	if len(m) != 2 {
		t.Fatalf("could not extract Session ID from response:\n%s", body)
	}
	return m[1]
}

// TestIntegration_SessionScenario walks the realistic M4 lifecycle an AI
// agent would follow:
//
//   tl_session_start  → returns sid
//   tl_save (× 2)      with session_id=sid → both attached
//   tl_session_summary → closes the session, persists digest
//   tl_session_summary → second call rejected ("already ended")
//   tl_save with sid    → still allowed; M4 design: post-mortem attach OK
//   tl_save cross-project session → rejected
//
// Each step builds an actual mcp.CallToolRequest with a JSON-shaped args
// map, runs it through the same decode + handler pair the registered tool
// would, and asserts on the formatted text response.
func TestIntegration_SessionScenario(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{Version: "test", DefaultProject: "enchanted-inn"}

	// Wiring sanity — building the server with M4 tools must not panic.
	if srv := New(st, cfg); srv == nil {
		t.Fatalf("New returned nil server")
	}

	// 1. Open a session.
	startRes, err := doSessionStart(ctx, st, cfg, decodeSessionStartArgs(buildReq("tl_session_start", map[string]any{
		"agent_label": "claude-code",
	})))
	if err != nil || startRes.IsError {
		t.Fatalf("tl_session_start failed: err=%v body=%s", err, textContent(startRes))
	}
	sid := extractSessionID(t, textContent(startRes))

	// 2. Two saves attached to the session via session_id.
	saveA, err := doSave(ctx, st, cfg, decodeSaveArgs(buildReq("tl_save", map[string]any{
		"title":      "Lantern bake notes",
		"content":    "lantern lantern bake bake",
		"type":       "scene-pattern",
		"topic_key":  "scene/lantern",
		"session_id": sid,
	})))
	if err != nil || saveA.IsError {
		t.Fatalf("save A failed: err=%v body=%s", err, textContent(saveA))
	}
	if !strings.Contains(textContent(saveA), "Session: "+sid) {
		t.Errorf("save A response should echo session id, got:\n%s", textContent(saveA))
	}

	saveB, err := doSave(ctx, st, cfg, decodeSaveArgs(buildReq("tl_save", map[string]any{
		"title":      "Bloom curve tweaks",
		"content":    "raised bloom threshold to 1.2 on Android",
		"type":       "perf-gotcha",
		"topic_key":  "perf/android/bloom",
		"session_id": sid,
	})))
	if err != nil || saveB.IsError {
		t.Fatalf("save B failed: err=%v body=%s", err, textContent(saveB))
	}

	// 2b. Verify the upsert-doesn't-clobber regression at the integration
	// level: re-save scene/lantern with different content, NO session_id.
	// The session linkage must survive.
	resave, err := doSave(ctx, st, cfg, decodeSaveArgs(buildReq("tl_save", map[string]any{
		"title":     "Lantern bake notes",
		"content":   "different content this time, no session_id passed",
		"type":      "scene-pattern",
		"topic_key": "scene/lantern",
	})))
	if err != nil || resave.IsError {
		t.Fatalf("resave failed: err=%v body=%s", err, textContent(resave))
	}
	if !strings.Contains(textContent(resave), "Session: "+sid) {
		t.Errorf("upsert without session_id must preserve prior session linkage, got:\n%s", textContent(resave))
	}

	// 3. Close the session with a summary.
	summaryRes, err := doSessionSummary(ctx, st, decodeSessionSummaryArgs(buildReq("tl_session_summary", map[string]any{
		"id":      sid,
		"summary": "## Goal\nBake lanterns.\n## Accomplished\n- baked lanterns\n- raised bloom threshold",
	})))
	if err != nil || summaryRes.IsError {
		t.Fatalf("tl_session_summary failed: err=%v body=%s", err, textContent(summaryRes))
	}
	body := textContent(summaryRes)
	if !strings.Contains(body, "Session closed") {
		t.Errorf("close response must confirm closure, got:\n%s", body)
	}
	if !strings.Contains(body, "Duration:") {
		t.Errorf("close response must include Duration, got:\n%s", body)
	}

	// 4. Second close must be rejected.
	repeat, err := doSessionSummary(ctx, st, decodeSessionSummaryArgs(buildReq("tl_session_summary", map[string]any{
		"id":      sid,
		"summary": "second attempt",
	})))
	if err != nil {
		t.Fatalf("repeat close: %v", err)
	}
	if !repeat.IsError {
		t.Fatalf("repeat close must yield error, got success: %s", textContent(repeat))
	}
	if !strings.Contains(strings.ToLower(textContent(repeat)), "already") {
		t.Errorf("repeat close error must say 'already ended', got:\n%s", textContent(repeat))
	}

	// 5. Post-mortem save attached to a closed session — by design, allowed.
	// (See PROGRESS.md M4 open question on this.)
	postMortem, err := doSave(ctx, st, cfg, decodeSaveArgs(buildReq("tl_save", map[string]any{
		"title":      "Late note",
		"content":    "remembered something after closing the session",
		"type":       "convention",
		"topic_key":  "convention/late",
		"session_id": sid,
	})))
	if err != nil || postMortem.IsError {
		t.Fatalf("post-mortem save must be allowed: err=%v body=%s", err, textContent(postMortem))
	}

	// 6. Cross-project session must be rejected.
	otherProj, err := doSessionStart(ctx, st, cfg, decodeSessionStartArgs(buildReq("tl_session_start", map[string]any{
		"project": "another-project",
	})))
	if err != nil || otherProj.IsError {
		t.Fatalf("other-project start failed: err=%v body=%s", err, textContent(otherProj))
	}
	otherSID := extractSessionID(t, textContent(otherProj))

	mismatch, err := doSave(ctx, st, cfg, decodeSaveArgs(buildReq("tl_save", map[string]any{
		"title":      "Mismatched save",
		"content":    "should not be allowed",
		"type":       "convention",
		"topic_key":  "convention/mismatch",
		"session_id": otherSID, // session belongs to another-project
		// No project arg → defaults to enchanted-inn (cfg.DefaultProject)
	})))
	if err != nil {
		t.Fatalf("mismatch save: %v", err)
	}
	if !mismatch.IsError {
		t.Fatalf("cross-project session_id must be rejected, got success: %s", textContent(mismatch))
	}
}
