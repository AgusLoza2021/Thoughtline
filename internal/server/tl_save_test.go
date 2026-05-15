package server

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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
		{"whitespace-only content", func(a *saveArgs) { a.Content = "   \t\n  " }, "'content' is required"},
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

// ---------------------------------------------------------------------------
// A1 — Observation truncation tests (REQ-CAPS-A1-001)
// ---------------------------------------------------------------------------

// truncationMarker is the exact marker appended when content is truncated.
// Tests use this constant so the expected value stays in sync with the
// implementation without duplicating the literal.
const truncationMarker = "\n\n…[truncated by Thoughtline at 50000 chars]"

// TestDoSave_UnderCap_Passthrough verifies that content below the rune cap is
// stored unchanged and no truncation marker is appended.
func TestDoSave_UnderCap_Passthrough(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{}

	tests := []struct {
		name    string
		content string
	}{
		{"49999 runes", strings.Repeat("a", 49999)},
		{"1 rune", "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := validArgs()
			args.Content = tc.content
			res, err := doSave(ctx, st, cfg, args)
			if err != nil {
				t.Fatalf("doSave error: %v", err)
			}
			if res.IsError {
				t.Fatalf("unexpected error result: %s", textContent(res))
			}
			// Verify stored content via a round-trip read.
			stored := storedContentByResult(t, st, res)
			if stored != tc.content {
				t.Errorf("content changed: got %d runes, want %d runes; ends with %q",
					utf8.RuneCountInString(stored),
					utf8.RuneCountInString(tc.content),
					lastN(stored, 50))
			}
		})
	}
}

// TestDoSave_AtCapBoundary_NoTruncate verifies that content of exactly 50000
// runes is stored without modification.
func TestDoSave_AtCapBoundary_NoTruncate(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{}

	content := strings.Repeat("b", memory.MaxObservationChars)
	args := validArgs()
	args.Content = content

	res, err := doSave(ctx, st, cfg, args)
	if err != nil {
		t.Fatalf("doSave error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", textContent(res))
	}
	stored := storedContentByResult(t, st, res)
	if utf8.RuneCountInString(stored) != memory.MaxObservationChars {
		t.Errorf("rune count = %d, want %d", utf8.RuneCountInString(stored), memory.MaxObservationChars)
	}
	if strings.HasSuffix(stored, truncationMarker) {
		t.Error("at-cap content should not have truncation marker")
	}
}

// TestDoSave_TruncatesOversizedContent_AddsMarker verifies that content of
// 50001 runes is truncated to exactly MaxObservationChars runes with the
// marker as suffix.
func TestDoSave_TruncatesOversizedContent_AddsMarker(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{}

	content := strings.Repeat("c", memory.MaxObservationChars+1)
	args := validArgs()
	args.Content = content

	res, err := doSave(ctx, st, cfg, args)
	if err != nil {
		t.Fatalf("doSave error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", textContent(res))
	}
	stored := storedContentByResult(t, st, res)
	runeCount := utf8.RuneCountInString(stored)
	if runeCount != memory.MaxObservationChars {
		t.Errorf("truncated rune count = %d, want %d", runeCount, memory.MaxObservationChars)
	}
	if !strings.HasSuffix(stored, truncationMarker) {
		t.Errorf("truncated content should end with marker; got suffix: %q", lastN(stored, 60))
	}
}

// TestDoSave_MultibyteRunes_TruncatedByRuneCount verifies that truncation is
// counted by Unicode code points (runes), not bytes. The fixture places a
// 3-byte CJK rune exactly at the truncation boundary so the test confirms
// the implementation slices on a rune boundary (no partial rune in result)
// and that the stored rune count is exactly MaxObservationChars.
//
// Fixture design: 49999 ASCII 'x' + '界' (3-byte, U+754C) at rune 50000 +
// several more ASCII chars — total rune count > 50000, total byte count stays
// well under MaxContentBytes so Validate does not fire before the truncation
// assertion can be checked.
func TestDoSave_MultibyteRunes_TruncatedByRuneCount(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{}

	// Build: 49999 ASCII runes + one 3-byte CJK rune + 10 more ASCII runes.
	// Rune count = 50010 > MaxObservationChars. Byte count ≈ 50012 < 64 KiB.
	content := strings.Repeat("x", memory.MaxObservationChars-1) + "界" + strings.Repeat("y", 10)
	args := validArgs()
	args.Content = content

	res, err := doSave(ctx, st, cfg, args)
	if err != nil {
		t.Fatalf("doSave error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", textContent(res))
	}
	stored := storedContentByResult(t, st, res)

	// Rune count must be exactly MaxObservationChars.
	runeCount := utf8.RuneCountInString(stored)
	if runeCount != memory.MaxObservationChars {
		t.Errorf("rune count = %d, want %d", runeCount, memory.MaxObservationChars)
	}
	// Must be valid UTF-8 — no partial rune.
	if !utf8.ValidString(stored) {
		t.Error("stored content is not valid UTF-8 (partial rune at boundary)")
	}
	// Must end with the marker.
	if !strings.HasSuffix(stored, truncationMarker) {
		t.Errorf("truncated content should end with marker; got suffix: %q", lastN(stored, 60))
	}
}

