package knowledge

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func newSyncStorage(t *testing.T) *storage.Storage {
	t.Helper()
	st, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// copyCorpus duplicates testdata so a test can edit and delete files without
// touching the fixture other tests scan.
func copyCorpus(t *testing.T, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "corpus")
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o644)
	})
	if err != nil {
		t.Fatalf("copy corpus: %v", err)
	}
	return dst
}

func mustScan(t *testing.T, dir string) Corpus {
	t.Helper()
	c, err := Scan(dir)
	if err != nil {
		t.Fatalf("scan %s: %v", dir, err)
	}
	return c
}

func sampleEntry() Entry {
	return Entry{
		Slug:     "camera-shake",
		Title:    "Camera Shake as an Additive Local Position Offset",
		Tags:     []string{"camera", "feel"},
		Source:   "external-knowledge-2026-09",
		Group:    "Camera and feel",
		Template: TemplateProcedural,
		Stacks:   []string{"svn"},
		Body:     "## Guidance\n\nAdd the shake on top.\n",
		Hash:     "0123456789abcdef0123456789abcdef01234567",
	}
}

func TestToMemory_TypeMapping(t *testing.T) {
	cases := []struct {
		template Template
		want     memory.Type
	}{
		{TemplateProcedural, memory.TypeConvention},
		{TemplateRecipe, memory.TypeScriptPattern},
		{TemplateOther, memory.TypeConvention},
	}

	for _, tc := range cases {
		e := sampleEntry()
		e.Template = tc.template
		got := ToMemory(e, SyncOptions{}).Type
		if got != tc.want {
			t.Errorf("template %q -> type %q, want %q", tc.template, got, tc.want)
		}
	}
}

func TestToMemory_DefaultsAndScope(t *testing.T) {
	m := ToMemory(sampleEntry(), SyncOptions{})
	if m.Project != DefaultSyncProject {
		t.Errorf("project = %q, want %q", m.Project, DefaultSyncProject)
	}
	if m.Scope != memory.ScopeProject {
		t.Errorf("scope = %q, want project", m.Scope)
	}
	if m.TopicKey != "knowledge/camera-shake" {
		t.Errorf("topic_key = %q", m.TopicKey)
	}
}

func TestToMemory_ValidatesForRealEntries(t *testing.T) {
	for _, e := range mustScan(t, "testdata/corpus").Entries {
		m := ToMemory(e, SyncOptions{})
		if err := memory.Validate(m); err != nil {
			t.Errorf("%s: %v (topic_key=%q tags=%v)", e.Slug, err, m.TopicKey, m.Tags)
		}
	}
}

func TestTopicKey_LongSlugStaysValid(t *testing.T) {
	e := sampleEntry()
	e.Slug = strings.Repeat("a", 120)
	m := ToMemory(e, SyncOptions{})

	if len(m.TopicKey) > memory.MaxTopicKeyLen+1 {
		t.Fatalf("topic_key is %d chars: %q", len(m.TopicKey), m.TopicKey)
	}
	if !strings.HasPrefix(m.TopicKey, TopicPrefix+strings.Repeat("a", 110)+"-") {
		t.Fatalf("truncated slug keeps no hash suffix: %q", m.TopicKey)
	}
	if err := memory.Validate(m); err != nil {
		t.Fatalf("long-slug entry failed validation: %v", err)
	}

	// Distinct long slugs must not collapse onto the same key.
	other := e
	other.Slug = strings.Repeat("a", 119) + "b"
	if got := ToMemory(other, SyncOptions{}).TopicKey; got == m.TopicKey {
		t.Fatalf("two long slugs share the key %q", got)
	}
}

func TestTopicKey_ScrubsCharactersTheRegexRejects(t *testing.T) {
	cases := map[string]string{
		"v1.2 notes":    "knowledge/v1-2-notes",
		"  UPPER Case ": "knowledge/upper-case",
		"...":           "knowledge/entry",
	}
	for slug, want := range cases {
		if got := TopicKey(slug); got != want {
			t.Errorf("TopicKey(%q) = %q, want %q", slug, got, want)
		}
	}
}

