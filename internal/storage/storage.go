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

	"github.com/AgusLoza2021/Thoughtline/internal/config"
	"github.com/AgusLoza2021/Thoughtline/internal/events"
	"github.com/AgusLoza2021/Thoughtline/internal/memory"

	_ "modernc.org/sqlite"
)

// ErrMemoryNotFound is returned by UpdateByID, SoftDelete, and other
// id-keyed mutations when no active (non-soft-deleted) row exists for the
// given id. Callers (and the server layer) should errors.Is against this
// sentinel to render a clean "not found" response.
var ErrMemoryNotFound = errors.New("storage: memory not found")

// ErrBrainRequired is returned when a per-brain storage method is called
// with brainID == 0. This is a programmer error — every write/read must
// be scoped to a specific brain.
var ErrBrainRequired = errors.New("storage: brainID must be non-zero")

// ErrBrainMismatch is returned when the memory.BrainID set by the caller
// does not match the brainID argument passed to Save. Always trust the arg;
// a mismatch indicates a bug in the caller.
var ErrBrainMismatch = errors.New("storage: memory.BrainID does not match brainID argument")

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
	db     *sql.DB
	now    func() time.Time
	brains brainCache
	bus    *events.Bus // optional; nil means no event emission
}

// SetBus attaches an event bus to the storage. After this call, every
// successful Save, UpdateByID, and SoftDelete emits the corresponding event.
// Safe to call at most once during app bootstrap. Passing nil is a no-op.
func (s *Storage) SetBus(bus *events.Bus) {
	s.bus = bus
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

	// Record v3 in the schema_version audit trail (INSERT OR IGNORE so
	// existing rows are preserved; v4 migration will add its own row).
	if _, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_version(version, applied_at) VALUES (3, ?)`,
		time.Now().UnixMilli()); err != nil {
		return fmt.Errorf("record schema v3: %w", err)
	}

	// v4 step: brains, global_config, memory_links tables + backfill.
	// Check whether v4 has already been applied by looking for the version row.
	var v4Applied int
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM schema_version WHERE version = 4`).Scan(&v4Applied)
	if v4Applied == 0 {
		if err := s.migrateV4(ctx); err != nil {
			return fmt.Errorf("migrate v4: %w", err)
		}
	}

	// v5 step: add 'rejected' to pending_events status CHECK via table recreate.
	var v5Applied int
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM schema_version WHERE version = 5`).Scan(&v5Applied)
	if v5Applied == 0 {
		if err := s.migrateV5(ctx); err != nil {
			return fmt.Errorf("migrate v5: %w", err)
		}
	}

	// v6 step: add idx_memories_hash partial covering index for DedupeCheck.
	var v6Applied int
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM schema_version WHERE version = 6`).Scan(&v6Applied)
	if v6Applied == 0 {
		if err := s.migrateV6(ctx); err != nil {
			return fmt.Errorf("migrate v6: %w", err)
		}
	}

	// Record / refresh the current schema version. INSERT OR IGNORE keeps the
	// first applied_at intact per row, leaving an audit trail per version.
	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_version(version, applied_at) VALUES (?, ?)`,
		currentSchemaVersion, time.Now().UnixMilli(),
	)
	return err
}

// migrateV5 recreates pending_events with 'rejected' added to the status
// CHECK constraint. Uses the SQLite 12-step rename-copy-drop sequence.
func (s *Storage) migrateV5(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Create the new table with the extended CHECK.
	if _, err := tx.ExecContext(ctx, schemaV5PendingEventsSQL); err != nil {
		return fmt.Errorf("v5 create pending_events_v5: %w", err)
	}

	// 2. Copy all existing rows.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pending_events_v5
		SELECT id, sync_id, project, session_id, event_type, tool_name, tool_use_id,
		       payload, event_hash, status, promoted_memory_id, promoted_at,
		       archived_at, created_at, captured_at
		FROM pending_events`); err != nil {
		return fmt.Errorf("v5 copy rows: %w", err)
	}

	// 3. Drop old table and rename new one. Indexes are dropped with the old table.
	if _, err := tx.ExecContext(ctx, `DROP TABLE pending_events`); err != nil {
		return fmt.Errorf("v5 drop old table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE pending_events_v5 RENAME TO pending_events`); err != nil {
		return fmt.Errorf("v5 rename: %w", err)
	}

	// 4. Recreate indexes (dropped with the old table).
	if _, err := tx.ExecContext(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_events_dedup
		    ON pending_events(project, event_hash)`); err != nil {
		return fmt.Errorf("v5 recreate dedup index: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_pending_events_triage
		    ON pending_events(project, status, captured_at DESC)`); err != nil {
		return fmt.Errorf("v5 recreate triage index: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_pending_events_session
		    ON pending_events(session_id, captured_at DESC)
		    WHERE session_id IS NOT NULL`); err != nil {
		return fmt.Errorf("v5 recreate session index: %w", err)
	}

	// 5. Record migration.
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_version(version, applied_at) VALUES (5, ?)`,
		time.Now().UnixMilli()); err != nil {
		return fmt.Errorf("v5 record version: %w", err)
	}

	return tx.Commit()
}

// migrateV6 creates idx_memories_hash — a partial covering index on
// (brain_id, normalized_hash, created_at) WHERE deleted_at IS NULL — used
// by DedupeCheck to locate duplicate observations without a full table scan.
// CREATE INDEX IF NOT EXISTS makes the index DDL idempotent; the
// schema_version INSERT OR IGNORE guard prevents double-application.
func (s *Storage) migrateV6(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, schemaV6IndexSQL); err != nil {
		return fmt.Errorf("v6 index: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_version(version, applied_at) VALUES (6, ?)`,
		time.Now().UnixMilli()); err != nil {
		return fmt.Errorf("v6 record version: %w", err)
	}

	return tx.Commit()
}

// migrateV4 creates the brains, global_config, and memory_links tables, adds
// memories.brain_id, seeds global_config, backfills brains from distinct
// memories.project values, and sets memories.brain_id. The entire operation
// runs in a single transaction so a failure rolls back cleanly to v3 state.
func (s *Storage) migrateV4(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Run v4 DDL (CREATE IF NOT EXISTS for new tables + indexes).
	if _, err := tx.ExecContext(ctx, schemaV4SQL); err != nil {
		return fmt.Errorf("v4 ddl: %w", err)
	}

	// 2. Add memories.brain_id if absent. PRAGMA-driven, idempotent.
	hasBrainID, err := columnExists(ctx, s.db, "memories", "brain_id")
	if err != nil {
		return fmt.Errorf("check brain_id column: %w", err)
	}
	if !hasBrainID {
		if _, err := tx.ExecContext(ctx,
			`ALTER TABLE memories ADD COLUMN brain_id INTEGER REFERENCES brains(id)`); err != nil {
			return fmt.Errorf("add brain_id: %w", err)
		}
	}
	// Create the brain_id index on memories (must be after ALTER TABLE adds the column).
	if _, err := tx.ExecContext(ctx, schemaV4MemoriesBrainIndexSQL); err != nil {
		return fmt.Errorf("v4 brain index: %w", err)
	}

	now := s.nowMillis().UnixMilli()

	// 3. Seed global_config singleton with compiled defaults if missing.
	// INSERT OR IGNORE preserves any pre-existing custom config (idempotency).
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO global_config(id, config_json, updated_at) VALUES (1, ?, ?)`,
		config.DefaultGlobalJSON(), now); err != nil {
		return fmt.Errorf("seed global_config: %w", err)
	}

	// 4. Backfill brains: one row per DISTINCT memories.project, kind='real'.
	// INSERT OR IGNORE so re-runs are no-ops.
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO brains (slug, display_name, kind, description,
		                              config_json, created_at, updated_at)
		SELECT project, project, 'real', 'Backfilled from v3 project',
		       '{}', ?, ?
		FROM memories
		WHERE project IS NOT NULL AND project <> ''
		GROUP BY project`, now, now); err != nil {
		return fmt.Errorf("backfill brains: %w", err)
	}

	// 5. UPDATE memories.brain_id from brains.slug.
	if _, err := tx.ExecContext(ctx, `
		UPDATE memories
		SET brain_id = (SELECT id FROM brains WHERE brains.slug = memories.project)
		WHERE brain_id IS NULL`); err != nil {
		return fmt.Errorf("backfill brain_id: %w", err)
	}

	// 6. FAIL LOUDLY if any active memory still has NULL brain_id.
	//    These are memories with NULL or empty project — unresolvable.
	var orphans int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM memories
		WHERE deleted_at IS NULL
		  AND (brain_id IS NULL OR project IS NULL OR project = '')`).Scan(&orphans); err != nil {
		return err
	}
	if orphans > 0 {
		ids := collectOrphanIDs(ctx, tx, 10)
		return fmt.Errorf("migrate v4: %d memories have NULL/empty project (ids: %v); cannot backfill brain_id", orphans, ids)
	}

	// 7. Bump schema_version.
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_version(version, applied_at) VALUES (4, ?)`,
		now); err != nil {
		return err
	}
	return tx.Commit()
}

