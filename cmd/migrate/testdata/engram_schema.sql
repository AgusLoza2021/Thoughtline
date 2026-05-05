-- Engram observations table schema used by reader_test.go and
-- integration_test.go to build deterministic fake Engram databases.
--
-- Column set matches the Engram production schema as observed in
-- docs/research/engram-anatomy.md. Columns tool_name, duplicate_count, and
-- last_seen_at are read and discarded by the reader — they exist here for
-- schema fidelity but are never mapped to EngramObservation.
CREATE TABLE IF NOT EXISTS observations (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id         TEXT    NOT NULL UNIQUE,
    type            TEXT    NOT NULL,
    title           TEXT    NOT NULL,
    content         TEXT    NOT NULL DEFAULT '',
    project         TEXT    NOT NULL DEFAULT '',
    scope           TEXT    NOT NULL DEFAULT 'project',
    topic_key       TEXT,
    normalized_hash TEXT,
    revision_count  INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT    NOT NULL,
    updated_at      TEXT    NOT NULL,
    deleted_at      TEXT,
    tool_name       TEXT,
    duplicate_count INTEGER NOT NULL DEFAULT 0,
    last_seen_at    TEXT
);