func TestToMemory_TagsFilterDedupeAndCap(t *testing.T) {
	e := sampleEntry()
	// One tag the memory regex rejects, one duplicate of a derived tag.
	e.Tags = []string{"Camera Bonus", "kind:knowledge", "has space and CAPS?!"}
	e.Stacks = []string{"svn"}

	tags := ToMemory(e, SyncOptions{}).Tags

	wantPrefix := []string{"kind:knowledge", "corpus:unity", "template:procedural", "group:camera-and-feel", "stack:svn"}
	for i, want := range wantPrefix {
		if i >= len(tags) || tags[i] != want {
			t.Fatalf("tags = %v, want prefix %v", tags, wantPrefix)
		}
	}

	tagShape := regexp.MustCompile(`^[a-z0-9][a-z0-9:_-]{0,40}$`)
	seen := map[string]bool{}
	for _, tag := range tags {
		if !tagShape.MatchString(tag) {
			t.Errorf("tag %q does not match the memory tag pattern", tag)
		}
		if seen[tag] {
			t.Errorf("tag %q appears twice", tag)
		}
		seen[tag] = true
	}
	if seen["kind:knowledge"] != true {
		t.Fatal("kind:knowledge missing")
	}
	if strings.Count(strings.Join(tags, " "), "kind:knowledge") != 1 {
		t.Errorf("derived and frontmatter tag were not deduped: %v", tags)
	}
	if len(tags) > maxTags {
		t.Errorf("got %d tags, want <= %d", len(tags), maxTags)
	}

	// Cap check with far more valid tags than the limit.
	e.Tags = nil
	for i := 0; i < maxTags+10; i++ {
		e.Tags = append(e.Tags, "tag"+string(rune('a'+i%26))+string(rune('a'+i/26)))
	}
	if got := ToMemory(e, SyncOptions{}).Tags; len(got) != maxTags {
		t.Errorf("got %d tags, want the cap of %d", len(got), maxTags)
	}
}

func TestToMemory_FooterPresent(t *testing.T) {
	e := sampleEntry()
	m := ToMemory(e, SyncOptions{Corpus: "demo"})

	// The footer carries the hash and nothing else: its words used to be indexed
	// as content, so the group name alone made every entry of a group a match
	// for any task naming it.
	if !strings.HasSuffix(m.Content, "\n\n---\n_kh:0123456789ab_") {
		t.Fatalf("footer missing or malformed:\n%s", m.Content)
	}
	footer := m.Content[strings.LastIndex(m.Content, "\n---\n"):]
	for _, word := range []string{"Camera and feel", "external-knowledge-2026-09", "camera-shake", "demo"} {
		if strings.Contains(footer, word) {
			t.Errorf("footer must not carry %q (FTS indexes it as content):\n%s", word, footer)
		}
	}
	if !strings.HasPrefix(m.Content, "## Guidance") {
		t.Fatalf("content must start with the body:\n%s", m.Content)
	}
}

// TestSync_FooterChangeIsNotABodyChange guards the price of the shorter footer:
// the hash still lives in the content, so re-running the importer is still a
// no-op and a real body edit is still an update.
func TestSync_FooterChangeIsNotABodyChange(t *testing.T) {
	dir := copyCorpus(t, "testdata/corpus")
	st := newSyncStorage(t)
	ctx := context.Background()

	if rep, err := Sync(ctx, st, mustScan(t, dir), SyncOptions{}); err != nil || rep.Created != 4 {
		t.Fatalf("first sync = %+v, err %v", rep, err)
	}
	rep, err := Sync(ctx, st, mustScan(t, dir), SyncOptions{})
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if rep.Unchanged != 4 || rep.Updated != 0 {
		t.Fatalf("re-run report = %+v, want 4 unchanged", rep)
	}

	shake := filepath.Join(dir, "entries", "camera-shake.md")
	raw, err := os.ReadFile(shake)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if err := os.WriteFile(shake, append(raw, []byte("\n## Notes\n\nA body edit.\n")...), 0o644); err != nil {
		t.Fatalf("write entry: %v", err)
	}
	rep, err = Sync(ctx, st, mustScan(t, dir), SyncOptions{})
	if err != nil {
		t.Fatalf("third sync: %v", err)
	}
	if rep.Updated != 1 || rep.Unchanged != 3 {
		t.Fatalf("body edit report = %+v, want 1 updated", rep)
	}
}

