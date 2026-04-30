# Changelog

All notable changes to Thoughtline will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) starting at the M1 release.

## [Unreleased]

### Added — M4 (Sessions)
- **`tl_session_start` MCP tool** — opens a session, returns its UUIDv7 id. Optional `agent_label` for cross-session forensics ("claude-code", "cursor", "zed", ...). Project defaults to working directory basename.
- **`tl_session_summary` MCP tool** — closes a session, persists the structured digest. Sessions are append-once: a second call returns "already ended". Summary required, ≤ 64 KB.
- **`tl_save` learns an optional `session_id` argument** — when present, attaches the memory to that session. Empty = unattached (preserves M1 default behaviour exactly).
- **Domain `Session` type** (`internal/memory/session.go`) — UUIDv7 id, project, optional agent_label (≤ 64 chars), started_at, optional ended_at, summary. `IsOpen()` and `Duration()` helpers. `ValidateSession` enforces UUIDv7 format, project required, label/summary size caps, ended_at >= started_at.
- **`memory.SessionID` field** added to `Memory` with UUIDv7 format validation in `memory.Validate`.
- **Schema v2 migration** — bumped `currentSchemaVersion` from 1 to 2. Adds `sessions` table + `idx_sessions_recent` (CREATE IF NOT EXISTS) and adds `memories.session_id TEXT REFERENCES sessions(id) ON DELETE SET NULL` via idempotent `ALTER TABLE` guarded by a `PRAGMA table_info` check. Fresh DBs and existing v1 DBs both migrate cleanly on first Open.
- **Storage CRUD for sessions** (`internal/storage/sessions.go`):
  - `StartSession(ctx, project, agentLabel)` — assigns UUIDv7, persists, returns the Session.
  - `EndSession(ctx, id, summary)` — sets ended_at and summary; rejects already-ended.
  - `GetSession(ctx, id)` / `RecentSessions(ctx, project, limit)`.
  - `ErrSessionNotFound`, `ErrSessionAlreadyEnded`, `ErrSessionProjectMismatch` exported sentinels.
  - **Cross-table integrity**: `Save` now calls `validateSessionLink` before inserting. A non-empty `m.SessionID` must point to an existing session in the same project, otherwise `ErrSessionNotFound` or `ErrSessionProjectMismatch`.
  - **Sticky session_id on upsert**: re-saving with the same `topic_key` and an empty `SessionID` PRESERVES the prior session linkage (`COALESCE(?, session_id)` in the UPDATE). To overwrite, pass an explicit `SessionID`. This prevents accidental session detachment.
- **Tests added**:
  - `internal/memory/session_test.go` — every validation rule.
  - `internal/storage/sessions_test.go` — full CRUD coverage, regression guards for upsert-clobber-session, cross-project rejection, unknown-session rejection, schema v2 idempotency on reopen.
  - `internal/server/tl_session_start_test.go`, `tl_session_summary_test.go`, `tl_save_session_test.go` — handler-level coverage for happy path, validation errors, not-found, already-closed.
  - `internal/server/integration_session_test.go` — end-to-end: start → save (× 2) → upsert preserves session → close → second close rejected → post-mortem save allowed → cross-project session rejected.

### Added — M3 (Context, Update, Delete)
- **`tl_context` MCP tool** — recent memories for the active project, ordered by `updated_at DESC`, soft-deleted rows excluded. Returns the same per-result envelope as `tl_search` so an AI parses both with one parser.
- **`tl_update` MCP tool** — patch a memory by id. Mutable fields: `title`, `content`, `tags`. Identity-defining fields (`type`, `topic_key`, `project`, `scope`) are intentionally NOT mutable. Empty patch = noop. Real changes bump `revision_count`, refresh `updated_at`, preserve `id`/`sync_id`/`created_at`. Re-validates the merged memory before persisting.
- **`tl_delete` MCP tool** — soft-delete by id. Sets `deleted_at`, hides the row from search/context/get_observation, frees the `topic_key` for a fresh `tl_save`. Repeating delete on an already-deleted row returns "not found".
- **`storage.Recent(ctx, project, limit)`** — uses the existing `idx_memories_recent` index. Project parameter is required (never silently spans projects). Limit clamped to `[1, 50]`, default 10.
- **`storage.UpdateByID(ctx, id, UpdatePatch)`** — `UpdatePatch` uses pointer fields (`*string`, `*[]string`) so callers can distinguish "leave unchanged" (nil) from "set to zero value" (e.g. `&[]string{}` to clear all tags).
- **`storage.SoftDelete(ctx, id)`** — partial unique index on `(project, topic_key)` where `deleted_at IS NULL` means the topic_key is automatically freed by soft-delete.
- **`storage.ErrMemoryNotFound`** — exported sentinel for `errors.Is` in callers when an id doesn't exist or has been soft-deleted.
- **Soft-delete regression guard**: M2's `Search` already filtered `deleted_at IS NULL` on the FTS path; pinned that behaviour in `TestSearch_ExcludesSoftDeletedRegression` plus the topic-key shortcut path.
- **Integration scenario test** (`internal/server/integration_test.go`): exercises every `tl_*` tool end-to-end through real `mcp.CallToolRequest` decoding — save → search (FTS + topic-key shortcut) → get_observation → context → update → search post-update → delete → search post-delete → context post-delete → get/update on deleted id (both not-found) → re-save reusing the freed topic_key.

