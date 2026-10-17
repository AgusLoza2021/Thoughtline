package knowledge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
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

	// DefaultMinCoverage is the absolute gate: how many distinct query terms an
	// entry must actually claim to be about. See coverageOf.
	DefaultMinCoverage = 2

	// HookMinCoverage is the floor a PreToolUse hook uses. A hook has no task
	// text, so its query is a file name and the anchor has to carry the whole
	// signal: below two terms any file sharing one word gets an injection.
	HookMinCoverage = 2

	// minWeightRareFraction is what "a token appearing in 10% of the corpus"
	// means when the default MinWeight is derived: a token that common carries
	// almost no signal, so a hit that needs two of them is the floor of a real
	// match. Deriving it from N (instead of pinning a number) keeps the gate
	// meaningful as the corpus grows from a dozen entries to hundreds.
	minWeightRareFraction = 0.1

	// defaultMinWeightFactor is how many 10%-common tokens a hit must add up to.
	defaultMinWeightFactor = 2.0

	// knowledgeTopicPrefix is the namespace corpus entries live under. Sync owns
	// it, so it is also how the corpus is enumerated for document frequencies.
	knowledgeTopicPrefix = "knowledge/"

	// projectRuleLimit is how many project conventions/decisions are reported.
	// They outrank recipes, but a page of rules defeats the purpose.
	projectRuleLimit = 3

	// candidateFactor over-fetches before foreign stacks are dropped, so
	// excluding them does not leave the answer shorter than Limit.
	candidateFactor = 3

	// sectionMaxRunes caps one rendered section. Sections are reminders, not
	// the entry itself; the full text stays one tl_get_observation away.
	sectionMaxRunes = 600

	// projectRuleAnchorRunes is how much of a project rule counts as its anchor.
	// A saved rule has no template, so its opening lines are the closest thing
	// to "Applies when" it has.
	projectRuleAnchorRunes = 400

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
	MinCoverage      int      // distinct query terms an entry must match; defaults to DefaultMinCoverage
	MinWeight        float64  // IDF-weighted sum of the matched terms; defaults to a value derived from corpus size
}

// CheckHit is one applicable entry, with the sections a caller needs to decide
// whether to read the whole thing.
type CheckHit struct {
	Source      string   `json:"source"`
	ID          int64    `json:"id"`
	TopicKey    string   `json:"topic_key"`
	Title       string   `json:"title"`
	Score       float64  `json:"score"`  // 0..1, relative to the best hit of the same source
	Weight      float64  `json:"weight"` // sum of IDF over the matched terms: specific terms weigh more
	Coverage    int      `json:"coverage"`
	Matched     []string `json:"matched,omitempty"` // distinct query terms found in the anchor
	AppliesWhen string   `json:"applies_when,omitempty"`
	Validate    string   `json:"validate,omitempty"`
	Limits      string   `json:"limits,omitempty"`
	Stacks      []string `json:"stacks,omitempty"`
}