func TestToMemory_TitleTruncatedByRunes(t *testing.T) {
	e := sampleEntry()
	e.Title = strings.Repeat("ñ", memory.MaxTitleChars+20)

	m := ToMemory(e, SyncOptions{})
	if got := len([]rune(m.Title)); got != memory.MaxTitleChars {
		t.Fatalf("title has %d runes, want %d", got, memory.MaxTitleChars)
	}
	if err := memory.Validate(m); err != nil {
		t.Fatalf("truncated title must validate: %v", err)
	}
}

// TestSync_Lifecycle walks the whole contract: first import, idempotent re-run,
// edited entry, deleted file, and a dry run that writes nothing.
func TestSync_Lifecycle(t *testing.T) {
	dir := copyCorpus(t, "testdata/corpus")
	st := newSyncStorage(t)
	ctx := context.Background()
	opt := SyncOptions{Prune: true}

	rep, err := Sync(ctx, st, mustScan(t, dir), opt)
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if rep.Scanned != 4 || rep.Created != 4 || rep.Updated != 0 || rep.Unchanged != 0 || rep.Deleted != 0 || rep.Skipped != 0 {
		t.Fatalf("first sync report = %+v", rep)
	}

	rep, err = Sync(ctx, st, mustScan(t, dir), opt)
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if rep.Created != 0 || rep.Updated != 0 || rep.Unchanged != 4 || rep.Deleted != 0 {
		t.Fatalf("second sync report = %+v, want 4 unchanged", rep)
	}

	// Edit one entry: exactly one update.
	shake := filepath.Join(dir, "entries", "camera-shake.md")
	raw, err := os.ReadFile(shake)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	edited := strings.Replace(string(raw), "Zero the offset when the curve ends.", "Zero the offset when the curve ends, then blend out.", 1)
	if edited == string(raw) {
		t.Fatal("fixture text changed; update this test")
	}
	if err := os.WriteFile(shake, []byte(edited), 0o644); err != nil {
		t.Fatalf("write entry: %v", err)
	}
	rep, err = Sync(ctx, st, mustScan(t, dir), opt)
	if err != nil {
		t.Fatalf("third sync: %v", err)
	}
	if rep.Created != 0 || rep.Updated != 1 || rep.Unchanged != 3 || rep.Deleted != 0 {
		t.Fatalf("after edit report = %+v, want 1 updated", rep)
	}

	// Dry run on a changed file reports the update and writes nothing.
	other := filepath.Join(dir, "entries", "animated-fill-bar.md")
	raw, err = os.ReadFile(other)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if err := os.WriteFile(other, append(raw, []byte("\nA dry-run-only sentence.\n")...), 0o644); err != nil {
		t.Fatalf("write entry: %v", err)
	}
	dry := opt
	dry.DryRun = true
	rep, err = Sync(ctx, st, mustScan(t, dir), dry)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if rep.Updated != 1 || rep.Unchanged != 3 || rep.Created != 0 || rep.Deleted != 0 {
		t.Fatalf("dry run report = %+v, want 1 updated", rep)
	}
	rep, err = Sync(ctx, st, mustScan(t, dir), opt)
	if err != nil {
		t.Fatalf("sync after dry run: %v", err)
	}
	if rep.Updated != 1 {
		t.Fatalf("dry run wrote to the database: %+v", rep)
	}

	// Removing a file soft-deletes its row.
	if err := os.Remove(filepath.Join(dir, "entries", "legacy-svn-audit.md")); err != nil {
		t.Fatalf("remove entry: %v", err)
	}
	rep, err = Sync(ctx, st, mustScan(t, dir), opt)
	if err != nil {
		t.Fatalf("fourth sync: %v", err)
	}
	if rep.Deleted != 1 || rep.Created != 0 || rep.Updated != 0 || rep.Unchanged != 3 {
		t.Fatalf("after delete report = %+v, want 1 deleted", rep)
	}

	rows, err := st.ListByTopicPrefix(ctx, brainIDOf(t, st), TopicPrefix)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d active rows, want 3", len(rows))
	}
	for _, m := range rows {
		if m.TopicKey == "knowledge/legacy-svn-audit" {
			t.Fatal("deleted entry is still active")
		}
	}
}

