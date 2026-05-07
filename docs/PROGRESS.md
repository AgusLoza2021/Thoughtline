# Progress

Single source of truth for "what's done, what's next, what's blocking publishing." Updated at the end of every working session.

---

## Current state — 2026-05-07

**Passive capture: 🟢 done.** Schema v3, 12 MCP tools, `thoughtline hook`, `thoughtline worker`. See ADR 0004.

**Milestone M5 (Dashboard): 🟢 done.** M0–M4 are also 🟢 done — see history below. Nine MCP tools + interactive TUI shipped in M5. M6 (Smarts / embeddings) is deferred per ADR 0002.

---

### What got done — passive capture hooks (2026-05-07)

- **Schema v3** (`internal/storage/schema.go`): `pending_events` table with `event_hash`-based dedup, `(project, event_hash)` unique index, status check constraint, 3 indexes.
- **Domain package** (`internal/pending/`): `Event` struct, `Status` constants, `Validate`, `ComputeHash` (SHA-256 over type+session+tool_use_id+floored-second+canonical-payload).
- **Storage layer** (`internal/storage/pending.go`): `InsertPending`, `ListPending`, `GetPendingByID`, `MarkPromoted`, `SweepPending` with `SweepResult`. Uses `INSERT OR IGNORE` for idempotency.
- **Hook CLI** (`cmd/thoughtline/hook.go`): opt-in fast-path, 1 MiB stdin cap, JSON validation, SHA-256 dedup hash, always exits 0 on user-facing errors.
- **Worker CLI** (`cmd/thoughtline/worker.go`): one-shot retention janitor; `--retention` (default 7d) and `--hard-delete` (default 30d) flags; cron/Task-Scheduler friendly.
- **3 new MCP tools** (`internal/server/`): `tl_pending_list`, `tl_pending_get`, `tl_promote`. Promote uses per-event transactions; partial success is the defined behavior.
- **Plugin hooks** (`plugin/claude-code/hooks/hooks.json`): 6 new entries for SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop, SessionEnd.
- **License hygiene** (`scripts/check-no-claude-mem.sh` + `.ps1`): CI guard against AGPL string leakage; added to `.github/workflows/ci.yml`.
- **Docs**: ADR 0004, integration guide, COMPARISON.md updated, README pointer added.
- **Verification**: `go test ./...` green; `go vet ./...` clean.

### What got done this session (M5)

- **Storage layer** (`internal/storage/stats.go` + `stats_test.go`):
  - `Stats(ctx, opts) (Stats, error)` — single-call snapshot returning total memories (active + soft-deleted), counts grouped by type / project / scope, open + closed session counts, recent memories + sessions.
  - `StatsOptions{Project, RecentLimit}`. Project="" = cross-project counts. Limit defaults 10, capped at 50.
  - Tests: empty DB, total + deleted counts, by-type / by-project / by-scope groupings, open/closed session counts, recent memories + limit + default, recent sessions, project filter.
- **MCP tool** (`internal/server/tl_stats.go` + `tl_stats_test.go`):
  - `tl_stats` registered alongside the existing 8 tools — tool count now 9.
  - `statsArgs.Project` with `*` sentinel for "ignore default project, show everything".
  - `formatStats` renders the Stats snapshot as readable text with deterministic ordering (alphabetical by project, taxonomy order by type).
  - Tests: happy path, empty DB, project filter, default-project fallback, `*` wildcard, type breakdown, sessions section, recent activity inclusion.
