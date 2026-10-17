package knowledge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCorpus materializes a throwaway corpus from relative paths to contents,
// so a keys test can state its whole fixture in one place.
func writeCorpus(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return dir
}

// curatedCorpus is the fixture the key-gate tests share: two curated entries,
// one uncurated entry that must keep the old coverage gate, and one curated
// entry carrying a negative key.
func curatedCorpus(t *testing.T) string {
	t.Helper()
	return writeCorpus(t, map[string]string{
		"INDEX.md": `# Fixture corpus

### Camera and feel (1)

- [Kick feedback](entries/kick-feedback.md) — A hit needs a kick.
- [Skeleton recolour](entries/skeleton-recolour.md) — Swap the skin.

### UI (1)

- [Animated fill bar](entries/progress-bar.md) — Stretch by width.
`,
		"keys.yaml": `# proposed by nan, owner reviews
kick-feedback: [camera-shake, screen-shake, !screen-flash]
progress-bar: [progress-bar, fillamount]
skeleton-recolour: [spine-skeleton, setskin, !spine-anatomy]
`,
		"entries/kick-feedback.md": `---
title: Camera Shake as an Additive Local Position Offset
tags: [camera, feel]
---

## Applies when

A hit or impact needs a short physical kick.

## Validate the result

Trigger the shake while a pan runs.

## Limits

Does not replace hit-stop.
`,
		"entries/progress-bar.md": `---
title: Animated fill bar
tags: [progress]
---

## Context

A progress bar sweeps toward a new value.

## Example

Set the RectTransform width.

## Notes

Sliced, not fillAmount.
`,
		"entries/skeleton-recolour.md": `---
title: Recolour a shared spine skeleton at runtime
tags: [spine]
---

## Applies when

One skeleton must serve every damage state.

## Validate the result

Compare a recoloured skin with the authored one.
`,
		"entries/plain-note.md": `---
title: Plain uncurated note
tags: [misc]
---

## Applies when

A hit flash needs a flash.
`,
	})
}

func TestParseKeysFile(t *testing.T) {
	raw := []byte(`# proposed by nan, owner reviews

camera-shake-as-an-additive-local-position-offset: [camera-shake, screen-shake, "impact-feedback"]
ui-fill: [Progress Bar, !spine-anatomy]
no-brackets: camera-shake
malformed line without a colon
`)
	got := parseKeysFile(raw)

	shake, ok := got["camera-shake-as-an-additive-local-position-offset"]
	if !ok {
		t.Fatalf("missing entry, got %+v", got)
	}
	if strings.Join(shake.Keys, ",") != "camera-shake,screen-shake,impact-feedback" {
		t.Errorf("Keys = %v", shake.Keys)
	}
	if len(shake.NegKeys) != 0 {
		t.Errorf("NegKeys should be empty, got %v", shake.NegKeys)
	}

	fill := got["ui-fill"]
	if strings.Join(fill.Keys, ",") != "progress-bar" {
		t.Errorf("Keys should be normalized to lowercase hyphens, got %v", fill.Keys)
	}
	if strings.Join(fill.NegKeys, ",") != "spine-anatomy" {
		t.Errorf("NegKeys = %v", fill.NegKeys)
	}
	if strings.Join(got["no-brackets"].Keys, ",") != "camera-shake" {
		t.Errorf("a bare list should parse for a tolerant author, got %v", got["no-brackets"].Keys)
	}
	// The comment and the malformed line are skipped, not fatal.
	if _, ok := got["# proposed by nan, owner reviews"]; ok {
		t.Error("comment line parsed as an entry")
	}
}

func TestScan_KeysSidecarAttachesKeysAndSkipsMetaFiles(t *testing.T) {
	corpus := mustScan(t, curatedCorpus(t))

	if len(corpus.Entries) != 4 {
		slugs := make([]string, len(corpus.Entries))
		for i, e := range corpus.Entries {
			slugs[i] = e.Slug
		}
		t.Fatalf("scanned %d entries (%v), want 4: keys.yaml and INDEX.md are not recipes",
			len(corpus.Entries), slugs)
	}

	bySlug := make(map[string]Entry, len(corpus.Entries))
	for _, e := range corpus.Entries {
		bySlug[e.Slug] = e
	}

	shake := bySlug["kick-feedback"]
	if strings.Join(shake.Keys, ",") != "camera-shake,screen-shake" {
		t.Errorf("Keys = %v", shake.Keys)
	}
	if strings.Join(shake.NegKeys, ",") != "screen-flash" {
		t.Errorf("NegKeys = %v", shake.NegKeys)
	}
	if shake.Group != "Camera and feel" {
		t.Errorf("Group = %q, want the INDEX.md group", shake.Group)
	}
	if plain := bySlug["plain-note"]; len(plain.Keys) != 0 {
		t.Errorf("an entry absent from keys.yaml must stay uncurated, got %v", plain.Keys)
	}
}

