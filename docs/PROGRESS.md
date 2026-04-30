# Progress

Single source of truth for "what's done, what's next, what's blocking publishing." Updated at the end of every working session.

---

## Current state — 2026-04-30

**Milestone M3 (Context, Update, Delete): 🟢 done.** M0–M2 are also 🟢 done — see history below.

### What got done this session (M3)

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

### What's NOT done (intentionally)

- No `tl_session_*` — that's M4.
- No hard-delete tool. Recovery is admin work via SQLite directly.
- No `tl_update` for `type` / `topic_key` / `project` / `scope` (identity-defining). The AI can `tl_delete` + `tl_save` if it really needs to re-cast a memory.
- LICENSE copyright still says "Thoughtline contributors" — pre-publish TODO.

---

## Next session — Milestone M4 (Sessions)

**Goal**: bookend coding sessions so the AI has a stable narrative across compactions.

### M4 definition of done

- [ ] Domain layer:
  - [ ] `Session` type with `id` (UUIDv7), `project`, `started_at`, `ended_at`, optional `summary`.
  - [ ] Validation: project required, ended_at >= started_at, summary ≤ MaxContentBytes.
- [ ] Storage layer:
  - [ ] `sessions` table + index on `(project, started_at DESC)`.
  - [ ] `StartSession(ctx, project) (Session, error)`.
  - [ ] `EndSession(ctx, sessionID, summary) (Session, error)`.
  - [ ] `GetSession(ctx, id) (Session, error)`.
  - [ ] Tests: start, end without summary, end with summary, end-twice rejection, project filter.
- [ ] Server layer:
  - [ ] `tl_session_start` — returns the session id; AI prepends it to subsequent saves (or stores it client-side).
  - [ ] `tl_session_summary` — takes session id + structured summary text, persists, returns confirmation.
  - [ ] Decide: do we add `session_id` as a column on `memories` to associate saves to a session? Likely yes — opens future "what did we discuss in session X?" queries.
- [ ] Docs:
  - [ ] CHANGELOG entry.
  - [ ] README tool catalogue + Examples (sections 9 and 10).
  - [ ] Mark M4 done here, sketch M5 (embeddings) plan or close as deferred.

### M4 open questions

| # | Question | Plan to resolve |
|---|----------|----------------|
| 1 | Where does the AI store the session id between calls? | Tool response includes the id; the AI is expected to thread it through subsequent saves. We don't keep server-side per-client state. |
| 2 | Should `memories.session_id` be a foreign key or a free-form text? | Foreign key with `ON DELETE SET NULL`. Sessions are durable; deleting a session shouldn't cascade-delete its memories. |
| 3 | Auto-end stale sessions? | No in M4. If the AI doesn't call `tl_session_summary`, the row stays open. We can add an `auto_close_after` heuristic in M5+ if this gets messy. |

---

## Milestone history

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