// collectOrphanIDs returns up to n memory IDs that are active but have
// NULL brain_id or NULL/empty project. Used in migration failure messages.
func collectOrphanIDs(ctx context.Context, tx *sql.Tx, n int) []int64 {
	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM memories
		WHERE deleted_at IS NULL
		  AND (brain_id IS NULL OR project IS NULL OR project = '')
		LIMIT ?`, n)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
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
//
// brainID must be non-zero (returns ErrBrainRequired if zero).
// If m.BrainID is set and differs from brainID, returns ErrBrainMismatch.
// Storage always trusts the brainID argument; m.BrainID and m.Project are
// stored as-is for denorm compat but brain_id column is set from brainID arg.
func (s *Storage) Save(ctx context.Context, brainID int64, m memory.Memory) (memory.Memory, UpsertAction, error) {
	if brainID == 0 {
		return memory.Memory{}, 0, ErrBrainRequired
	}
	if m.BrainID != 0 && m.BrainID != brainID {
		return memory.Memory{}, 0, fmt.Errorf("%w: memory has %d, arg is %d",
			ErrBrainMismatch, m.BrainID, brainID)
	}
	if err := s.validateSessionLink(ctx, m); err != nil {
		return memory.Memory{}, 0, err
	}
	if m.TopicKey == "" {
		saved, err := s.insertNew(ctx, brainID, m)
		if err != nil {
			return memory.Memory{}, ActionCreated, err
		}
		return saved, ActionCreated, nil
	}
	return s.upsertByTopicKey(ctx, brainID, m)
}

func (s *Storage) insertNew(ctx context.Context, brainID int64, m memory.Memory) (memory.Memory, error) {
	now := s.nowMillis()
	syncID, err := newSyncID()
	if err != nil {
		return memory.Memory{}, err
	}

	hash := NormalizedHash(m.Title, m.Content)
	tagsJSON, err := encodeTags(m.Tags)
	if err != nil {
		return memory.Memory{}, err
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO memories (
			sync_id, project, scope, type, topic_key,
			title, content, tags, normalized_hash, revision_count,
			created_at, updated_at, session_id, brain_id
		) VALUES (?, ?, ?, ?, NULL, ?, ?, ?, ?, 0, ?, ?, ?, ?)`,
		syncID, m.Project, string(m.Scope), string(m.Type),
		m.Title, m.Content, tagsJSON, hash,
		now.UnixMilli(), now.UnixMilli(),
		nullIfEmpty(m.SessionID),
		brainID,
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
	m.BrainID = brainID
	m.NormalizedHash = hash
	m.RevisionCount = 0
	m.CreatedAt = now
	m.UpdatedAt = now
	if s.bus != nil {
		s.bus.Publish(events.MemoryCreated{Brain: brainID, MemoryID: id, SyncID: syncID, At: now})
	}
	return m, nil
}

