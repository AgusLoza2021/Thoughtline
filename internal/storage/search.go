package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// Search defaults & caps. Exported so the server layer can document them.
const (
	DefaultSearchLimit = 10
	MaxSearchLimit     = 50

	// SnippetMaxChars caps the per-result preview returned to callers. The
	// underlying FTS5 snippet() is in token units; we trim to char units after.
	SnippetMaxChars = 300

	// topicKeyShortcutScore is assigned to results retrieved by the
	// topic_key GLOB shortcut so they sort ahead of FTS hits when callers
	// want a unified ranking. Mirrors Engram's -1000 sentinel.
	topicKeyShortcutScore = -1000.0
)

// SearchOptions filters and paginates a Search call. Empty string filters
// (Project, Scope, Type, TopicKey) mean "no filter for this column".
type SearchOptions struct {
	Project  string
	Scope    string
	Type     string
	TopicKey string // GLOB pattern (e.g. "design/*"); exact when no wildcard
	Limit    int
	Offset   int
}

// SearchResult is a single hit from Search. Snippet is at most SnippetMaxChars.
// Score is the BM25 rank from SQLite FTS5 — lower (more negative) means a
// stronger match. Topic-key shortcut hits use a synthetic score of -1000 so
// they always rank ahead of any FTS hit.
type SearchResult struct {
	ID            int64
	SyncID        string
	Project       string
	Scope         memory.Scope
	Type          memory.Type
	TopicKey      string
	Title         string
	Snippet       string
	Tags          []string
	Score         float64
	RevisionCount int
	UpdatedAt     time.Time
}

// Search returns memories matching the query, ranked by BM25 (or by recency
// for topic-key shortcut hits). Filters are applied as exact-match SQL WHERE
// predicates, except TopicKey which is a GLOB pattern.
//
// Behaviour summary:
//   - If query contains "/", a topic_key GLOB lookup runs first. If it returns
//     rows, those rows are the result set (FTS does not run). If it returns
//     zero rows, we fall through to the FTS path. This is the
//     short-circuit behaviour documented in PROGRESS.md (M2 Q2).
//   - Otherwise the FTS5 path runs, with each query token wrapped in quotes
//     to neutralize FTS5 operator characters in user input.
//   - Soft-deleted rows (deleted_at IS NOT NULL) are never returned.
func (s *Storage) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	if strings.Contains(query, "/") {
		hits, err := s.searchByTopicKey(ctx, query, opts, limit, offset)
		if err != nil {
			return nil, err
		}
		if len(hits) > 0 {
			return hits, nil
		}
		// Shortcut miss → fall through to FTS path.
	}

	return s.searchByFTS(ctx, query, opts, limit, offset)
}