func TestSync_KeyTagsRespectTheTagRegex(t *testing.T) {
	ctx := context.Background()
	st := newSyncStorage(t)
	dir := curatedCorpus(t)
	// A key long enough to fail memory.Validate must be dropped, not fail sync.
	file := filepath.Join(dir, "keys.yaml")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read keys.yaml: %v", err)
	}
	long := strings.Repeat("a", 60)
	if err := os.WriteFile(file,
		append(raw, []byte("plain-note: ["+long+"]\n")...), 0o644); err != nil {
		t.Fatalf("append keys.yaml: %v", err)
	}

	rep, err := Sync(ctx, st, mustScan(t, dir), SyncOptions{})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	for _, e := range rep.Errors {
		t.Errorf("sync reported %s", e)
	}

	brainID, err := st.ResolveBrainID(ctx, DefaultSyncProject)
	if err != nil {
		t.Fatalf("resolve brain: %v", err)
	}
	rows, err := st.ListByTopicPrefix(ctx, brainID, TopicPrefix)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	tags := make(map[string]string, len(rows))
	for _, m := range rows {
		tags[m.TopicKey] = strings.Join(m.Tags, " ")
	}

	if got := tags[TopicKey("kick-feedback")]; !strings.Contains(got, "key:camera-shake") ||
		!strings.Contains(got, "key:screen-shake") || !strings.Contains(got, "notkey:screen-flash") {
		t.Errorf("key tags missing from %q", got)
	}
	if got := tags[TopicKey("progress-bar")]; !strings.Contains(got, "key:fillamount") {
		t.Errorf("key tags missing from %q", got)
	}
	if got := tags[TopicKey("plain-note")]; strings.Contains(got, "key:") {
		t.Errorf("an invalid key should be dropped, got %q", got)
	}
}