func (s *Storage) upsertByTopicKey(ctx context.Context, brainID int64, m memory.Memory) (memory.Memory, UpsertAction, error) {
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
				created_at, updated_at, session_id, brain_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)`,
			syncID, m.Project, string(m.Scope), string(m.Type), m.TopicKey,
			m.Title, m.Content, tagsJSON, hash,
			now.UnixMilli(), now.UnixMilli(),
			nullIfEmpty(m.SessionID),
			brainID,
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
		m.BrainID = brainID
		m.NormalizedHash = hash
		m.RevisionCount = 0
		m.CreatedAt = now
		m.UpdatedAt = now
		if s.bus != nil {
			s.bus.Publish(events.MemoryCreated{Brain: brainID, MemoryID: newID, SyncID: syncID, At: now})
		}
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
	m.BrainID = brainID
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
	if s.bus != nil {
		s.bus.Publish(events.MemoryUpdated{Brain: brainID, MemoryID: existingID, SyncID: existingSyncID, At: now})
	}
	return m, ActionUpdated, nil
}

// GetByID fetches the memory with the given id, scoped to brainID.
// If the row exists but belongs to a different brain, ErrMemoryNotFound is
// returned — the existence of the row in another brain is not leaked.
func (s *Storage) GetByID(ctx context.Context, brainID int64, id int64) (memory.Memory, error) {
	m, err := s.getByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memory.Memory{}, ErrMemoryNotFound
		}
		return memory.Memory{}, err
	}
	if m.BrainID != brainID {
		return memory.Memory{}, ErrMemoryNotFound
	}
	return m, nil
}

// GetByIDUnscoped fetches the memory with the given id without enforcing
// brain isolation. Use only from the server layer for operations (delete,
// update, get-observation) that operate on a globally-unique memory ID and
// need to discover which brain owns the row before calling the
// brain-scoped API. Soft-deleted rows ARE returned (caller decides).
func (s *Storage) GetByIDUnscoped(ctx context.Context, id int64) (memory.Memory, error) {
	m, err := s.getByID(ctx, id)
	if err != nil {
		return memory.Memory{}, err
	}
	return m, nil
}

// getByID is the internal fetch used by Save's upsert path. It does NOT
// enforce cross-brain isolation (that is the caller's responsibility).
func (s *Storage) getByID(ctx context.Context, id int64) (memory.Memory, error) {
	var (
		m           memory.Memory
		topicKey    sql.NullString
		tagsJSON    sql.NullString
		createdAtMS int64
		updatedAtMS int64
		deletedAtMS sql.NullInt64
		sessionID   sql.NullString
		brainID     sql.NullInt64
	)
	row := s.db.QueryRowContext(ctx, `
		SELECT id, sync_id, project, scope, type, topic_key,
		       title, content, tags, normalized_hash, revision_count,
		       created_at, updated_at, deleted_at, session_id, brain_id
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
		&createdAtMS, &updatedAtMS, &deletedAtMS, &sessionID, &brainID,
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
	if brainID.Valid {
		m.BrainID = brainID.Int64
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

// NormalizedHash returns a SHA-256 fingerprint of (title, content) suitable
// for deduplication and noop detection on upsert. Whitespace at the very
// edges of each field is trimmed; everything else (case, internal whitespace)
// is significant.
//
// Recipe (INTERNAL — callers MUST NOT depend on the exact algorithm; it may
// change between binary versions): SHA-256(trimSpace(title) + NUL +
// trimSpace(content)), hex-encoded.
func NormalizedHash(title, content string) string {
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

// configDefaultGlobalJSON exposes config.DefaultGlobalJSON() to package-level
// tests without requiring them to import internal/config directly.
func configDefaultGlobalJSON() string { return config.DefaultGlobalJSON() }

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
