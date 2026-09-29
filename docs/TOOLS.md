# Tool catalogue & end-to-end examples

---

The full reference for every MCP tool Thoughtline exposes, plus working examples of the JSON arguments an MCP client sends. Skim the catalogue, then jump to the example block that matches what you're trying to do.

## Catalogue

All MCP tools share the `tl_` prefix.

| Tool                  | Purpose                                                                                  | Status |
| --------------------- | ---------------------------------------------------------------------------------------- | ------ |
| `tl_save`             | Persist a memory; upserts on `topic_key`; identical re-saves are noops                   | ✅ M1  |
| `tl_search`           | FTS5 + BM25 search with optional filters (`type`, `scope`, `project`, `topic_key`, `tags`) | ✅ M2  |
| `tl_get_observation`  | Fetch full untruncated content of a memory by id                                         | ✅ M2  |
| `tl_context`          | Recent memories for the active project, ordered by `updated_at DESC`                     | ✅ M3  |
| `tl_update`           | Patch `title` / `content` / `tags` of an existing memory by id                           | ✅ M3  |
| `tl_delete`           | Soft-delete a memory by id; frees its `topic_key` for reuse                              | ✅ M3  |
| `tl_session_start`    | Open a session; returns a UUIDv7 you thread through subsequent `tl_save` calls           | ✅ M4  |
| `tl_session_summary`  | Close a session; persists a structured end-of-session digest. Append-once.               | ✅ M4  |
| `tl_stats`            | Snapshot of memory + session counts. Optional `project` filter (`*` for all)             | ✅ M5  |
| `tl_pending_list`     | Browse hook events captured but not yet filed                                            | ✅ M5  |
| `tl_pending_get`      | Read one captured event, payload included                                                | ✅ M5  |
| `tl_promote`          | File captured events as memories, under the fields you supply                            | ✅ M5  |
| `tl_judge`            | Format a comparison between a stored memory and one you are about to write. Writes nothing. | ✅ M6  |
| `tl_link`             | Record a typed edge from one memory to another                                           | ✅ M6  |
| `tl_related`          | Read every edge on a memory, in either direction                                         | ✅ M6  |


`Status` is the release window a tool first shipped in, not a maturity claim. The pending trio landed with the M5 passive-capture work (2026-05-07); `tl_judge`, `tl_link` and `tl_related` followed in `8f4a6c4` (2026-05-14), which is also when `tl_search` grew its `tags` and `recent_first` arguments.
Architecture: [`ARCHITECTURE.md`](ARCHITECTURE.md). Taxonomy: [`design/memory-domain.md`](design/memory-domain.md). Tags: [`design/tag-conventions.md`](design/tag-conventions.md).

---

## Parameter reference

### `tl_save`

| Param       | Required | Notes                                                                                          |
|-------------|----------|------------------------------------------------------------------------------------------------|
| `title`     | yes      | Short, searchable headline (≤ 200 chars)                                                       |
| `content`   | yes      | Markdown body. Recommended structure: **What** / **Why** / **Where** / **Learned**             |
| `type`      | yes      | One of the fourteen catalogued types — see [`design/memory-domain.md`](design/memory-domain.md). Anything else is refused, never folded to a default |
| `scope`     | no       | `project` (default) or `personal`. The `preference` type auto-defaults to `personal`           |
| `topic_key` | no       | Stable key for evolving topics. Re-saves on the same key replace the stored title, content and tags. Must match `^[a-z0-9][a-z0-9/_.-]{1,128}$` — lowercase, no spaces, no leading slash |
| `project`   | no       | Precedence: `THOUGHTLINE_PROJECT`, then the working-directory basename, then `default`. Pass it explicitly to file into another project |
| `tags`      | no       | Lowercase tags, optionally `key:value`. See [`design/tag-conventions.md`](design/tag-conventions.md) for the canonical vocabulary. |
| `session_id` | no      | Optional UUIDv7 returned by `tl_session_start`. Attaches the memory to that session. Must belong to the same project as the save. Sticky on upsert (omitting it preserves prior linkage). |

### `tl_search`