- **Dashboard package** (`internal/dashboard/`):
  - `model.go` / `update.go` / `view.go` — Bubbletea Model with explicit Init / Update / View split. State: `loaded`, `loading`, `err`, `width`, `height`, `Quitting`. `loadStatsCmd` runs storage.Stats off the main goroutine and posts a `statsLoadedMsg`.
  - `roadmap.go` — hardcoded `Roadmap()` returning M0–M6 statuses + `StatusGlyph`. Hardcoded so PR review covers milestone state changes alongside the corresponding code change.
  - `view.go` — lipgloss styles + 4-panel layout: header, Stats panel, Recent Activity panel (side-by-side via `JoinHorizontal`), Roadmap panel, footer with key bindings.
  - `run.go` — `Run(ctx, st, cfg)` boots a `tea.Program` with `WithAltScreen()` and `WithContext` for clean shutdown.
  - Tests follow `~/.claude/skills/go-testing/SKILL.md` Pattern 2 (direct `Model.Update()`) — 11 tests covering Init / quit (q/ctrl+c/esc) / refresh on r / window resize / stats loaded clears loading / err propagation / View renders panels / Quitting view is empty / loading hint / Roadmap structure / StatusGlyph mapping.
- **CLI subcommands** (`cmd/thoughtline/main.go`):
  - `thoughtline` (no args) and `thoughtline serve` → MCP stdio server (default; what an MCP client launches).
  - `thoughtline ui` (or `thoughtline dashboard`) → opens the TUI.
  - `thoughtline version` (or `-v`/`--version`) → prints version.
  - `thoughtline help` (or `-h`/`--help`) → prints usage.
  - Unknown subcommand → friendly error + usage + exit 2.
- **Dependencies added**:
  - `github.com/charmbracelet/bubbletea v1.3.10`
  - `github.com/charmbracelet/lipgloss v1.1.0`
  - Plus their transitive dependencies (lucasb-eyer/go-colorful, mattn/go-runewidth, muesli/termenv, etc.) — all standard for Go TUIs.
- **Roadmap renumbered**: M5 was previously "Smarts (embeddings)" deferred; now M5 is "Dashboard" (this release) and Smarts is M6, still deferred per ADR 0002.
- **Verification**: `go vet ./...` clean; `go test ./...` all green across `memory`, `storage`, `server`, `dashboard` packages.

### What's NOT done (intentionally)

- **No interactive search / drill-down in the TUI**: the v1 dashboard is a status-at-a-glance read-only view. Search interactivo (presionás `s`, escribís query, ves resultados live) lo agregamos solo si lo extrañamos.
- **No soft-deleted recovery view**: `DeletedMemories` count is shown but there's no UI to restore. Recovery is admin work via SQLite directly (or `UPDATE memories SET deleted_at = NULL WHERE id = ?`).
- **No tag breakdown panel**: tags by frequency would be a nice future addition. Trivial to add to `Stats` if requested.
- **No live polling**: TUI loads stats once on Init and only re-polls on `r`. A live ticker (every N seconds) is easy to add but adds load on the SQLite file for an unclear win.
- **No pre-publish TODO closed**: `_engram-research/` paths in research docs still need permalink replacement before going public.

---

## Next session — Milestone M6 (Smarts) — DEFERRED by default

Per [ADR 0002](decisions/0002-search-strategy-fts5-first.md), M6 only happens if user feedback shows lexical recall failures dominate complaints. Engram has run in production without embeddings for months, so there's no urgency.

**If/when M6 happens**, the ADR sketch is:
- Add `embedding_dim` column (the only schema change needed; reserved BLOB columns already exist).
- Optional companion table `memory_embeddings` (sync_id PK, vector BLOB, model, dim, created_at).
- Provider-agnostic: store model name + dim per row so multiple providers can coexist during transition.
- Hybrid ranking: BM25 score + vector score combined via reciprocal rank fusion (RRF) or tunable linear blend.
- Behind a feature flag, off by default.
- `tl_reindex` background tool to backfill embeddings for memories worth re-embedding.

For now, **v0.0.1 ships with M0–M5 complete**. Nine MCP tools + interactive dashboard.

---

## Milestone history

### 2026-04-30 — M4 Sessions 🟢

