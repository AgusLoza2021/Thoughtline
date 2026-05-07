# Changelog

All notable changes to Thoughtline will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) starting at the M1 release.

## [Unreleased]

## [0.1.0] - 2026-05-07

This is the first numbered release. It folds in the M5 launch-readiness work (TUI overhaul, distribution, plugin) plus two structured changes built under SDD: passive capture from Claude Code hooks, and the v2 workstation-style TUI.

### Added — Passive capture (`passive-capture-hooks`)
- New SQLite table `pending_events` (schema v3) with UNIQUE on `(project, event_hash)` for idempotent inserts
- `thoughtline hook <event-name>` subcommand — reads JSON from stdin, fail-silent, never breaks the host Claude Code session
- `thoughtline worker` subcommand — retention janitor with archive + hard-delete windows (default 7d / 30d)
- 3 new MCP tools: `tl_pending_list`, `tl_pending_get`, `tl_promote` (per-event partial-success batch)
- Plugin integration — `plugin/claude-code/hooks.json` registers all 6 Claude Code hook events
- Opt-in only via `THOUGHTLINE_PASSIVE_CAPTURE=1`; default OFF
- License hygiene: `scripts/check-no-claude-mem.{sh,ps1}` blocks accidental copy of AGPL strings
- ADR 0004 documents the design and the known v1 limitation around the `tl_promote` non-atomic seam

### Added — TUI v2 workstation (`tui-redesign`)
- Welcome screen (engram-inspired) — ASCII logo, stat card, 5-action menu
- Workstation screen — 3-pane layout (sidebar / center / right detail) with operational top bar and bottom status line
- 4 new drill-in screens: SearchScreen, RecentScreen, BrowseProjectsScreen / Projects, PendingScreen, DetailScreen
- Screen-stack navigation (`Screen` interface, push/pop semantics) with vim-style keys (`hjkl`, `gg`, `G`, `/`, `r`)
- Single Rose-Pine-Moon adapted palette; multi-theme infrastructure deprecated
- Cross-platform disk-free measurement (`diskfree_unix.go` / `diskfree_windows.go`)
- New storage methods backing the workstation: `CountPending`, `MostRecentProjects`, `RecentAll`, project-scoped recent

### Fixed
- `q` no longer pops out of `SearchScreen` while typing — queries containing the letter `q` now work; `esc` is the universal back
- `tl_search` license hygiene script no longer trips on legitimate doc references

### Added — Public-launch readiness

This batch lands the work needed to flip the repo public: cross-platform plugin, Claude Code marketplace metadata, brand polish, distribution, and a Bubbletea TUI restyle.

#### Distribution & repo hygiene
- **GoReleaser config** (`.goreleaser.yaml`) — cross-compile for linux / darwin / windows × amd64 / arm64, tar.gz / zip archives, checksums, version + commit + date injected via ldflags.
- **Release workflow** (`.github/workflows/release.yml`) — fires on `v*.*.*` tags, runs GoReleaser, publishes artifacts.
- **`.gitignore` tightened** — explicit ignores for bare-name binaries (`/thoughtline`, `/thoughtline-exe`). The tracked Linux binary was untracked in the same change.
- **Community files** — `SECURITY.md` (private vulnerability reporting), `.github/ISSUE_TEMPLATE/{bug_report.yml,feature_request.yml,config.yml}`, `.github/PULL_REQUEST_TEMPLATE.md`.

#### Dashboard TUI overhaul
- **Theme system** (`internal/dashboard/themes.go`) — three palettes shippable on day one:
  - `brand` — violet + cyan, the original tech-SaaS look
  - `zbrush` — warm tactile palette inspired by Pixologic ZBrush (`#D68A3C` amber on `#2D2A26` warm-dark, `#E8DCC4` cream text)
  - `mono` — minimalist grayscale for screenshots and slides
  Switchable at runtime with the `[t]` hotkey or via `--theme {brand|zbrush|mono}`. `ApplyTheme(t)` rebuilds every package-level style on swap.
