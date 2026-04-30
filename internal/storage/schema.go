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
`

// currentSchemaVersion is bumped whenever schemaSQL changes in a way that
// requires a migration. M1 ships at version 1.
const currentSchemaVersion = 1
