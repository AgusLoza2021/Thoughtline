package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
)

// TagCount is one row of the TopTags result.
type TagCount struct {
	Tag   string
	Count int
}

// TopTags returns the most-used tags across active memories scoped to brainID.
// When brainID is 0 the filter is omitted (cross-brain, used by stats only).
// limit <= 0 returns the default cap (20). Tags are stored as JSON arrays in
// the memories.tags column; SQLite does not have native JSON aggregation in
// the modernc driver, so we decode in Go.
//
// Soft-deleted memories are excluded.
func (s *Storage) TopTags(ctx context.Context, brainID int64, limit int) ([]TagCount, error) {
	if limit <= 0 {
		limit = 20
	}

	args := []any{}
	where := "deleted_at IS NULL AND tags IS NOT NULL"
	if brainID != 0 {
		where += " AND brain_id = ?"
		args = append(args, brainID)
	}

	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT tags FROM memories WHERE %s`, where),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("top tags query: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var tagsJSON sql.NullString
		if err := rows.Scan(&tagsJSON); err != nil {
			return nil, fmt.Errorf("scan tag row: %w", err)
		}
		if !tagsJSON.Valid || tagsJSON.String == "" {
			continue
		}
		var tags []string
		if err := json.Unmarshal([]byte(tagsJSON.String), &tags); err != nil {
			// Skip malformed rows rather than failing the whole stat call.
			continue
		}
		for _, t := range tags {
			if t == "" {
				continue
			}
			counts[t]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tag rows: %w", err)
	}

	out := make([]TagCount, 0, len(counts))
	for tag, n := range counts {
		out = append(out, TagCount{Tag: tag, Count: n})
	}
	// Sort by count desc, then tag asc for stable output.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Tag < out[j].Tag
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
