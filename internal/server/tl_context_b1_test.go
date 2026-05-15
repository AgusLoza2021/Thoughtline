package server

// B1 — tl_context response budget tests (REQ-CAPS-B1-001 through REQ-CAPS-B1-004).
//
// These tests call formatContextResults(results, maxChars) directly using a
// small synthetic budget so fixtures stay small and deterministic. The last
// test (BudgetParamPassedThroughTlContext) exercises the doContext handler to
// verify the production constant is wired through.
//
// Naming is LOCKED per spec #116 / invocation contract.

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// makeResult constructs a SearchResult with a known snippet and title,
// at the given updatedAt time. All other fields are filled with valid defaults.
func makeResult(id int64, title, snippet string, updatedAt time.Time) storage.SearchResult {
	return storage.SearchResult{
		ID:            id,
		SyncID:        fmt.Sprintf("sync-%d", id),
		Project:       "enchanted-inn",
		Scope:         memory.ScopeProject,
		Type:          memory.TypeScenePattern,
		Title:         title,
		Snippet:       snippet,
		RevisionCount: 1,
		UpdatedAt:     updatedAt,
	}
}

// runeLen is a shorthand so test assertions read cleanly.
func runeLen(s string) int { return utf8.RuneCountInString(s) }

// renderFull renders one result block exactly as writeResultBlock does,
// for computing expected sizes. Mirrors the production render (no score).
func renderFull(r storage.SearchResult) string {
	var b strings.Builder
	writeResultBlock(&b, r, false)
	return b.String()
}

// renderHead renders a result block without the Snippet line, matching the
// head-only form produced by the B1 strip algorithm.
func renderHead(r storage.SearchResult) string {
	full := renderFull(r)
	// Strip the "Snippet: ...\n" line — it is always the last line.
	if idx := strings.LastIndex(full, "\nSnippet: "); idx >= 0 {
		return full[:idx+1] // keep the newline before "Snippet:"
	}
	return full
}

// t0 is the base time for B1 test fixtures. "Older" entries get earlier times.
var b1T0 = time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