- **Cockpit status bar** (`renderStatusBar`) — single line at the top of every tab: `◆ THOUGHTLINE ONLINE · MEM N · SESSIONS N · vX.Y.Z`. When an update is available, an amber pill appears: `↑ vX.Y.Z available · [u] open release`.
- **Background update check** (`internal/dashboard/updatecheck.go`) — non-blocking GitHub releases lookup with 3s timeout. `[u]` opens the release URL in the user's browser cross-platform (`rundll32` / `open` / `xdg-open`). Toggleable via `--no-update-check`.
- **Engram-style stats** — right-aligned numerals in `renderStatsPanel`. Bold brand-colored numbers, muted labels.
- **Animated header cube** (`internal/dashboard/cube.go`) — 3D wireframe ASCII cube that oscillates through 5 frames. Plain ASCII (CMD-safe) and "fancy" box-drawing modes. 220ms per frame default.
- **Splash screen** (`internal/dashboard/splash.go`) — cube + ASCII wordmark + tagline, vertically centered with `lipgloss.Place`. Auto-dismiss after 1.5s (configurable). Skip with `--no-splash`.
- **CLI flags** — `--theme`, `--no-splash`, `--no-update-check`, `--splash-ms N` on `thoughtline ui`.
- **Help tab** — new rows for `[t]` (cycle theme) and `[u]` (open release).
- **Hero composition** — Overview tab uses `lipgloss.JoinHorizontal` to stack the cube next to the brand pill, tagline, and breadcrumb metadata.

#### Claude Code plugin
- **`thoughtline protocol` subcommand** (`cmd/thoughtline/protocol.go`) — single source of truth for the active-protocol markdown. Replaces the duplicated heredocs in three different files. Cross-platform (Go binary, no shell needed). Flags: `--event {session-start|post-compaction}`, `--project NAME`, `-o FILE`. Versioned via `Protocol-Version: 1` header so plugins can detect drift.
- **`plugin/claude-code/` overhaul** — the plugin now ships:
  - `hooks/hooks.json` — calls `thoughtline protocol` directly, no bash. Adds `resume` to the matcher so re-opening a project re-injects the protocol.
  - `commands/{tl-recent, tl-search, tl-stats, tl-ui, tl-export}.md` — slash commands surfaced in autocomplete.
  - `agents/tl-archivist.md` — specialist subagent for memory hygiene (dedup, prune, audit). Read-mostly, never deletes without explicit confirmation.
  - `examples/{saved-memory.md, session-transcript.md}` — calibration set for what "good" memories look like and what a real session flow looks like.
  - `skills/memory/SKILL.md` — extended with `## Examples` block (save triggers, search triggers, NOT-a-trigger).
  - `.claude-plugin/plugin.json` — bumped with `repository`, `homepage`, `keywords`, `protocolVersion`. Version synced to the binary's (`0.0.1`).
  - `.claude-plugin/marketplace.json` — listing manifest for Claude Code plugin marketplaces (publisher, displayName, summary, tags, requirements).
  - `LICENSE` — MIT, copied from the repo root so the plugin directory is legally self-contained.
  - `README.md` — install, prerequisites, file layout, configuration, uninstall.
- **Bash scripts removed** — `plugin/claude-code/scripts/{session-start.sh, post-compaction.sh, _helpers.sh}`. The binary subcommand replaces them; Windows users no longer need WSL or Git Bash.

#### Plugin CI
- **`.github/workflows/plugin.yml`** — JSON validation (`jq -e .` on every plugin JSON file), front-matter check on commands and agents, build of the binary, smoke test of `thoughtline protocol` for both events, and a guard that fails CI if `hooks.json` ever references a `.sh` script again.

#### Repo presentation
- **README.md hook section** — `Quick start` block at the top with `go install`, MCP JSON config, `thoughtline ui`, and a Claude Code plugin install one-liner.

