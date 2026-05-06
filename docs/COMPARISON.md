# Thoughtline vs Engram vs claude-mem

Three projects, one job: **give your AI coding assistant a long-term memory that survives across sessions**. This page exists so you can pick the right tool the first time, and so we can be honest about what Thoughtline is and isn't.

---

## TL;DR

| If you... | Use |
|-----------|-----|
| Ship games and want a memory that already speaks `scene-pattern`, `asset-reference`, `perf-gotcha`, `pipeline-step`, `script-pattern` out of the box | **Thoughtline** |
| Want the most general agent-agnostic memory binary, with optional cloud replication and the broadest agent matrix | **[Engram](https://github.com/Gentleman-Programming/engram)** |
| Live entirely inside Claude Code and want zero-touch automatic capture & compression instead of explicit `tl_save`/`mem_save` calls | **[claude-mem](https://github.com/thedotmack/claude-mem)** |

All three are local-first and respect your data. There is no winner — they make different bets.

---

## Provenance — credit where it's due

Thoughtline cites Engram as its foundation in the [README](../README.md). The MCP tool shape (`tl_save` / `tl_search` / `tl_context` / `tl_get_observation` / `tl_session_summary` / ...), the SQLite + FTS5 storage layer, and the `topic_key` upsert pattern are direct echoes of Engram's design — credited and intentional. Alan Buscaglia and the Gentleman.Programming community did the hard architectural work; we built on top of it.

Where Thoughtline diverges is the **memory taxonomy**:

- **Engram's vocabulary** is general-purpose: `bugfix`, `decision`, `architecture`, `pattern`, `config`, `discovery`, `learning`. Built for software engineering at large.
- **Thoughtline's vocabulary** is gamedev-first: `scene-pattern`, `asset-reference`, `perf-gotcha`, `pipeline-step`, `script-pattern`, `shader-tweak`, `animation-curve`, `audio-mix`, `build-config`, plus tags tuned for engines (`engine:unity`, `engine:godot`, `platform:switch`, `pipeline:fbx-to-godot`).

Same shape, different language. Both can coexist on the same machine.

---

## Side by side

| Dimension | Thoughtline | Engram | claude-mem |
|-----------|-------------|--------|------------|
| Language / runtime | Go (single binary) | Go (single binary) | Node.js (npm) |
| Storage | SQLite + FTS5 | SQLite + FTS5 | SQLite + ChromaDB embeddings |
| Memory model | Explicit save | Explicit save | **Implicit** capture + AI compression |
| Vocabulary | **Gamedev-specialized** | General software | General software |
| MCP tools | 9 (M5: save, search, context, get_observation, update, delete, session_start, session_summary, stats) | 19 (incl. conflict surfacing, semantic LLM judging) | Search-focused MCP tools |
| Built-in TUI | Yes — Bubbletea, three themes (`brand` / `zbrush` / `mono`), animated header cube | Yes — Catppuccin Mocha, dashboard / detail / search views | Web viewer UI on `localhost:37777` |
| HTTP API | No (planned) | Yes | No |
| Cloud sync | No | Yes (opt-in replication, beta) | No |
| Cross-machine sync | Manual | Git sync built in | Manual |
| Auto-capture from agent activity | No (explicit by design) | No (explicit by design) | **Yes** |
| Install (macOS) | `go install` (Homebrew planned) | `brew install gentleman-programming/tap/engram` | `npx claude-mem install` |
| Install (Windows) | `irm .../scripts/install.ps1 \| iex` | Documented in INSTALLATION.md | `npx claude-mem install` |
| License | MIT | MIT | AGPL-3.0 |
| Releases at time of writing | 0 (v0.0.1 imminent) | 78+ | 260+ |

---

## Where Thoughtline tries to earn its place

1. **Gamedev taxonomy out of the box.** You don't have to invent your own tag scheme to remember "the lantern texture import settings that didn't blow out the bloom" — there's already a category for it. The taxonomy lives in [docs/design/tag-conventions.md](design/tag-conventions.md).
2. **Engine-aware integration docs from day one.** Cursor, Zed, and Rider-Unity have dedicated setup pages in [docs/integrations/](integrations/) — the Rider one is written specifically for Unity workflows.
3. **A TUI that ships with multiple themes by default.** The `zbrush` palette (warm amber) is tuned for long art-pipeline sessions and is the visual differentiator from generic dev tooling.

That's the entire pitch. Three things. If those don't matter to you, one of the others will fit better.

---

## Where you should pick something else

**Pick Engram if:**
- You need cloud sync across machines, today (Engram has it; we don't).
- You want the larger MCP surface (19 tools incl. conflict surfacing and semantic judging).
- Your workflow is general software, not games — Engram's vocabulary fits you better.
- You value the most active community in this niche right now.

**Pick claude-mem if:**
- You want to *not think about what to save*. claude-mem auto-captures tool usage and compresses it for you. Thoughtline (and Engram) require explicit `tl_save` / `mem_save` calls — that's a deliberate trade for precision over coverage.
- You live inside Claude Code and don't need agent-agnostic.
- You want a polished web viewer + the largest community of any memory tool for AI coding assistants.

---

## Can I run more than one?

Yes. They write to different SQLite files in different directories and expose different MCP tool names (`tl_*` vs `mem_*` vs claude-mem's tool set). Nothing stops a project from registering all three MCP servers in `~/.claude.json`. The cost is duplicated context — the AI will save the same fact in multiple places. In practice you pick one and stick with it.

---

## We're not the same project, intentionally

Engram and claude-mem solve the *general* memory problem. Thoughtline is what happens when you take Engram's architecture, drop the cloud / HTTP surface, and specialize the vocabulary for the workflows of game studios. That's a deliberate trade — fewer features, sharper fit.

If your assistant currently misremembers which `Materials/lantern_emissive_01.mat` you settled on after three iterations, this one is for you.
