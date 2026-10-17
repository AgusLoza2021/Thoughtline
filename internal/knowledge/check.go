package knowledge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// Check defaults & caps. A check answers "does anything I wrote apply here?" —
// it is a nudge before writing code, so the answer must stay small enough to
// read inside a prompt.
const (
	DefaultCheckLimit = 5
	MaxCheckLimit     = 10

	// DefaultMinScore keeps a hit only when it is at least this fraction as
	// relevant as the best hit of the same source. See normaliseAndGate.
	DefaultMinScore = 0.35

	// projectRuleLimit is how many project conventions/decisions are reported.
	// They outrank recipes, but a page of rules defeats the purpose.
	projectRuleLimit = 3

	// candidateFactor over-fetches before foreign stacks are dropped, so
	// excluding them does not leave the answer shorter than Limit.
	candidateFactor = 3

	// sectionMaxRunes caps one rendered section. Sections are reminders, not
	// the entry itself; the full text stays one tl_get_observation away.
	sectionMaxRunes = 600

	// maxQueryTokens bounds the generated FTS query. Beyond this the OR chain
	// matches almost every entry, which ranks by noise instead of relevance.
	maxQueryTokens = 16

	// SourceKnowledge marks a hit that came from the corpus brain.
	SourceKnowledge = "knowledge"
	// SourceProjectRule marks a hit that came from the caller's own project
	// brain. Project rules win over generic recipes, so they are listed first.
	SourceProjectRule = "project-rule"

	// KindTag identifies corpus entries. The memory type only distinguishes
	// procedural from recipe, so the tag is the reliable marker.
	KindTag        = "kind:knowledge"
	StackTagPrefix = "stack:"
)

// CheckRequest is one "what applies to this change?" question.
type CheckRequest struct {
	Task             string   // free-text description of the work
	Paths            []string // files the change touches
	Project          string   // current project, searched for project rules
	KnowledgeProject string   // corpus brain; defaults to DefaultSyncProject
	ExcludeStacks    []string // foreign stacks to drop, e.g. svn, fiber, tactile
	Limit            int      // knowledge hits; defaults to DefaultCheckLimit, max MaxCheckLimit
	MinScore         float64  // relevance gate; defaults to DefaultMinScore
}

// CheckHit is one applicable entry, with the sections a caller needs to decide
// whether to read the whole thing.
type CheckHit struct {
	Source      string   `json:"source"`
	ID          int64    `json:"id"`
	TopicKey    string   `json:"topic_key"`
	Title       string   `json:"title"`
	Score       float64  `json:"score"` // 0..1, relative to the best hit of the same source
	AppliesWhen string   `json:"applies_when,omitempty"`
	Validate    string   `json:"validate,omitempty"`
	Limits      string   `json:"limits,omitempty"`
	Stacks      []string `json:"stacks,omitempty"`
}

// CheckResult is what a check found. Query is echoed because it is a pure
// function of the request and the fastest way to explain an empty answer.
type CheckResult struct {
	Query    string     `json:"query"`
	Hits     []CheckHit `json:"hits"`
	Excluded int        `json:"excluded"`
}

// projectRuleTypes are the memory types that carry a rule the current project
// already agreed on. Discovery/config/pattern are not rules one can violate.
var projectRuleTypes = []memory.Type{
	memory.TypeConvention,
	memory.TypeDecision,
	memory.TypeArchitecture,
}

// Check answers one question: which corpus entries and project rules apply to
// this task and these paths? It never writes and never creates a brain: a
// check runs on every edit hook, so it must be a pure read.
func Check(ctx context.Context, st *storage.Storage, req CheckRequest) (CheckResult, error) {
	req = withCheckDefaults(req)
	res := CheckResult{Query: BuildQuery(req.Task, req.Paths)}
	if res.Query == "" || st == nil {
		return res, nil
	}

	rules, err := projectRuleHits(ctx, st, req, res.Query)
	if err != nil {
		return res, err
	}
	hits, excluded, err := knowledgeHits(ctx, st, req, res.Query)
	if err != nil {
		return res, err
	}
	res.Excluded = excluded
	res.Hits = append(rules, hits...)
	return res, nil
}

