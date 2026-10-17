package knowledge

import (
	"errors"
	"strings"
	"testing"
)

const proceduralFixture = `---
title: "Camera Shake as an Additive Local Position Offset"
tags: [Camera, camera, Camera Shake, Feel]
source: external-knowledge-2026-09
---

# Camera Shake

## Guidance

Keep the shake additive.

## Applies when

A hit needs an impact kick.

## Procedure

1. Sample the curve.

## Limits

Not a hit-stop replacement.

## Notes from the field

Extra heading that belongs to neither template.
`

const recipeFixture = `---
title: Animated fill bar
tags: [ui, fill]
source: external-knowledge-2026-09
---

## Context

fillAmount snaps at the ends.

## Example

Drive the RectTransform width instead.

## Notes

Clamp to the track width.
`

func mustParse(t *testing.T, path, text string) Entry {
	t.Helper()
	e, err := ParseEntry(path, []byte(text))
	if err != nil {
		t.Fatalf("ParseEntry(%s): unexpected error %v", path, err)
	}
	return e
}

func TestParseEntry_ProceduralTemplate(t *testing.T) {
	e := mustParse(t, "testdata/camera-shake.md", proceduralFixture)

	if e.Slug != "camera-shake" {
		t.Fatalf("slug = %q, want camera-shake", e.Slug)
	}
	if want := "Camera Shake as an Additive Local Position Offset"; e.Title != want {
		t.Fatalf("title = %q, want %q", e.Title, want)
	}
	if e.Source != "external-knowledge-2026-09" {
		t.Fatalf("source = %q", e.Source)
	}
	if e.Template != TemplateProcedural {
		t.Fatalf("template = %q, want %q", e.Template, TemplateProcedural)
	}
	// Guidance, Applies when, Procedure, Limits, Notes from the field — the
	// level-one title is not a section.
	if len(e.Sections) != 5 {
		t.Fatalf("sections = %d, want 5: %+v", len(e.Sections), e.Sections)
	}
	if want := "Keep the shake additive."; e.Section("Guidance") != want {
		t.Fatalf("Guidance = %q, want %q", e.Section("Guidance"), want)
	}
	if e.Section("applies WHEN") != "A hit needs an impact kick." {
		t.Fatalf("case-insensitive lookup failed: %q", e.Section("applies WHEN"))
	}
	if e.Section("Missing") != "" {
		t.Fatalf("absent section should be empty, got %q", e.Section("Missing"))
	}
	if got := e.Section("Notes from the field"); got != "Extra heading that belongs to neither template." {
		t.Fatalf("extra heading not kept, got %q", got)
	}
	if strings.Contains(e.Body, "title:") {
		t.Fatalf("body still contains frontmatter: %q", e.Body)
	}
	if len(e.Hash) != 64 {
		t.Fatalf("hash = %q, want 64 hex chars", e.Hash)
	}
}

func TestParseEntry_RecipeTemplate(t *testing.T) {
	e := mustParse(t, "animated-fill-bar.md", recipeFixture)

	if e.Template != TemplateRecipe {
		t.Fatalf("template = %q, want %q", e.Template, TemplateRecipe)
	}
	if want := "Drive the RectTransform width instead."; e.Section("Example") != want {
		t.Fatalf("Example = %q, want %q", e.Section("Example"), want)
	}
	if e.Title != "Animated fill bar" {
		t.Fatalf("unquoted title = %q", e.Title)
	}
}

func TestParseEntry_OtherTemplate(t *testing.T) {
	e := mustParse(t, "odd.md", "---\ntitle: Odd\n---\n\n## Whatever\n\nText.\n")
	if e.Template != TemplateOther {
		t.Fatalf("template = %q, want %q", e.Template, TemplateOther)
	}
}

