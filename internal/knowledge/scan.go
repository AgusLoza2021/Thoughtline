package knowledge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Corpus is a scanned directory of recipes. Entries are sorted by Slug so
// callers (and tests) get a deterministic order without re-sorting.
type Corpus struct {
	Dir     string
	Entries []Entry
}

// Group returns the entries of one topic group, preserving corpus order.
func (c Corpus) Group(name string) []Entry {
	var out []Entry
	for _, e := range c.Entries {
		if strings.EqualFold(e.Group, name) {
			out = append(out, e)
		}
	}
	return out
}

var (
	// filenames that describe the corpus itself rather than one recipe.
	indexFileNames = map[string]bool{"skill.md": true, "readme.md": true, "index.md": true}
	groupHeadingRe = regexp.MustCompile(`^###\s+(.+)$`)
	groupCountRe   = regexp.MustCompile(`\s*\(\d+\)\s*$`)
	indexLinkRe    = regexp.MustCompile(`\]\(([^)\s]+\.md)\)`)
)

// Scan reads a corpus directory. Recipe files live in <dir>/entries when that
// folder exists, otherwise directly in <dir>; the index file, when present,
// supplies each entry's topic Group.
func Scan(dir string) (Corpus, error) {
	corpus := Corpus{Dir: dir}

	listDir := dir
	if info, err := os.Stat(filepath.Join(dir, "entries")); err == nil && info.IsDir() {
		listDir = filepath.Join(dir, "entries")
	}

	dirents, err := os.ReadDir(listDir)
	if err != nil {
		return Corpus{}, fmt.Errorf("knowledge: read %s: %w", listDir, err)
	}

	for _, de := range dirents {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.EqualFold(filepath.Ext(name), ".md") || indexFileNames[strings.ToLower(name)] {
			continue
		}
		path := filepath.Join(listDir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			return Corpus{}, fmt.Errorf("knowledge: read %s: %w", path, err)
		}
		entry, err := ParseEntry(path, raw)
		if errors.Is(err, ErrEmptyEntry) {
			// An empty file is an authoring accident, not a reason to lose the
			// other 175 recipes.
			continue
		}
		if err != nil {
			return Corpus{}, fmt.Errorf("knowledge: parse %s: %w", path, err)
		}
		corpus.Entries = append(corpus.Entries, entry)
	}

	if groups := readGroups(dir); len(groups) > 0 {
		for i := range corpus.Entries {
			corpus.Entries[i].Group = groups[corpus.Entries[i].Slug]
		}
	}

	sort.Slice(corpus.Entries, func(i, j int) bool {
		if corpus.Entries[i].Slug != corpus.Entries[j].Slug {
			return corpus.Entries[i].Slug < corpus.Entries[j].Slug
		}
		return corpus.Entries[i].Path < corpus.Entries[j].Path
	})
	return corpus, nil
}

// readGroups loads the index file if the corpus has one. A missing index is
// normal (flat corpora); an unreadable one just means ungrouped entries.
func readGroups(dir string) map[string]string {
	for _, name := range []string{"SKILL.md", "INDEX.md"} {
		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if groups := ParseGroups(normalizeText(string(raw))); len(groups) > 0 {
			return groups
		}
	}
	return nil
}

// ParseGroups maps entry slug to topic group from an index file such as
// SKILL.md, where every "### Group (12)" heading is followed by bullets like
// "- [Title](entries/slug.md) — first sentence".
func ParseGroups(indexMarkdown string) map[string]string {
	groups := make(map[string]string)
	group := ""
	for _, line := range strings.Split(normalizeText(indexMarkdown), "\n") {
		line = strings.TrimSpace(line)
		if m := groupHeadingRe.FindStringSubmatch(line); m != nil {
			group = strings.TrimSpace(groupCountRe.ReplaceAllString(m[1], ""))
			continue
		}
		if group == "" {
			continue
		}
		m := indexLinkRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		slug := slugOf(m[1])
		if slug == "" {
			continue
		}
		if _, ok := groups[slug]; !ok {
			groups[slug] = group
		}
	}
	return groups
}
