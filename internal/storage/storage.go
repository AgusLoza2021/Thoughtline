// Package storage owns persistence for Thoughtline.
//
// Backend: SQLite via modernc.org/sqlite (a pure-Go driver — no CGO, FTS5
// baked in). The default database file is opened with WAL journal mode and
// foreign keys on.
//
// Public API (M1):
//
//   Open(ctx, path) → *Storage
//   (*Storage).Close()
//   (*Storage).Save(ctx, memory.Memory) → (memory.Memory, UpsertAction, error)
//
// The single Save entry point dispatches:
//
//   - empty TopicKey ⇒ insert a brand-new row, ActionCreated.
//   - non-empty TopicKey ⇒ upsert keyed on (project, topic_key):
//       - first save ⇒ ActionCreated.
//       - re-save with changed content ⇒ UPDATE bumps revision_count, keeps
//         id / sync_id / created_at, ActionUpdated.
//       - re-save with identical content ⇒ ActionNoop, no row mutation.
package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"

	_ "modernc.org/sqlite"
)

// ErrMemoryNotFound is returned by UpdateByID, SoftDelete, and other
// id-keyed mutations when no active (non-soft-deleted) row exists for the
// given id. Callers (and the server layer) should errors.Is against this
// sentinel to render a clean "not found" response.
var ErrMemoryNotFound = errors.New("storage: memory not found")

// UpsertAction reports what Save did with the row.
type UpsertAction int

const (
	ActionCreated UpsertAction = iota
	ActionUpdated
	ActionNoop
)

func (a UpsertAction) String() string {
	switch a {
	case ActionCreated:
		return "created"
	case ActionUpdated:
		return "updated"
	case ActionNoop:
		return "noop"
	default:
		return "unknown"
	}
}

// Storage owns a SQLite connection pool and is safe for concurrent use.
type Storage struct {
	db  *sql.DB
	now func() time.Time
}

// Open opens or creates the database at path and runs migrations. Use ":memory:"
// for an ephemeral in-process database (intended for tests).
func Open(ctx context.Context, path string) (*Storage, error) {
	dsn := path
	if path != ":memory:" {
		// Enable WAL + FK + busy timeout via DSN query params.
		dsn = path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	// Tighten DB file permissions to 0600 (owner-only read/write).
	// SQLite creates the file with the OS umask, which on most Linux
	// distros leaves the file world-readable (0644). The DB can contain
	// raw user prompts and tool I/O via passive capture — keep it private.
	// On Windows os.Chmod only toggles the read-only bit, which is the
	// expected no-op for this case.
	if path != ":memory:" {
		if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
			// Log via fmt.Errorf chain but don't fail Open — restrictive
			// perms are best-effort. A user-visible failure here would
			// break first-run on filesystems that don't honour chmod.
			_ = err
		}
	}

	s := &Storage{db: db, now: time.Now}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// Close releases the underlying connection pool.
func (s *Storage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// SetClock replaces the storage's wall clock. Tests use this to make
// timestamps deterministic.
func (s *Storage) SetClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

// nowMillis returns the storage clock truncated to millisecond precision.
// Storage persists timestamps as unix epoch ms; truncating in-memory values
// keeps Save's returned Memory bit-equal to what GetByID would return.
func (s *Storage) nowMillis() time.Time {
	return time.UnixMilli(s.now().UnixMilli())
}

func (s *Storage) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return err
	}

	// v2 step: add memories.session_id column if it isn't already there.
	// SQLite has no ALTER TABLE ... IF NOT EXISTS, so we introspect via
	// PRAGMA table_info. This is idempotent — fresh DBs already pick up the
	// column from the alter, v1 DBs get it added here.
	hasSession, err := columnExists(ctx, s.db, "memories", "session_id")
	if err != nil {
		return fmt.Errorf("check session_id column: %w", err)
	}
	if !hasSession {
		if _, err := s.db.ExecContext(ctx, `
			ALTER TABLE memories ADD COLUMN session_id TEXT
				REFERENCES sessions(id) ON DELETE SET NULL`); err != nil {
			return fmt.Errorf("add session_id column: %w", err)
		}
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_memories_session
			ON memories(session_id)
			WHERE session_id IS NOT NULL`); err != nil {
		return fmt.Errorf("create session_id index: %w", err)
	}

	// Record / refresh the schema version. Using INSERT OR IGNORE keeps the
	// first applied_at intact per row; bumping currentSchemaVersion to a new
	// integer adds a separate row, leaving an audit trail of when each
	// version was first applied.
	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_version(version, applied_at) VALUES (?, ?)`,
		currentSchemaVersion, time.Now().UnixMilli(),
	)
	return err
}

