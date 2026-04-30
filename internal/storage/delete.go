package storage

import (
	"context"
	"fmt"
)

// SoftDelete marks the memory with the given id as deleted by setting
// deleted_at to the current clock time. Subsequent Search, Recent, and
// UpdateByID calls treat the row as not found.
//
// The unique index on (project, topic_key) is partial — it only covers
// rows where deleted_at IS NULL — so soft-deleting a row frees its
// topic_key for a fresh Save in the same project.
//
// Returns ErrMemoryNotFound if no active row matches id (either the id is
// unknown, or the row is already soft-deleted). The contract is symmetric
// with UpdateByID: only "active" rows are addressable.
//
// FTS5 sync: the memories_au trigger fires on the underlying UPDATE and
// will re-index the row's title/content. Search-level filtering on
// deleted_at IS NULL is what actually hides the row from results — the
// FTS index itself is left populated, which is fine because there is no
// public API that reads memories_fts without joining back to memories.
func (s *Storage) SoftDelete(ctx context.Context, id int64) error {
	now := s.nowMillis().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
		UPDATE memories
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("soft delete: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("soft delete rows affected: %w", err)
	}
	if affected == 0 {
		return ErrMemoryNotFound
	}
	return nil
}
