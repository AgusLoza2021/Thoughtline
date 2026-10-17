package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
)

// Template classifies which body shape an entry uses. The corpora mix two
// hand-written templates and a few one-off layouts, and callers surface
// different sections depending on which one they are looking at ("Applies when"
// for procedural entries, "Example" for recipes).
type Template string

const (
	TemplateProcedural Template = "procedural" // has "Applies when" or "Procedure"
	TemplateRecipe     Template = "recipe"     // has "Context" or "Example"
	TemplateOther      Template = "other"
)

// ErrEmptyEntry is returned for a file that holds no content at all. Callers
// scanning a directory should skip these rather than abort.
var ErrEmptyEntry = errors.New("knowledge: empty entry")

// Section is one "## Heading" block of an entry. Heading keeps the author's
// casing because it is shown to humans; use Entry.Section for lookups.
type Section struct {
	Heading string
	Body    string
}

// Entry is one parsed recipe file.
type Entry struct {
	Slug     string    // file base name without .md, lowercased
	Path     string    // path as given to ParseEntry
	Title    string    // frontmatter title, else first "# " heading, else Slug
	Tags     []string  // frontmatter tags, normalized and deduped
	Source   string    // frontmatter source
	Group    string    // topic group from the index file; "" if not indexed
	Template Template  // body shape
	Sections []Section // in file order
	Stacks   []string  // foreign-stack markers detected in title+body
	Body     string    // full body after frontmatter, BOM removed, LF endings
	Hash     string    // hex sha256 of the raw file bytes
}

// Section returns the body of the named section, matching the heading
// case-insensitively, or "" when the entry has no such section.
func (e Entry) Section(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for _, s := range e.Sections {
		if strings.EqualFold(strings.TrimSpace(s.Heading), name) {
			return s.Body
		}
	}
	return ""
}

// ParseEntry parses one corpus file. Only empty input is an error: everything
// else (missing frontmatter, unquoted titles, unknown headings) degrades to a
// best-effort Entry, because a single sloppy file must not poison a scan.
func ParseEntry(path string, raw []byte) (Entry, error) {
	text := normalizeText(string(raw))
	if strings.TrimSpace(text) == "" {
		return Entry{}, ErrEmptyEntry
	}

	sum := sha256.Sum256(raw)
	e := Entry{
		Slug: slugOf(path),
		Path: path,
		Hash: hex.EncodeToString(sum[:]),
	}

	meta, body := splitFrontmatter(text)
	e.Body = body
	e.Title = unquote(meta["title"])
	e.Source = unquote(meta["source"])
	e.Tags = normalizeTags(meta["tags"])
	if e.Title == "" {
		e.Title = firstHeading(body)
	}
	if e.Title == "" {
		e.Title = e.Slug
	}
	e.Sections = parseSections(body)
	e.Template = classify(e.Sections)
	// Stacks can hide in the title only (e.g. "SVN externals audit"), so the
	// title is part of the detection text.
	e.Stacks = DetectStacks(e.Title + "\n" + body)
	return e, nil
}

// normalizeText drops a UTF-8 BOM and folds CRLF/CR to LF so sections, hashes
// and stack detection do not depend on how a file happens to be saved.
func normalizeText(s string) string {
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// splitFrontmatter separates an optional leading "---" block from the body.
// The block is parsed as flat key: value pairs: the corpora use exactly
// title/tags/source, all scalars, so a YAML dependency buys nothing here.
func splitFrontmatter(text string) (map[string]string, string) {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, text
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "---" {
			continue
		}
		return parseMeta(lines[1:i]), strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
	}
	// Unbalanced fence: treat the whole file as body so a half-written entry
	// still contributes its text instead of disappearing.
	return nil, text
}

func parseMeta(lines []string) map[string]string {
	meta := make(map[string]string, len(lines))
	for _, line := range lines {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(key))
		if k == "" {
			continue
		}
		meta[k] = strings.TrimSpace(value)
	}
	return meta
}

// parseSections splits the body on "## " headings. "### " sub-headings and
// prose before the first heading stay with their nearest parent section; the
// latter is dropped because it is not addressable by name anyway.
func parseSections(body string) []Section {
	var (
		sections []Section
		current  *Section
		buf      []string
	)
	flush := func() {
		if current == nil {
			return
		}
		current.Body = strings.TrimSpace(strings.Join(buf, "\n"))
		sections = append(sections, *current)
		buf = nil
	}
	for _, line := range strings.Split(body, "\n") {
		if heading, ok := sectionHeading(line); ok {
			flush()
			current = &Section{Heading: heading}
			continue
		}
		if current != nil {
			buf = append(buf, line)
		}
	}
	flush()
	return sections
}

func sectionHeading(line string) (string, bool) {
	if !strings.HasPrefix(line, "## ") {
		return "", false
	}
	heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
	return heading, heading != ""
}

func classify(sections []Section) Template {
	has := make(map[string]bool, len(sections))
	for _, s := range sections {
		has[strings.ToLower(strings.TrimSpace(s.Heading))] = true
	}
	// Procedural wins when both shapes appear: an entry that says how to apply
	// itself needs its "Applies when" gate honoured before its example.
	if has["applies when"] || has["procedure"] {
		return TemplateProcedural
	}
	if has["context"] || has["example"] {
		return TemplateRecipe
	}
	return TemplateOther
}

func firstHeading(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return ""
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return strings.TrimSpace(s[1 : len(s)-1])
		}
	}
	return s
}

// Slug is the stable identity of an entry inside a corpus, so it is derived
// from the filename and lowercased to survive case-insensitive filesystems.
func slugOf(path string) string {
	base := filepath.Base(path)
	if strings.EqualFold(filepath.Ext(base), ".md") {
		base = base[:len(base)-len(filepath.Ext(base))]
	}
	return strings.ToLower(strings.TrimSpace(base))
}

// normalizeTags turns a frontmatter list into tags that later feed memory
// validation (^[a-z0-9][a-z0-9:_-]{0,40}$): lowercase, spaces to hyphens,
// everything else dropped, order preserved, duplicates removed.
func normalizeTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]")

	tags := make([]string, 0, 4)
	seen := make(map[string]bool, 4)
	for _, part := range strings.Split(raw, ",") {
		tag := normalizeTag(part)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

func normalizeTag(part string) string {
	part = strings.Trim(unquote(part), `"'`)
	var b strings.Builder
	b.Grow(len(part))
	for _, r := range strings.ToLower(part) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == ':', r == '_', r == '-':
			b.WriteRune(r)
		case r == ' ' || r == '\t':
			b.WriteRune('-')
		}
	}
	return b.String()
}