### Added — M2 (Search)
- **`tl_search` MCP tool** — keyword search backed by SQLite FTS5 + BM25 ranking. Parameters: `query` (required), optional `type`, `scope`, `project`, `topic_key` (GLOB filter), `limit` (default 10, hard cap 50), `offset`.
- **Topic-key shortcut** — queries containing `/` are first matched against `topic_key` as a GLOB pattern (so `design/auth/*` or an exact `scene/playcanvas/inn-cellar` lookup is O(1)). If any rows match the shortcut, FTS does not run. Misses fall through to FTS cleanly.
- **FTS5 query sanitization** — every whitespace-delimited token is wrapped in literal quotes; FTS5 special operators (`:`, `*`, `^`, `(`, `)`, `NEAR`, `OR`, ...) in user input become inert. No way to crash the engine with malformed input.
- **`tl_get_observation` MCP tool** — fetch the full untruncated content + metadata of a single memory by `id`. Companion to `tl_search`'s 300-char snippets. Soft-deleted rows are filtered out.
- **`storage.Search`** — returns `[]SearchResult` with `id`, `sync_id`, `title`, `snippet` (≤300 chars from FTS5 `snippet()`), `score` (BM25 rank), `topic_key`, `tags`, `revision_count`, `updated_at`. Filters: `project`, `scope`, `type`, `topic_key` GLOB. Limit/offset pagination.
- **Tests** — storage layer covers basic FTS5 match, BM25 ordering, every filter, both topic-key shortcut paths (hit and fall-through-on-miss), GLOB wildcards, snippet truncation, FTS operator sanitization, default + clamped limits, offset pagination, empty result, revision count round-trip. Server layer covers happy path, missing query, default-project scoping, explicit project override, type filter, topic-key shortcut end-to-end, get-observation hint, limit override.

### Added — M1 (Save)
- **`tl_save` MCP tool** — first working tool. Persists a memory with `title`, `content`, `type`, optional `scope`/`topic_key`/`project`/`tags`. Returns `id`, `sync_id`, `action` (`created`/`updated`/`noop`).
- **Storage layer** (`internal/storage`) — SQLite via `modernc.org/sqlite`, FTS5 contentless virtual table kept in sync via three triggers, `(project, topic_key)` unique index for upserts, reserved `embedding*` columns for M5.
- **Domain layer** (`internal/memory`) — `Memory`, `Type`, `Scope` types and `Validate(Memory) error` with all rules from `docs/design/memory-domain.md`.
- **Topic-key upsert semantics** (mirrors Engram): same project + same topic_key reuses `id`, `sync_id`, `created_at` and bumps `revision_count`. Identical re-saves are noops.
- **CLI / runtime** — `cmd/thoughtline` resolves `THOUGHTLINE_DB` / `THOUGHTLINE_HOME` env vars, opens the database (creating the parent dir if missing), boots the MCP stdio server, exits cleanly on EOF / SIGINT / SIGTERM.
- **Tests** — `go test ./...` covers domain validation (every rule), storage (insert, upsert, noop, FTS sync, persistence across reopen), and the `tl_save` handler (happy path, validation errors, scope auto-defaults, end-to-end upsert flow).

### Added — M0 (Bootstrap)
- Project skeleton: `cmd/thoughtline`, `internal/{server,storage,memory}` with package docs.
- README, LICENSE (MIT, with attribution to Engram), `.gitignore`, `go.mod` targeting Go 1.25.
- Architectural reconnaissance of [Engram](https://github.com/Gentleman-Programming/engram) — three deep-dive docs in `docs/research/`.
- ADR 0001: Architecture baseline — copy Engram's pattern set, customize taxonomy.
- ADR 0002: Search strategy — FTS5 + BM25 in v1, embeddings reserved for M5.
- Memory taxonomy design (`docs/design/memory-domain.md`) with gamedev-first types.
- Architecture overview (`docs/ARCHITECTURE.md`) with mermaid diagrams.
- Example MCP client config in `examples/mcp-config.example.json`.
- Basic CI workflow: `go vet` + `go build` + `go test` on Linux/macOS/Windows.
- CONTRIBUTING.md with PR and ADR conventions.

### Status
- M0–M4 complete. Eight MCP tools live: `tl_save`, `tl_search`, `tl_get_observation`, `tl_context`, `tl_update`, `tl_delete`, `tl_session_start`, `tl_session_summary`. M5 (semantic embeddings) remains deferred per ADR 0002 — schema reserved, opt-in if/when needed.

[Unreleased]: https://github.com/AgusLoza2021/Thoughtline/compare/HEAD...HEAD