// CheckResult is what a check found. Query is echoed because it is a pure
// function of the request and the fastest way to explain an empty answer.
//
// Best carries the closest candidate when nothing survived the gates, so a
// caller can tell "the corpus was read and nothing applies" from "the corpus
// was never consulted".
type CheckResult struct {
	Query    string     `json:"query"`
	Hits     []CheckHit `json:"hits"`
	Excluded int        `json:"excluded"`
	Best     *CheckHit  `json:"best_candidate,omitempty"`
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
	tokens := queryTokens(BuildQuery(req.Task, req.Paths))
	res := CheckResult{Query: strings.Join(tokens, " ")}
	if len(tokens) == 0 || st == nil {
		return res, nil
	}

	// The corpus is scanned once per check: its size and its per-token document
	// frequencies decide both the hit weights and the default MinWeight, and
	// recomputing them per candidate would repeat the same 176-row walk.
	table, brainID, err := corpusStats(ctx, st, req, tokens)
	if err != nil {
		return res, err
	}
	if req.MinWeight <= 0 {
		req.MinWeight = table.defaultMinWeight()
	}

	rules, err := projectRuleCandidates(ctx, st, req, tokens, table)
	if err != nil {
		return res, err
	}
	knowledge, excluded, err := knowledgeCandidates(ctx, st, brainID, req, tokens, table)
	if err != nil {
		return res, err
	}
	res.Excluded = excluded
	res.Hits = append(finishCandidates(rules, req, projectRuleLimit, false),
		finishCandidates(knowledge, req, req.Limit, true)...)
	if len(res.Hits) == 0 {
		res.Best = bestCandidate(rules, knowledge)
	}
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
	if req.MinCoverage <= 0 {
		req.MinCoverage = DefaultMinCoverage
	}
	// MinWeight stays as given here: its default depends on the corpus size,
	// which is only known once the corpus has been scanned. See Check.
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

	// Generic code words. Measured on noisy hits: these appeared on tangents as
	// often as on the right entry, so an entry claiming two of them was about
	// nothing in particular. They are English because the corpus is authored in
	// English; Spanish equivalents are covered above.
	"game": true, "games": true, "state": true, "states": true,
	"setup": true, "one": true, "per": true, "every": true, "all": true,
	"core": true, "view": true, "views": true, "module": true,
	"modules": true, "manager": true, "config": true, "data": true,
	"tool": true, "tools": true, "test": true, "tests": true,
	"editor": true, "base": true, "util": true, "utils": true,
	"helper": true, "helpers": true, "common": true, "main": true,
	"behind": true, "inside": true, "first": true, "show": true,
	"shows": true, "feat": true, "set": true, "start": true, "starts": true,
}

// genericPathTokens are directory names and extensions that say nothing about
// the subject of a file. The last two are this owner's repositories: every path
// in them contains that name, so keeping it would match every entry evenly.
// They are re-listed here (the vendor-segment filter already drops them) so a
// joined form that survives the split still cannot reach the query.
var genericPathTokens = map[string]bool{
	"assets": true, "scripts": true, "cs": true, "src": true, "editor": true,
	"runtime": true, "gorodiskigames": true, "survivalio": true,
	"library": true, "packages": true, "temp": true, "tests": true,
	"repo": true, "repos": true,
}

// vendorPathSegments are path segments that name where a file lives rather than
// what it is about: vendored third-party trees and the directories Unity, git
// and Go own. They are dropped as WHOLE segments before CamelCase splitting, so
// "GorodiskiGames/Survivalio" vanishes instead of degrading into the generic
// words "games" and "survivalio" that matched every entry at the same rate.
var vendorPathSegments = map[string]bool{
	"gorodiskigames": true, "survivalio": true, "library": true,
	"packages": true, "temp": true, "obj": true, "logs": true,
	".git": true, ".claude": true, "externals": true, "thirdparty": true,
	"third-party": true, "vendor": true, "plugins": true,
}

