package storage

import (
	"context"
	"strings"
	"testing"
)

// MatchAny exists because a check builds its own multi-token query out of a
// task description; requiring every token in one row is far too strict there.
func TestSearch_MatchAny_OrSemantics(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	for _, word := range []string{"alpha", "beta", "gamma"} {
		m := sampleMemory()
		m.TopicKey = "scene/" + word
		m.Title = strings.ToUpper(word) + " note"
		m.Content = "This note talks only about " + word + "."
		seed(t, st, m)
	}

	// Implicit AND: no single row holds both words.
	andHits, err := st.Search(ctx, brainID, "alpha gamma", SearchOptions{})
	if err != nil {
		t.Fatalf("AND search: %v", err)
	}
	if len(andHits) != 0 {
		t.Fatalf("implicit AND should not match, got %d hits", len(andHits))
	}

	// MatchAny: either word is enough.
	anyHits, err := st.Search(ctx, brainID, "alpha gamma", SearchOptions{MatchAny: true})
	if err != nil {
		t.Fatalf("MatchAny search: %v", err)
	}
	if len(anyHits) != 2 {
		t.Fatalf("MatchAny should return 2 hits, got %d", len(anyHits))
	}
	for _, r := range anyHits {
		if !strings.Contains(strings.ToLower(r.Title), "alpha") && !strings.Contains(strings.ToLower(r.Title), "gamma") {
			t.Errorf("unexpected hit %q", r.Title)
		}
	}
}

// The default path must stay byte-identical: a two-token query still filters
// by AND and a single token still matches.
func TestSearch_MatchAny_DefaultUnchanged(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	both := sampleMemory()
	both.TopicKey = "scene/both"
	both.Title = "Alpha and gamma together"
	both.Content = "Mentions alpha and gamma in one body."
	seed(t, st, both)

	andHits, err := st.Search(ctx, brainID, "alpha gamma", SearchOptions{})
	if err != nil {
		t.Fatalf("AND search: %v", err)
	}
	if len(andHits) != 1 {
		t.Fatalf("implicit AND should return the row holding both tokens, got %d", len(andHits))
	}

	// OR mode keeps the same row, it just also allows partial matches.
	anyHits, err := st.Search(ctx, brainID, "alpha gamma", SearchOptions{MatchAny: true})
	if err != nil {
		t.Fatalf("MatchAny search: %v", err)
	}
	if len(anyHits) != 1 {
		t.Fatalf("MatchAny should still return exactly that row, got %d", len(anyHits))
	}
}

// OR mode must not let an FTS5 operator through sanitization.
func TestSearch_MatchAny_SanitizesOperators(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.TopicKey = "scene/inject"
	m.Title = "NEAR injection attempt"
	m.Content = "Body mentioning NEAR and parenthesis ( and quote \" characters."
	seed(t, st, m)

	if _, err := st.Search(ctx, brainID, `NEAR( "quote`, SearchOptions{MatchAny: true}); err != nil {
		t.Fatalf("MatchAny with operator-ish input must not error: %v", err)
	}
}

func TestSanitizeFTSQueryMode_Joiners(t *testing.T) {
	if got, want := sanitizeFTSQueryMode("a b", false), `"a" "b"`; got != want {
		t.Errorf("AND joiner = %q, want %q", got, want)
	}
	if got, want := sanitizeFTSQueryMode("a b", true), `"a" OR "b"`; got != want {
		t.Errorf("OR joiner = %q, want %q", got, want)
	}
	if got := sanitizeFTSQuery(`" "`); got != "" {
		t.Errorf("quote-only query should sanitize to empty, got %q", got)
	}
}
