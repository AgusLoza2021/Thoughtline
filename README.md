<div align="center">

# Thoughtline

**Persistent, project-aware memory for AI coding assistants — built for game-development workflows.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](go.mod)
[![Status](https://img.shields.io/badge/status-M3%20done-yellow)](docs/PROGRESS.md)
[![MCP](https://img.shields.io/badge/MCP-stdio-7C3AED)](#install-planned)

*Save your project's lore. Recall it from any session. Forever.*

</div>

---

## What is Thoughtline?

Thoughtline is an **MCP (Model Context Protocol) server** that gives AI assistants like Claude Code, Cursor, or Zed a long-term memory shaped for how a **game studio actually works**: design decisions, asset references, performance gotchas, scene patterns, pipeline steps. It runs **locally, per developer**, in a single Go binary backed by a SQLite database you own.

Thoughtline stands on the shoulders of [**Engram**](https://github.com/Gentleman-Programming/engram) by Alan Buscaglia — we deliberately reuse Engram's MCP shape, storage layout, and the clever bits like **FTS5 full-text search** and **`topic_key` upserts**. What we add is a **gamedev-first memory taxonomy** and a vocabulary tuned for engines like PlayCanvas, Unity, Unreal, and Godot.

> **Status: M3 done.** Six tools live: `tl_save`, `tl_search`, `tl_get_observation`, `tl_context`, `tl_update`, `tl_delete`. The session bookends land in M4. See [`docs/PROGRESS.md`](docs/PROGRESS.md).

---

## Why this exists

Generic AI memory tools speak the language of backend engineers: `bugfix`, `decision`, `architecture`, `pattern`. Useful — but nowhere near the texture of building games.

When a gamedev team works on a real title, they routinely need to remember things like:

- *"Why is the inn entity hierarchy split into `world/static` and `world/interactive`?"*
- *"What were the import settings we landed on for the lantern texture so it didn't blow out the bloom?"*
- *"Which script handles the camera nudge when the player enters the cellar?"*
- *"That batching gotcha on Android with the chairs — what was the exact draw-call count we couldn't cross?"*

None of these fit cleanly into a generic memory schema. **Thoughtline captures them as first-class memory types** so search, recall, and cross-session context all stay sharp instead of being squeezed into a `pattern` bucket.

---

## What it is — and isn't

| ✅ Thoughtline IS                                                              | ❌ Thoughtline is NOT                                                |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------- |
| A local MCP server, one binary, single SQLite file you own                     | A cloud service or shared team store                                |
| Single-user. Each dev runs their own instance                                  | Multi-tenant or multi-user                                          |
| A tool an AI assistant calls — not something a human queries by hand most days | A project management tool, wiki, or note-taking app                 |
| Opinionated about gamedev memory types                                         | A general-purpose key-value store                                   |
| Architecture-compatible with Engram (same MCP shape, similar storage)          | A drop-in fork of Engram                                            |

If you are **not** a gamedev and want a generic memory MCP, **use [Engram](https://github.com/Gentleman-Programming/engram) directly** — it is excellent. Thoughtline only exists because we wanted gamedev concepts at the schema level.

---

## Architecture at a glance

```mermaid
graph LR
    Client["AI client<br/>(Claude Code, Cursor, ...)"] -- "MCP / stdio JSON-RPC" --> Server["thoughtline<br/>MCP server"]
    Server --> Tools["Tool layer<br/>tl_save · tl_search · tl_context · ..."]
    Tools --> Domain["Memory domain<br/>(types, topic keys, scopes)"]
    Domain --> Storage["Storage<br/>(SQLite + FTS5 / BM25)"]
    Storage --> DB[("thoughtline.db<br/>local file")]

    classDef ext fill:#1e1e2e,stroke:#7C3AED,color:#fff
    classDef core fill:#181825,stroke:#00ADD8,color:#fff
    classDef store fill:#11111b,stroke:#a6e3a1,color:#fff
    class Client ext
    class Server,Tools,Domain core
    class Storage,DB store
```

Full breakdown in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). The decisions behind it (and why we picked FTS5 over embeddings for v1) live in [`docs/decisions/`](docs/decisions/).

---

## The Thoughtline memory taxonomy (preview)

The full taxonomy lives in [`docs/design/memory-domain.md`](docs/design/memory-domain.md). Headline types:

| Type                   | What it captures                                                         |
| ---------------------- | ------------------------------------------------------------------------ |
| `game-design-decision` | Design choices and the reasoning behind them                             |
| `scene-pattern`        | Recurring entity hierarchies / component setups in your engine of choice |
| `asset-reference`      | Path, version, import settings, and origin of a model/texture/sound      |
| `perf-gotcha`          | Performance traps you only learn by hitting them (drawcalls, GC, batching) |
| `pipeline-step`        | A reproducible step in your asset/build pipeline (Blender → engine, etc.) |
| `script-pattern`       | An engine-script idiom worth remembering                                 |
| `bugfix`               | Bug + root cause + fix, with engine/platform tags                        |
| `convention`           | Naming, structure, project-wide rules                                    |
| `preference`           | Per-developer ergonomics                                                 |

Each memory carries the same envelope: `topic_key`, `scope`, `project`, `created_at`, `revision_count`, free-form content.

---

## Roadmap

The work is sliced into milestones. Each one has a definition of done, so progress is unambiguous.

| Milestone        | Goal                                                                          | Status         |
| ---------------- | ----------------------------------------------------------------------------- | -------------- |
| **M0 Bootstrap** | Skeleton repo, full docs, ADRs, taxonomy design, research baseline            | 🟢 done         |
| **M1 Save**      | `tl_save` end-to-end with SQLite, FTS5 schema, topic-key upsert               | 🟢 done         |
| **M2 Search**    | `tl_search` with FTS5 + BM25 ranking, paginated, filterable by type/scope/project; `tl_get_observation` companion | 🟢 done         |
| **M3 Context**   | `tl_context` (recent activity), `tl_update` (patch by id), `tl_delete` (soft delete) | 🟢 done         |
| **M4 Sessions**  | `tl_session_start` + `tl_session_summary` to bookend coding sessions          | 🟡 next         |
| **M5 Smarts**    | Optional embeddings layer for semantic recall — **schema reserved from M1**   | 🔵 deferred     |

See [`docs/PROGRESS.md`](docs/PROGRESS.md) for the live milestone status and pre-publish TODOs.

---

## Install

> ✅ As of M3, six tools are live: `tl_save`, `tl_search`, `tl_get_observation`, `tl_context`, `tl_update`, `tl_delete`. Session bookends (`tl_session_*`) come in M4.

### From source

```bash
git clone https://github.com/AgusLoza2021/Thoughtline.git
cd Thoughtline
go build ./cmd/thoughtline
```

This produces a `thoughtline` binary in the project root. Move it onto your `PATH` or reference it by absolute path in your MCP client config.

### Once a release is tagged

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
```

### Wire it into your MCP client

Claude Code (`.claude/mcp.json` or per-IDE equivalent):

```jsonc
{
  "mcpServers": {
    "thoughtline": {
      "command": "thoughtline",
      "args": []
    }
  }
}
```

A complete example lives at [`examples/mcp-config.example.json`](examples/mcp-config.example.json).

### Configuration (env vars)

| Variable | Purpose | Default |
|----------|---------|---------|
| `THOUGHTLINE_DB` | Full path to the SQLite file. Wins over `THOUGHTLINE_HOME` if set. | (unset) |
| `THOUGHTLINE_HOME` | Directory holding the database file (`thoughtline.db`). | OS user-cache dir + `/thoughtline` |
| `THOUGHTLINE_PROJECT` | Default project identifier when a tool call omits `project`. | basename of working directory |

---

## Tool catalogue

All MCP tools share the `tl_` prefix.

| Tool                  | Purpose                                                                                  | Status |
| --------------------- | ---------------------------------------------------------------------------------------- | ------ |
| `tl_save`             | Persist a memory; upserts on `topic_key`; identical re-saves are noops                   | ✅ M1  |
| `tl_search`           | FTS5 + BM25 search with optional filters (`type`, `scope`, `project`, `topic_key` glob) | ✅ M2  |
| `tl_get_observation`  | Fetch full untruncated content of a memory by id                                         | ✅ M2  |
| `tl_context`          | Recent memories for the active project, ordered by `updated_at DESC`                     | ✅ M3  |
| `tl_update`           | Patch `title` / `content` / `tags` of an existing memory by id                           | ✅ M3  |
| `tl_delete`           | Soft-delete a memory by id; frees its `topic_key` for reuse                              | ✅ M3  |
| `tl_session_start`    | Mark the start of a coding session, anchor a session id                                  | ⏳ M4  |
| `tl_session_summary`  | Save a structured end-of-session digest                                                  | ⏳ M4  |

Full architecture in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). Memory taxonomy in [`docs/design/memory-domain.md`](docs/design/memory-domain.md).

### `tl_save` parameters

| Param       | Required | Notes                                                                                          |
|-------------|----------|------------------------------------------------------------------------------------------------|
| `title`     | yes      | Short, searchable headline (≤ 200 chars)                                                       |
| `content`   | yes      | Markdown body. Recommended structure: **What** / **Why** / **Where** / **Learned**             |
| `type`      | yes      | One of the 9 catalogued types — see [memory-domain.md](docs/design/memory-domain.md)           |
| `scope`     | no       | `project` (default) or `personal`. The `preference` type auto-defaults to `personal`           |
| `topic_key` | no       | Stable key for evolving topics. Re-saves on the same key upsert (lowercase / `[a-z0-9/_-]`)    |
| `project`   | no       | Defaults to the working-directory basename (or `THOUGHTLINE_PROJECT` if set)                   |
| `tags`      | no       | Lowercase tags, optionally `key:value` (e.g. `engine:playcanvas`, `platform:android`)          |

### `tl_search` parameters

| Param       | Required | Notes                                                                                          |
|-------------|----------|------------------------------------------------------------------------------------------------|
| `query`     | yes      | Keyword query. Each whitespace-delimited token is a literal phrase (implicit AND). Containing `/` triggers the topic-key GLOB shortcut |
| `type`      | no       | Filter — exact match against one of the catalogued types                                       |
| `scope`     | no       | Filter — `project` or `personal`                                                               |
| `project`   | no       | Filter — defaults to the working-directory basename. Pass explicitly to query other projects   |
| `topic_key` | no       | GLOB filter on `topic_key` (e.g. `design/auth/*`)                                              |
| `limit`     | no       | Max results. Default 10, hard cap 50                                                           |
| `offset`    | no       | Pagination offset; pair with `limit`                                                           |

Results carry: `id`, `sync_id`, `title`, `snippet` (≤ 300 chars from FTS5 `snippet()`), `score` (BM25 — lower = better; topic-key shortcut hits get a synthetic `-1000`), plus all metadata. The response footer always points at `tl_get_observation` for full content.

### `tl_get_observation` parameters

| Param | Required | Notes                                                              |
|-------|----------|--------------------------------------------------------------------|
| `id`  | yes      | The local row id from a `tl_save` or `tl_search` response          |

Returns the full untruncated content + every metadata field. Soft-deleted rows return a "not found" error.

### `tl_context` parameters

| Param     | Required | Notes                                                                                  |
|-----------|----------|----------------------------------------------------------------------------------------|
| `project` | no       | Defaults to the working-directory basename. Pass explicitly to peek at another project |
| `limit`   | no       | Max recent memories. Default 10, hard cap 50                                            |

Returns the most recently updated memories (by `updated_at DESC`) for the active project, soft-deleted rows excluded. Same per-result envelope as `tl_search` so an AI parses both with one parser.

### `tl_update` parameters

| Param     | Required | Notes                                                                                          |
|-----------|----------|------------------------------------------------------------------------------------------------|
| `id`      | yes      | The local row id of the memory to patch                                                         |
| `title`   | no       | New title (≤ 200 chars). Omit to leave unchanged                                                |
| `content` | no       | New markdown body. Omit to leave unchanged                                                      |
| `tags`    | no       | New tag list. Pass an empty array (`[]`) to clear all tags. Omit to leave unchanged             |

Identity-defining fields (`type`, `topic_key`, `project`, `scope`) cannot be changed. To "rename" a memory's identity, `tl_delete` the row and `tl_save` a fresh one. Empty patch (no fields supplied) is a noop. Real changes bump `revision_count`, refresh `updated_at`, preserve `id` / `sync_id` / `created_at`. The merged memory is re-validated before persisting.

### `tl_delete` parameters

| Param | Required | Notes                                                                          |
|-------|----------|--------------------------------------------------------------------------------|
| `id`  | yes      | The local row id of the memory to soft-delete                                  |

Sets `deleted_at` and hides the row from `tl_search` / `tl_context` / `tl_get_observation`. The `topic_key` (if any) is freed for a fresh `tl_save`. This is **soft** — the row remains on disk and can be recovered manually. A second `tl_delete` on the same id returns "not found".

---

## Examples — end-to-end flow

The shape your team will use day-to-day. Each example is the **JSON arguments** an MCP client (Claude Code, Cursor, Zed, ...) sends; the AI never types these by hand — it picks them automatically based on the tool descriptions.

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

### 3. Topic-key shortcut — exact lookup or GLOB

A query containing `/` is matched against `topic_key` first. If any rows match, FTS5 does **not** run.

```jsonc
// Exact lookup → O(1)
{ "query": "scene/playcanvas/inn-cellar" }

// GLOB lookup → all auth design notes
{ "query": "design/auth/*" }
```

If the shortcut returns zero rows, the query falls through to the FTS5 path automatically.

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

---

## Repo layout

```
thoughtline/
├── cmd/thoughtline/         # binary entry point
├── internal/
│   ├── server/              # MCP server wiring (mark3labs/mcp-go)
│   ├── storage/             # SQLite persistence (modernc.org/sqlite)
│   └── memory/              # domain types: Memory, Type, Scope, TopicKey
├── docs/
│   ├── ARCHITECTURE.md      # how the system fits together
│   ├── PROGRESS.md          # live milestone status + pre-publish TODOs
│   ├── research/            # deep-dives into Engram (the reference architecture)
│   ├── design/              # taxonomies, schemas, vocabulary
│   └── decisions/           # ADRs — every load-bearing decision lives here
├── examples/                # MCP client config examples
├── .github/workflows/       # CI: build + vet on every push
├── go.mod
├── LICENSE                  # MIT, with attribution to Engram
├── CHANGELOG.md
├── CONTRIBUTING.md
└── README.md                # you are here
```

---

## Documentation index

| Doc | Purpose |
|-----|---------|
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Components, data flow, schema sketch |
| [`docs/PROGRESS.md`](docs/PROGRESS.md) | What's done, what's next, pre-publish TODOs |
| [`docs/design/memory-domain.md`](docs/design/memory-domain.md) | Full memory type taxonomy with examples |
| [`docs/decisions/0001-architecture-baseline.md`](docs/decisions/0001-architecture-baseline.md) | Why we built on Engram's foundations |
| [`docs/decisions/0002-search-strategy-fts5-first.md`](docs/decisions/0002-search-strategy-fts5-first.md) | FTS5 in v1, embeddings deferred (with reasoning) |
| [`docs/research/engram-anatomy.md`](docs/research/engram-anatomy.md) | Engram codebase map |
| [`docs/research/flow-mem-save.md`](docs/research/flow-mem-save.md) | Engram's `mem_save` traced end-to-end |
| [`docs/research/flow-mem-search.md`](docs/research/flow-mem-search.md) | Engram's search internals — the most important doc |

---

## Credits and lineage

Thoughtline would not exist without **Engram** by [Alan Buscaglia](https://github.com/Gentleman-Programming) and the Gentleman-Programming community. We read its source carefully (see `docs/research/`), credit the design choices we keep, and stay MIT-compatible so improvements can flow back upstream.

If you are evaluating memory tools for a non-gamedev team, **use Engram directly** — it is more general and more battle-tested. Thoughtline is a gamedev-flavored sibling, not a competitor.

---

## License

MIT — see [LICENSE](LICENSE).

## Contributing

PRs, issues, and design feedback welcome. See [CONTRIBUTING.md](CONTRIBUTING.md). Be kind, be precise, cite source lines.