// TestDoSave_AlreadyTruncatedMarker_NotDoubled verifies that re-saving content
// that was already truncated (ends with marker, exactly MaxObservationChars
// runes) does not append the marker a second time.
func TestDoSave_AlreadyTruncatedMarker_NotDoubled(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{}

	// Construct content that is already exactly MaxObservationChars runes and
	// ends with the marker — simulating a previously-truncated observation.
	markerRunes := utf8.RuneCountInString(truncationMarker)
	bodyRunes := memory.MaxObservationChars - markerRunes
	alreadyTruncated := strings.Repeat("d", bodyRunes) + truncationMarker

	if utf8.RuneCountInString(alreadyTruncated) != memory.MaxObservationChars {
		t.Fatalf("test fixture rune count = %d, want %d (fix the test)",
			utf8.RuneCountInString(alreadyTruncated), memory.MaxObservationChars)
	}

	args := validArgs()
	args.Content = alreadyTruncated

	res, err := doSave(ctx, st, cfg, args)
	if err != nil {
		t.Fatalf("doSave error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", textContent(res))
	}
	stored := storedContentByResult(t, st, res)

	// Rune count must still be exactly MaxObservationChars.
	runeCount := utf8.RuneCountInString(stored)
	if runeCount != memory.MaxObservationChars {
		t.Errorf("rune count = %d, want %d (marker doubled?)", runeCount, memory.MaxObservationChars)
	}
	// Marker must appear exactly once.
	count := strings.Count(stored, truncationMarker)
	if count != 1 {
		t.Errorf("marker appears %d times, want exactly 1", count)
	}
}

// ---------------------------------------------------------------------------
// Helpers used by A1 tests
// ---------------------------------------------------------------------------

// storedContentByResult retrieves the full stored content of the memory whose
// ID is embedded in the doSave result text. Parses "ID: <n>" from the text,
// then fetches the row via GetByIDUnscoped so the full Content field is
// available (SearchResult only carries a 300-char snippet).
func storedContentByResult(t *testing.T, st *storage.Storage, res *mcp.CallToolResult) string {
	t.Helper()
	body := textContent(res)
	// Parse "ID: <digits>" from the result body.
	var id int64
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID: ") {
			if _, err := fmt.Sscanf(line, "ID: %d", &id); err == nil {
				break
			}
		}
	}
	if id == 0 {
		t.Fatalf("could not parse ID from save result:\n%s", body)
	}
	m, err := st.GetByIDUnscoped(context.Background(), id)
	if err != nil {
		t.Fatalf("GetByIDUnscoped(%d): %v", id, err)
	}
	return m.Content
}

// lastN returns the last n characters of s (for readable error messages).
func lastN(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[len(r)-n:])
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

// ---------------------------------------------------------------------------
// A2 — Dedup window tests (REQ-CAPS-A2-001 through REQ-CAPS-A2-004)
// ---------------------------------------------------------------------------

// dedupeBaseArgs returns a validArgs with no TopicKey, suitable for dedup testing.
func dedupeBaseArgs() saveArgs {
	return saveArgs{
		Title:   "Lock player loot UI to a 4x6 grid",
		Content: "**What**: chose grid\n**Why**: cognitive load on mobile\n**Where**: ui/loot/grid.js\n",
		Type:    string(memory.TypeGameDesignDecision),
		Project: "enchanted-inn",
	}
}

