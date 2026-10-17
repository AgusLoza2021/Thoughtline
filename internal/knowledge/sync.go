package knowledge

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// TopicPrefix is the namespace every corpus entry lives under. Prune and list
// both work on this prefix, so it is exported rather than repeated as a literal.
const TopicPrefix = "knowledge/"

// Defaults for the sync destination. A separate project keeps the recipes out
// of the owner's real project brains (search noise) while staying queryable.
const (
	DefaultSyncProject = "knowledge-unity"
	DefaultSyncCorpus  = "unity"
)

const (
	// maxTags mirrors what tl_check can render without drowning a prompt.
	maxTags = 20
	// topicKeySlugBudget is the longest slug that still fits the topic_key
	// regex (^[a-z0-9][a-z0-9/_-]{1,128}$) after TopicPrefix and the 9-char
	// "-<hash8>" suffix: 10 + 110 + 9 = 129.
	topicKeySlugBudget = 110
	// footerHashChars is enough of a sha256 to tell revisions apart by eye.
	footerHashChars = 12
)

// tagRe mirrors memory's (unexported) tag pattern so a frontmatter tag that
// would fail memory.Validate can be dropped here instead of failing the entry.
var tagRe = regexp.MustCompile(`^[a-z0-9][a-z0-9:_-]{0,40}$`)

// SyncOptions configures an import run.
type SyncOptions struct {
	Project string // brain slug to write to; defaults to DefaultSyncProject
	Corpus  string // corpus id used in tags and the footer; defaults to DefaultSyncCorpus
	DryRun  bool   // compute the report without writing anything
	Prune   bool   // soft-delete knowledge rows whose file is gone
}

// SyncReport is what a run did (or, under DryRun, what it would do).
type SyncReport struct {
	Scanned   int
	Created   int
	Updated   int
	Unchanged int
	Deleted   int
	Skipped   int
	Errors    []string
}

// Sync imports a scanned corpus into storage, keyed by topic_key so re-running
// is idempotent. A single unusable entry is recorded in the report and skipped:
// one sloppy recipe must not block the other 175.
func Sync(ctx context.Context, st *storage.Storage, c Corpus, opt SyncOptions) (SyncReport, error) {
	var rep SyncReport

	opt.Project = orDefault(strings.TrimSpace(opt.Project), DefaultSyncProject)
	opt.Corpus = orDefault(strings.TrimSpace(opt.Corpus), DefaultSyncCorpus)
	rep.Scanned = len(c.Entries)

	// Plan the whole corpus before touching the database: it lets DryRun share
	// one code path, and it means a duplicate-slug mistake is reported before
	// any half-migrated state exists.
	plans := make([]memory.Memory, 0, len(c.Entries))
	produced := make(map[string]bool, len(c.Entries))
	for _, e := range c.Entries {
		m := ToMemory(e, opt)
		if err := memory.Validate(m); err != nil {
			rep.Errors = append(rep.Errors, fmt.Sprintf("%s: %v", relPath(c, e), err))
			rep.Skipped++
			continue
		}
		if produced[m.TopicKey] {
			// Two files whose names collapse to the same slug would fight over
			// one row; keep the first and surface the collision.
			rep.Errors = append(rep.Errors, fmt.Sprintf("%s: duplicate topic_key %s", relPath(c, e), m.TopicKey))
			rep.Skipped++
			continue
		}
		produced[m.TopicKey] = true
		plans = append(plans, m)
	}

	brainID, err := syncBrainID(ctx, st, opt)
	if err != nil {
		return rep, err
	}

	// One listing serves both DryRun classification and prune: it is the only
	// way to know which rows this importer owns.
	existing := map[string]memory.Memory{}
	if brainID != 0 {
		rows, err := st.ListByTopicPrefix(ctx, brainID, TopicPrefix)
		if err != nil {
			return rep, fmt.Errorf("knowledge: list existing entries: %w", err)
		}
		for _, m := range rows {
			existing[m.TopicKey] = m
		}
	}

	if opt.DryRun {
		classifyDryRun(&rep, plans, existing)
		if opt.Prune && len(c.Entries) > 0 {
			for key := range existing {
				if !produced[key] {
					rep.Deleted++
				}
			}
		}
		return rep, nil
	}

	for _, m := range plans {
		_, action, err := st.Save(ctx, brainID, m)
		if err != nil {
			rep.Errors = append(rep.Errors, fmt.Sprintf("%s: %v", m.TopicKey, err))
			rep.Skipped++
			continue
		}
		switch action {
		case storage.ActionCreated:
			rep.Created++
		case storage.ActionUpdated:
			rep.Updated++
		default:
			rep.Unchanged++
		}
	}

	if opt.Prune {
		if len(c.Entries) == 0 {
			// An empty scan is almost always a mistyped path, not a deleted
			// corpus. Refuse to wipe the namespace on that signal alone.
			rep.Errors = append(rep.Errors, "refusing to prune: scanned corpus is empty")
			return rep, nil
		}
		for key, m := range existing {
			if produced[key] {
				continue
			}
			if err := st.SoftDelete(ctx, brainID, m.ID); err != nil {
				rep.Errors = append(rep.Errors, fmt.Sprintf("%s: soft delete: %v", key, err))
				continue
			}
			rep.Deleted++
		}
	}
	return rep, nil
}

// ToMemory maps one parsed entry onto the memory shape. It is pure so the
// mapping can be tested without a database.
func ToMemory(e Entry, opt SyncOptions) memory.Memory {
	corpus := orDefault(strings.TrimSpace(opt.Corpus), DefaultSyncCorpus)
	return memory.Memory{
		Project:  orDefault(strings.TrimSpace(opt.Project), DefaultSyncProject),
		Scope:    memory.ScopeProject,
		Type:     typeFor(e.Template),
		TopicKey: TopicKey(e.Slug),
		Title:    truncateRunes(strings.TrimSpace(e.Title), memory.MaxTitleChars),
		Content:  contentFor(e),
		Tags:     tagsFor(e, corpus),
	}
}

