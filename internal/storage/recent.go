package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// Recent returns the most recently updated active memories for a project,
// ordered by updated_at DESC. Soft-deleted rows are excluded. The returned
// SearchResult mirrors what tl_search yields, except Score is always 0
// (recency is not a relevance signal) and Snippet is the content prefix
// truncated to SnippetMaxChars (no FTS5 match centering).
//
// project must be non-empty: passing "" would silently return rows from
// every project, which is a security/data-leak shape we refuse to support.
// Callers that genuinely want cross-project recency should call Recent
// once per project.
//
// limit follows the same semantics as Search: <=0 means DefaultSearchLimit,
// values above MaxSearchLimit are clamped down.
func (s *Storage) Recent(ctx context.Context, project string, limit int) ([]SearchResult, error) {
	if project == "" {
		return nil, errors.New("storage: Recent requires a non-empty project")
	}
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sync_id, project, scope, type, topic_key,
		       title, content, tags, revision_count, updated_at
		FROM memories
		WHERE project = ? AND deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT ?`,
		project, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent query: %w", err)
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
			return nil, fmt.Errorf("scan recent row: %w", err)
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
		r.UpdatedAt = time.UnixMilli(updatedMS)
		// Score stays at zero — Recent has no relevance signal.
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent rows: %w", err)
	}
	return out, nil
}
