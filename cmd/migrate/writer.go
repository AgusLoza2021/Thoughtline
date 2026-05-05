package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// writeRow persists one migrated memory to the Thoughtline database.
//
// It performs three-step idempotent write:
//  1. Pre-check: if sync_id already exists → skip (duplicate re-run safety)
//  2. Pre-check: if (project, topic_key) already exists in Thoughtline → skip
//     (prevents overwriting Thoughtline-native data — design.md Medium risk)
//  3. storage.Save() — writes the row, generates a fresh UUIDv7 sync_id
//  4. Post-UPDATE — overwrites the auto-generated sync_id with the Engram
//     sync_id so provenance is preserved 1:1
//
// rawDB is a raw *sql.DB to the same thoughtline.db file. storage.Storage
// does not expose its internal *sql.DB, so the caller opens a second handle.
// This is safe under WAL mode — SQLite supports multiple readers/writers
// to the same file in WAL journal mode.
func writeRow(ctx context.Context, st *storage.Storage, rawDB *sql.DB, m memory.Memory, engramSyncID string) (RowResult, error) {
	result := RowResult{SyncID: engramSyncID}

	// Step 1 — duplicate sync_id pre-check.
	var dummy int
	err := rawDB.QueryRowContext(ctx,
		"SELECT 1 FROM memories WHERE sync_id = ?", engramSyncID,
	).Scan(&dummy)
	if err == nil {
		// Row with this sync_id already in destination.
		result.Action = "skipped-duplicate"
		result.Reason = "sync_id already exists"
		return result, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return result, fmt.Errorf("pre-check sync_id %s: %w", engramSyncID, err)
	}

	// Step 2 — (project, topic_key) collision pre-check.
	// Only relevant when the migrated row has a topic_key — empty topic_keys
	// always INSERT rather than UPSERT in storage.Save().
	if m.TopicKey != "" {
		err = rawDB.QueryRowContext(ctx,
			"SELECT 1 FROM memories WHERE project = ? AND topic_key = ? AND deleted_at IS NULL",
			m.Project, m.TopicKey,
		).Scan(&dummy)
		if err == nil {
			// A Thoughtline-native row with this (project, topic_key) exists.
			// Skipping protects the existing row from being silently overwritten.
			result.Action = "skipped-topic-collision"
			result.Reason = fmt.Sprintf("topic_key %q already exists for project %q", m.TopicKey, m.Project)
			return result, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return result, fmt.Errorf("pre-check topic_key collision for %s: %w", engramSyncID, err)
		}
	}

	// Step 3 — storage.Save() writes the row and generates a fresh UUIDv7.
	saved, action, err := st.Save(ctx, m)
	if err != nil {
		return result, fmt.Errorf("storage.Save for %s: %w", engramSyncID, err)
	}
	if action == storage.ActionNoop {
		// Content hash collision — storage treated it as a no-op. This can
		// happen if two Engram rows have identical content. Treat as duplicate.
		result.Action = "skipped-duplicate"
		result.Reason = "content hash matched existing row (ActionNoop)"
		return result, nil
	}

	// Step 4 — overwrite the auto-generated UUIDv7 with the Engram sync_id.
	// We use the row's id from the Save result for precision — last_insert_rowid()
	// is race-prone when two connections share the WAL file.
	_, err = rawDB.ExecContext(ctx,
		"UPDATE memories SET sync_id = ? WHERE id = ?",
		engramSyncID, saved.ID,
	)
	if err != nil {
		return result, fmt.Errorf("post-update sync_id for %s (id=%d): %w", engramSyncID, saved.ID, err)
	}

	result.Action = "created"
	return result, nil
}
