package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// StatsOptions configures a Stats() call.
type StatsOptions struct {
	// Project, when non-empty, scopes ALL counts to that project. Sessions
	// and memories from other projects are excluded entirely.
	Project string
	// RecentLimit caps how many recent memories AND sessions are included
	// in the snapshot. <=0 means DefaultSearchLimit (10), >MaxSearchLimit
	// is clamped to MaxSearchLimit (50).
	RecentLimit int
}

// Stats is a snapshot of the database's contents — counts and the most
// recent items — designed to drive the dashboard UI and the tl_stats MCP
// tool. All fields reflect "active" state (deleted_at IS NULL) except
// DeletedMemories which is the count of soft-deleted rows.
type Stats struct {
	GeneratedAt time.Time

	TotalMemories   int
	DeletedMemories int

	ByType    map[memory.Type]int
	ByProject map[string]int
	ByScope   map[memory.Scope]int

	OpenSessions   int
	ClosedSessions int

	RecentMemories []SearchResult
	RecentSessions []memory.Session
}

// Stats returns a snapshot of the database for dashboards. Errors propagate
// from the underlying queries; partial Stats are never returned.
func (s *Storage) Stats(ctx context.Context, opts StatsOptions) (Stats, error) {
	limit := opts.RecentLimit
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}

	out := Stats{
		GeneratedAt: s.nowMillis(),
		ByType:      make(map[memory.Type]int),
		ByProject:   make(map[string]int),
		ByScope:     make(map[memory.Scope]int),
	}

	// Memory totals + soft-deleted count, optionally scoped to a project.
	if err := s.queryMemoryCounts(ctx, opts.Project, &out); err != nil {
		return Stats{}, err
	}
	if err := s.queryMemoryGroupings(ctx, opts.Project, &out); err != nil {
		return Stats{}, err
	}
	if err := s.querySessionCounts(ctx, opts.Project, &out); err != nil {
		return Stats{}, err
	}

	// Recent memories: project-scoped if requested, all-projects otherwise.
	recent, err := s.recentMemoriesForStats(ctx, opts.Project, limit)
	if err != nil {
		return Stats{}, err
	}
	out.RecentMemories = recent

	recentSess, err := s.recentSessionsForStats(ctx, opts.Project, limit)
	if err != nil {
		return Stats{}, err
	}
	out.RecentSessions = recentSess

	return out, nil
}

func (s *Storage) queryMemoryCounts(ctx context.Context, project string, out *Stats) error {
	args := []any{}
	where := "1=1"
	if project != "" {
		where = "project = ?"
		args = append(args, project)
	}

	row := s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT count(*) FROM memories WHERE %s AND deleted_at IS NULL`, where),
		args...,
	)
	if err := row.Scan(&out.TotalMemories); err != nil {
		return fmt.Errorf("count active memories: %w", err)
	}

	row = s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT count(*) FROM memories WHERE %s AND deleted_at IS NOT NULL`, where),
		args...,
	)
	if err := row.Scan(&out.DeletedMemories); err != nil {
		return fmt.Errorf("count deleted memories: %w", err)
	}
	return nil
}

func (s *Storage) queryMemoryGroupings(ctx context.Context, project string, out *Stats) error {
	args := []any{}
	where := "1=1"
	if project != "" {
		where = "project = ?"
		args = append(args, project)
	}

	// By type.
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT type, count(*) FROM memories WHERE %s AND deleted_at IS NULL GROUP BY type`, where),
		args...,
	)
	if err != nil {
		return fmt.Errorf("group by type: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var typ string
		var n int
		if err := rows.Scan(&typ, &n); err != nil {
			return fmt.Errorf("scan type: %w", err)
		}
		out.ByType[memory.Type(typ)] = n
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// By project.
	rows2, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT project, count(*) FROM memories WHERE %s AND deleted_at IS NULL GROUP BY project`, where),
		args...,
	)
	if err != nil {
		return fmt.Errorf("group by project: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var p string
		var n int
		if err := rows2.Scan(&p, &n); err != nil {
			return fmt.Errorf("scan project: %w", err)
		}
		out.ByProject[p] = n
	}
	if err := rows2.Err(); err != nil {
		return err
	}

	// By scope.
	rows3, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT scope, count(*) FROM memories WHERE %s AND deleted_at IS NULL GROUP BY scope`, where),
		args...,
	)
	if err != nil {
		return fmt.Errorf("group by scope: %w", err)
	}
	defer rows3.Close()
	for rows3.Next() {
		var sc string
		var n int
		if err := rows3.Scan(&sc, &n); err != nil {
			return fmt.Errorf("scan scope: %w", err)
		}
		out.ByScope[memory.Scope(sc)] = n
	}
	return rows3.Err()
}

func (s *Storage) querySessionCounts(ctx context.Context, project string, out *Stats) error {
	args := []any{}
	where := "1=1"
	if project != "" {
		where = "project = ?"
		args = append(args, project)
	}

	row := s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT count(*) FROM sessions WHERE %s AND ended_at IS NULL`, where),
		args...,
	)
	if err := row.Scan(&out.OpenSessions); err != nil {
		return fmt.Errorf("count open sessions: %w", err)
	}

	row = s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT count(*) FROM sessions WHERE %s AND ended_at IS NOT NULL`, where),
		args...,
	)
	if err := row.Scan(&out.ClosedSessions); err != nil {
		return fmt.Errorf("count closed sessions: %w", err)
	}
	return nil
}

// recentMemoriesForStats returns up to `limit` of the most recently updated
// active memories, optionally scoped to a project. Same envelope as Recent()
// so dashboard rendering is uniform.
func (s *Storage) recentMemoriesForStats(ctx context.Context, project string, limit int) ([]SearchResult, error) {
	if project != "" {
		return s.Recent(ctx, project, limit)
	}
	// Cross-project recency. Soft-deleted excluded.
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, sync_id, project, scope, type, topic_key,
		       title, content, tags, revision_count, updated_at
		FROM memories
		WHERE deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent memories cross-project: %w", err)
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
		out = append(out, r)
	}
	return out, rows.Err()
}

// recentSessionsForStats returns up to `limit` of the most recently STARTED
// sessions. Project-scoped or cross-project depending on the option.
func (s *Storage) recentSessionsForStats(ctx context.Context, project string, limit int) ([]memory.Session, error) {
	if project != "" {
		return s.RecentSessions(ctx, project, limit)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project, agent_label, started_at, ended_at, summary
		FROM sessions
		ORDER BY started_at DESC
		LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent sessions cross-project: %w", err)
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
			return nil, fmt.Errorf("scan session row: %w", err)
		}
		sess.StartedAt = time.UnixMilli(startedMS)
		if endedMS.Valid {
			t := time.UnixMilli(endedMS.Int64)
			sess.EndedAt = &t
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}