// columnExists reports whether the given table has a column with the given
// name. Used by migrate() to keep ALTER TABLE idempotent.
func columnExists(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%q)`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			name      string
			typ       string
			notnull   int
			dflt      sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// Save persists m and returns the stored Memory (with id, sync_id, timestamps
// populated) along with the action performed. It assumes m has already passed
// memory.Validate; it does NOT re-validate the memory shape itself, but it DOES
// enforce the cross-table invariants that domain validation can't see — namely
// that an attached SessionID exists and belongs to the same project.
func (s *Storage) Save(ctx context.Context, m memory.Memory) (memory.Memory, UpsertAction, error) {
	if err := s.validateSessionLink(ctx, m); err != nil {
		return memory.Memory{}, 0, err
	}
	if m.TopicKey == "" {
		saved, err := s.insertNew(ctx, m)
		if err != nil {
			return memory.Memory{}, ActionCreated, err
		}
		return saved, ActionCreated, nil
	}
	return s.upsertByTopicKey(ctx, m)
}

func (s *Storage) insertNew(ctx context.Context, m memory.Memory) (memory.Memory, error) {
	now := s.nowMillis()
	syncID, err := newSyncID()
	if err != nil {
		return memory.Memory{}, err
	}

	hash := normalizedHash(m.Title, m.Content)
	tagsJSON, err := encodeTags(m.Tags)
	if err != nil {
		return memory.Memory{}, err
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO memories (
			sync_id, project, scope, type, topic_key,
			title, content, tags, normalized_hash, revision_count,
			created_at, updated_at, session_id
		) VALUES (?, ?, ?, ?, NULL, ?, ?, ?, ?, 0, ?, ?, ?)`,
		syncID, m.Project, string(m.Scope), string(m.Type),
		m.Title, m.Content, tagsJSON, hash,
		now.UnixMilli(), now.UnixMilli(),
		nullIfEmpty(m.SessionID),
	)
	if err != nil {
		return memory.Memory{}, fmt.Errorf("insert memory: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return memory.Memory{}, fmt.Errorf("last insert id: %w", err)
	}

	m.ID = id
	m.SyncID = syncID
	m.NormalizedHash = hash
	m.RevisionCount = 0
	m.CreatedAt = now
	m.UpdatedAt = now
	return m, nil
}

func (s *Storage) upsertByTopicKey(ctx context.Context, m memory.Memory) (memory.Memory, UpsertAction, error) {
	hash := normalizedHash(m.Title, m.Content)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return memory.Memory{}, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // safe no-op after Commit

	var (
		existingID            int64
		existingSyncID        string
		existingHash          string
		existingRevision      int
		existingCreatedAtMS   int64
		existingSessionID     sql.NullString
	)
	err = tx.QueryRowContext(ctx, `
		SELECT id, sync_id, normalized_hash, revision_count, created_at, session_id
		FROM memories
		WHERE project = ? AND topic_key = ? AND deleted_at IS NULL`,
		m.Project, m.TopicKey,
	).Scan(&existingID, &existingSyncID, &existingHash, &existingRevision, &existingCreatedAtMS, &existingSessionID)

	now := s.nowMillis()
	tagsJSON, encErr := encodeTags(m.Tags)
	if encErr != nil {
		return memory.Memory{}, 0, encErr
	}

	switch {
	case errors.Is(err, sql.ErrNoRows):
		// First time we see this (project, topic_key) — insert.
		syncID, idErr := newSyncID()
		if idErr != nil {
			return memory.Memory{}, 0, idErr
		}
		res, ierr := tx.ExecContext(ctx, `
			INSERT INTO memories (
				sync_id, project, scope, type, topic_key,
				title, content, tags, normalized_hash, revision_count,
				created_at, updated_at, session_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?)`,
			syncID, m.Project, string(m.Scope), string(m.Type), m.TopicKey,
			m.Title, m.Content, tagsJSON, hash,
			now.UnixMilli(), now.UnixMilli(),
			nullIfEmpty(m.SessionID),
		)
		if ierr != nil {
			return memory.Memory{}, 0, fmt.Errorf("insert memory: %w", ierr)
		}
		newID, lerr := res.LastInsertId()
		if lerr != nil {
			return memory.Memory{}, 0, lerr
		}
		if cerr := tx.Commit(); cerr != nil {
			return memory.Memory{}, 0, cerr
		}
		m.ID = newID
		m.SyncID = syncID
		m.NormalizedHash = hash
		m.RevisionCount = 0
		m.CreatedAt = now
		m.UpdatedAt = now
		return m, ActionCreated, nil

	case err != nil:
		return memory.Memory{}, 0, fmt.Errorf("lookup topic_key: %w", err)
	}

	// Found an existing row.
	if existingHash == hash {
		// Identical content — return the stored row unchanged.
		if cerr := tx.Commit(); cerr != nil {
			return memory.Memory{}, 0, cerr
		}
		stored, gerr := s.getByID(ctx, existingID)
		if gerr != nil {
			return memory.Memory{}, 0, gerr
		}
		return stored, ActionNoop, nil
	}

	// Real update — bump revision_count and refresh updated_at. session_id
	// is sticky: an upsert that omits SessionID preserves the prior value
	// (COALESCE returns the second arg when the first is NULL). To overwrite,
	// the caller must pass an explicit SessionID. To clear, today there is no
	// path — by design — once a memory is attached to a session it stays
	// historically attached. (See M4 design notes in PROGRESS.md.)
	_, uerr := tx.ExecContext(ctx, `
		UPDATE memories
		SET title = ?, content = ?, tags = ?, normalized_hash = ?,
		    revision_count = revision_count + 1, updated_at = ?,
		    type = ?, scope = ?,
		    session_id = COALESCE(?, session_id)
		WHERE id = ?`,
		m.Title, m.Content, tagsJSON, hash,
		now.UnixMilli(),
		string(m.Type), string(m.Scope),
		nullIfEmpty(m.SessionID),
		existingID,
	)
	if uerr != nil {
		return memory.Memory{}, 0, fmt.Errorf("update memory: %w", uerr)
	}
	if cerr := tx.Commit(); cerr != nil {
		return memory.Memory{}, 0, cerr
	}

	m.ID = existingID
	m.SyncID = existingSyncID
	m.NormalizedHash = hash
	m.RevisionCount = existingRevision + 1
	m.CreatedAt = time.UnixMilli(existingCreatedAtMS)
	m.UpdatedAt = now
	// Mirror what COALESCE(?, session_id) wrote to the row: if the caller
	// provided no SessionID, the existing one survives — propagate that into
	// the returned struct so callers see what's actually in the DB.
	if m.SessionID == "" && existingSessionID.Valid {
		m.SessionID = existingSessionID.String
	}
	return m, ActionUpdated, nil
}