func withCheckDefaults(req CheckRequest) CheckRequest {
	req.Task = strings.TrimSpace(req.Task)
	req.Project = strings.TrimSpace(req.Project)
	req.KnowledgeProject = orDefault(strings.TrimSpace(req.KnowledgeProject), DefaultSyncProject)
	switch {
	case req.Limit <= 0:
		req.Limit = DefaultCheckLimit
	case req.Limit > MaxCheckLimit:
		req.Limit = MaxCheckLimit
	}
	if req.MinScore <= 0 {
		req.MinScore = DefaultMinScore
	}
	req.ExcludeStacks = normalizeStacks(req.ExcludeStacks)
	return req
}

// BuildQuery turns a task description and a set of paths into the FTS query a
// check runs. It is pure and exported because the query explains the whole
// result: an empty answer is almost always a query problem, not an empty
// corpus.
//
// Ordering matters: task words come first so the cap drops path noise before
// it drops what the caller actually asked about.
func BuildQuery(task string, paths []string) string {
	seen := make(map[string]bool, maxQueryTokens)
	tokens := make([]string, 0, maxQueryTokens)
	add := func(tok string) {
		if len(tokens) == maxQueryTokens || seen[tok] {
			return
		}
		if len([]rune(tok)) < 3 || queryNoise[tok] || genericPathTokens[tok] {
			return
		}
		seen[tok] = true
		tokens = append(tokens, tok)
	}

	for _, tok := range splitWords(task) {
		add(tok)
	}
	for _, p := range paths {
		for _, tok := range pathWords(p) {
			add(tok)
		}
	}
	return strings.Join(tokens, " ")
}

// queryNoise holds words that carry no matching signal. Two languages because
// the owner works in Spanish and the corpus is written in English; the length
// filter already removes most one- and two-letter Spanish particles.
var queryNoise = map[string]bool{
	// English
	"the": true, "and": true, "for": true, "with": true, "when": true,
	"then": true, "than": true, "that": true, "this": true, "these": true,
	"those": true, "into": true, "onto": true, "from": true, "over": true,
	"under": true, "about": true, "after": true, "before": true, "while": true,
	"should": true, "would": true, "could": true, "must": true, "might": true,
	"have": true, "has": true, "had": true, "been": true, "being": true,
	"does": true, "did": true, "was": true, "were": true, "are": true,
	"add": true, "adding": true, "adds": true, "make": true, "makes": true,
	"making": true, "use": true, "uses": true, "using": true, "need": true,
	"needs": true, "want": true, "wants": true, "get": true, "gets": true,
	"getting": true, "fix": true, "fixes": true, "fixing": true,
	"implement": true, "implementing": true, "create": true, "creating": true,
	"write": true, "writing": true, "change": true, "changes": true,
	"refactor": true, "code": true, "there": true, "here": true, "what": true,
	"which": true, "where": true, "how": true, "why": true, "also": true,
	"just": true, "like": true, "only": true, "some": true, "more": true,
	"most": true, "very": true, "new": true, "now": true, "out": true,
	"them": true, "they": true, "their": true, "your": true, "you": true,
	"its": true, "our": true, "ours": true, "his": true, "her": true,
	// Spanish
	"los": true, "las": true, "una": true, "unos": true, "unas": true,
	"del": true, "para": true, "por": true, "con": true, "que": true,
	"como": true, "cuando": true, "donde": true, "entre": true, "sobre": true,
	"hay": true, "esta": true, "este": true, "esto": true, "estos": true,
	"son": true, "ser": true, "sus": true, "mis": true, "mas": true,
	"pero": true, "muy": true, "tambien": true, "todo": true, "toda": true,
	"todos": true, "todas": true, "otro": true, "otra": true, "cada": true,
	"usar": true, "usando": true, "puede": true, "debe": true, "tiene": true,
	"hacer": true, "agregar": true, "agrega": true, "crear": true,
	"cambiar": true, "arreglar": true, "porque": true,
}

// genericPathTokens are directory names and extensions that say nothing about
// the subject of a file. The last two are this owner's repositories: every path
// in them contains that name, so keeping it would match every entry evenly.
var genericPathTokens = map[string]bool{
	"assets": true, "scripts": true, "cs": true, "src": true, "editor": true,
	"runtime": true, "gorodiskigames": true, "survivalio": true,
	"library": true, "packages": true, "temp": true, "tests": true,
	"repo": true, "repos": true,
}