func TestCheck_KeyGateSelectsByCuratedKey(t *testing.T) {
	ctx := context.Background()
	st := newSyncStorage(t)
	if _, err := Sync(ctx, st, mustScan(t, curatedCorpus(t)), SyncOptions{}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	res, err := Check(ctx, st, CheckRequest{Task: "add camera shake on impact"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("hits = %+v, want only the curated camera-shake entry", res.Hits)
	}
	hit := res.Hits[0]
	if hit.TopicKey != "knowledge/kick-feedback" {
		t.Fatalf("topic_key = %q", hit.TopicKey)
	}
	if strings.Join(hit.MatchedKeys, ",") != "camera-shake" {
		t.Errorf("MatchedKeys = %v, want camera-shake", hit.MatchedKeys)
	}
}

// TestCheck_KeyGateRejectsACuratedEntryWithoutAKeyMatch is the behaviour the
// curated gate exists for: the entry's own words reach the query (its title is
// "Animated fill bar"), so the old coverage gate would have kept it. A key that
// did not fire means the author says this is not that work.
func TestCheck_KeyGateRejectsACuratedEntryWithoutAKeyMatch(t *testing.T) {
	ctx := context.Background()
	st := newSyncStorage(t)
	if _, err := Sync(ctx, st, mustScan(t, curatedCorpus(t)), SyncOptions{}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	res, err := Check(ctx, st, CheckRequest{
		Task:       "fill bar width",
		Limit:      MaxCheckLimit,
		MinWeight:  0.0001, // isolate the key gate from the corpus-scaled default
		MinCoverage: 1,
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	for _, h := range res.Hits {
		if h.TopicKey == "knowledge/progress-bar" {
			t.Fatalf("curated entry kept without a key match: %+v", h)
		}
	}
	if res.Best == nil || res.Best.TopicKey != "knowledge/progress-bar" {
		t.Errorf("Best = %+v, want the gated progress-bar entry as the closest miss", res.Best)
	}

	// The same query does select it through its key.
	res, err = Check(ctx, st, CheckRequest{Task: "progress bar width"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(res.Hits) != 1 || res.Hits[0].TopicKey != "knowledge/progress-bar" {
		t.Fatalf("key match hits = %+v", res.Hits)
	}
	if strings.Join(res.Hits[0].MatchedKeys, ",") != "progress-bar" {
		t.Errorf("MatchedKeys = %v", res.Hits[0].MatchedKeys)
	}
}

// TestCheck_UncuratedEntryFallsBackToCoverage pins that a corpus nobody curated
// keeps working: entries without keys are gated by coverage and IDF weight.
func TestCheck_UncuratedEntryFallsBackToCoverage(t *testing.T) {
	dir := curatedCorpus(t)
	if err := os.Remove(filepath.Join(dir, "keys.yaml")); err != nil {
		t.Fatalf("remove keys.yaml: %v", err)
	}

	ctx := context.Background()
	st := newSyncStorage(t)
	if _, err := Sync(ctx, st, mustScan(t, dir), SyncOptions{}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	res, err := Check(ctx, st, CheckRequest{Task: "fill bar width", MinCoverage: 2, MinWeight: 0.0001})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(res.Hits) == 0 || res.Hits[0].TopicKey != "knowledge/progress-bar" {
		t.Fatalf("uncurated entry lost its coverage gate: %+v", res.Hits)
	}
	if len(res.Hits[0].MatchedKeys) != 0 {
		t.Errorf("uncurated hit should carry no keys, got %v", res.Hits[0].MatchedKeys)
	}
}

func TestCheck_NegativeKeyBlocksAHit(t *testing.T) {
	ctx := context.Background()
	st := newSyncStorage(t)
	if _, err := Sync(ctx, st, mustScan(t, curatedCorpus(t)), SyncOptions{}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	// spine-skeleton matches, but spine-anatomy is exactly the false friend the
	// negative key names, and it vetoes the entry.
	blocked, err := Check(ctx, st, CheckRequest{Task: "spine skeleton anatomy diagram"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	for _, h := range blocked.Hits {
		if h.TopicKey == "knowledge/skeleton-recolour" {
			t.Fatalf("negative key did not block the hit: %+v", h)
		}
	}

	kept, err := Check(ctx, st, CheckRequest{Task: "recolour spine skeleton damage state"})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(kept.Hits) != 1 || kept.Hits[0].TopicKey != "knowledge/skeleton-recolour" {
		t.Fatalf("hits = %+v, want the skeleton entry", kept.Hits)
	}
	if strings.Join(kept.Hits[0].MatchedKeys, ",") != "spine-skeleton" {
		t.Errorf("MatchedKeys = %v", kept.Hits[0].MatchedKeys)
	}
}

func TestKeyMatches(t *testing.T) {
	words := map[string]bool{"camera": true, "shake": true, "hit": true}
	cases := []struct {
		key  string
		want bool
	}{
		{key: "camera-shake", want: true},
		{key: "camera", want: true},
		{key: "camera-flash", want: false},
		{key: "shake-camera-hit", want: true},
		{key: "-", want: false},
		{key: "hits", want: true}, // a trailing plural is the one tolerated inflection
		{key: "hitz", want: false},
	}
	for _, tc := range cases {
		if got := keyMatches(tc.key, words); got != tc.want {
			t.Errorf("keyMatches(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

// TestSync_KeysEditReachesTheDatabase pins the reason a curated entry's hash
// absorbs its keys: the recipe file did not change, so without it every row
// would report "unchanged" and keep the old tags forever.
func TestSync_KeysEditReachesTheDatabase(t *testing.T) {
	ctx := context.Background()
	st := newSyncStorage(t)
	dir := curatedCorpus(t)

	first, err := Sync(ctx, st, mustScan(t, dir), SyncOptions{})
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if first.Created != 4 {
		t.Fatalf("first sync created %d, want 4", first.Created)
	}

	second, err := Sync(ctx, st, mustScan(t, dir), SyncOptions{})
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if second.Unchanged != 4 {
		t.Errorf("re-sync should be idempotent, got created=%d updated=%d unchanged=%d",
			second.Created, second.Updated, second.Unchanged)
	}

	// Curate an entry that had no keys. Only keys.yaml changes.
	file := filepath.Join(dir, "keys.yaml")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read keys.yaml: %v", err)
	}
	if err := os.WriteFile(file, append(raw, []byte("plain-note: [hit-flash]\n")...), 0o644); err != nil {
		t.Fatalf("append keys.yaml: %v", err)
	}

	third, err := Sync(ctx, st, mustScan(t, dir), SyncOptions{})
	if err != nil {
		t.Fatalf("third sync: %v", err)
	}
	if third.Updated != 1 || third.Unchanged != 3 {
		t.Errorf("keys.yaml edit should update exactly the curated entry, got updated=%d unchanged=%d",
			third.Updated, third.Unchanged)
	}

	brainID, err := st.ResolveBrainID(ctx, DefaultSyncProject)
	if err != nil {
		t.Fatalf("resolve brain: %v", err)
	}
	rows, err := st.ListByTopicPrefix(ctx, brainID, TopicPrefix)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, m := range rows {
		if m.TopicKey == TopicKey("plain-note") && !strings.Contains(strings.Join(m.Tags, " "), "key:hit-flash") {
			t.Errorf("new key did not reach the stored row: %v", m.Tags)
		}
	}
}