// pathWords tokenizes a file path, dropping vendor segments and the extension
// because ".cs" or ".md" appears on every candidate and only dilutes the query.
func pathWords(path string) []string {
	p := strings.ReplaceAll(path, `\`, "/")
	segments := make([]string, 0, 8)
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || vendorPathSegments[strings.ToLower(seg)] {
			continue
		}
		segments = append(segments, seg)
	}
	p = strings.Join(segments, "/")
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

// candidate is one search row plus the parsed body that both the coverage gate
// and the section extractor need. The body is read once per candidate: reading
// it in two passes doubled the queries on every check.
type candidate struct {
	hit   CheckHit
	entry Entry
	valid bool // the body was readable, so its gate section can be used
}

// knowledgeCandidates searches the corpus brain for entries that mention any
// query token, drops entries belonging to a foreign stack, and computes each
// survivor's coverage and weight. brainID is resolved once by corpusStats and
// passed in so the same call does not resolve it twice.
func knowledgeCandidates(ctx context.Context, st *storage.Storage, brainID int64, req CheckRequest, tokens []string, table idfTable) ([]candidate, int, error) {
	if brainID == 0 {
		// Nothing synced yet is an empty corpus, not a failure.
		return nil, 0, nil
	}

	rows, err := st.Search(ctx, brainID, strings.Join(tokens, " "), storage.SearchOptions{
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

	kept := make([]storage.SearchResult, 0, len(rows))
	excluded := 0
	for _, r := range rows {
		if anyStackExcluded(stackTags(r.Tags), excludedStacks) {
			excluded++
			continue
		}
		kept = append(kept, r)
	}
	return loadCandidates(ctx, st, brainID, SourceKnowledge, kept, tokens, false, table), excluded, nil
}

// projectRuleCandidates searches the caller's own brain for rules of the
// current project. SearchOptions.Type takes one value, so the three rule types
// are three searches whose union is re-ranked by coverage, weight and then bm25.
func projectRuleCandidates(ctx context.Context, st *storage.Storage, req CheckRequest, tokens []string, table idfTable) ([]candidate, error) {
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
		found, err := st.Search(ctx, brainID, strings.Join(tokens, " "), storage.SearchOptions{
			Type:     string(t),
			MatchAny: true,
			Limit:    projectRuleLimit,
		})
		if err != nil {
			return nil, fmt.Errorf("knowledge: search project rules: %w", err)
		}
		rows = append(rows, found...)
	}
	return loadCandidates(ctx, st, brainID, SourceProjectRule, rows, tokens, true, table), nil
}

// loadCandidates reads each row's body and scores its coverage and IDF weight
// over the anchor text. A row whose body cannot be read keeps the coverage its
// title and tags earn: a disappearing memory must not sink the whole answer.
func loadCandidates(ctx context.Context, st *storage.Storage, brainID int64, source string, rows []storage.SearchResult, tokens []string, projectRule bool, table idfTable) []candidate {
	out := make([]candidate, 0, len(rows))
	for _, r := range rows {
		c := candidate{hit: newHit(source, r)}
		body, readable := "", false
		if m, err := st.GetByID(ctx, brainID, r.ID); err == nil {
			// Stored content carries no frontmatter (Sync writes the parsed
			// body plus a footer), so re-parsing it is only a way to reach the
			// section API. The title already came from the row.
			body, readable = stripFooter(m.Content), true
		}
		anchor, entry, valid := rowAnchor(r.TopicKey, r.Title, r.Tags, body, readable, projectRule)
		c.entry, c.valid = entry, valid
		c.hit.Coverage, c.hit.Matched = coverageOf(tokens, anchor)
		c.hit.Weight = table.weightOf(c.hit.Matched)
		out = append(out, c)
	}
	return out
}

// rowAnchor is the small, hand-written part of an entry that says what it is
// about: its title, its tags and the section that gates applicability. It is
// also the text document frequencies are counted over, so the gate and the
// weight never disagree about what an entry claims.
//
// WHY not the whole body: a long procedure mentions half its domain somewhere,
// and the full-text index used to rank that as relevance. The measured result
// was a 20-25% precise top hit, so the gate reads only what the entry claims to
// be about.
func rowAnchor(topicKey, title string, tags []string, body string, readable, projectRule bool) (string, Entry, bool) {
	var b strings.Builder
	b.WriteString(title)
	if words := tagWords(tags); words != "" {
		b.WriteString("\n")
		b.WriteString(words)
	}
	if !readable {
		return b.String(), Entry{}, false
	}
	entry, _ := ParseEntry(topicKey, []byte(body))
	b.WriteString("\n")
	switch {
	case projectRule:
		// A saved rule has no template, so its opening prose is its gate. It is
		// capped because a rule's applicability is stated up front or not at all.
		b.WriteString(truncateRunes(entry.Body, projectRuleAnchorRunes))
	case entry.Template == TemplateRecipe:
		b.WriteString(entry.Section("Context"))
	default:
		b.WriteString(entry.Section("Applies when"))
	}
	return b.String(), entry, true
}

// idfTable is the corpus statistics one check needs: how many knowledge rows
// exist (N) and in how many of them each query token occurs as a whole word.
//
// WHY IDF at all: coverage counted generic words (game, state, setup) as much
// as specific ones (camera+shake), so a tangent that shared two common words
// passed the gate. Weighting a match by how rare its tokens are is what makes
// "camera+shake" outrank "game+state" without raising the coverage floor.
type idfTable struct {
	n  int
	df map[string]int
}

// idf is the smoothed inverse document frequency of a token. The +1 offsets
// keep it defined for an empty corpus and for a token that appears nowhere, and
// the +1 outside keeps even a token in every row worth 1.0 so a weight is
// always a sum of positive terms.
func (t idfTable) idf(token string) float64 {
	return math.Log(float64(t.n+1)/float64(t.df[token]+1)) + 1
}

// weightOf sums the IDF of the distinct terms a hit matched.
func (t idfTable) weightOf(matched []string) float64 {
	total := 0.0
	for _, tok := range matched {
		total += t.idf(tok)
	}
	return total
}

// defaultMinWeight is the weight a hit needs when the caller set none: the
// equivalent of two tokens that appear in a tenth of the corpus. It is derived
// from N, not pinned, so the same gate keeps meaning as the corpus grows.
func (t idfTable) defaultMinWeight() float64 {
	rare := int(math.Round(float64(t.n) * minWeightRareFraction))
	if rare < 1 {
		rare = 1
	}
	return defaultMinWeightFactor * (math.Log(float64(t.n+1)/float64(rare+1)) + 1)
}

// corpusStats enumerates the corpus once and builds the IDF table for the query
// tokens. It also returns the resolved brain id (zero when nothing was synced)
// so the caller does not resolve it a second time.
func corpusStats(ctx context.Context, st *storage.Storage, req CheckRequest, tokens []string) (idfTable, int64, error) {
	table := idfTable{df: make(map[string]int, len(tokens))}
	brainID, err := st.ResolveBrainID(ctx, req.KnowledgeProject)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return table, 0, nil
		}
		return table, 0, fmt.Errorf("knowledge: resolve %q: %w", req.KnowledgeProject, err)
	}

	rows, err := st.ListByTopicPrefix(ctx, brainID, knowledgeTopicPrefix)
	if err != nil {
		return table, brainID, fmt.Errorf("knowledge: list corpus: %w", err)
	}

	wanted := make(map[string]bool, len(tokens))
	for _, tok := range tokens {
		wanted[tok] = true
	}
	for _, m := range rows {
		if !hasTag(m.Tags, KindTag) {
			continue
		}
		table.n++
		anchor, _, _ := rowAnchor(m.TopicKey, m.Title, m.Tags, stripFooter(m.Content), true, false)
		seen := make(map[string]bool, len(tokens))
		for _, w := range splitWords(anchor) {
			if !wanted[w] || seen[w] {
				continue
			}
			seen[w] = true
			table.df[w]++
		}
	}
	return table, brainID, nil
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

// tagWords flattens tags into anchor words. A tag carries structure in a prefix
// ("group:camera-and-feel", "stack:svn") and only its value names a subject, so
// the prefix is dropped; splitWords then splits the hyphens.
func tagWords(tags []string) string {
	parts := make([]string, 0, len(tags))
	for _, t := range tags {
		if _, value, ok := strings.Cut(t, ":"); ok {
			t = value
		}
		if t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

// queryTokens is BuildQuery's output read back as a set of terms. BuildQuery
// joins with spaces and never emits a space inside a token.
func queryTokens(query string) []string {
	if query == "" {
		return nil
	}
	return strings.Fields(query)
}

// coverageOf counts how many DISTINCT query terms occur as whole words in the
// anchor, and returns the matched terms in query order.
//
// WHY whole words through splitWords: the anchor is tokenized the same way the
// query was, so "cameraShake" in a title really does contain "shake", while
// "hits" must not satisfy "hit".
func coverageOf(tokens []string, anchor string) (int, []string) {
	if len(tokens) == 0 || anchor == "" {
		return 0, nil
	}
	words := make(map[string]bool, 32)
	for _, w := range splitWords(anchor) {
		words[w] = true
	}
	matched := make([]string, 0, 4)
	seen := make(map[string]bool, 4)
	for _, tok := range tokens {
		if seen[tok] || !words[tok] {
			continue
		}
		seen[tok] = true
		matched = append(matched, tok)
	}
	return len(matched), matched
}

// finishCandidates applies the gates in order: the absolute coverage gate, the
// IDF weight gate, the per-source limit, then the relative MinScore gate on the
// survivors.
//
// Order matters. Coverage runs first because it is the only gate that can
// reject the top hit: normaliseAndGate divides by the best hit of the same
// source, so the best one always scores 1.00 and MinScore could never drop it.
// Weight runs second because it needs no ranking either: it is an absolute sum
// of the rarity of the terms the hit actually matched.
//
// WHY applyWeight is false for project rules: their weight is computed from the
// CORPUS's document frequencies, which say nothing about a rule the owner wrote
// for this project. Gating them on it silently dropped the rule that mattered
// ("camera feel lives in FeelConfig") on any corpus where "camera" is common,
// and the corpus holds at most three of them anyway.
func finishCandidates(cands []candidate, req CheckRequest, limit int, applyWeight bool) []CheckHit {
	rankCandidates(cands, applyWeight)
	kept := cands[:0]
	for _, c := range cands {
		if c.hit.Coverage < req.MinCoverage {
			continue
		}
		if applyWeight && c.hit.Weight < req.MinWeight {
			continue
		}
		kept = append(kept, c)
	}
	if len(kept) > limit {
		kept = kept[:limit]
	}

	hits := make([]CheckHit, len(kept))
	for i, c := range kept {
		hits[i] = c.hit
		if c.valid {
			hits[i].AppliesWhen, hits[i].Validate, hits[i].Limits = checkSections(c.entry)
		}
	}
	return normaliseAndGate(hits, req.MinScore)
}

// rankCandidates orders survivors by IDF weight first and by raw relevance
// second: weight is what separates a specific match (camera+shake) from one
// built out of generic words (game+state), and bm25 only breaks ties inside a
// weight level. Coverage is already gated, so it no longer needs to lead.
//
// Project rules are ranked by bm25 alone (applyWeight false) because their
// weight would be measured against the corpus, not against the project.
func rankCandidates(cands []candidate, applyWeight bool) {
	sort.SliceStable(cands, func(i, j int) bool {
		if applyWeight && cands[i].hit.Weight != cands[j].hit.Weight {
			return cands[i].hit.Weight > cands[j].hit.Weight
		}
		return cands[i].hit.Score > cands[j].hit.Score
	})
}

// bestCandidate reports the closest miss so an empty answer can say what it
// nearly found. Candidates with coverage 0 are not worth naming: they share no
// term with the task at all.
func bestCandidate(lists ...[]candidate) *CheckHit {
	var best *CheckHit
	for _, list := range lists {
		for i := range list {
			hit := list[i].hit
			if hit.Coverage == 0 {
				continue
			}
			if best == nil || hit.Coverage > best.Coverage ||
				(hit.Coverage == best.Coverage && hit.Weight > best.Weight) {
				best = &hit
			}
		}
	}
	return best
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
// It carries only the content hash.
const footerMarker = "\n---\n_kh:"

// legacyFooterMarker is the pre-gate footer shape ("_knowledge: corpus/slug ·
// group: …_"). Entries synced before the change still carry it until their
// next sync, and its words are exactly the noise the coverage gate exists to
// ignore, so it is still stripped from sections and from anchor text.
const legacyFooterMarker = "\n---\n_knowledge: "

// stripFooter removes that provenance footer: it is metadata for a human
// reading the row, and it would otherwise land inside the last section.
func stripFooter(content string) string {
	for _, marker := range []string{footerMarker, legacyFooterMarker} {
		if i := strings.LastIndex(content, marker); i >= 0 {
			return content[:i]
		}
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