// pathWords tokenizes a file path, dropping the extension because ".cs" or
// ".md" appears on every candidate and only dilutes the query.
func pathWords(path string) []string {
	p := strings.ReplaceAll(path, `\`, "/")
	cut := len(p)
	if i := strings.LastIndex(p, "/"); i >= 0 {
		if j := strings.LastIndex(p[i+1:], "."); j > 0 {
			cut = i + 1 + j
		}
	} else if j := strings.LastIndex(p, "."); j > 0 {
		cut = j
	}
	return splitWords(p[:cut])
}

// splitWords tokenizes on anything that is not a letter or digit, plus
// CamelCase boundaries, and lowercases the result: "EnemyHitFlashSystem" is
// four words, "SVNExternals" is two.
func splitWords(s string) []string {
	runes := []rune(s)
	out := make([]string, 0, 8)
	start := -1
	flush := func(end int) {
		if start >= 0 && end > start {
			out = append(out, strings.ToLower(string(runes[start:end])))
		}
		start = -1
	}
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush(i)
			continue
		}
		if start < 0 {
			start = i
			continue
		}
		prev := runes[i-1]
		switch {
		case unicode.IsUpper(r) && unicode.IsLower(prev):
			flush(i)
			start = i
		case unicode.IsUpper(r) && unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]):
			// An acronym that starts a new word: the boundary sits before its
			// last capital, so "SVNExternals" yields svn + externals.
			flush(i)
			start = i
		}
	}
	flush(len(runes))
	return out
}

// knowledgeHits searches the corpus brain for entries that mention any query
// token, drops entries belonging to a foreign stack, and keeps the best Limit.
func knowledgeHits(ctx context.Context, st *storage.Storage, req CheckRequest, query string) ([]CheckHit, int, error) {
	brainID, err := st.ResolveBrainID(ctx, req.KnowledgeProject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Nothing synced yet is an empty corpus, not a failure.
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("knowledge: resolve %q: %w", req.KnowledgeProject, err)
	}

	rows, err := st.Search(ctx, brainID, query, storage.SearchOptions{
		Tags:     []string{KindTag},
		MatchAny: true,
		Limit:    req.Limit * candidateFactor,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("knowledge: search corpus: %w", err)
	}

	excludedStacks := make(map[string]bool, len(req.ExcludeStacks))
	for _, s := range req.ExcludeStacks {
		excludedStacks[s] = true
	}

	kept := make([]CheckHit, 0, len(rows))
	excluded := 0
	for _, r := range rows {
		hit := newHit(SourceKnowledge, r)
		if anyStackExcluded(hit.Stacks, excludedStacks) {
			excluded++
			continue
		}
		kept = append(kept, hit)
	}
	if len(kept) > req.Limit {
		kept = kept[:req.Limit]
	}
	kept = normaliseAndGate(kept, req.MinScore)
	fillSections(ctx, st, brainID, kept)
	return kept, excluded, nil
}

// projectRuleHits searches the caller's own brain for rules of the current
// project. SearchOptions.Type takes one value, so the three rule types are
// three searches whose union is re-ranked: bm25 rank is computed over the whole
// FTS table and the type filter is applied afterwards, so the ranks in the
// union are directly comparable.
func projectRuleHits(ctx context.Context, st *storage.Storage, req CheckRequest, query string) ([]CheckHit, error) {
	if req.Project == "" {
		return nil, nil
	}
	brainID, err := st.ResolveBrainID(ctx, req.Project)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// A project with no memories has no rules yet.
			return nil, nil
		}
		return nil, fmt.Errorf("knowledge: resolve project %q: %w", req.Project, err)
	}

	var rows []storage.SearchResult
	for _, t := range projectRuleTypes {
		found, err := st.Search(ctx, brainID, query, storage.SearchOptions{
			Type:     string(t),
			MatchAny: true,
			Limit:    projectRuleLimit,
		})
		if err != nil {
			return nil, fmt.Errorf("knowledge: search project rules: %w", err)
		}
		rows = append(rows, found...)
	}

	// FTS rank is negative and lower is better.
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Score < rows[j].Score })
	if len(rows) > projectRuleLimit {
		rows = rows[:projectRuleLimit]
	}

	hits := make([]CheckHit, 0, len(rows))
	for _, r := range rows {
		hits = append(hits, newHit(SourceProjectRule, r))
	}
	hits = normaliseAndGate(hits, req.MinScore)
	fillSections(ctx, st, brainID, hits)
	return hits, nil
}

// newHit maps a search row onto the part of a hit that the row already has.
// Sections need the full body, which Search does not return (it caps the
// snippet at 300 chars), so they are filled in later and only for survivors.
func newHit(source string, r storage.SearchResult) CheckHit {
	return CheckHit{
		Source:   source,
		ID:       r.ID,
		TopicKey: r.TopicKey,
		Title:    r.Title,
		// FTS5 bm25 rank is negative with more negative = better. Exposing
		// -rank keeps the intermediate value positive so the ratio in
		// normaliseAndGate means what it reads like.
		Score:  -r.Score,
		Stacks: stackTags(r.Tags),
	}
}

// fillSections reads each surviving hit's body and extracts the three sections
// a caller needs. A row that cannot be read keeps its title and score: a
// disappearing memory must not sink the whole answer.
func fillSections(ctx context.Context, st *storage.Storage, brainID int64, hits []CheckHit) {
	for i := range hits {
		m, err := st.GetByID(ctx, brainID, hits[i].ID)
		if err != nil {
			continue
		}
		// Stored content carries no frontmatter (Sync writes the parsed body
		// plus a footer), so re-parsing it is only a way to reach the section
		// API. The title already came from the row.
		entry, _ := ParseEntry(hits[i].TopicKey, []byte(stripFooter(m.Content)))
		hits[i].AppliesWhen, hits[i].Validate, hits[i].Limits = checkSections(entry)
	}
}

// checkSections picks the three sections a caller needs. The corpora mix two
// hand-written templates: a recipe has no "Applies when", so its Context is
// the gate and its Example is what "Validate" points at. Nothing else maps
// closely enough to be useful, and a heading nobody agreed on is not guessed.
func checkSections(e Entry) (appliesWhen, validate, limits string) {
	when, val, lim := "Applies when", "Validate the result", "Limits"
	if e.Template == TemplateRecipe {
		when, val, lim = "Context", "Example", "Notes"
	}
	appliesWhen = trimSection(e.Section(when))
	validate = trimSection(e.Section(val))
	limits = trimSection(e.Section(lim))
	if appliesWhen == "" && len(e.Sections) > 0 {
		// TemplateOther: the entry still has prose, and its first section is
		// the closest thing to a gate it has.
		appliesWhen = trimSection(e.Sections[0].Body)
	}
	return appliesWhen, validate, limits
}

// footerMarker starts the provenance line Sync appends to every stored entry.
const footerMarker = "\n---\n_knowledge: "

// stripFooter removes that provenance footer: it is metadata for a human
// reading the row, and it would otherwise land inside the last section.
func stripFooter(content string) string {
	if i := strings.LastIndex(content, footerMarker); i >= 0 {
		return content[:i]
	}
	return content
}

func trimSection(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= sectionMaxRunes {
		return s
	}
	return strings.TrimSpace(string(runes[:sectionMaxRunes-1])) + "…"
}

// normaliseAndGate turns raw relevance into a 0..1 score relative to the best
// hit of the same source and drops hits below minScore.
//
// WHY relative: FTS5's bm25() is unbounded and its magnitude depends on the
// query and on the corpus, so an absolute threshold would mean something
// different for every task — the number that looked right for "camera shake"
// would reject everything for "pool VFX particles". Dividing by the best hit of
// the same source makes the threshold a ratio ("keep hits at least 35% as
// relevant as the best one"), which holds across queries. Ties at the top get
// 1.0; a non-positive best (no rank data) means every hit tied, so they all
// survive rather than silently vanishing.
func normaliseAndGate(hits []CheckHit, minScore float64) []CheckHit {
	if len(hits) == 0 {
		return nil
	}
	best := 0.0
	for _, h := range hits {
		if h.Score > best {
			best = h.Score
		}
	}
	out := hits[:0]
	for _, h := range hits {
		if best <= 0 {
			h.Score = 1
		} else {
			h.Score = h.Score / best
		}
		if h.Score >= minScore {
			out = append(out, h)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// normalizeStacks accepts both "svn" and "stack:svn" so the same value works as
// an env list item and as a tag.
func normalizeStacks(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, s := range raw {
		s = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), StackTagPrefix)))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// stackTags extracts the stack names a row is tagged with.
func stackTags(tags []string) []string {
	var out []string
	for _, t := range tags {
		if name, ok := strings.CutPrefix(t, StackTagPrefix); ok && name != "" {
			out = append(out, name)
		}
	}
	return out
}

func anyStackExcluded(stacks []string, excluded map[string]bool) bool {
	if len(excluded) == 0 {
		return false
	}
	for _, s := range stacks {
		if excluded[s] {
			return true
		}
	}
	return false
}
