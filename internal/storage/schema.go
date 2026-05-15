package storage

// schemaSQL is the full DDL for a fresh Thoughtline database.
//
// Layout sketch is documented in docs/ARCHITECTURE.md and ADR 0002. Two
// load-bearing properties:
//
//  1. The `embedding*` columns are reserved for M5 — never written by v1.
//     Including them now lets M5 add semantics additively, no migration.
//  2. memories_fts is a contentless FTS5 virtual table. Three triggers keep
//     it in lockstep with the canonical `memories` table. The pattern is
//     copied from engram (internal/store/store.go) and from the official
//     SQLite FTS5 docs.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS memories (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id         TEXT    NOT NULL UNIQUE,
    project         TEXT    NOT NULL,
    scope           TEXT    NOT NULL CHECK (scope IN ('project','personal')),
    type            TEXT    NOT NULL,
    topic_key       TEXT,
    title           TEXT    NOT NULL,
    content         TEXT    NOT NULL,
    tags            TEXT,
    normalized_hash TEXT    NOT NULL,
    revision_count  INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    deleted_at      INTEGER,

    embedding             BLOB,
    embedding_model       TEXT,
    embedding_created_at  INTEGER
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_memories_topic
    ON memories(project, topic_key)
    WHERE topic_key IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_memories_recent
    ON memories(project, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    title,
    content,
    content       = memories,
    content_rowid = id,
    tokenize      = 'unicode61 remove_diacritics 2'
);

CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(rowid, title, content)
        VALUES (new.id, new.title, new.content);
END;

CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, title, content)
        VALUES('delete', old.id, old.title, old.content);
    INSERT INTO memories_fts(rowid, title, content)
        VALUES (new.id, new.title, new.content);
END;

CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, title, content)
        VALUES('delete', old.id, old.title, old.content);
END;

CREATE TABLE IF NOT EXISTS schema_version (
    version    INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);

-- v2: sessions bookend coding interactions. CREATE IF NOT EXISTS makes this
-- safe to run on both fresh databases and ones that were initially v1.
-- The session_id column on memories is added separately by migrateV2 because
-- ALTER TABLE has no IF NOT EXISTS variant in SQLite.
CREATE TABLE IF NOT EXISTS sessions (
    id           TEXT    PRIMARY KEY,
    project      TEXT    NOT NULL,
    agent_label  TEXT    NOT NULL DEFAULT '',
    started_at   INTEGER NOT NULL,
    ended_at     INTEGER,
    summary      TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sessions_recent
    ON sessions(project, started_at DESC);

-- v3: passive capture queue. pending_events holds raw hook payloads from
-- Claude Code until the model promotes them to memories via tl_promote.
-- They are NEVER surfaced by tl_search — they live in this separate table only.
CREATE TABLE IF NOT EXISTS pending_events (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id             TEXT    NOT NULL UNIQUE,
    project             TEXT    NOT NULL,
    session_id          TEXT,
    event_type          TEXT    NOT NULL,
    tool_name           TEXT,
    tool_use_id         TEXT,
    payload             TEXT    NOT NULL,
    event_hash          TEXT    NOT NULL,
    status              TEXT    NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending','promoted','archived')),
    promoted_memory_id  INTEGER,
    promoted_at         INTEGER,
    archived_at         INTEGER,
    created_at          INTEGER NOT NULL,
    captured_at         INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_events_dedup
    ON pending_events(project, event_hash);

CREATE INDEX IF NOT EXISTS idx_pending_events_triage
    ON pending_events(project, status, captured_at DESC);

CREATE INDEX IF NOT EXISTS idx_pending_events_session
    ON pending_events(session_id, captured_at DESC)
    WHERE session_id IS NOT NULL;
`

// v4 DDL is appended to schemaSQL. All new statements use CREATE … IF NOT EXISTS
// so the concatenated Exec is idempotent on every Open.
const schemaV4SQL = `
CREATE TABLE IF NOT EXISTS brains (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    slug          TEXT    NOT NULL,
    display_name  TEXT    NOT NULL,
    kind          TEXT    NOT NULL CHECK (kind IN ('real','synthetic','sandbox')),
    description   TEXT    NOT NULL DEFAULT '',
    config_json   TEXT    NOT NULL DEFAULT '{}',
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    archived_at   INTEGER
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_brains_slug_active
    ON brains(slug)
    WHERE archived_at IS NULL;

CREATE TABLE IF NOT EXISTS global_config (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    config_json TEXT    NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS memory_links (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    brain_id   INTEGER NOT NULL REFERENCES brains(id)   ON DELETE CASCADE,
    from_id    INTEGER NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    to_id      INTEGER NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    relation   TEXT    NOT NULL CHECK (relation IN (
                  'supersedes','contradicts','refines','depends_on',
                  'references','related','derived_from')),
    weight     REAL    NOT NULL DEFAULT 1.0,
    source     TEXT    NOT NULL DEFAULT 'manual'
                       CHECK (source IN ('manual','auto','imported')),
    note       TEXT    NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,

    UNIQUE(from_id, to_id, relation)
);

CREATE INDEX IF NOT EXISTS idx_links_brain_from ON memory_links(brain_id, from_id);
CREATE INDEX IF NOT EXISTS idx_links_brain_to   ON memory_links(brain_id, to_id);
`

// schemaV4MemoriesBrainIndexSQL is applied after the ALTER TABLE that adds
// memories.brain_id, since SQLite requires the column to exist before indexing it.
const schemaV4MemoriesBrainIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_memories_brain
    ON memories(brain_id, updated_at DESC)
    WHERE deleted_at IS NULL;
`

// schemaV5SQL adds the 'rejected' status to the pending_events CHECK constraint.
// SQLite cannot ALTER a CHECK constraint, so we recreate the table via the
// 12-step rename-copy-drop sequence inside a transaction.
const schemaV5PendingEventsSQL = `
CREATE TABLE IF NOT EXISTS pending_events_v5 (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id             TEXT    NOT NULL UNIQUE,
    project             TEXT    NOT NULL,
    session_id          TEXT,
    event_type          TEXT    NOT NULL,
    tool_name           TEXT,
    tool_use_id         TEXT,
    payload             TEXT    NOT NULL,
    event_hash          TEXT    NOT NULL,
    status              TEXT    NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending','promoted','archived','rejected')),
    promoted_memory_id  INTEGER,
    promoted_at         INTEGER,
    archived_at         INTEGER,
    created_at          INTEGER NOT NULL,
    captured_at         INTEGER NOT NULL
);
`

// schemaV6IndexSQL creates the covering partial index used by DedupeCheck to
// locate duplicate non-deleted, non-topic-keyed memories efficiently.
// The index includes created_at so the dedup query's ORDER BY and window
// filter are resolved without a separate sort or table scan.
// CREATE INDEX IF NOT EXISTS makes the statement idempotent on re-open.
const schemaV6IndexSQL = `CREATE INDEX IF NOT EXISTS idx_memories_hash ON memories(brain_id, normalized_hash, created_at) WHERE deleted_at IS NULL`

// currentSchemaVersion is bumped whenever schemaSQL changes in a way that
// requires a migration. v1: M1 baseline. v2: M4 sessions table + memories.session_id.
// v3: passive-capture pending_events table. v4: brains, global_config, memory_links.
// v5: pending_events CHECK constraint adds 'rejected' status.
// v6: idx_memories_hash partial covering index for DedupeCheck.
const currentSchemaVersion = 6