#### Game-dev launch polish
- **"For game devs, in 60 seconds"** hook section above the fold — pain-point list (re-explaining hierarchies, import settings, batching gotchas) followed by the value prop in three lines.
- **Comparison table** — Thoughtline vs Engram vs Cursor memories vs ChatGPT memory vs manual notes. Honest tradeoffs; explicit "use Engram if you don't ship games" callout.
- **`docs/TOOLS.md`** — full parameter reference + 10 end-to-end examples extracted from README (the README dropped from 704 → ~330 lines).
- **`docs/design/tag-conventions.md`** — canonical tag vocabulary: engine, platform, pipeline, asset, phase, tooling, performance buckets. Examples per category.
- **`docs/integrations/cursor.md`** — wiring Thoughtline into Cursor (MCP config + `.cursorrules` snippet).
- **`docs/integrations/zed.md`** — wiring into Zed's assistant context server config.
- **`docs/integrations/rider-unity.md`** — JetBrains Rider for Unity, with a "day 1 saves" cheat sheet (folder layout, MonoBehaviour conventions, Android asset bundles).
- **`docs/media/`** — screenshots directory with capture-tip README; placeholder image references in main README.
- **README integrations directory** — every README install section links the per-IDE guide instead of dumping every JSON config inline.

### Added — M5 (Dashboard)
- **`thoughtline ui` subcommand** — opens an interactive Bubbletea TUI with four panels: header (version + DB path + active project), stats (counts by type / project / scope), recent activity (last 10 memories + 5 sessions), and roadmap (M0–M6 status). Keys: `r` refresh, `q` / `ctrl+c` / `esc` quit. Reads from the same SQLite store the MCP server uses.
- **`tl_stats` MCP tool** — programmatic access to the same stats snapshot. Optional `project` argument; pass `*` to see counts across all projects. Returns text-formatted breakdown the AI can read aloud or summarize.
- **`storage.Stats(ctx, opts) (Stats, error)`** — single query interface returning total memories (active + soft-deleted), counts grouped by type / project / scope, open + closed session counts, and the most recent N memories + sessions. Supports project filter and configurable RecentLimit (default 10, capped at 50).
- **CLI subcommands** — `thoughtline help`, `thoughtline version`, `thoughtline serve` (default), `thoughtline ui`. Friendly error message + usage on unknown subcommand.
- **`internal/dashboard` package** — Bubbletea Model / Update / View split, lipgloss styling, hardcoded `Roadmap()` so PRs review milestone status changes alongside the corresponding code change.
- **Tests** — `internal/storage/stats_test.go` covers every aggregation (totals, by type, by project, by scope, open/closed sessions, recent memories with limit + default, recent sessions, project filter). `internal/server/tl_stats_test.go` covers happy path, empty DB, project filter, default-project fallback, `*` wildcard, type breakdown, recent activity inclusion. `internal/dashboard/model_test.go` follows the SKILL.md patterns: direct `Model.Update()` tests for state transitions (q / ctrl+c / esc / r / window resize / stats loaded), and a basic View test pinning the panels render their headers + roadmap entries.
- **Roadmap renumbered** — M5 is now Dashboard (this release). Smarts (semantic embeddings) moves to M6 and remains deferred per [ADR 0002](docs/decisions/0002-search-strategy-fts5-first.md). Schema reservation for embeddings is still in place from M1.

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
- M0–M5 complete. Nine MCP tools live: `tl_save`, `tl_search`, `tl_get_observation`, `tl_context`, `tl_update`, `tl_delete`, `tl_session_start`, `tl_session_summary`, `tl_stats`. Plus a `thoughtline ui` interactive dashboard. M6 (semantic embeddings) remains deferred per ADR 0002 — schema reserved, opt-in if/when needed.

[Unreleased]: https://github.com/AgusLoza2021/Thoughtline/compare/HEAD...HEAD
