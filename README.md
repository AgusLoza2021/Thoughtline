<div align="center">

# Thoughtline

**Persistent, project-aware memory for AI coding assistants — built for game-development workflows.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go)](go.mod)
[![Status](https://img.shields.io/badge/status-M5%20done-brightgreen)](docs/PROGRESS.md)
[![MCP](https://img.shields.io/badge/MCP-stdio-7C3AED)](#install-planned)

*Save your project's lore. Recall it from any session. Forever.*

</div>

---

## What is this?

**Memory for your AI coding assistant. Local. Forever.**

Your AI forgets everything between sessions. Thoughtline is a single SQLite file on your machine that the AI writes to while you work and reads from when you come back. No cloud, no account, no subscription.

### What it actually saves you

You stop re-explaining things like:

- Why the inn entity is split into `world/static` and `world/interactive`
- The lantern texture import settings that don't blow out the bloom
- Which script handles the camera nudge when the player enters the cellar
- That Android batching gotcha on the chairs — the draw-call count you couldn't cross
- The animation curve numbers your animator finally locked in

Memories are typed (`scene-pattern`, `asset-reference`, `perf-gotcha`, `pipeline-step`, `script-pattern`) and tagged (`engine:unity`, `platform:switch`, `pipeline:fbx-to-godot`) so you can search them later.

**Any engine** — Unity, Unreal, Godot, PlayCanvas, Bevy, your own.
**Any AI tool that speaks MCP** — Claude Code, Cursor, Zed, Rider, Visual Studio, JetBrains.

> 📸 _Screenshots of the dashboard go here. Until then: `thoughtline ui` and see for yourself — a six-tab memory workspace (Home, Memories, Search, Inbox, Sessions, Help) with global quick actions, an Inbox you can accept/edit/reject from, and a read-only Detail view with [C] copy-to-clipboard. Golden snapshots of every tab live under `internal/dashboard/testdata/*.golden` for layout reference._

---

## Quick start

### Windows — one command (auto-installer)

From **PowerShell** (recommended):

```powershell
irm https://raw.githubusercontent.com/AgusLoza2021/Thoughtline/main/scripts/install.ps1 | iex
```

From **classic `cmd.exe`**:

```cmd
powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/AgusLoza2021/Thoughtline/main/scripts/install.ps1 | iex"
```

The installer is **idempotent** and **does not need admin/UAC**. It will:

1. Install Go via `winget` if missing
2. Run `go install` for the `thoughtline` binary
3. Add `%USERPROFILE%\go\bin` to your user PATH
4. Register `thoughtline` as an MCP server in `%USERPROFILE%\.claude.json` (Claude Code CLI **and** the VS Code extension share this config — one entry covers both)
5. Verify with `thoughtline version`

After it finishes, **restart Claude Code** (or VS Code with the Claude Code extension) and the `tl_*` tools are live.

Already cloned the repo? You can run the script locally:

```powershell
.\scripts\install.ps1                  # default
.\scripts\install.ps1 -Version v0.0.1  # pin a version
.\scripts\install.ps1 -Force           # overwrite an existing MCP entry
```

Flags: `-Version <ref>`, `-SkipPathSetup`, `-SkipMcp`, `-Force`. See [scripts/install.ps1](scripts/install.ps1).

### Manual install (macOS / Linux / Windows)

**Install** (Go 1.25+):

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
```

Or grab a prebuilt binary from the [latest release](https://github.com/AgusLoza2021/Thoughtline/releases/latest).

**Add to Claude Code** — drop this into `~/.claude.json` (or `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "thoughtline": {
      "command": "thoughtline",
      "args": ["serve"]
    }
  }
}
```

**Open the dashboard:**

```bash
thoughtline ui
```

That's it. Restart your AI client and the `tl_save`, `tl_search`, `tl_context`, … tools are available.

### Or install the Claude Code plugin

If you use Claude Code, the [official plugin](plugin/claude-code/) wires everything up — MCP server registration, `SessionStart`/post-compaction hooks that inject the active-protocol markdown, slash commands (`/tl-recent`, `/tl-search`, `/tl-stats`, `/tl-ui`, `/tl-export`), a `tl-archivist` subagent for memory hygiene, and the `thoughtline-memory` skill that tells Claude when to save and recall.

```bash
claude plugin install github:AgusLoza2021/Thoughtline/plugin/claude-code
```

The plugin is **cross-platform** — it shells out to the `thoughtline protocol` subcommand instead of bash scripts, so Windows users do not need WSL or Git Bash.

See [plugin/claude-code/README.md](plugin/claude-code/README.md) for the full plugin reference.

---

## What is Thoughtline?

Thoughtline is an **MCP (Model Context Protocol) server** that gives AI assistants like Claude Code, Cursor, or Zed a long-term memory shaped for how a **game studio actually works**: design decisions, asset references, performance gotchas, scene patterns, pipeline steps. It runs **locally, per developer**, in a single Go binary backed by a SQLite database you own.

Thoughtline stands on the shoulders of [**Engram**](https://github.com/Gentleman-Programming/engram) by Alan Buscaglia — we deliberately reuse Engram's MCP shape, storage layout, and the clever bits like **FTS5 full-text search** and **`topic_key` upserts**. What we add is a **gamedev-first memory taxonomy** and a vocabulary tuned for engines like PlayCanvas, Unity, Unreal, and Godot.

> Curious how Thoughtline lines up against Engram and [claude-mem](https://github.com/thedotmack/claude-mem)? See **[docs/COMPARISON.md](docs/COMPARISON.md)** for an honest side-by-side.

> **Status: M5 done + passive capture.** Twelve tools live: `tl_save`, `tl_search`, `tl_get_observation`, `tl_context`, `tl_update`, `tl_delete`, `tl_session_start`, `tl_session_summary`, `tl_stats`, plus `tl_pending_list`, `tl_pending_get`, `tl_promote` for opt-in passive capture of Claude Code hook events. See [docs/integrations/claude-code-passive-capture.md](docs/integrations/claude-code-passive-capture.md) for the setup guide. M6 (semantic embeddings) remains deferred per [ADR 0002](docs/decisions/0002-search-strategy-fts5-first.md). See [`docs/PROGRESS.md`](docs/PROGRESS.md).

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

## How it compares

There are several memory tools out there. Here's where Thoughtline sits:

| Capability                              | Thoughtline      | [Engram](https://github.com/Gentleman-Programming/engram) | Cursor "memories"   | ChatGPT memory       | Manual notes (Notion/Obsidian) |
|-----------------------------------------|------------------|-----------|---------------------|----------------------|---------------------------------|
| Local-first, your data on your machine  | ✅               | ✅         | ❌ cloud             | ❌ cloud              | ✅                              |
| Works across **multiple AI clients**    | ✅ MCP           | ✅ MCP     | ❌ Cursor only       | ❌ ChatGPT only       | ✅ but read-only to AI          |
| AI auto-saves (no copy-paste)           | ✅               | ✅         | ✅ implicit          | ✅ implicit           | ❌ you write everything         |
| AI auto-recalls in new sessions         | ✅               | ✅         | ✅                   | ✅                    | ❌ you have to remind it        |
| Survives context compactions            | ✅               | ✅         | ⚠️ partial           | ⚠️ partial            | ✅                              |
| Per-project scoping (no cross-bleed)    | ✅               | ✅         | ⚠️ workspace-only    | ❌ global             | ✅ folder structure             |
| Game-dev memory types at the schema level | ✅ 11 types    | ❌ generic | ❌ free-form         | ❌ free-form          | ❌ you invent the structure     |
| Engine / platform / pipeline tag vocab  | ✅ canonical     | ❌         | ❌                   | ❌                    | ❌                              |
| Topic-key upserts (no duplicates)       | ✅               | ✅         | ❌                   | ❌                    | manual                          |
| Sessions with structured digests        | ✅ UUIDv7        | ⚠️ basic   | ❌                   | ❌                    | manual                          |
| Interactive TUI (`thoughtline ui`)      | ✅ Bubbletea     | ✅ TUI     | ❌                   | ❌                    | n/a                             |
| Cost                                    | free, MIT        | free, MIT  | bundled with Cursor  | bundled with ChatGPT  | free–paid                       |

**TL;DR**: if you ship games and your team uses MCP-capable tools, Thoughtline is the one shaped for you. If you don't ship games, [Engram](https://github.com/Gentleman-Programming/engram) is the better generic choice — Thoughtline literally builds on it.

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
| `preference`           | Per-developer ergonomics (scope = personal)                              |
| `decision`             | Technical or architectural decisions with rationale (project-scoped)     |
| `architecture`         | System-level structural knowledge: packages, boundaries, contracts       |

Each memory carries the same envelope: `topic_key`, `scope`, `project`, `created_at`, `revision_count`, free-form content.

---

## Migrating from Engram

If you're already using [Engram](https://github.com/Gentleman-Programming/engram) and want to move your memories to Thoughtline, `cmd/migrate` is your tool.

**Prerequisite — back up first:**
```powershell
Compress-Archive -Path "$env:USERPROFILE\.engram\*" -DestinationPath "$env:USERPROFILE\.engram-backup-$(Get-Date -Format 'yyyy-MM-dd').zip" -Force
```

**Three commands:**
```bash
# 1. Build the migrator
go build -o migrate.exe ./cmd/migrate/cmd

# 2. Dry run — verify counts without writing
./migrate.exe --dry-run --verbose

# 3. Production run
./migrate.exe
```

See [`cmd/migrate/README.md`](cmd/migrate/README.md) for the full guide: type mapping table, error handling, idempotency guarantee, and log file location.

---

## Roadmap

The work is sliced into milestones. Each one has a definition of done, so progress is unambiguous.

| Milestone        | Goal                                                                          | Status         |
| ---------------- | ----------------------------------------------------------------------------- | -------------- |
| **M0 Bootstrap** | Skeleton repo, full docs, ADRs, taxonomy design, research baseline            | 🟢 done         |
| **M1 Save**      | `tl_save` end-to-end with SQLite, FTS5 schema, topic-key upsert               | 🟢 done         |
| **M2 Search**    | `tl_search` with FTS5 + BM25 ranking, paginated, filterable by type/scope/project; `tl_get_observation` companion | 🟢 done         |
| **M3 Context**   | `tl_context` (recent activity), `tl_update` (patch by id), `tl_delete` (soft delete) | 🟢 done         |
| **M4 Sessions**  | `tl_session_start` + `tl_session_summary`; `tl_save` learns optional `session_id`    | 🟢 done         |
| **M5 Dashboard** | `thoughtline ui` interactive TUI + `tl_stats` MCP tool                              | 🟢 done         |
| **M6 Smarts**    | Optional embeddings layer for semantic recall — **schema reserved from M1**   | 🔵 deferred     |

See [`docs/PROGRESS.md`](docs/PROGRESS.md) for the live milestone status and pre-publish TODOs.

---

## Install

> ✅ As of M5, nine MCP tools + an interactive dashboard ship: `tl_save`, `tl_search`, `tl_get_observation`, `tl_context`, `tl_update`, `tl_delete`, `tl_session_start`, `tl_session_summary`, `tl_stats`, plus `thoughtline ui`.

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

Per-tool guides:

- **Claude Code** — see [`docs/integrations/claude-code-protocol.md`](docs/integrations/claude-code-protocol.md), or use the [bundled plugin](plugin/claude-code/README.md) for one-line install
- **Cursor** — [`docs/integrations/cursor.md`](docs/integrations/cursor.md)
- **Zed** — [`docs/integrations/zed.md`](docs/integrations/zed.md)
- **JetBrains Rider (Unity)** — [`docs/integrations/rider-unity.md`](docs/integrations/rider-unity.md)
- **Visual Studio (Unreal)** — see Cursor or Rider guides; same MCP shape

Generic MCP config (any client):

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

All MCP tools share the `tl_` prefix. The full reference — every parameter, every response shape, every edge case — lives in [**`docs/TOOLS.md`**](docs/TOOLS.md). Skim the table below; jump there when you need the details.

| Tool                  | Purpose                                                                                  | Status |
| --------------------- | ---------------------------------------------------------------------------------------- | ------ |
| `tl_save`             | Persist a memory; upserts on `topic_key`; identical re-saves are noops                   | ✅ M1  |
| `tl_search`           | FTS5 + BM25 search with optional filters (`type`, `scope`, `project`, `topic_key` glob) | ✅ M2  |
| `tl_get_observation`  | Fetch full untruncated content of a memory by id                                         | ✅ M2  |
| `tl_context`          | Recent memories for the active project, ordered by `updated_at DESC`                     | ✅ M3  |
| `tl_update`           | Patch `title` / `content` / `tags` of an existing memory by id                           | ✅ M3  |
| `tl_delete`           | Soft-delete a memory by id; frees its `topic_key` for reuse                              | ✅ M3  |
| `tl_session_start`    | Open a session; returns a UUIDv7 you thread through subsequent `tl_save` calls           | ✅ M4  |
| `tl_session_summary`  | Close a session; persists a structured end-of-session digest. Append-once.               | ✅ M4  |
| `tl_stats`            | Snapshot of memory + session counts. Optional `project` filter (`*` for all)             | ✅ M5  |

### One concrete example — save & search

```jsonc
// tl_save — capture an inn-cellar scene pattern
{
  "title": "Inn cellar entity hierarchy",
  "type": "scene-pattern",
  "topic_key": "scene/playcanvas/inn-cellar",
  "tags": ["engine:playcanvas", "platform:android"],
  "content": "**What**: cellar split into world/static and world/interactive. **Why**: keeps batching tight on Android. **Learned**: chairs in interactive caused +38 draw calls."
}

// Later — recall it
{ "query": "scene/playcanvas/*" }    // topic-key GLOB, O(1)
{ "query": "lantern bloom android" } // FTS5 + BM25 keyword search
```

For ten more end-to-end examples — sessions, soft-delete, cross-project queries, sticky session_id — see [`docs/TOOLS.md`](docs/TOOLS.md#end-to-end-flow).

Full architecture in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). Memory taxonomy in [`docs/design/memory-domain.md`](docs/design/memory-domain.md). Tag conventions in [`docs/design/tag-conventions.md`](docs/design/tag-conventions.md).

<!-- Tool parameter tables and end-to-end examples moved to docs/TOOLS.md to keep this README scannable. -->

_(Full parameter reference for every tool moved to [`docs/TOOLS.md`](docs/TOOLS.md#parameter-reference).)_

---

## Interactive dashboard (`thoughtline ui`)

When you want a visual at-a-glance view of what's stored — without firing up the AI — Thoughtline ships an interactive terminal **memory workspace** (Bubbletea TUI). It is organised as six tabs reachable via the digit keys `1`–`6` or `tab` / `shift+tab` to cycle:

| # | Tab | What it does |
|---|------|--------------|
| 1 | Home | Project health card, latest memories, recent activity, first-run empty state |
| 2 | Memories | Paginated list with type/tag/scope/sort filters; `f` cycles focus, `c` clears |
| 3 | Search | Full-text FTS5 search with live results |
| 4 | Inbox | Pending captures — `[A]` accept as-is, `[E]` edit-then-promote, `[R]` reject |
| 5 | Sessions | Recent sessions across all projects (read-only) |
| 6 | Help | Keybindings reference + roadmap snapshot |

```bash
thoughtline ui                    # launch the workspace
```

> 📸 _TODO: replace these placeholders with real PNGs/GIFs before launch._
>
> Layout snapshots for every tab live under `internal/dashboard/testdata/*.golden` (rendered at 100×30) and are the source of truth for visual contracts.

**Global hotkeys** (active when no input is focused): `[S]` save (CLI/MCP for now), `[/]` jump to Search, `[M]` jump to Memories, `[I]` jump to Inbox.
**Navigation**: `1`-`6` jump to tab · `tab` / `shift+tab` cycle · `esc` pop overlay · `q` / `ctrl+c` quit.
**Memory Detail**: `↑↓` scroll, `[C]` copy to OS clipboard (Windows `clip.exe`, macOS `pbcopy`, Linux `wl-copy` → `xclip`).

The Memories tab and the Detail view are **read-only by design** — no edit/delete from the TUI to prevent accidental data loss. The Inbox is the only mutation path, and it always promotes to a memory via the same save pipeline that CLI/MCP use.

For programmatic access from the AI, use the equivalent **`tl_stats` MCP tool**:

```jsonc
// Default: scoped to current project
{}

// Cross-project: see counts across the whole DB
{ "project": "*" }

// Specific project
{ "project": "sort-factory-v4" }
```

The text output mirrors the dashboard's stats panel so the AI can describe the state of memory back to you.

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
| [`docs/INSTALLATION.md`](docs/INSTALLATION.md) | Install on Windows / macOS / Linux, configure, verify, troubleshoot |
| [`docs/AGENT-SETUP.md`](docs/AGENT-SETUP.md) | Per-editor setup index (Claude Code, Cursor, Zed, Windsurf, OpenCode, Gemini CLI, Rider) |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Components, data flow, schema, MCP surface, TUI |
| [`docs/COMPARISON.md`](docs/COMPARISON.md) | Honest side-by-side vs Engram and claude-mem |
| [`docs/PROGRESS.md`](docs/PROGRESS.md) | What's done, what's next, pre-publish TODOs |
| [`docs/design/memory-domain.md`](docs/design/memory-domain.md) | Full memory type taxonomy with examples |
| [`docs/design/tag-conventions.md`](docs/design/tag-conventions.md) | Tag scheme: engine, platform, pipeline |
| [`docs/decisions/0001-architecture-baseline.md`](docs/decisions/0001-architecture-baseline.md) | Why we built on Engram's foundations |
| [`docs/decisions/0002-search-strategy-fts5-first.md`](docs/decisions/0002-search-strategy-fts5-first.md) | FTS5 in v1, embeddings deferred (with reasoning) |
| [`docs/decisions/0003-state-dir-not-cache-dir.md`](docs/decisions/0003-state-dir-not-cache-dir.md) | Store memory data in user data dir, not user cache dir (with auto-migration) |
| [`docs/integrations/`](docs/integrations/) | Per-editor setup: Cursor, Zed, Windsurf, OpenCode, Gemini CLI, Rider-Unity |
| [`docs/research/engram-anatomy.md`](docs/research/engram-anatomy.md) | Engram codebase map |
| [`docs/research/flow-mem-save.md`](docs/research/flow-mem-save.md) | Engram's `mem_save` traced end-to-end |
| [`docs/research/flow-mem-search.md`](docs/research/flow-mem-search.md) | Engram's search internals — the most important doc |

---

## Credits and lineage

Thoughtline would not exist without **Engram** by [Alan Buscaglia](https://github.com/Gentleman-Programming) and the Gentleman-Programming community. We read its source carefully (see `docs/research/`), credit the design choices we keep, and stay MIT-compatible so improvements can flow back upstream.

Thoughtline now ships a first-class migration path for users coming from Engram: the `cmd/migrate` binary reads your Engram database, maps types and columns to Thoughtline's format, and writes through the same `storage.Save()` path the MCP server uses — FTS5 indexing, `sync_id` preservation, and all validation included. See [`cmd/migrate/README.md`](cmd/migrate/README.md).

If you are evaluating memory tools for a non-gamedev team, **use Engram directly** — it is more general and more battle-tested. Thoughtline is a gamedev-flavored sibling, not a competitor.

---

## License

MIT — see [LICENSE](LICENSE).

## Contributing

PRs, issues, and design feedback welcome. See [CONTRIBUTING.md](CONTRIBUTING.md). Be kind, be precise, cite source lines.