| Param       | Required | Notes                                                                                          |
|-------------|----------|------------------------------------------------------------------------------------------------|
| `query`     | yes      | Keyword query. Each whitespace-delimited token is a literal phrase (implicit AND). A query containing `/` is tried against `topic_key` first — see the [shortcut](#3-topic-key-shortcut-tried-before-full-text-search) |
| `type`      | no       | Filter — exact match against one of the catalogued types                                       |
| `scope`     | no       | Filter — `project` or `personal`                                                               |
| `project`   | no       | Filter — defaults to the working-directory basename. Pass explicitly to query other projects   |
| `topic_key` | no       | GLOB filter on `topic_key` (e.g. `design/auth/*`). A bare `design/auth` matches the exact key only |
| `limit`     | no       | Max results. Default 10, hard cap 50                                                           |
| `tags`      | no       | Filter — returns memories carrying **every** tag in the list. A tag that matches nothing returns an empty result, not an error |
| `recent_first` | no    | Sort by `updated_at DESC` instead of relevance. Useful when the question is "what did we just do" rather than "what do we know" |
| `offset`    | no       | Pagination offset; pair with `limit` |

Each hit carries `Title`, `ID`, `Sync ID`, `Project`, `Type`, `Scope`, `Topic` (when keyed), `Revision`, `Score`, `Updated`, `Tags` (when present) and `Snippet` (≤ 300 chars, trimmed from the FTS5 `snippet()` in token units). `Score` is BM25 — lower is better, and a topic-key shortcut hit gets a synthetic `-1000` so it sorts above full-text matches. Nothing else is attached: a hit has no health state and no relation annotation. The footer always points at `tl_get_observation` for the full body.

### `tl_get_observation`

| Param | Required | Notes                                                              |
|-------|----------|--------------------------------------------------------------------|
| `id`  | yes      | The local row id from a `tl_save` or `tl_search` response          |

Returns the full untruncated content + every metadata field. Soft-deleted rows return a "not found" error.

### `tl_context`

| Param     | Required | Notes                                                                                  |
|-----------|----------|----------------------------------------------------------------------------------------|
| `project` | no       | Defaults to the working-directory basename. Pass explicitly to peek at another project |
| `limit`   | no       | Max recent memories. Default 10, hard cap 50                                            |

Returns the most recently updated memories (by `updated_at DESC`) for the active project, soft-deleted rows excluded. Same per-result envelope as `tl_search` so an AI parses both with one parser.

### `tl_update`

| Param     | Required | Notes                                                                                          |
|-----------|----------|------------------------------------------------------------------------------------------------|
| `id`      | yes      | The local row id of the memory to patch                                                         |
| `title`   | no       | New title (≤ 200 chars). Omit to leave unchanged                                                |
| `content` | no       | New markdown body. Omit to leave unchanged                                                      |
| `tags`    | no       | New tag list. Pass an empty array (`[]`) to clear all tags. Omit to leave unchanged             |

Identity-defining fields (`type`, `topic_key`, `project`, `scope`) cannot be changed. To "rename" a memory's identity, `tl_delete` the row and `tl_save` a fresh one. Empty patch (no fields supplied) is a noop. Real changes bump `revision_count`, refresh `updated_at`, preserve `id` / `sync_id` / `created_at`. The merged memory is re-validated before persisting.

### `tl_delete`

| Param | Required | Notes                                                                          |
|-------|----------|--------------------------------------------------------------------------------|
| `id`  | yes      | The local row id of the memory to soft-delete                                  |

Sets `deleted_at` and hides the row from `tl_search` / `tl_context` / `tl_get_observation`. The `topic_key` (if any) is freed for a fresh `tl_save`. This is **soft** — the row remains on disk and can be recovered manually. A second `tl_delete` on the same id returns "not found".

### `tl_session_start`

| Param         | Required | Notes                                                                                   |
|---------------|----------|-----------------------------------------------------------------------------------------|
| `project`     | no       | Defaults to the working-directory basename                                              |
| `agent_label` | no       | Tag like `claude-code`, `cursor`, `zed`, `rider`, `visual-studio`. ≤ 64 chars.          |

Returns a UUIDv7 session id. Multiple open sessions per project are allowed — the server is stateless and does not track an "active" session. Thread the returned `session_id` through subsequent `tl_save` calls until you close the session with `tl_session_summary`.

### `tl_session_summary`

| Param     | Required | Notes                                                                                       |
|-----------|----------|---------------------------------------------------------------------------------------------|
| `id`      | yes      | The session id from `tl_session_start`                                                      |
| `summary` | yes      | Structured end-of-session digest (markdown). Recommended: `## Goal / ## Discoveries / ## Accomplished / ## Next Steps / ## Relevant Files`. ≤ 64 KB |

Closes a session **once** — a second `tl_session_summary` on the same id returns "already ended". The digest is durable plain text written for a future session that has no other context.

### `tl_stats`

| Param      | Required | Notes                                                                                    |
|------------|----------|------------------------------------------------------------------------------------------|
| `project`  | no       | Filter to a single project, pass `*` for cross-project totals. Defaults to active project.|

Returns a text snapshot: counts by type / project / scope, the number of open and closed sessions, the four most recently active projects, and the most recent memories and sessions. `RecentLimit` is a storage option defaulting to 10 and clamped at 50, and it is **not** an MCP argument — there is no way to ask for more over the wire.

---
### `tl_judge`

| Param             | Required | Notes                                                                                  |
|-------------------|----------|----------------------------------------------------------------------------------------|
| `existing_id`     | yes      | The stored memory you are comparing against                                            |
| `incoming_title`  | yes      | Title of the memory you are about to write                                             |
| `incoming_content`| yes      | Body of the memory you are about to write                                              |
| `relation`        | yes      | One of `supersedes`, `compatible`, `conflicts_with`, `scoped`, `not_conflict`           |
| `note`            | no       | Free-text rationale, echoed into the output                                            |

Formats a side-by-side comparison and a recommended action. It is **read-only**: it writes nothing, records nothing and detects nothing. You supply the verdict, and the relation set here is deliberately different from `tl_link`'s — it is a vocabulary for judging, not for storing. Use `tl_link` when you want the decision to persist.

### `tl_link`

| Param       | Required | Notes                                                                          |
|-------------|----------|--------------------------------------------------------------------------------|
| `from_id`   | yes      | Source memory                                                                  |
| `to_id`     | yes      | Target memory                                                                  |
| `relation`  | yes      | One of `supersedes`, `contradicts`, `refines`, `depends_on`, `references`, `related`, `derived_from` |
| `note`      | no       | Why the edge exists                                                            |
| `project`   | no       | Defaults to the working-directory basename                                     |

Records a directed edge between two memories. The edge is the only thing that survives a contradiction: nothing in the server detects one, so a memory that supersedes another says so because a writer linked it. Both ids must exist and be live.

### `tl_related`

| Param       | Required | Notes                                                                        |
|-------------|----------|------------------------------------------------------------------------------|
| `id`        | yes      | The memory whose edges you want                                             |
| `direction` | no       | `out`, `in` or `both`. Defaults to `both`                                   |
| `project`   | no       | Defaults to the working-directory basename                                  |

Reads the edges back, with the memory at the other end of each one. This is how you find the replacement for something a search result calls superseded.

### `tl_pending_list`

| Param        | Required | Notes                                                                              |
|--------------|----------|------------------------------------------------------------------------------------|
| `project`    | no       | Defaults to the working-directory basename                                         |
| `status`     | no       | `pending` (default), `promoted` or `dismissed`                                     |
| `event_type` | no       | Filter by hook event — `session-start`, `user-prompt-submit`, `pre-tool-use`, `post-tool-use`, `stop`, `session-end` |
| `since`      | no       | Only events captured after this moment                                            |
| `limit`      | no       | Default 10, hard cap 50                                                            |
| `offset`     | no       | Pagination offset                                                                  |

Lists hook events captured by the Claude Code plugin. Payloads are stored raw — nothing is summarised or extracted — and they are invisible to `tl_search` until promoted. Capture is opt-in: the hooks no-op unless `THOUGHTLINE_PASSIVE_CAPTURE=1` is set. Reads only; this tool declares itself read-only on the wire.

### `tl_pending_get`

| Param | Required | Notes                                    |
|-------|----------|------------------------------------------|
| `id`  | yes      | The pending-event id from `tl_pending_list` |

Returns one captured event with its full payload, which is what you need before deciding whether it is worth filing.

### `tl_promote`

| Param   | Required | Notes                                                                                    |
|---------|----------|------------------------------------------------------------------------------------------|
| `items` | yes      | Array of objects; each has `pending_event_id` plus the memory fields to file it under    |

Each item takes `pending_event_id` and then `type`, `topic_key`, `title`, `content`, `scope` and `tags` — the same vocabulary as `tl_save`, because a promoted event *is* a save. A raw payload does not know what a memory looks like, so it cannot fill these in for you.

Promotion runs per event in its own transaction, so a batch that fails halfway leaves the successful ones saved and the rest pending. This is the only tool that answers with JSON rather than text.

---

## End-to-end flow

The shape your team will use day-to-day. Each example is the **JSON arguments** an MCP client (Claude Code, Cursor, Zed, Rider, …) sends; the AI never types these by hand — it picks them automatically based on the tool descriptions.

### 1. Save a scene-pattern with a stable topic key

```jsonc
// tl_save
{
  "title": "Inn cellar entity hierarchy",
  "type": "scene-pattern",
  "topic_key": "scene/playcanvas/inn-cellar",
  "tags": ["engine:playcanvas", "platform:android"],
  "content": "**What**: cellar split into world/static (chairs, walls) and world/interactive (cellar door).\n**Why**: keeps batching tight on Android.\n**Where**: scenes/inn-cellar.scene\n**Learned**: chairs were originally in interactive — caused 38 extra draw calls."
}
```

Response (abridged):

```
Saved (action=created): "Inn cellar entity hierarchy"
ID: 12
Sync ID: 019dde…-7d70-9044-…
Project: enchanted-inn
Type: scene-pattern
Scope: project
Topic: scene/playcanvas/inn-cellar
Revision: 0
```

A second `tl_save` with the same `topic_key` and identical content is a **noop** (no row mutation). Same key with changed content is an **update** that bumps `revision_count` and refreshes `updated_at` while keeping `id` and `sync_id` stable.

### 2. Search by keyword — FTS5 + BM25

```jsonc
// tl_search
{
  "query": "lantern bloom android",
  "type": "perf-gotcha",
  "limit": 5
}
```

Response sketch:

```
Found 2 result(s) for "lantern bloom android":

Title: Lantern emissive blew out bloom on Android
ID: 7
Sync ID: 019dde…
Project: enchanted-inn
Type: perf-gotcha
Topic: perf/android/lantern-bloom
Revision: 1
Score: -2.4173
Snippet: …reduced lantern emissive intensity from 4.0 to 1.6; bloom threshold raised to 1.2 on Android…
---
Title: …
…

---
Snippets above are previews (≤300 chars, may include '…' ellipses from FTS5). Call tl_get_observation(id: <ID>) to read the full untruncated content of a specific match.
```

### 3. Topic-key shortcut — tried before full-text search

A query containing `/` is matched against `topic_key` first, and FTS5 does not run if it hits.

```jsonc
// Bare key → an exact match, then an implicit trailing wildcard
{ "query": "scene/playcanvas/inn-cellar" }

// Explicit GLOB → all auth design notes. No fallback is added when a
// wildcard is already present.
{ "query": "design/auth/*" }
```

The second attempt is the reason `design/auth` finds `design/auth/jwt`: the key is used as typed, and if that matches nothing it is retried with an implicit `*`. Both attempts have to miss before the query falls through to the full-text path automatically — so no query that returns keyed rows today can change its answer.

The trigger is the `/` in the query, not the shape of the key. A key like `inn-entity-hierarchy` has no slash, so it is only reachable through full-text search.
### 4. Read the full content with `tl_get_observation`

```jsonc
// tl_get_observation
{ "id": 7 }
```

Returns the full untruncated `content`, every metadata field, and the tags list. Use this whenever a `tl_search` snippet is a preview and the AI needs the rest.

### 5. Cross-project lookup

By default `tl_search` filters by the current working directory's project. To query a different project, pass `project` explicitly:

```jsonc
{ "query": "playcanvas batching", "project": "sort-factory-v4" }
```

### 6. Recover context after a session compaction (`tl_context`)

```jsonc
// tl_context — no args needed, defaults to the active project
{}
```

Response sketch:

```
Recent 4 memorie(s) for project "enchanted-inn" (newest first):

Title: Inn cellar entity hierarchy
ID: 12
Sync ID: 019dde…
Project: enchanted-inn
Type: scene-pattern
Topic: scene/playcanvas/inn-cellar
Revision: 0
Updated: 2026-04-30 14:22:11 UTC
Snippet: world/static for chairs; world/interactive for the cellar door…
---
Title: Lantern emissive blew out bloom on Android
…
```

Use this at the start of a session, or right after a context compaction, to recover what was being worked on. Same envelope shape as `tl_search` so the AI parses both with one parser.

### 7. Patch an existing memory (`tl_update`)

```jsonc
// Rename + add a tag
{
  "id": 12,
  "title": "Inn cellar entity hierarchy (post-bake refactor)",
  "tags": ["engine:playcanvas", "platform:android", "post-m4"]
}
```

Response (abridged):

```
Saved (action=updated): "Inn cellar entity hierarchy (post-bake refactor)"
ID: 12
Sync ID: 019dde…   ← preserved
Project: enchanted-inn
Type: scene-pattern  ← cannot be changed via tl_update
Topic: scene/playcanvas/inn-cellar  ← cannot be changed
Revision: 1   ← bumped
```

Type, `topic_key`, `project`, and `scope` are intentionally NOT mutable — they define identity. To "rename" identity: `tl_delete` then `tl_save` fresh. Pass `"tags": []` (empty array) to clear all tags. Omit a field entirely to leave it unchanged.

### 8. Soft-delete a memory (`tl_delete`)

```jsonc
// tl_delete
{ "id": 12 }
```

Response:

```
Deleted memory id=12 (soft delete — row hidden from search/context, topic_key freed).
```

After this:
- `tl_search` no longer returns the row
- `tl_context` no longer lists it
- `tl_get_observation` reports "not found"
- The `topic_key` is available again — a new `tl_save` with the same key creates a fresh row (different `id`, fresh `created_at`)
- A second `tl_delete` on the same id reports "not found"

This is intentionally **soft** so a misclick is recoverable manually from the SQLite file. Hard delete is admin work, off-band.

### 9. Bookend a coding session (`tl_session_start` + `tl_session_summary`)

Open a session at the start of a working block:

```jsonc
// tl_session_start
{ "agent_label": "claude-code" }
```

Response:

```
Session opened.
Session ID: 019dde…-7585-a286-fcb6bf225f45
Project: enchanted-inn
Agent: claude-code
Started: 2026-04-30 14:00:00 UTC

Thread this Session ID through subsequent tl_save calls (session_id arg) to attach memories. Close the session with tl_session_summary when work is done.
```

Then thread that id through every `tl_save` while the session is open:

```jsonc
{
  "title": "Lantern bake notes",
  "content": "raised bloom threshold to 1.2 on Android",
  "type": "perf-gotcha",
  "topic_key": "perf/android/bloom",
  "session_id": "019dde…-7585-a286-fcb6bf225f45"
}
```

**Sticky `session_id`**: if you re-save the same `topic_key` later without passing `session_id`, the prior linkage is preserved. To overwrite, pass an explicit (different) `session_id`. By design, there is no path to "detach" a memory from its session — once attached, it stays attached historically.

When the working block is done, close with a structured digest:

```jsonc
// tl_session_summary
{
  "id": "019dde…-7585-a286-fcb6bf225f45",
  "summary": "## Goal\nBake lanterns and tame bloom on Android.\n## Accomplished\n- Lantern emissive 4.0 → 1.6\n- Bloom threshold 1.0 → 1.2\n## Next Steps\n- Verify on Pixel 6"
}
```

A session can only be closed **once**. Calling `tl_session_summary` again on the same id returns "already ended".

### 10. Cross-project session attachment is rejected

A session belongs to exactly one project. Trying to attach a memory in project A to a session in project B is rejected at save time:

```jsonc
// session was opened with project="alpha"
// but this save is for project="beta" → rejected
{
  "title": "Mismatched save",
  "content": "...",
  "type": "convention",
  "project": "beta",
  "session_id": "<uuid for alpha session>"
}
```

Returns an error: *"session_id … belongs to a different project than this save. Sessions are scoped to a single project."*
