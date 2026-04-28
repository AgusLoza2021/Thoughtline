// Package storage owns persistence for Thoughtline.
//
// Backend: SQLite via modernc.org/sqlite (a pure-Go driver — no CGO, no
// platform-specific build flags, FTS5 baked in). The default database file
// lives at $THOUGHTLINE_HOME/thoughtline.db, falling back to a per-OS data
// directory when the env var is unset.
//
// Schema sketch (firmed up in milestone M1, see docs/ARCHITECTURE.md):
//
//   CREATE TABLE memories (
//       id              INTEGER PRIMARY KEY AUTOINCREMENT,
//       sync_id         TEXT NOT NULL UNIQUE,           -- stable across upserts
//       project         TEXT NOT NULL,
//       scope           TEXT NOT NULL CHECK (scope IN ('project','personal')),
//       type            TEXT NOT NULL,
//       topic_key       TEXT,                            -- nullable; upsert key
//       title           TEXT NOT NULL,
//       content         TEXT NOT NULL,
//       normalized_hash TEXT NOT NULL,                   -- for conflict detection
//       revision_count  INTEGER NOT NULL DEFAULT 0,
//       created_at      INTEGER NOT NULL,                -- unix epoch ms
//       updated_at      INTEGER NOT NULL,
//       deleted_at      INTEGER,                          -- soft delete
//
//       -- M5 reserved (embeddings layer; never written by v1):
//       embedding             BLOB,
//       embedding_model       TEXT,
//       embedding_created_at  INTEGER
//   );
//
//   CREATE VIRTUAL TABLE memories_fts USING fts5(
//       title, content, content=memories, content_rowid=id
//   );
//   -- triggers: keep memories_fts in sync on INSERT/UPDATE/DELETE
//
// Search uses FTS5 + BM25 ranking by default; queries containing a "/" are
// matched against topic_key first (a trick borrowed from Engram). Embeddings
// are explicitly deferred to milestone M5 — schema is reserved so adding them
// is a non-breaking migration.
//
// Status: skeleton. Schema, migrations, and queries land in M1.
package storage