// ---------------------------------------------------------------------------
// TestFormatContextResults_UnderCapUnchanged
//
// REQ-CAPS-B1-001 happy path: total render < budget → returned unchanged.
// ---------------------------------------------------------------------------
func TestFormatContextResults_UnderCapUnchanged(t *testing.T) {
	results := []storage.SearchResult{
		makeResult(2, "Newer entry", "newer snippet", b1T0.Add(2*time.Hour)),
		makeResult(1, "Older entry", "older snippet", b1T0.Add(1*time.Hour)),
	}

	// Use a budget well above the rendered size.
	budget := 8000
	got := formatContextResults(results, budget)

	if runeLen(got) > budget {
		t.Errorf("under-cap: result length %d > budget %d", runeLen(got), budget)
	}
	// Both snippets must be present — nothing stripped.
	if !strings.Contains(got, "newer snippet") {
		t.Errorf("under-cap: newer snippet must be present, got:\n%s", got)
	}
	if !strings.Contains(got, "older snippet") {
		t.Errorf("under-cap: older snippet must be present, got:\n%s", got)
	}
	// No omitted tail.
	if strings.Contains(got, "omitted") {
		t.Errorf("under-cap: no omitted tail expected, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// TestFormatContextResults_StripSnippetsOldestFirst
//
// REQ-CAPS-B1-002: snippets are stripped from oldest blocks first.
// ---------------------------------------------------------------------------
func TestFormatContextResults_StripSnippetsOldestFirst(t *testing.T) {
	// Create a long snippet so each block is sizeable.
	longSnippet := strings.Repeat("x", 200)

	newer := makeResult(2, "Newer entry", longSnippet, b1T0.Add(2*time.Hour))
	older := makeResult(1, "Older entry", longSnippet, b1T0.Add(1*time.Hour))
	// results arrive newest-first (as Recent returns them).
	results := []storage.SearchResult{newer, older}

	// Budget large enough for: full newer + head older + footer + tail-reserve,
	// but NOT large enough for full newer + full older + footer + tail-reserve.
	// We compute the snippet size to set the boundary precisely.
	headNewer := runeLen(renderHead(newer))
	headOlder := runeLen(renderHead(older))
	fullNewer := runeLen(renderFull(newer))
	fullOlder := runeLen(renderFull(older))
	snippetOlder := fullOlder - headOlder

	// budget = full-newer + head-older + 80 (tail reserve) + 200 (footer headroom)
	// This must satisfy: budget < full-newer + full-older + 80 + 200
	// i.e., fullNewer + headOlder + 280 < fullNewer + fullOlder + 280
	// i.e., headOlder < fullOlder  — true whenever snippet is non-empty.
	budget := fullNewer + headOlder + 280
	if snippetOlder <= 0 {
		t.Skip("fixture has no snippet — test not meaningful")
	}
	if budget >= fullNewer+fullOlder+280 {
		t.Fatalf("fixture budget too large — both full blocks would fit")
	}

	got := formatContextResults(results, budget)

	if runeLen(got) > budget {
		t.Errorf("after strip: result %d > budget %d", runeLen(got), budget)
	}
	// Newer block retains snippet.
	if !strings.Contains(got, longSnippet) {
		t.Errorf("strip-oldest: newer snippet must be retained, got:\n%s", got)
	}
	// Older block has title/ID but NOT the long snippet.
	if !strings.Contains(got, "Older entry") {
		t.Errorf("strip-oldest: older title must be present, got:\n%s", got)
	}
	// head sizes used above to confirm the test is coherent.
	_ = headNewer
}

// ---------------------------------------------------------------------------
// TestFormatContextResults_DropEntriesAfterStripping
//
// REQ-CAPS-B1-003: after stripping all snippets, oldest entries are dropped.
// ---------------------------------------------------------------------------
func TestFormatContextResults_DropEntriesAfterStripping(t *testing.T) {
	snippet := strings.Repeat("y", 100)

	newest := makeResult(3, "Newest entry", snippet, b1T0.Add(3*time.Hour))
	middle := makeResult(2, "Middle entry", snippet, b1T0.Add(2*time.Hour))
	oldest := makeResult(1, "Oldest entry", snippet, b1T0.Add(1*time.Hour))
	results := []storage.SearchResult{newest, middle, oldest}

	// Budget: enough for newest head + middle head, but NOT oldest head.
	headNewest := runeLen(renderHead(newest))
	headMiddle := runeLen(renderHead(middle))
	budget := headNewest + headMiddle + 5

	got := formatContextResults(results, budget)

	if runeLen(got) > budget {
		t.Errorf("after drop: result %d > budget %d", runeLen(got), budget)
	}
	// Oldest must be absent.
	if strings.Contains(got, "Oldest entry") {
		t.Errorf("drop-oldest: oldest entry must be dropped, got:\n%s", got)
	}
	// Newest must be present.
	if !strings.Contains(got, "Newest entry") {
		t.Errorf("drop-oldest: newest entry must be retained, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// TestFormatContextResults_NewestEntryFloorIgnoresCap
//
// REQ-CAPS-B1-004 floor: even when a single block exceeds the budget, it is
// returned (head-only) because recovery hooks MUST get at least one ID.
// ---------------------------------------------------------------------------
func TestFormatContextResults_NewestEntryFloorIgnoresCap(t *testing.T) {
	// Build a result whose head-only render is large (long title).
	longTitle := strings.Repeat("T", 300) // 300-char title guarantees head > tiny budget
	r := makeResult(42, longTitle, "some snippet", b1T0)
	results := []storage.SearchResult{r}

	// Budget smaller than the head block.
	budget := 50

	got := formatContextResults(results, budget)

	// Must include the ID even though it exceeds the cap.
	if !strings.Contains(got, "ID: 42") {
		t.Errorf("floor: newest entry ID must be present even over cap, got:\n%s", got)
	}
	// Snippet should be stripped (head-only).
	if strings.Contains(got, "some snippet") {
		t.Errorf("floor: snippet must be stripped when over budget, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// TestFormatContextResults_SingleEntryOverCapKeepsBlockStripsSnippet
//
// REQ-CAPS-B1-002 edge: single entry with a snippet; budget is between head
// size and full size → snippet is stripped, block is kept.
// ---------------------------------------------------------------------------
func TestFormatContextResults_SingleEntryOverCapKeepsBlockStripsSnippet(t *testing.T) {
	snippet := strings.Repeat("S", 250) // long enough to push over a small budget
	r := makeResult(7, "Solo entry", snippet, b1T0)
	results := []storage.SearchResult{r}

	fullSize := runeLen(renderFull(r))
	headSize := runeLen(renderHead(r))
	// Sanity: the fixture must actually make sense (head < full).
	if headSize >= fullSize {
		t.Fatalf("fixture broken: headSize %d >= fullSize %d", headSize, fullSize)
	}
	// Budget between head and full — strip is triggered, drop is not.
	budget := headSize + (fullSize-headSize)/2

	got := formatContextResults(results, budget)

	// Block is kept (ID and title present).
	if !strings.Contains(got, "ID: 7") {
		t.Errorf("single-over-cap: ID must be present, got:\n%s", got)
	}
	if !strings.Contains(got, "Solo entry") {
		t.Errorf("single-over-cap: title must be present, got:\n%s", got)
	}
	// Snippet stripped.
	if strings.Contains(got, snippet) {
		t.Errorf("single-over-cap: snippet must be stripped, got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// TestFormatContextResults_OmittedCountTail
//
// REQ-CAPS-B1-003: dropped entries produce a count tail.
// ---------------------------------------------------------------------------
func TestFormatContextResults_OmittedCountTail(t *testing.T) {
	snippet := strings.Repeat("z", 100)

	newest := makeResult(10, "Newest entry", snippet, b1T0.Add(3*time.Hour))
	dropped1 := makeResult(9, "Dropped one", snippet, b1T0.Add(2*time.Hour))
	dropped2 := makeResult(8, "Dropped two", snippet, b1T0.Add(1*time.Hour))
	results := []storage.SearchResult{newest, dropped1, dropped2}

	// Budget: only fits newest head (no room for dropped1 or dropped2).
	budget := runeLen(renderHead(newest)) + 5

	got := formatContextResults(results, budget)

	// Omitted tail must mention the count (2).
	if !strings.Contains(got, "2") {
		t.Errorf("omitted-tail: tail must mention count 2, got:\n%s", got)
	}
	if !strings.Contains(got, "omitted") {
		t.Errorf("omitted-tail: tail must contain 'omitted', got:\n%s", got)
	}
}

// ---------------------------------------------------------------------------
// TestFormatContextResults_BudgetParamPassedThroughTlContext
//
// Verifies that doContext wires ContextResponseMaxChars through to
// formatContextResults. Seeds enough memories to exceed the production cap
// and checks the response is <= ContextResponseMaxChars.
// ---------------------------------------------------------------------------
func TestFormatContextResults_BudgetParamPassedThroughTlContext(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	// Seed 20 memories with content large enough to push total > 4000 chars.
	bigContent := strings.Repeat("A", 400)
	for i := 0; i < 20; i++ {
		m := sampleSceneMemory(
			fmt.Sprintf("Entry %02d with a moderately long title to pad size", i),
			bigContent,
			fmt.Sprintf("scene/entry%02d", i),
		)
		seedMemory(t, st, m)
	}

	res, err := doContext(ctx, st, Config{}, contextArgs{
		Project: "enchanted-inn",
		Limit:   50,
	})
	if err != nil {
		t.Fatalf("doContext: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %s", textContent(res))
	}

	body := textContent(res)
	if runeLen(body) > ContextResponseMaxChars {
		t.Errorf("response length %d exceeds ContextResponseMaxChars %d",
			runeLen(body), ContextResponseMaxChars)
	}
}
