package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MemoryLink represents a row in memory_links.
type MemoryLink struct {
	ID        int64
	BrainID   int64
	FromID    int64
	ToID      int64
	Relation  string
	Weight    float64
	Source    string
	Note      string
	CreatedAt time.Time
}

// CreateLink inserts a memory_links row. Returns (link, false, nil) on success,
// (zero, true, nil) if the link already exists (UNIQUE conflict = noop).
func (s *Storage) CreateLink(ctx context.Context, brainID, fromID, toID int64, relation, note string) (MemoryLink, bool, error) {
	now := s.nowMillis()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO memory_links (brain_id, from_id, to_id, relation, weight, source, note, created_at)
		VALUES (?, ?, ?, ?, 1.0, 'manual', ?, ?)`,
		brainID, fromID, toID, relation, note, now.UnixMilli(),
	)
	if err != nil {
		// SQLite UNIQUE constraint violation: "UNIQUE constraint failed: memory_links.from_id, memory_links.to_id, memory_links.relation"
		if isUniqueConflict(err) {
			return MemoryLink{}, true, nil
		}
		return MemoryLink{}, false, fmt.Errorf("create link: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return MemoryLink{}, false, fmt.Errorf("create link last insert id: %w", err)
	}
	return MemoryLink{
		ID:        id,
		BrainID:   brainID,
		FromID:    fromID,
		ToID:      toID,
		Relation:  relation,
		Weight:    1.0,
		Source:    "manual",
		Note:      note,
		CreatedAt: now,
	}, false, nil
}

// GetLinks returns all links for a memory. direction: "from" = only from_id=id,
// "to" = only to_id=id, "" = both.
func (s *Storage) GetLinks(ctx context.Context, brainID, memoryID int64, direction string) ([]MemoryLink, error) {
	var query string
	var args []any

	switch direction {
	case "from":
		query = `SELECT id, brain_id, from_id, to_id, relation, weight, source, note, created_at
		         FROM memory_links WHERE brain_id = ? AND from_id = ? ORDER BY created_at`
		args = []any{brainID, memoryID}
	case "to":
		query = `SELECT id, brain_id, from_id, to_id, relation, weight, source, note, created_at
		         FROM memory_links WHERE brain_id = ? AND to_id = ? ORDER BY created_at`
		args = []any{brainID, memoryID}
	default:
		query = `SELECT id, brain_id, from_id, to_id, relation, weight, source, note, created_at
		         FROM memory_links WHERE brain_id = ? AND (from_id = ? OR to_id = ?) ORDER BY created_at`
		args = []any{brainID, memoryID, memoryID}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get links: %w", err)
	}
	defer rows.Close()

	var links []MemoryLink
	for rows.Next() {
		var l MemoryLink
		var createdAtMS int64
		if err := rows.Scan(&l.ID, &l.BrainID, &l.FromID, &l.ToID, &l.Relation,
			&l.Weight, &l.Source, &l.Note, &createdAtMS); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		l.CreatedAt = time.UnixMilli(createdAtMS)
		links = append(links, l)
	}
	return links, rows.Err()
}

// isUniqueConflict reports whether err is a SQLite UNIQUE constraint violation.
// modernc.org/sqlite wraps the error in a string that contains "UNIQUE constraint failed".
func isUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	// Check by unwrapping to see if it's a sqlite error code 2067 (SQLITE_CONSTRAINT_UNIQUE)
	// or by string match as a fallback (modernc driver uses string errors).
	type sqliteErr interface {
		Code() int
	}
	var se sqliteErr
	if errors.As(err, &se) {
		return se.Code() == 2067
	}
	// Fallback: string match. Both mattn and modernc drivers include this phrase.
	return containsUniqueMsg(err.Error())
}

func containsUniqueMsg(msg string) bool {
	// "UNIQUE constraint failed" is the standard SQLite error text.
	for i := 0; i+23 <= len(msg); i++ {
		if msg[i:i+6] == "UNIQUE" {
			return true
		}
	}
	return false
}

// DeleteLink removes a specific link by ID, scoped to brainID.
// Returns sql.ErrNoRows if not found.
func (s *Storage) DeleteLink(ctx context.Context, brainID, linkID int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM memory_links WHERE id = ? AND brain_id = ?`, linkID, brainID)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
