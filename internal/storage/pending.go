package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
)

// ErrPendingNotFound is returned when a pending_events lookup finds no row.
var ErrPendingNotFound = errors.New("storage: pending event not found")

// ErrAlreadyPromoted is returned when MarkPromoted is called on a row that is
// already in status=promoted.
var ErrAlreadyPromoted = errors.New("storage: pending event already promoted")

// ListPendingParams carries the optional filters for ListPending.
type ListPendingParams struct {
	Project   string
	Status    string    // "" = all statuses
	EventType string    // "" = all types
	Since     time.Time // zero = no lower bound
	Limit     int       // 0 = default (50)
	Offset    int
}

// SweepResult holds the counts from a SweepPending call.
type SweepResult struct {
	Archived int64
	Deleted  int64
}

// InsertPending inserts a new row into pending_events using INSERT OR IGNORE for
// idempotency. Returns ActionCreated on a new row, ActionNoop on a duplicate
// (project, event_hash).
func (s *Storage) InsertPending(ctx context.Context, ev pending.Event) (UpsertAction, error) {
	now := s.now()
	createdAt := ev.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	capturedAt := ev.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = now
	}

	syncID := ev.SyncID
	if syncID == "" {
		var err error
		syncID, err = newSyncID()
		if err != nil {
			return 0, fmt.Errorf("new sync_id: %w", err)
		}
	}

	status := ev.Status
	if status == "" {
		status = pending.StatusPending
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO pending_events
		(sync_id, project, session_id, event_type, tool_name, tool_use_id,
		 payload, event_hash, status, created_at, captured_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		syncID,
		ev.Project,
		nullIfEmpty(ev.SessionID),
		ev.EventType,
		nullIfEmpty(ev.ToolName),
		nullIfEmpty(ev.ToolUseID),
		ev.Payload,
		ev.Hash,
		string(status),
		createdAt.UnixMilli(),
		capturedAt.UnixMilli(),
	)
	if err != nil {
		return 0, fmt.Errorf("insert pending event: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ActionNoop, nil
	}
	return ActionCreated, nil
}

// ListPending returns pending events filtered by the given params, ordered by
// captured_at DESC.
func (s *Storage) ListPending(ctx context.Context, p ListPendingParams) ([]pending.Event, error) {
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}

	// Build query dynamically based on which filters are active.
	query := `
		SELECT id, sync_id, project, session_id, event_type, tool_name, tool_use_id,
		       payload, event_hash, status, promoted_memory_id, promoted_at,
		       archived_at, created_at, captured_at
		FROM pending_events
		WHERE project = ?`
	args := []any{p.Project}

	if p.Status != "" {
		query += " AND status = ?"
		args = append(args, p.Status)
	}
	if p.EventType != "" {
		query += " AND event_type = ?"
		args = append(args, p.EventType)
	}
	if !p.Since.IsZero() {
		query += " AND captured_at > ?"
		args = append(args, p.Since.UnixMilli())
	}
	query += " ORDER BY captured_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, p.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list pending: %w", err)
	}
	defer rows.Close()

	var events []pending.Event
	for rows.Next() {
		ev, err := scanPendingEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pending event: %w", err)
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

// GetPendingByID retrieves a single pending event by its integer ID, including
// the full payload. Returns ErrPendingNotFound if no row exists.
func (s *Storage) GetPendingByID(ctx context.Context, id int64) (pending.Event, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, sync_id, project, session_id, event_type, tool_name, tool_use_id,
		       payload, event_hash, status, promoted_memory_id, promoted_at,
		       archived_at, created_at, captured_at
		FROM pending_events
		WHERE id = ?`, id)

	ev, err := scanPendingEventRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return pending.Event{}, ErrPendingNotFound
	}
	return ev, err
}

// MarkPromoted transitions a pending event to promoted status and records the
// memory ID that was created. Returns ErrAlreadyPromoted if the row is already
// in promoted state.
func (s *Storage) MarkPromoted(ctx context.Context, pendingID, memoryID int64) error {
	// Check current status first.
	var status string
	err := s.db.QueryRowContext(ctx,
		`SELECT status FROM pending_events WHERE id = ?`, pendingID,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPendingNotFound
	}
	if err != nil {
		return fmt.Errorf("check status: %w", err)
	}
	if status == string(pending.StatusPromoted) {
		return ErrAlreadyPromoted
	}

	now := s.now()
	res, err := s.db.ExecContext(ctx, `
		UPDATE pending_events
		SET status = 'promoted', promoted_memory_id = ?, promoted_at = ?
		WHERE id = ? AND status = 'pending'`,
		memoryID, now.UnixMilli(), pendingID,
	)
	if err != nil {
		return fmt.Errorf("mark promoted: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		// Race condition or status changed between check and update.
		return ErrAlreadyPromoted
	}
	return nil
}

// SweepPending runs two operations in sequence using the provided 'now' as the
// reference clock (so tests can inject a fixed time):
//
//  1. Archive: pending rows with captured_at older than retentionDur → status=archived
//  2. Delete: archived rows with archived_at older than hardDeleteDur → DELETE
//
// Promoted rows are never touched.
func (s *Storage) SweepPending(ctx context.Context, retentionDur, hardDeleteDur time.Duration, now time.Time) (SweepResult, error) {
	retainCutoff := now.Add(-retentionDur).UnixMilli()
	deleteCutoff := now.Add(-hardDeleteDur).UnixMilli()
	archiveNow := now.UnixMilli()

	archiveRes, err := s.db.ExecContext(ctx, `
		UPDATE pending_events
		SET status = 'archived', archived_at = ?
		WHERE status = 'pending' AND captured_at < ?`,
		archiveNow, retainCutoff,
	)
	if err != nil {
		return SweepResult{}, fmt.Errorf("archive sweep: %w", err)
	}
	archived, _ := archiveRes.RowsAffected()

	deleteRes, err := s.db.ExecContext(ctx, `
		DELETE FROM pending_events
		WHERE status = 'archived' AND archived_at < ?`,
		deleteCutoff,
	)
	if err != nil {
		return SweepResult{}, fmt.Errorf("hard delete sweep: %w", err)
	}
	deleted, _ := deleteRes.RowsAffected()

	return SweepResult{Archived: archived, Deleted: deleted}, nil
}

// scanPendingEvent scans a row from a multi-row query into a pending.Event.
func scanPendingEvent(rows *sql.Rows) (pending.Event, error) {
	return scanPending(func(dest ...any) error { return rows.Scan(dest...) })
}

// scanPendingEventRow scans a single-row query into a pending.Event.
func scanPendingEventRow(row *sql.Row) (pending.Event, error) {
	return scanPending(func(dest ...any) error { return row.Scan(dest...) })
}

func scanPending(scan func(...any) error) (pending.Event, error) {
	var (
		ev            pending.Event
		sessionID     sql.NullString
		toolName      sql.NullString
		toolUseID     sql.NullString
		promotedMemID sql.NullInt64
		promotedAtMS  sql.NullInt64
		archivedAtMS  sql.NullInt64
		createdAtMS   int64
		capturedAtMS  int64
		status        string
	)
	if err := scan(
		&ev.ID, &ev.SyncID, &ev.Project, &sessionID, &ev.EventType,
		&toolName, &toolUseID, &ev.Payload, &ev.Hash, &status,
		&promotedMemID, &promotedAtMS, &archivedAtMS,
		&createdAtMS, &capturedAtMS,
	); err != nil {
		return pending.Event{}, err
	}
	ev.Status = pending.Status(status)
	ev.CreatedAt = time.UnixMilli(createdAtMS)
	ev.CapturedAt = time.UnixMilli(capturedAtMS)
	if sessionID.Valid {
		ev.SessionID = sessionID.String
	}
	if toolName.Valid {
		ev.ToolName = toolName.String
	}
	if toolUseID.Valid {
		ev.ToolUseID = toolUseID.String
	}
	if promotedMemID.Valid {
		ev.PromotedMemoryID = promotedMemID.Int64
	}
	if promotedAtMS.Valid {
		ev.PromotedAt = time.UnixMilli(promotedAtMS.Int64)
	}
	if archivedAtMS.Valid {
		ev.ArchivedAt = time.UnixMilli(archivedAtMS.Int64)
	}
	return ev, nil
}