func TestSync_NoPruneKeepsOrphans(t *testing.T) {
	dir := copyCorpus(t, "testdata/corpus")
	st := newSyncStorage(t)
	ctx := context.Background()

	if _, err := Sync(ctx, st, mustScan(t, dir), SyncOptions{Prune: true}); err != nil {
		t.Fatalf("seed sync: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, "entries", "camera-shake.md")); err != nil {
		t.Fatalf("remove entry: %v", err)
	}

	rep, err := Sync(ctx, st, mustScan(t, dir), SyncOptions{Prune: false})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if rep.Deleted != 0 {
		t.Fatalf("deleted = %d with pruning off", rep.Deleted)
	}
	rows, err := st.ListByTopicPrefix(ctx, brainIDOf(t, st), TopicPrefix)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want the orphan kept", len(rows))
	}
}

func TestSync_DryRunDoesNotCreateTheBrain(t *testing.T) {
	st := newSyncStorage(t)
	ctx := context.Background()

	rep, err := Sync(ctx, st, mustScan(t, "testdata/corpus"), SyncOptions{DryRun: true, Prune: true})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if rep.Created != 4 {
		t.Fatalf("dry run on an empty database should report 4 created: %+v", rep)
	}
	if _, err := st.ResolveBrainID(ctx, DefaultSyncProject); err == nil {
		t.Fatal("dry run created the project brain")
	}
}

func TestSync_EmptyCorpusRefusesToPrune(t *testing.T) {
	st := newSyncStorage(t)
	ctx := context.Background()

	if _, err := Sync(ctx, st, mustScan(t, "testdata/corpus"), SyncOptions{Prune: true}); err != nil {
		t.Fatalf("seed sync: %v", err)
	}

	empty := t.TempDir()
	rep, err := Sync(ctx, st, Corpus{Dir: empty}, SyncOptions{Prune: true})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if rep.Deleted != 0 {
		t.Fatalf("an empty scan pruned %d rows", rep.Deleted)
	}
	if len(rep.Errors) == 0 {
		t.Fatal("expected a reported refusal")
	}
}

func TestSync_SkipsInvalidEntryAndReportsIt(t *testing.T) {
	st := newSyncStorage(t)
	ctx := context.Background()
	c := Corpus{Dir: "corpus", Entries: []Entry{
		sampleEntry(),
		{Slug: "no-body", Title: "Empty body", Template: TemplateOther},
	}}

	rep, err := Sync(ctx, st, c, SyncOptions{})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if rep.Created != 1 || rep.Skipped != 1 || rep.Scanned != 2 {
		t.Fatalf("report = %+v, want 1 created and 1 skipped", rep)
	}
	if len(rep.Errors) != 1 || !strings.Contains(rep.Errors[0], "no-body") {
		t.Fatalf("errors = %v", rep.Errors)
	}
}

// brainIDOf resolves the default sync brain for assertions that need to query
// storage directly.
func brainIDOf(t *testing.T, st *storage.Storage) int64 {
	t.Helper()
	id, err := st.ResolveBrainID(context.Background(), DefaultSyncProject)
	if err != nil {
		t.Fatalf("resolve brain: %v", err)
	}
	return id
}