// TopicKey builds the stable identity of an entry. Over-long slugs (or ones
// carrying characters the topic_key regex rejects) are sanitized and, when they
// still do not fit, shortened with a hash suffix so distinct files cannot
// collapse onto one row.
func TopicKey(slug string) string {
	s := sanitizeSlug(slug)
	if s == "" {
		s = "entry"
	}
	if len(s) > topicKeySlugBudget {
		sum := sha256.Sum256([]byte(s))
		s = strings.TrimRight(s[:topicKeySlugBudget], "-") + "-" + hex.EncodeToString(sum[:4])
	}
	return TopicPrefix + s
}

// typeFor maps a template onto the closed memory.Type set. The domain has no
// "pattern"/"discovery" type, so the precise template travels in the
// `template:` tag and the type only distinguishes a procedure from a code
// sample.
func typeFor(t Template) memory.Type {
	switch t {
	case TemplateRecipe:
		return memory.TypeScriptPattern
	default:
		return memory.TypeConvention
	}
}

// contentFor builds the stored body plus its provenance footer.
//
// The footer carries no words on purpose: FTS indexes it as content, and it
// used to repeat the topic group ("group: Camera and feel") and the source, so
// every entry of a group became a lexical match for any task naming the group.
// Group, source and slug already travel in the tags and the topic_key; the hash
// stays here because it is what makes a body change detectable on re-sync.
func contentFor(e Entry) string {
	body := strings.TrimRight(normalizeText(e.Body), "\n")
	footer := fmt.Sprintf("---\n_kh:%s_", truncateBytes(e.Hash, footerHashChars))
	if body == "" {
		// A file with frontmatter and no prose is an authoring accident: leave
		// the content empty so Validate reports it and Sync skips the entry.
		return ""
	}
	return body + "\n\n" + footer
}

func tagsFor(e Entry, corpus string) []string {
	candidates := make([]string, 0, 6+len(e.Stacks)+len(e.Tags))
	candidates = append(candidates,
		"kind:knowledge",
		"corpus:"+corpus,
		"template:"+templateTag(e.Template),
	)
	if group := strings.TrimSpace(e.Group); group != "" {
		candidates = append(candidates, fitTag("group:", slugify(group)))
	}
	for _, stack := range e.Stacks {
		candidates = append(candidates, "stack:"+stack)
	}
	candidates = append(candidates, e.Tags...)

	tags := make([]string, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, tag := range candidates {
		if !tagRe.MatchString(tag) || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
		if len(tags) == maxTags {
			break
		}
	}
	return tags
}

func templateTag(t Template) string {
	switch t {
	case TemplateProcedural:
		return "procedural"
	case TemplateRecipe:
		return "recipe"
	default:
		return "other"
	}
}

// fitTag appends suffix to prefix and keeps the result inside the tag length
// limit, trimming separator hyphens so the tag stays valid.
func fitTag(prefix, suffix string) string {
	budget := 1 + memory.MaxTagLen - len(prefix)
	if len(suffix) > budget {
		suffix = strings.TrimRight(suffix[:budget], "-")
	}
	return prefix + suffix
}

// slugify reduces a topic group to tag material: lowercase, spaces to hyphens,
// everything the tag regex rejects dropped.
func slugify(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	lastHyphen := true // leading hyphens would make the tag invalid
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case r == ' ' || r == '\t' || r == '_':
			if !lastHyphen {
				b.WriteRune('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// sanitizeSlug keeps only characters the topic_key regex allows, so a filename
// such as "v1.2 notes.md" still maps to a storable key.
func sanitizeSlug(slug string) string {
	var b strings.Builder
	b.Grow(len(slug))
	for _, r := range strings.ToLower(strings.TrimSpace(slug)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func classifyDryRun(rep *SyncReport, plans []memory.Memory, existing map[string]memory.Memory) {
	for _, m := range plans {
		prev, ok := existing[m.TopicKey]
		switch {
		case !ok:
			rep.Created++
		case prev.NormalizedHash == storage.NormalizedHash(m.Title, m.Content):
			rep.Unchanged++
		default:
			rep.Updated++
		}
	}
}

// syncBrainID resolves the destination brain without creating one during a dry
// run, so `--dry-run` on a fresh database leaves it untouched.
func syncBrainID(ctx context.Context, st *storage.Storage, opt SyncOptions) (int64, error) {
	if !opt.DryRun {
		id, err := st.ResolveOrCreateBrainID(ctx, opt.Project)
		if err != nil {
			return 0, fmt.Errorf("knowledge: resolve project %q: %w", opt.Project, err)
		}
		return id, nil
	}
	id, err := st.ResolveBrainID(ctx, opt.Project)
	if errors.Is(err, sql.ErrNoRows) {
		// No brain means no rows: the dry run reports everything as created.
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("knowledge: resolve project %q: %w", opt.Project, err)
	}
	return id, nil
}

// relPath keeps report lines readable when the corpus was scanned by absolute
// path: the corpus-relative name alone identifies the entry.
func relPath(c Corpus, e Entry) string {
	if e.Path == "" {
		return e.Slug
	}
	if c.Dir != "" {
		if rel, err := filepath.Rel(c.Dir, e.Path); err == nil {
			return rel
		}
	}
	return e.Path
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return strings.TrimSpace(string(runes[:max]))
}

func truncateBytes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
