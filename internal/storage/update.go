package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// UpdatePatch describes a partial mutation applied by UpdateByID. Pointer
// fields distinguish "leave unchanged" (nil) from "set to this value"
// (non-nil — including a pointer to the zero value, which would for example
// clear the tag list when set to &[]string{}).
//
// Only mutable fields are exposed: Title, Content, Tags. Type, TopicKey,
// Project, and Scope are identity-defining and cannot be changed by
// UpdateByID — the AI is expected to delete + re-save when those need to
// move.
type UpdatePatch struct {
	Title   *string
	Content *string
	Tags    *[]string
}

// IsEmpty reports whether the patch carries no changes.
func (p UpdatePatch) IsEmpty() bool {
	return p.Title == nil && p.Content == nil && p.Tags == nil
}

// UpdateByID applies a partial mutation to the memory with the given id,
// scoped to brainID. Cross-brain access returns ErrMemoryNotFound (existence
// of the row in another brain is not leaked).
//
// Behaviour:
//   - If the row does not exist, is soft-deleted, or belongs to a different
//     brain, returns ErrMemoryNotFound.
//   - Empty patch (all nil) is a true noop: returns the current row unchanged
//     without bumping revision_count or updated_at.
//   - The merged Memory is re-validated against memory.Validate; any rule
//     violation is returned verbatim (callers can errors.Is against the
//     specific memory.Err* sentinels).
//   - On real change: bumps revision_count, refreshes updated_at, recomputes
//     normalized_hash, preserves id / sync_id / created_at / type / topic_key
//     / project / scope.
//   - FTS5 stays in sync via the existing memories_au trigger (eviction +
//     re-insert with new title/content).
func (s *Storage) UpdateByID(ctx context.Context, brainID int64, id int64, patch UpdatePatch) (memory.Memory, error) {
	current, err := s.getByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memory.Memory{}, ErrMemoryNotFound
		}
		return memory.Memory{}, fmt.Errorf("update lookup: %w", err)
	}
	if current.DeletedAt != nil {
		return memory.Memory{}, ErrMemoryNotFound
	}
	if current.BrainID != brainID {
		return memory.Memory{}, ErrMemoryNotFound
	}

	if patch.IsEmpty() {
		return current, nil
	}

	next := current
	if patch.Title != nil {
		next.Title = *patch.Title
	}
	if patch.Content != nil {
		next.Content = *patch.Content
	}
	if patch.Tags != nil {
		next.Tags = *patch.Tags
	}

	// Re-validate the merged memory with the same rules tl_save applies. This
	// catches an empty title, oversized content, malformed tags, etc. before
	// we touch the database.
	if err := memory.Validate(next); err != nil {
		return memory.Memory{}, err
	}

	now := s.nowMillis()
	hash := normalizedHash(next.Title, next.Content)
	tagsJSON, err := encodeTags(next.Tags)
	if err != nil {
		return memory.Memory{}, err
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE memories
		SET title = ?, content = ?, tags = ?, normalized_hash = ?,
		    revision_count = revision_count + 1, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		next.Title, next.Content, tagsJSON, hash,
		now.UnixMilli(),
		id,
	)
	if err != nil {
		return memory.Memory{}, fmt.Errorf("update memory: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return memory.Memory{}, fmt.Errorf("update rows affected: %w", err)
	}
	if affected == 0 {
		// Race: row was soft-deleted between getByID and UPDATE.
		return memory.Memory{}, ErrMemoryNotFound
	}

	next.NormalizedHash = hash
	next.RevisionCount = current.RevisionCount + 1
	next.UpdatedAt = now
	return next, nil
}