// GetByID is exported for tests — production callers use Save.
func (s *Storage) GetByID(ctx context.Context, id int64) (memory.Memory, error) {
	return s.getByID(ctx, id)
}

func (s *Storage) getByID(ctx context.Context, id int64) (memory.Memory, error) {
	var (
		m              memory.Memory
		topicKey       sql.NullString
		tagsJSON       sql.NullString
		createdAtMS    int64
		updatedAtMS    int64
		deletedAtMS    sql.NullInt64
		sessionID      sql.NullString
	)
	row := s.db.QueryRowContext(ctx, `
		SELECT id, sync_id, project, scope, type, topic_key,
		       title, content, tags, normalized_hash, revision_count,
		       created_at, updated_at, deleted_at, session_id
		FROM memories
		WHERE id = ?`,
		id,
	)
	var (
		scope string
		typ   string
	)
	if err := row.Scan(
		&m.ID, &m.SyncID, &m.Project, &scope, &typ, &topicKey,
		&m.Title, &m.Content, &tagsJSON, &m.NormalizedHash, &m.RevisionCount,
		&createdAtMS, &updatedAtMS, &deletedAtMS, &sessionID,
	); err != nil {
		return memory.Memory{}, err
	}
	m.Scope = memory.Scope(scope)
	m.Type = memory.Type(typ)
	if topicKey.Valid {
		m.TopicKey = topicKey.String
	}
	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &m.Tags); err != nil {
			return memory.Memory{}, fmt.Errorf("decode tags: %w", err)
		}
	}
	m.CreatedAt = time.UnixMilli(createdAtMS)
	m.UpdatedAt = time.UnixMilli(updatedAtMS)
	if deletedAtMS.Valid {
		t := time.UnixMilli(deletedAtMS.Int64)
		m.DeletedAt = &t
	}
	if sessionID.Valid {
		m.SessionID = sessionID.String
	}
	return m, nil
}

// FTSCount returns the row count of the memories_fts virtual table. Used by
// tests to verify the FTS5 sync triggers fired correctly.
func (s *Storage) FTSCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM memories_fts`).Scan(&n)
	return n, err
}

// FTSContains returns true if the FTS5 index has at least one row matching q.
// Used by tests to verify fresh content is searchable after Save.
func (s *Storage) FTSContains(ctx context.Context, q string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FROM memories_fts WHERE memories_fts MATCH ?`,
		q,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func newSyncID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("uuidv7: %w", err)
	}
	return id.String(), nil
}

// normalizedHash returns a sha256 fingerprint of (title, content) suitable
// for noop detection on upsert. Whitespace at the very edges of each field
// is trimmed; everything else (case, internal whitespace) is significant.
func normalizedHash(title, content string) string {
	h := sha256.New()
	h.Write([]byte(strings.TrimSpace(title)))
	h.Write([]byte{0})
	h.Write([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(h.Sum(nil))
}

// nullIfEmpty maps "" → SQL NULL, anything else → a valid string. Used at
// every Save/upsert site that handles the optional session_id column so the
// SQL stays referentially honest (NULL for "no session", a valid UUIDv7
// otherwise — the foreign key into sessions(id) requires that distinction).
func nullIfEmpty(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func encodeTags(tags []string) (sql.NullString, error) {
	if len(tags) == 0 {
		return sql.NullString{Valid: false}, nil
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return sql.NullString{}, fmt.Errorf("encode tags: %w", err)
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}
