<div align="center">

# Thoughtline

**Persistent, project-aware memory for AI coding assistants — built for game-development workflows.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](go.mod)
[![Status](https://img.shields.io/badge/status-bootstrap-orange)](docs/PROGRESS.md)
[![MCP](https://img.shields.io/badge/MCP-stdio-7C3AED)](#install-planned)

*Save your project's lore. Recall it from any session. Forever.*

</div>

---

## What is Thoughtline?

Thoughtline is an **MCP (Model Context Protocol) server** that gives AI assistants like Claude Code, Cursor, or Zed a long-term memory shaped for how a **game studio actually works**: design decisions, asset references, performance gotchas, scene patterns, pipeline steps. It runs **locally, per developer**, in a single Go binary backed by a SQLite database you own.

Thoughtline stands on the shoulders of [**Engram**](https://github.com/Gentleman-Programming/engram) by Alan Buscaglia — we deliberately reuse Engram's MCP shape, storage layout, and the clever bits like **FTS5 full-text search** and **`topic_key` upserts**. What we add is a **gamedev-first memory taxonomy** and a vocabulary tuned for engines like PlayCanvas, Unity, Unreal, and Godot.

> **Status: bootstrap milestone.** The skeleton is in place; tools are registered in milestone M1. See [`docs/PROGRESS.md`](docs/PROGRESS.md).

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
| **M1 Save**      | `tl_save` end-to-end with SQLite, FTS5 schema, topic-key upsert               | 🟡 next         |
| **M2 Search**    | `tl_search` with FTS5 + BM25 ranking, paginated, filterable by type/scope/project | ⏳ pending     |
| **M3 Context**   | `tl_context` returning recent activity for the active project                 | ⏳ pending     |
| **M4 Sessions**  | `tl_session_start` + `tl_session_summary` to bookend coding sessions          | ⏳ pending     |
| **M5 Smarts**    | Optional embeddings layer for semantic recall — **schema reserved from M1**   | 🔵 deferred     |

See [`docs/PROGRESS.md`](docs/PROGRESS.md) for the live status of M0 and pre-publish TODOs.

---

## Install (planned, not working yet)

> ⚠️ The current `main()` only prints a banner. Real installation lands with M1.

### Once shipped

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
```

Then in your Claude Code MCP config (`.claude/mcp.json` or per-IDE equivalent):

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

A working example lives at [`examples/mcp-config.example.json`](examples/mcp-config.example.json).

### Build from source today

```bash
git clone https://github.com/AgusLoza2021/Thoughtline.git
cd Thoughtline
go build ./cmd/thoughtline
./thoughtline    # prints skeleton banner and exits cleanly
```

---

## Tool catalogue (planned)

All MCP tools share the `tl_` prefix.

| Tool                  | Purpose                                                                                  | Milestone |
| --------------------- | ---------------------------------------------------------------------------------------- | --------- |
| `tl_save`             | Persist a memory; upserts on `topic_key`                                                 | M1        |
| `tl_search`           | FTS5 + BM25 search with optional filters (`type`, `scope`, `project`, `topic_key` glob) | M2        |
| `tl_context`          | Recent activity for the current project (last N memories)                                | M3        |
| `tl_get_observation`  | Fetch full untruncated content of a memory by id                                         | M2        |
| `tl_session_start`    | Mark the start of a coding session, anchor a session id                                  | M4        |
| `tl_session_summary`  | Save a structured end-of-session digest                                                  | M4        |
| `tl_update`           | Update a specific memory by id                                                           | M2        |
| `tl_delete`           | Soft-delete a memory                                                                     | M3        |

The full schema for each will be in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) before M1 implementation.

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
