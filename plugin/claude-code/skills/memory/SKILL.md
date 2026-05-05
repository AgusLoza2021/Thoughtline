---
name: thoughtline-memory
description: Persistent, project-aware memory for AI coding agents. Use thoughtline whenever you make a decision, fix a bug, learn something non-obvious, or need to recall prior work.
---

# Thoughtline Memory Protocol

Thoughtline is a local-first SQLite-backed memory system. It survives across sessions and compactions. This protocol is **MANDATORY** and **ALWAYS ACTIVE** when the plugin is installed.

## Tools (namespace `tl`)

Core (always available, no ToolSearch needed):
- `tl_save` — persist a memory
- `tl_search` — keyword + GLOB search (FTS5 + BM25)
- `tl_context` — recent activity for the active project
- `tl_get_observation` — full untruncated content of one memory
- `tl_session_start` — open a session
- `tl_session_summary` — close a session with a structured digest

Via ToolSearch: `tl_update`, `tl_delete`, `tl_stats`.

## Proactive Save — do NOT wait for the user to ask

Call `tl_save` IMMEDIATELY after any of:

- Architecture or design decision made
- Team convention documented or established
- Tool / library choice made with tradeoffs
- Bug fixed (include root cause)
- Feature implemented with a non-obvious approach
- Configuration change or environment setup done
- Non-obvious discovery, gotcha, or edge case
- Pattern established (naming, structure, convention)
- User preference or constraint learned
- User confirms or rejects an approach

**Self-check after every task**: "Did I or the user decide, confirm, prefer, fix, learn, or establish a convention? If yes → `tl_save` NOW."

## Recommended content structure

```
**What**: [one sentence — what was done or decided]
**Why**: [the reason — user request, bug, perf, etc.]
**Where**: [files or paths affected]
**Learned**: [gotchas, edge cases, surprises — omit if none]
```

## Topic key convention

`category/subject` or `category/area/subject`. Re-saving the same `topic_key` upserts (same `sync_id`, bumped `revision_count`) — no duplicates.

| Memory type | Topic key pattern |
|-------------|-------------------|
| `decision` | `decision/<area>/<choice>` |
| `architecture` | `architecture/<area>` |
| `bugfix` | `bug/<area>` (or omit for one-off bugs) |
| `convention` | `convention/<area>` |
| `preference` | `preference/<area>` |
| `game-design-decision` | `design/<system>/<choice>` |
| `scene-pattern` | `scene/<engine>/<pattern>` |
| `asset-reference` | `asset/<category>/<name>` |
| `perf-gotcha` | `perf/<platform>/<area>` |
| `pipeline-step` | `pipeline/<source>-to-<target>/<asset>` |
| `script-pattern` | `script/<engine>/<concept>` |

## Search workflow

1. `tl_search` returns id, title, snippet (≤300 chars), score, metadata.
2. A query containing `/` short-circuits to a topic_key GLOB lookup.
3. If a snippet looks promising → `tl_get_observation` for full content.

Search proactively when the user says: "remember", "what did we do", "how did we solve", "acordate", "qué hicimos", or references prior work.

## Session lifecycle

- Open with `tl_session_start` (returns a UUIDv7 id) when you begin meaningful work.
- Close with `tl_session_summary` before saying "done" / "listo". Include: **Goal**, **Discoveries**, **Accomplished**, **Next Steps**, **Relevant Files**.

## After a compaction

The post-compaction hook injects an explicit recovery sequence: persist the compaction summary via `tl_session_summary`, recover state via `tl_context`, then continue work. Follow it in order.

## Examples

These are the kinds of moments where this skill should fire — use them as a calibration set, not an exhaustive list.

### Save triggers

```
user: vamos a usar WAL en SQLite por las escrituras concurrentes
assistant: [calls tl_save with type=decision, topic_key=decision/sqlite-wal,
            content has What/Why/Where/Learned]
```

```
user: ah, encontré por qué fallaba — el FTS5 contentless table no soporta DELETE en cascada
assistant: [calls tl_save with type=bugfix, content describing root cause + fix]
```

```
user: dale, perfecto el approach de un PR bundleado
assistant: [calls tl_save with type=preference, scope=personal,
            topic_key=preference/pr-strategy — confirmed convention]
```

### Search triggers

```
user: ¿cómo habíamos resuelto lo del crash en startup?
assistant: [calls tl_search with query="crash startup",
            inspects top results, then tl_get_observation on the best match]
```

```
user: arrancamos a tocar el storage layer
assistant: [calls tl_search with query="architecture/storage-layer" (GLOB shortcut),
            grabs relevant decisions before writing any code]
```

### NOT a save trigger

```
user: corré los tests
assistant: [runs tests — no decision, no learning, nothing to save]
```

```
user: explicame qué hace este archivo
assistant: [explains — pure information request, no project memory]
```