func TestParseEntry_TagsNormalized(t *testing.T) {
	e := mustParse(t, "tags.md", proceduralFixture)

	want := []string{"camera", "camera-shake", "feel"}
	if len(e.Tags) != len(want) {
		t.Fatalf("tags = %v, want %v", e.Tags, want)
	}
	for i := range want {
		if e.Tags[i] != want[i] {
			t.Fatalf("tags = %v, want %v", e.Tags, want)
		}
	}

	t.Run("invalid chars dropped and empties skipped", func(t *testing.T) {
		got := normalizeTags(`[UI/Canvas, "3D!", , grid_board, ok:tag, A-B]`)
		want := []string{"uicanvas", "3d", "grid_board", "ok:tag", "a-b"}
		if len(got) != len(want) {
			t.Fatalf("tags = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("tags = %v, want %v", got, want)
			}
		}
	})

	t.Run("empty list yields nil", func(t *testing.T) {
		if got := normalizeTags("[]"); got != nil {
			t.Fatalf("tags = %v, want nil", got)
		}
	})
}

func TestParseEntry_ToleratesBOMAndCRLF(t *testing.T) {
	crlf := "\ufeff" + strings.ReplaceAll(recipeFixture, "\n", "\r\n")
	e := mustParse(t, "animated-fill-bar.md", crlf)

	if e.Slug != "animated-fill-bar" {
		t.Fatalf("slug = %q", e.Slug)
	}
	if e.Title != "Animated fill bar" {
		t.Fatalf("title = %q (BOM leaked into frontmatter?)", e.Title)
	}
	if e.Template != TemplateRecipe {
		t.Fatalf("template = %q, want %q", e.Template, TemplateRecipe)
	}
	if strings.Contains(e.Body, "\r") {
		t.Fatalf("body still has CR: %q", e.Body)
	}
	if got := e.Section("Notes"); got != "Clamp to the track width." {
		t.Fatalf("Notes = %q", got)
	}
	if e.Hash == mustParse(t, "animated-fill-bar.md", recipeFixture).Hash {
		t.Fatal("hash must be over raw bytes, so LF and CRLF differ")
	}
}

func TestParseEntry_MissingFrontmatter(t *testing.T) {
	t.Run("falls back to first h1", func(t *testing.T) {
		e := mustParse(t, "no-frontmatter.md", "# Legacy note\n\n## Notes\n\nText.\n")
		if e.Title != "Legacy note" {
			t.Fatalf("title = %q, want Legacy note", e.Title)
		}
		if e.Source != "" || e.Tags != nil {
			t.Fatalf("expected no metadata, got source=%q tags=%v", e.Source, e.Tags)
		}
		if e.Section("Notes") != "Text." {
			t.Fatalf("Notes = %q", e.Section("Notes"))
		}
	})

	t.Run("falls back to slug", func(t *testing.T) {
		e := mustParse(t, "Bare-Name.md", "Just prose, no headings.\n")
		if e.Title != "bare-name" {
			t.Fatalf("title = %q, want bare-name", e.Title)
		}
	})
}

func TestParseEntry_EmptyInput(t *testing.T) {
	for _, raw := range [][]byte{nil, {}, []byte("   \n\t\n")} {
		if _, err := ParseEntry("empty.md", raw); !errors.Is(err, ErrEmptyEntry) {
			t.Fatalf("ParseEntry(%q) err = %v, want ErrEmptyEntry", raw, err)
		}
	}
}

func TestParseEntry_HashStable(t *testing.T) {
	first := mustParse(t, "camera-shake.md", proceduralFixture)
	second := mustParse(t, "camera-shake.md", proceduralFixture)
	if first.Hash != second.Hash {
		t.Fatalf("hash not stable: %q vs %q", first.Hash, second.Hash)
	}
	edited := proceduralFixture + "\nextra line\n"
	if mustParse(t, "camera-shake.md", edited).Hash == first.Hash {
		t.Fatal("hash must change when the file changes")
	}
}

func TestParseEntry_DetectsStacks(t *testing.T) {
	e := mustParse(t, "legacy-svn-audit.md", "---\ntitle: SVN externals audit\n---\n\n## Context\n\nUses FiberCtrl.\n")
	if len(e.Stacks) != 2 || e.Stacks[0] != "fiber" || e.Stacks[1] != "svn" {
		t.Fatalf("stacks = %v, want [fiber svn]", e.Stacks)
	}
}
