package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// DedupeCheck reports whether a non-topic-keyed, non-deleted memory with the
// given normalized_hash exists for brainID with created_at >= windowStart.
//
// The query is scoped by two mandatory filters (both required per design D8):
//   - topic_key IS NULL — topic-keyed upserts are never dedup candidates
//   - deleted_at IS NULL — tombstones do not block re-saves
//
// Returns:
//   - found:             true when a matching row is located.
//   - existingID:        the row id when found; zero otherwise.
//   - existingCreatedAt: the matched row's created_at (millisecond precision,
//     suitable for "saved N minutes ago" rendering); zero time otherwise.
//   - err:               any database error; callers must check before using
//     the other return values.
//
// Recipe note (INTERNAL): the hash is computed by NormalizedHash, which
// applies SHA-256(trimSpace(title) + NUL + trimSpace(content)). The recipe
// is intentionally opaque to callers — only the server's doSave may call
// NormalizedHash directly; do not depend on the exact algorithm.
func (s *Storage) DedupeCheck(
	ctx context.Context,
	brainID int64,
	hash string,
	windowStart time.Time,
) (found bool, existingID int64, existingCreatedAt time.Time, err error) {
	var (
		id          int64
		createdAtMS int64
	)
	queryErr := s.db.QueryRowContext(ctx, `
		SELECT id, created_at
		FROM memories
		WHERE brain_id = ?
		  AND normalized_hash = ?
		  AND topic_key IS NULL
		  AND deleted_at IS NULL
		  AND created_at >= ?
		ORDER BY created_at DESC
		LIMIT 1`,
		brainID, hash, windowStart.UnixMilli(),
	).Scan(&id, &createdAtMS)

	if errors.Is(queryErr, sql.ErrNoRows) {
		return false, 0, time.Time{}, nil
	}
	if queryErr != nil {
		return false, 0, time.Time{}, fmt.Errorf("dedupe check: %w", queryErr)
	}
	return true, id, time.UnixMilli(createdAtMS), nil
}