// searchByTopicKey runs the GLOB shortcut. Rank is synthetic; ordering is by
// updated_at DESC (most recently touched first).
func (s *Storage) searchByTopicKey(ctx context.Context, glob string, opts SearchOptions, limit, offset int) ([]SearchResult, error) {
	var (
		where = []string{"topic_key GLOB ?", "deleted_at IS NULL"}
		args  = []any{glob}
	)
	addCommonFilters(&where, &args, opts)

	q := `
		SELECT id, sync_id, project, scope, type, topic_key,
		       title, content, tags, revision_count, updated_at
		FROM memories
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY updated_at DESC
		LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("topic_key shortcut: %w", err)
	}
	defer rows.Close()

	var out []SearchResult
	for rows.Next() {
		var (
			r          SearchResult
			scope, typ string
			topicKey   sql.NullString
			tagsJSON   sql.NullString
			content    string
			updatedMS  int64
		)
		if err := rows.Scan(
			&r.ID, &r.SyncID, &r.Project, &scope, &typ, &topicKey,
			&r.Title, &content, &tagsJSON, &r.RevisionCount, &updatedMS,
		); err != nil {
			return nil, fmt.Errorf("scan topic_key row: %w", err)
		}
		r.Scope = memory.Scope(scope)
		r.Type = memory.Type(typ)
		if topicKey.Valid {
			r.TopicKey = topicKey.String
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			if err := json.Unmarshal([]byte(tagsJSON.String), &r.Tags); err != nil {
				return nil, fmt.Errorf("decode tags: %w", err)
			}
		}
		r.Snippet = truncateRunes(content, SnippetMaxChars)
		r.Score = topicKeyShortcutScore
		r.UpdatedAt = time.UnixMilli(updatedMS)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topic_key rows: %w", err)
	}
	return out, nil
}

// searchByFTS runs the FTS5 + BM25 path. The query is sanitized so each token
// is a literal-quoted phrase; FTS5 special chars in user input become inert.
func (s *Storage) searchByFTS(ctx context.Context, raw string, opts SearchOptions, limit, offset int) ([]SearchResult, error) {
	q := sanitizeFTSQuery(raw)
	if q == "" {
		// Nothing left after sanitization (e.g. all whitespace) — empty result.
		return nil, nil
	}

	where := []string{"memories_fts MATCH ?", "m.deleted_at IS NULL"}
	args := []any{q}
	addCommonFiltersAliased(&where, &args, opts, "m")

	sqlStr := `
		SELECT m.id, m.sync_id, m.project, m.scope, m.type, m.topic_key,
		       m.title,
		       snippet(memories_fts, 1, '', '', '…', 32) AS snip,
		       m.tags, m.revision_count, m.updated_at, fts.rank
		FROM memories_fts fts
		JOIN memories m ON m.id = fts.rowid
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY fts.rank
		LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()

	var out []SearchResult
	for rows.Next() {
		var (
			r          SearchResult
			scope, typ string
			topicKey   sql.NullString
			tagsJSON   sql.NullString
			snip       string
			updatedMS  int64
		)
		if err := rows.Scan(
			&r.ID, &r.SyncID, &r.Project, &scope, &typ, &topicKey,
			&r.Title, &snip, &tagsJSON, &r.RevisionCount, &updatedMS, &r.Score,
		); err != nil {
			return nil, fmt.Errorf("scan fts row: %w", err)
		}
		r.Scope = memory.Scope(scope)
		r.Type = memory.Type(typ)
		if topicKey.Valid {
			r.TopicKey = topicKey.String
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			if err := json.Unmarshal([]byte(tagsJSON.String), &r.Tags); err != nil {
				return nil, fmt.Errorf("decode tags: %w", err)
			}
		}
		r.Snippet = truncateRunes(snip, SnippetMaxChars)
		r.UpdatedAt = time.UnixMilli(updatedMS)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fts rows: %w", err)
	}
	return out, nil
}

// addCommonFilters appends WHERE predicates and args for the topic-key path
// (no table alias).
func addCommonFilters(where *[]string, args *[]any, opts SearchOptions) {
	if opts.Project != "" {
		*where = append(*where, "project = ?")
		*args = append(*args, opts.Project)
	}
	if opts.Scope != "" {
		*where = append(*where, "scope = ?")
		*args = append(*args, opts.Scope)
	}
	if opts.Type != "" {
		*where = append(*where, "type = ?")
		*args = append(*args, opts.Type)
	}
	if opts.TopicKey != "" {
		*where = append(*where, "topic_key GLOB ?")
		*args = append(*args, opts.TopicKey)
	}
}

// addCommonFiltersAliased is like addCommonFilters but qualifies columns with
// the given table alias (used by the FTS join).
func addCommonFiltersAliased(where *[]string, args *[]any, opts SearchOptions, alias string) {
	a := alias + "."
	if opts.Project != "" {
		*where = append(*where, a+"project = ?")
		*args = append(*args, opts.Project)
	}
	if opts.Scope != "" {
		*where = append(*where, a+"scope = ?")
		*args = append(*args, opts.Scope)
	}
	if opts.Type != "" {
		*where = append(*where, a+"type = ?")
		*args = append(*args, opts.Type)
	}
	if opts.TopicKey != "" {
		*where = append(*where, a+"topic_key GLOB ?")
		*args = append(*args, opts.TopicKey)
	}
}

// sanitizeFTSQuery makes a user query safe for FTS5 MATCH by stripping all
// double-quote characters from each whitespace-delimited token and re-wrapping
// the token in quotes. Result tokens are joined with spaces, which FTS5 reads
// as implicit AND. After sanitization, the query is purely a sequence of
// literal phrases — no operator (`:` `*` `^` `(` `)` `NEAR` `OR` ...) survives.
//
// Empty input or input made entirely of quotes returns "".
func sanitizeFTSQuery(raw string) string {
	tokens := strings.Fields(raw)
	out := make([]string, 0, len(tokens))
	for _, w := range tokens {
		w = strings.ReplaceAll(w, `"`, "")
		if w == "" {
			continue
		}
		out = append(out, `"`+w+`"`)
	}
	return strings.Join(out, " ")
}

// truncateRunes returns s truncated to at most n runes (not bytes), preserving
// UTF-8 boundaries. Returns s unchanged if it is already short enough.
func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
