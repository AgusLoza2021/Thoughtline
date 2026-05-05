package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// ReadObservations reads all active (non-soft-deleted) observations from the
// Engram database and returns them as a slice of EngramObservation. It accepts
// a *sql.DB opened by the caller with read-only WAL pragmas — it never opens
// its own connection.
//
// Columns that Engram stores but Thoughtline has no equivalent for
// (tool_name, duplicate_count, last_seen_at) are scanned and discarded.
//
// The full result set is returned in one slice. At ~291 rows × ~2 KB avg the
// memory budget is well within reason.
func ReadObservations(ctx context.Context, db *sql.DB) ([]EngramObservation, error) {
	const q = `
		SELECT
			sync_id, type, title, content, project, scope,
			COALESCE(topic_key, ''),
			COALESCE(normalized_hash, ''),
			revision_count, created_at, updated_at, deleted_at
		FROM observations
		WHERE deleted_at IS NULL
		ORDER BY id ASC`

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("read engram observations: %w", err)
	}
	defer rows.Close()

	var out []EngramObservation
	for rows.Next() {
		var (
			o         EngramObservation
			deletedAt sql.NullString
		)
		if err := rows.Scan(
			&o.SyncID, &o.Type, &o.Title, &o.Content,
			&o.Project, &o.Scope,
			&o.TopicKey,
			&o.NormalizedHash,
			&o.RevisionCount, &o.CreatedAt, &o.UpdatedAt,
			&deletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan observation row: %w", err)
		}
		// deletedAt should always be NULL here due to WHERE clause, but guard
		// defensively.
		if deletedAt.Valid {
			s := deletedAt.String
			o.DeletedAt = &s
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate observation rows: %w", err)
	}
	return out, nil
}
