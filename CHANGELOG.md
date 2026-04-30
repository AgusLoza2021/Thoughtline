# Changelog

All notable changes to Thoughtline will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) starting at the M1 release.

## [Unreleased]

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
- M0 (Bootstrap) complete. M1 (`tl_save`) complete. M2 (`tl_search` + `tl_get_observation`) complete. M3 (`tl_context` + `tl_update` + `tl_delete`) complete. M4 (`tl_session_*`) is next.

[Unreleased]: https://github.com/AgusLoza2021/Thoughtline/compare/HEAD...HEAD