- Domain `Session` type + UUIDv7 validation + `Memory.SessionID` field.
- Storage: schema v2, sessions table, FK with `ON DELETE SET NULL`, idempotent migration via PRAGMA + ALTER, sessions CRUD (`StartSession`/`EndSession`/`GetSession`/`RecentSessions`), 3 sentinels (`ErrSessionNotFound`, `ErrSessionAlreadyEnded`, `ErrSessionProjectMismatch`), sticky session_id on upsert with regression tests.
- Server: `tl_session_start` + `tl_session_summary` MCP tools, `tl_save` learned `session_id` arg, cross-table error surfacing.
- Integration: 6-step scenario covering full lifecycle including upsert preserves session, post-mortem save allowed, cross-project rejection.
- Audit-fix moment: paused mid-sprint, caught 3 bugs (undefined helper, upsert clobber, missing cross-project validation) and fixed with regression tests. Re-established TDD discipline.
- Verification: `go vet ./...` clean; `go test ./...` all green.

### 2026-04-30 — M3 Context, Update, Delete 🟢



- **Storage layer**:
  - `internal/storage/recent.go` — `Recent(ctx, project, limit) → []SearchResult` using the existing `idx_memories_recent` index. Project required (refuses empty to prevent cross-project leak). Limit defaults 10, hard cap 50. Snippet = content prefix truncated to `SnippetMaxChars` runes (no FTS5 match centering since there's no query). Score = 0.
  - `internal/storage/update.go` — `UpdateByID(ctx, id, UpdatePatch)` with pointer fields (`*string`, `*[]string`) so callers can distinguish "leave unchanged" (nil) from "set to zero value". Mutable fields: Title, Content, Tags. Identity fields (Type, TopicKey, Project, Scope) deliberately not patchable. Empty patch = noop (returns current row, no DB write, no revision bump). Real changes re-validate via `memory.Validate`, bump `revision_count`, refresh `updated_at`, recompute `normalized_hash`, preserve id/sync_id/created_at. FTS5 stays in sync via the existing `memories_au` trigger.
  - `internal/storage/delete.go` — `SoftDelete(ctx, id)` sets `deleted_at = now`. Returns `ErrMemoryNotFound` if id is unknown OR already soft-deleted (symmetric with UpdateByID).
  - `storage.ErrMemoryNotFound` — exported sentinel for `errors.Is`.
  - **Tests**: `recent_test.go` covers ordering, project filter (and `Recent("")` rejection), soft-delete exclusion, default + clamp limits, snippet/metadata population, regression guard for soft-delete on `Search` (FTS path AND topic-key shortcut path), unknown-project empty result. `update_test.go` covers title-only patch, identity preservation across upsert, FTS index refresh, tag round-trip + clear, empty-patch noop, not-found, soft-deleted rejection, validation propagation, hash recomputation. `delete_test.go` covers happy path, removal from Search, not-found, second-delete = not-found, topic_key reuse after soft-delete.
- **Server layer**:
  - `tl_context.go` — full MCP tool with `contextArgs`/`doContext`/`formatContextResults`. Reuses `writeResultBlock` (extracted into `tl_search.go`) so `tl_context` and `tl_search` produce identically-shaped per-result blocks. No `Score:` line for context (recency, not relevance).
  - `tl_update.go` — `updateArgs` carries `Has{Title,Content,Tags}` flags so the decode path can distinguish "field absent in JSON" from "field explicitly set to empty". Maps to `storage.UpdatePatch`. Surfaces `ErrMemoryNotFound` and `memory.ErrEmpty*` validation errors with friendly messages via the existing `formatValidationError`.
  - `tl_delete.go` — `deleteArgs.ID` only. Plain-text confirmation on success.
  - `server.New` registers all three new tools alongside the M1+M2 set.
  - **Handler tests**: `tl_context_test.go` (happy path, default project, missing project, empty result, limit override, type-filter-not-exposed pin), `tl_update_test.go` (title-only, tag clear, noop, not-found, missing id, validation propagation), `tl_delete_test.go` (happy path, removal from search+context, not-found, missing id, double-delete = not-found).
- **Integration scenario** (`internal/server/integration_test.go`):
  - Builds the full `MCPServer` via `New(st, cfg)`.
  - Drives an 11-step end-to-end flow with real `mcp.CallToolRequest` decoding: save → search (FTS + topic-key shortcut) → get_observation → context → update → search post-update → delete → search post-delete → context post-delete → get/update on deleted id (both not-found) → re-save reusing freed topic_key.
- **Verification**: `go vet ./...` clean; `go test ./...` all green across `memory`, `storage`, `server` packages.

### 2026-04-30 — M2 Search 🟢



- **Storage layer** (`internal/storage/search.go`):
  - `Search(ctx, query, opts) ([]SearchResult, error)` — FTS5 MATCH + BM25 ranking, with the `topic_key` GLOB shortcut for queries containing `/`. Shortcut **short-circuits** on hit (no merge with FTS); falls through to FTS on miss.
  - `SearchResult` carries `id`, `sync_id`, `project`, `scope`, `type`, `topic_key`, `title`, `snippet` (≤300 chars), `tags`, `score` (BM25 rank; topic-key hits get synthetic `-1000`), `revision_count`, `updated_at`.
  - `SearchOptions` filters: `Project`, `Scope`, `Type`, `TopicKey` (GLOB), `Limit` (default 10, hard cap 50), `Offset`.
  - `sanitizeFTSQuery` strips all `"` from each whitespace-delimited token then re-wraps in literal quotes — FTS5 special chars (`:`, `*`, `^`, `(`, `)`, `NEAR`, `OR`, ...) in user input become inert.
  - Snippet generated via SQLite `snippet(memories_fts, 1, '', '', '…', 32)` then truncated to 300 runes (UTF-8 safe).
  - **Tests** (`search_test.go`): basic FTS match, BM25 ordering by relevance, filter by `type`/`scope`/`project`, topic-key shortcut exact + GLOB, fall-through on shortcut miss, snippet truncation on long content, sanitization of FTS operators, default + clamped limits, offset pagination, empty result, topic-key glob filter (independent of query), revision-count round-trip after upsert.
- **Server layer** (`internal/server`):
  - `tl_search.go`: full MCP tool with JSON schema, `searchArgs` decoder, `doSearch` testable core, `formatSearchResults` natural-language renderer with always-on `tl_get_observation` footer hint.
  - `tl_get_observation.go`: full MCP tool with `getObservationArgs`, `doGetObservation` core that filters soft-deleted rows, `formatObservation` renderer.
  - `server.go` updated to register both new tools alongside `tl_save`.
  - **Tests**: happy path, blank-query validation, default-project scoping, explicit project override, type filter, topic-key shortcut end-to-end, get-observation hint always present, limit override; `tl_get_observation` happy path, not-found, missing/zero id.
- **Verification**: `go vet ./...` clean; `go test ./...` all green.

### 2026-04-28 — M1 Save 🟢

- **Domain layer** (`internal/memory`): `types.go` with `Memory`/`Type`/`Scope` and the closed type catalogue, `validate.go` with every rule from `design/memory-domain.md`, `validate_test.go` covering all rules positively and negatively.
- **Storage layer** (`internal/storage`): `schema.go` with the full DDL (memories, FTS5 contentless virtual table, three triggers, reserved `embedding*` columns, `schema_version` table); `storage.go` exposing `Open`/`Close`/`Save`/`GetByID`/`FTSCount`/`FTSContains` with UUIDv7 sync_id and normalized-hash noop detection; tests covering insert, upsert, noop, FTS sync, persistence across reopen.
- **Server layer**: `server.go` builds the `mcp-go` MCPServer with instructions; `tl_save.go` registers the tool with full JSON schema; tests cover happy path, scope auto-defaults (`preference` → `personal`), default project from config, every validation error path, and an end-to-end upsert flow.
- **CLI**: resolves `THOUGHTLINE_DB` / `THOUGHTLINE_HOME` env vars, opens storage, derives default project from cwd basename or `THOUGHTLINE_PROJECT`, boots the MCP stdio server.
- **Verification**: `go vet ./...` clean; `go test ./...` green; `go build ./cmd/thoughtline` succeeds.

### 2026-04-28 — M0 Bootstrap 🟢

- Cloned [Engram](https://github.com/Gentleman-Programming/engram) (depth=50) into `_engram-research/` *outside* the project tree, so it never accidentally gets committed.
- Project skeleton on Desktop: `cmd/thoughtline/`, `internal/{server,storage,memory}/` with package `doc.go` files, `go.mod` targeting Go 1.25, `LICENSE` (MIT, with attribution to Engram), `.gitignore`.
- Architectural reconnaissance (delegated to a sub-agent) produced three deep-dive docs in `docs/research/` covering engram's anatomy and the `mem_save` / `mem_search` flows.
- **Critical insight from research**: Engram has reserved schema columns for embeddings but never uses them. Search is 100% FTS5 + BM25 with a topic-key shortcut. This drove ADR 0002.
- ADRs: `0001-architecture-baseline.md` (parallel project, not fork), `0002-search-strategy-fts5-first.md` (FTS5 in v1, embeddings deferred to M5).
- `ARCHITECTURE.md`, `design/memory-domain.md`, README, CHANGELOG, CONTRIBUTING, example MCP config, multi-OS CI workflow.

---

## Pre-publish TODOs

Items the maintainer must do before / right after pushing the repo to GitHub. Ordered.

### Before first commit

- [x] ~~**Decide the GitHub URL.**~~ Repo is `github.com/AgusLoza2021/Thoughtline`. Updated in:
  - [x] ~~[`go.mod`](../go.mod) — `module` line~~
  - [x] ~~[`README.md`](../README.md) — install command, clone command~~
  - [x] ~~[`CHANGELOG.md`](../CHANGELOG.md) — `[Unreleased]` compare URL~~
  - [x] ~~[`CONTRIBUTING.md`](../CONTRIBUTING.md) — clone command~~
- [x] ~~**LICENSE copyright**: updated to `Copyright (c) 2026 Agustín Lozano`. Engram attribution paragraph kept at the bottom.~~

### Before going public

- [ ] **Verify Engram attribution**. The LICENSE and README both credit Engram and link to it. Don't strip these — it's the right thing to do and keeps the door open for cross-pollination.
- [ ] **Replace the relative `_engram-research/` paths** in `docs/research/*.md` with permalinks to specific Engram commits on GitHub (e.g. `https://github.com/Gentleman-Programming/engram/blob/<sha>/internal/store/store.go#L789`). Right now they point at the local clone — fine for you, broken for anyone else.
- [ ] **Add a SECURITY.md** if accepting issues from the public (one paragraph: how to report security issues, expected response time).
- [ ] **Optional but nice**: GitHub issue templates (`bug.yml`, `feature.yml`) under `.github/ISSUE_TEMPLATE/`.
- [ ] **Optional**: a `CODE_OF_CONDUCT.md`. We've leaned on a one-liner in CONTRIBUTING for now; bring in a full document if the project grows.

### First push checklist

The repo is already initialized locally (`Desktop/thoughtline/.git`) with one auto-generated `Initial commit` that contains only `.gitattributes`. The remote has not been added yet. From the project root:

```bash
git add .
git commit -m "chore: M0 bootstrap — skeleton, docs, ADRs, taxonomy"
git remote add origin https://github.com/AgusLoza2021/Thoughtline.git
git push -u origin main
```

After pushing:

- [ ] Enable GitHub Actions for the repo (CI is already wired in `.github/workflows/ci.yml`).
- [ ] Tag `v0.0.1` once M1 is done — that's the first version we publish to `pkg.go.dev`.

---

## How to update this file

After every working session:

1. Move the **What got done this session** content to a `## YYYY-MM-DD — milestone-name` history section above this one (keeping `## Current state` at the top).
2. Update the M1 (or active milestone) checklist with what got checked off.
3. Add any new open questions to the table.
4. Strike-through items in the pre-publish TODO list as they're completed.

This file is a living artifact. If it goes stale, the rest of the docs lose their anchor.
