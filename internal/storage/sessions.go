package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// Session-related sentinels. Tests and the server layer match against
// these via errors.Is.
var (
	ErrSessionNotFound        = errors.New("storage: session not found")
	ErrSessionAlreadyEnded    = errors.New("storage: session already ended")
	ErrSessionProjectMismatch = errors.New("storage: memory.Project does not match session.Project")
)

// StartSession creates a fresh open session for the given project. agentLabel
// is optional (empty string skips it). Returns the persisted Session with
// its UUIDv7 id and started_at populated.
func (s *Storage) StartSession(ctx context.Context, project, agentLabel string) (memory.Session, error) {
	sess := memory.Session{
		Project:    project,
		AgentLabel: agentLabel,
		StartedAt:  s.nowMillis(),
	}
	// Pre-validate (project required, agent_label length, etc.) before we
	// hit storage. ID stays empty here — domain validation allows that and
	// storage assigns the UUIDv7 below.
	if err := memory.ValidateSession(sess); err != nil {
		return memory.Session{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return memory.Session{}, fmt.Errorf("uuidv7: %w", err)
	}
	sess.ID = id.String()

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, project, agent_label, started_at, ended_at, summary)
		VALUES (?, ?, ?, ?, NULL, '')`,
		sess.ID, sess.Project, sess.AgentLabel, sess.StartedAt.UnixMilli(),
	); err != nil {
		return memory.Session{}, fmt.Errorf("insert session: %w", err)
	}
	return sess, nil
}

// EndSession closes the session by setting ended_at and persisting the
// summary. A session can only be ended once: subsequent calls return
// ErrSessionAlreadyEnded. Summary length is bounded by
// memory.MaxSessionSummaryBytes.
func (s *Storage) EndSession(ctx context.Context, id, summary string) (memory.Session, error) {
	current, err := s.GetSession(ctx, id)
	if err != nil {
		return memory.Session{}, err
	}
	if current.EndedAt != nil {
		return memory.Session{}, ErrSessionAlreadyEnded
	}
	// Validate the merged shape before writing — catches oversized summary.
	now := s.nowMillis()
	merged := current
	merged.EndedAt = &now
	merged.Summary = summary
	if err := memory.ValidateSession(merged); err != nil {
		return memory.Session{}, err
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE sessions SET ended_at = ?, summary = ?
		WHERE id = ? AND ended_at IS NULL`,
		now.UnixMilli(), summary, id,
	)
	if err != nil {
		return memory.Session{}, fmt.Errorf("update session: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return memory.Session{}, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		// Race: someone else closed it between our GetSession and UPDATE.
		return memory.Session{}, ErrSessionAlreadyEnded
	}
	return merged, nil
}

// GetSession fetches a single session by id. Returns ErrSessionNotFound when
// no row matches.
func (s *Storage) GetSession(ctx context.Context, id string) (memory.Session, error) {
	var (
		sess      memory.Session
		startedMS int64
		endedMS   sql.NullInt64
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project, agent_label, started_at, ended_at, summary
		FROM sessions
		WHERE id = ?`, id,
	).Scan(&sess.ID, &sess.Project, &sess.AgentLabel, &startedMS, &endedMS, &sess.Summary)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memory.Session{}, ErrSessionNotFound
		}
		return memory.Session{}, fmt.Errorf("get session: %w", err)
	}
	sess.StartedAt = time.UnixMilli(startedMS)
	if endedMS.Valid {
		t := time.UnixMilli(endedMS.Int64)
		sess.EndedAt = &t
	}
	return sess, nil
}

// RecentSessions returns the most recently STARTED sessions for the given
// project, ordered by started_at DESC. Mirrors Recent() for memories: project
// required (refuses empty), limit clamped to [DefaultSearchLimit, MaxSearchLimit].
func (s *Storage) RecentSessions(ctx context.Context, project string, limit int) ([]memory.Session, error) {
	if project == "" {
		return nil, errors.New("storage: RecentSessions requires a non-empty project")
	}
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project, agent_label, started_at, ended_at, summary
		FROM sessions
		WHERE project = ?
		ORDER BY started_at DESC
		LIMIT ?`,
		project, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent sessions: %w", err)
	}
	defer rows.Close()

	var out []memory.Session
	for rows.Next() {
		var (
			sess      memory.Session
			startedMS int64
			endedMS   sql.NullInt64
		)
		if err := rows.Scan(
			&sess.ID, &sess.Project, &sess.AgentLabel,
			&startedMS, &endedMS, &sess.Summary,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sess.StartedAt = time.UnixMilli(startedMS)
		if endedMS.Valid {
			t := time.UnixMilli(endedMS.Int64)
			sess.EndedAt = &t
		}
		out = append(out, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return out, nil
}

// validateSessionLink checks that the optional m.SessionID points to an
// existing session in the same project. Called by Save before insert/upsert
// when m.SessionID != "". Empty SessionID is treated as "no session" and
// skips the check entirely.
func (s *Storage) validateSessionLink(ctx context.Context, m memory.Memory) error {
	if m.SessionID == "" {
		return nil
	}
	sess, err := s.GetSession(ctx, m.SessionID)
	if err != nil {
		return err // ErrSessionNotFound or wrapped scan error
	}
	if sess.Project != m.Project {
		return fmt.Errorf("%w: memory project %q vs session project %q",
			ErrSessionProjectMismatch, m.Project, sess.Project)
	}
	return nil
}
