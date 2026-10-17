package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGroups(t *testing.T) {
	index := `# Index

## How to use

- [Not a group](entries/ignored-before-groups.md) — preamble bullet.

### Camera and feel (12)

- [Camera Shake as an Additive Local Position Offset](entries/camera-shake.md) — Add an impact shake.
- [Animated fill bar](entries/animated-fill-bar.md) — Stretch by width.

### Agent/tool workflow notes (not Unity code — read with extra care) (23)

- [SVN externals audit](entries/legacy-svn-audit.md) — Audit externals.
`
	groups := ParseGroups(index)

	want := map[string]string{
		"camera-shake":      "Camera and feel",
		"animated-fill-bar": "Camera and feel",
		"legacy-svn-audit":  "Agent/tool workflow notes (not Unity code — read with extra care)",
	}
	if len(groups) != len(want) {
		t.Fatalf("groups = %v, want %d entries", groups, len(want))
	}
	for slug, group := range want {
		if groups[slug] != group {
			t.Fatalf("group[%q] = %q, want %q", slug, groups[slug], group)
		}
	}
	if _, ok := groups["ignored-before-groups"]; ok {
		t.Fatal("bullets outside a group heading must be ignored")
	}
}

func TestParseGroups_CRLFAndBOM(t *testing.T) {
	index := "\ufeff### Camera (2)\r\n\r\n- [Shake](entries/camera-shake.md) — x\r\n"
	groups := ParseGroups(index)
	if groups["camera-shake"] != "Camera" {
		t.Fatalf("groups = %v, want camera-shake -> Camera", groups)
	}
}

func TestScan_Testdata(t *testing.T) {
	corpus, err := Scan(filepath.Join("testdata", "corpus"))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	wantSlugs := []string{"animated-fill-bar", "camera-shake", "legacy-svn-audit", "no-frontmatter"}
	if len(corpus.Entries) != len(wantSlugs) {
		t.Fatalf("entries = %d, want %d: %+v", len(corpus.Entries), len(wantSlugs), corpus.Entries)
	}
	for i, slug := range wantSlugs {
		if corpus.Entries[i].Slug != slug {
			t.Fatalf("entry[%d] = %q, want %q (sorted by slug)", i, corpus.Entries[i].Slug, slug)
		}
	}

	bySlug := make(map[string]Entry, len(corpus.Entries))
	for _, e := range corpus.Entries {
		bySlug[e.Slug] = e
	}
	if _, ok := bySlug["skill"]; ok {
		t.Fatal("SKILL.md must not be scanned as an entry")
	}

	cases := []struct {
		slug     string
		group    string
		template Template
		stacks   string
	}{
		{"camera-shake", "Camera and feel", TemplateProcedural, ""},
		{"animated-fill-bar", "Camera and feel", TemplateRecipe, ""},
		{"legacy-svn-audit", "Build and version control", TemplateRecipe, "fiber,svn"},
		{"no-frontmatter", "", TemplateOther, "belt"},
	}
	for _, tc := range cases {
		e, ok := bySlug[tc.slug]
		if !ok {
			t.Fatalf("%s missing from corpus", tc.slug)
		}
		if e.Group != tc.group {
			t.Fatalf("%s group = %q, want %q", tc.slug, e.Group, tc.group)
		}
		if e.Template != tc.template {
			t.Fatalf("%s template = %q, want %q", tc.slug, e.Template, tc.template)
		}
		if got := strings.Join(e.Stacks, ","); got != tc.stacks {
			t.Fatalf("%s stacks = %q, want %q", tc.slug, got, tc.stacks)
		}
	}

	// The BOM+CRLF fixture must come back clean.
	fill := bySlug["animated-fill-bar"]
	if fill.Title != "Animated fill bar without fillAmount" {
		t.Fatalf("BOM fixture title = %q", fill.Title)
	}
	if fill.Section("notes") == "" {
		t.Fatal("BOM fixture lost its Notes section")
	}
}

func TestScan_FlatDirectorySkipsIndexFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("SKILL.md", "### Camera (1)\n\n- [Shake](entries/shake.md) — x\n")
	write("README.md", "# readme\n")
	write("INDEX.md", "# index\n")
	write("shake.md", "---\ntitle: Shake\n---\n\n## Guidance\n\nText.\n")

	corpus, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(corpus.Entries) != 1 || corpus.Entries[0].Slug != "shake" {
		t.Fatalf("entries = %+v, want just shake", corpus.Entries)
	}
	if corpus.Entries[0].Group != "Camera" {
		t.Fatalf("group = %q, want Camera", corpus.Entries[0].Group)
	}
}

func TestScan_MissingDirectory(t *testing.T) {
	if _, err := Scan(filepath.Join("testdata", "does-not-exist")); err == nil {
		t.Fatal("expected an error for a missing corpus directory")
	}
}

// TestScan_RealCorpus is opt-in: the real corpora live outside the repo, so CI
// must not depend on them. Run it with
//
//	THOUGHTLINE_KNOWLEDGE_CORPUS=<dir> go test ./internal/knowledge/ -run Real -v
func TestScan_RealCorpus(t *testing.T) {
	dir := os.Getenv("THOUGHTLINE_KNOWLEDGE_CORPUS")
	if dir == "" {
		t.Skip("THOUGHTLINE_KNOWLEDGE_CORPUS not set")
	}

	corpus, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan(%s): %v", dir, err)
	}
	if len(corpus.Entries) == 0 {
		t.Fatalf("Scan(%s) found no entries", dir)
	}

	templates := map[Template]int{}
	stacks := map[string]int{}
	grouped := 0
	for _, e := range corpus.Entries {
		templates[e.Template]++
		for _, s := range e.Stacks {
			stacks[s]++
		}
		if e.Group != "" {
			grouped++
		}
		if e.Title == "" {
			t.Errorf("%s has an empty title", e.Path)
		}
	}

	t.Logf("entries=%d grouped=%d templates: procedural=%d recipe=%d other=%d",
		len(corpus.Entries), grouped,
		templates[TemplateProcedural], templates[TemplateRecipe], templates[TemplateOther])
	for id, n := range stacks {
		t.Logf("stack %s: %d entries", id, n)
	}
}