// TestDoSave_DedupeReturnsExistingEnvelope verifies the end-to-end integration:
// a second non-topic-key save with identical title+content within the dedup
// window returns a "deduped" response envelope and does NOT insert a new row.
func TestDoSave_DedupeReturnsExistingEnvelope(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{}

	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	st.SetClock(func() time.Time { return baseTime })

	args := dedupeBaseArgs()

	// First save succeeds normally.
	res1, err := doSave(ctx, st, cfg, args)
	if err != nil || res1.IsError {
		t.Fatalf("first save: err=%v body=%s", err, textContent(res1))
	}
	body1 := textContent(res1)
	if !strings.Contains(body1, "action=created") {
		t.Fatalf("first save should be created, got:\n%s", body1)
	}

	// Resolve the brain to count rows later.
	brainID, err := st.ResolveOrCreateBrainID(ctx, args.Project)
	if err != nil {
		t.Fatalf("resolve brain: %v", err)
	}

	// Advance time to 5 minutes later — still within the 15-minute window.
	st.SetClock(func() time.Time { return baseTime.Add(5 * time.Minute) })

	// Second save with identical args → should deduplicate.
	res2, err := doSave(ctx, st, cfg, args)
	if err != nil {
		t.Fatalf("second save error: %v", err)
	}
	if res2.IsError {
		t.Fatalf("expected success result, got error: %s", textContent(res2))
	}
	body2 := textContent(res2)

	// Response must mention "deduped".
	if !strings.Contains(body2, "deduped") {
		t.Errorf("dedup response should contain 'deduped'; got:\n%s", body2)
	}

	// Extract the numeric ID from the first save's response so we can assert
	// the dedup envelope references the same row (design contract D12).
	var savedID int64
	for _, line := range strings.Split(body1, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID: ") {
			if _, err := fmt.Sscanf(line, "ID: %d", &savedID); err == nil {
				break
			}
		}
	}
	if savedID == 0 {
		t.Fatalf("could not parse ID from first save result:\n%s", body1)
	}
	// Dedup envelope must embed the existing ID.
	if !strings.Contains(body2, fmt.Sprintf("ID %d", savedID)) {
		t.Errorf("dedup response should reference existing ID %d; got:\n%s", savedID, body2)
	}
	// Dedup envelope must include the age fragment.
	if !strings.Contains(body2, "ago") {
		t.Errorf("dedup response should contain 'ago'; got:\n%s", body2)
	}
	// Dedup envelope must include the no-insert affordance.
	if !strings.Contains(body2, "No new row inserted") {
		t.Errorf("dedup response should contain 'No new row inserted'; got:\n%s", body2)
	}

	// No new row should be inserted — Recent should still return exactly 1 row.
	rows, err := st.Recent(ctx, brainID, 10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row after dedup (no new insert); got %d", len(rows))
	}
}

// TestDoSave_TopicKeyedSave_BypassesDedupeCheck verifies that a topic-keyed save
// with args identical to an existing non-topic-keyed memory does NOT trigger
// the dedup path — it should go through upsertByTopicKey normally.
func TestDoSave_TopicKeyedSave_BypassesDedupeCheck(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	cfg := Config{}

	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	st.SetClock(func() time.Time { return baseTime })

	// Save a non-topic-keyed row first.
	plain := dedupeBaseArgs()
	res1, err := doSave(ctx, st, cfg, plain)
	if err != nil || res1.IsError {
		t.Fatalf("plain save: err=%v body=%s", err, textContent(res1))
	}

	// Now save the SAME content with a topic_key — should bypass dedup,
	// create a new row via upsertByTopicKey.
	keyed := dedupeBaseArgs()
	keyed.TopicKey = "design/inventory/grid-layout"

	res2, err := doSave(ctx, st, cfg, keyed)
	if err != nil || res2.IsError {
		t.Fatalf("keyed save: err=%v body=%s", err, textContent(res2))
	}
	body2 := textContent(res2)

	// Must NOT be a dedup response — should be a normal created action.
	if strings.Contains(body2, "deduped") {
		t.Errorf("topic-keyed save must not trigger dedup; got:\n%s", body2)
	}
	if !strings.Contains(body2, "action=created") {
		t.Errorf("topic-keyed save should be action=created; got:\n%s", body2)
	}
}
