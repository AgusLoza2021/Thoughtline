package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// likeEscape is the ESCAPE character used for prefix matching. Backslash is
// not part of the topic_key alphabet, so it never collides with real data.
const likeEscape = '\\'

// ListByTopicPrefix returns the active memories of a brain whose topic_key
// starts with prefix, ordered by topic_key.
//
// This exists for corpus sync: the knowledge importer needs to know which
// `knowledge/...` rows it previously wrote so it can prune the ones whose file
// disappeared, and there is no other way to enumerate a namespace without
// loading the whole brain.
//
// brainID must be non-zero: a zero brain would span every project and let a
// sync in one project delete rows of another.
func (s *Storage) ListByTopicPrefix(ctx context.Context, brainID int64, prefix string) ([]memory.Memory, error) {
	if brainID == 0 {
		return nil, ErrBrainRequired
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id
		FROM memories
		WHERE brain_id = ? AND deleted_at IS NULL
		  AND topic_key LIKE ? ESCAPE '\'
		ORDER BY topic_key`,
		brainID, escapeLike(prefix)+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("list by topic prefix: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan topic prefix row: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list by topic prefix: %w", err)
	}

	// Fetch each row through getByID so there is exactly one place that knows
	// the column layout. The id list is a namespace-sized slice (hundreds at
	// most), so the extra round-trips cost nothing worth a duplicated scanner.
	out := make([]memory.Memory, 0, len(ids))
	for _, id := range ids {
		m, err := s.getByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("load memory %d: %w", id, err)
		}
		out = append(out, m)
	}
	return out, nil
}

// escapeLike makes a literal string safe to use as a LIKE prefix. Without it a
// topic_key of `knowledge/a_b` would also match `knowledge/axb`, and a caller
// pruning one namespace would delete rows of another.
func escapeLike(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	for _, r := range s {
		switch r {
		case likeEscape, '%', '_':
			b.WriteRune(likeEscape)
		}
		b.WriteRune(r)
	}
	return b.String()
}
