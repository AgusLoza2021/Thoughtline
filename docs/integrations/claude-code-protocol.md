# Thoughtline Persistent Memory — Protocol

<!-- retired-v0.1.0 -->
> **Updated for Engram.** This page was written for the v0.1.0 MCP server. The
> protocol below is live — what changed is the tool names (`tl_*` back then,
> `mem_*` now), where tags go, and a few claims we can no longer make now that the
> engine is not ours. The exact list is at the bottom, under
> [What changed](#what-changed-from-the-v010-version).
>
> For the vocabulary the protocol refers to, see the
> [memory domain](../design/memory-domain.md) and the
> [tag conventions](../design/tag-conventions.md). For the server itself, see
> [Engram](https://github.com/Gentleman-Programming/engram).

This document is the canonical Thoughtline memory protocol block for AI assistants (Claude Code and compatible clients). It defines **when** to save, **when** to search, **how** to use each tool, and **what** the session close protocol looks like.

Copy this block into your `CLAUDE.md` as the `## Thoughtline Persistent Memory — Protocol` section.

---

## Thoughtline Persistent Memory — Protocol

You have Engram's memory tools available.
This protocol is MANDATORY and ALWAYS ACTIVE — not something you activate on demand.

### PROACTIVE SAVE TRIGGERS (mandatory — do NOT wait for user to ask)

Call `mem_save` IMMEDIATELY and WITHOUT BEING ASKED after any of these:
- Architecture or design decision made
- Team convention documented or established
- Workflow change agreed upon
- Tool or library choice made with tradeoffs
- Bug fix completed (include root cause)
- Feature implemented with non-obvious approach
- Configuration change or environment setup done
- Non-obvious discovery about the codebase
- Gotcha, edge case, or unexpected behavior found
- Pattern established (naming, structure, convention)
- User preference or constraint learned

Self-check after EVERY task: "Did I make a decision, fix a bug, learn something non-obvious, or establish a convention? If yes, call `mem_save` NOW."

#### `mem_save` usage

```jsonc
// Required: title, content, type, scope
// Optional: topic_key, project, session_id
{
  "title": "Use WAL journal mode for all SQLite connections",
  "type": "decision",                  // one of the 11 catalogued types
  "scope": "project",                  // "project" (default) or "personal"
  "topic_key": "decision/sqlite-wal",  // stable key → replaces on re-save
  "project": "my-game",
  "content": "**Tags**: platform:windows, tool:claude-code\n\n**What**: Enable WAL via DSN pragma.\n**Why**: Concurrent tool calls need readers + writer simultaneously.\n**Where**: src/storage/db.ts\n**Learned**: busy_timeout=5000 prevents lock errors under load."
}
```

Note what is **not** in that payload: there is no `tags` field. Engram has none, so tags live on a `**Tags**:` line as the first line of `content`, comma-separated, in `key:value` form. Engram indexes the body, so the line stays findable — but it is not a filter, so treat it as an aid to recall rather than a schema. Full vocabulary: [tag conventions](../design/tag-conventions.md).

**Recommended content structure** (use **What** / **Why** / **Where** / **Learned**):
```
**What**: [one sentence — what was done or decided]
**Why**: [the reason — user request, bug, performance, etc.]
**Where**: [files or paths affected]
**Learned**: [gotchas, edge cases, surprises — omit if none]
```

**`topic_key` naming convention**: `category/subject` or `category/subcategory/subject`, lowercase, slash-separated.
- `architecture/storage-layer`
- `decision/auth/jwt-vs-session`
- `bugfix/fts5-shared-cache`
- `convention/script-naming`
- `preference/keybindings`

Re-saving the same `topic_key` **replaces** the stored title and content rather than appending a second copy. Reuse a key only for a topic that genuinely evolves; give a new key to a new topic.

### WHEN TO SEARCH MEMORY

On any variation of "remember", "recall", "what did we do", "how did we solve", "recordar", "qué hicimos", or references to past work:

1. Call `mem_search` with relevant keywords
2. If a result looks promising, call `mem_get_observation` for the full untruncated content

Also search PROACTIVELY when:
- Starting work on something that might have been done before
- User mentions a topic you have no context on
- User's FIRST message references the project, a feature, or a problem

#### `mem_search` usage

```jsonc
// Keyword search
{ "query": "WAL journal mode sqlite", "project": "my-game", "limit": 5 }

// Filter by type — one of the 11 catalogued values
{ "query": "crash on startup", "type": "bugfix" }

// Broader recall: "all" (default) is AND across keywords, "any" is OR
{ "query": "asset bundle android apk", "match_mode": "any" }

// Across every project, when you are not sure where it was saved
{ "query": "blender export", "all_projects": true }
```

#### `mem_get_observation` usage

```jsonc
// Call with the id from a mem_search result for full untruncated content
{ "id": 42 }
```

Search results contain `id`, `title`, `snippet`, and metadata. Always call `mem_get_observation` when a snippet is a preview and you need the full content.

### SESSION CLOSE PROTOCOL (mandatory)

Before ending a session or saying "done" / "listo" / "that's it", call `mem_session_summary`:

```jsonc
{
  "content": "## Goal\n[What we were working on this session]\n\n## Instructions\n[User preferences or constraints discovered — skip if none]\n\n## Discoveries\n- [Technical findings, gotchas, non-obvious learnings]\n\n## Accomplished\n- [Completed items with key details]\n\n## Next Steps\n- [What remains to be done — for the next session]\n\n## Relevant Files\n- path/to/file — [what it does or what changed]"
}
```

This is NOT optional. If you skip this, the next session starts blind.

### AFTER COMPACTION

After a context compaction (or if you see "FIRST ACTION REQUIRED"):
1. Call `mem_context` to recover the recent state of this project
2. Call `mem_search` with keywords from the current task to recover prior decisions
3. Only THEN continue working

### Topic Key Format (reference)

| Memory type | Suggested topic_key pattern |
|-------------|----------------------------|
| `decision` | `decision/<area>/<choice>` |
| `architecture` | `architecture/<area>` |
| `bugfix` | `bug/<area>` or omit (each bug is unique) |
| `convention` | `convention/<area>` |
| `preference` | `preference/<area>` |
| `game-design-decision` | `design/<system>/<choice>` |
| `scene-pattern` | `scene/<engine>/<pattern>` |
| `asset-reference` | `asset/<category>/<name>` |
| `perf-gotcha` | `perf/<platform>/<area>` |
| `pipeline-step` | `pipeline/<source>-to-<target>/<asset-type>` |
| `script-pattern` | `script/<engine>/<concept>` |

### SDD Artifact Naming (for SDD workflows)

When using the SDD (Spec-Driven Development) workflow, artifacts use these topic_key patterns:

| Artifact | topic_key |
|----------|-----------|
| Project context | `sdd-init/{project}` |
| Exploration | `sdd/{change-name}/explore` |
| Proposal | `sdd/{change-name}/proposal` |
| Spec | `sdd/{change-name}/spec` |
| Design | `sdd/{change-name}/design` |
| Tasks | `sdd/{change-name}/tasks` |
| Apply progress | `sdd/{change-name}/apply-progress` |
| Verify report | `sdd/{change-name}/verify-report` |
| Archive report | `sdd/{change-name}/archive-report` |

---

## What changed from the v0.1.0 version

| Then | Now | Why |
|------|-----|-----|
| `tl_save`, `tl_search`, `tl_get_observation`, `tl_session_summary` | `mem_save`, `mem_search`, `mem_get_observation`, `mem_session_summary` | The tools belong to Engram, not to this project. |
| `"tags": ["platform:windows"]` | A `**Tags**:` first line inside `content` | Engram's `mem_save` accepts no tags field. |
| "FTS5 + BM25 ranking" | "keyword search" | That was our engine's ranking. Never promise a mechanism the server does not document. |
| `{ "query": "decision/*" }`, and a `/` in the query matching `topic_key` first | Not used | Those were our query shortcuts. Engram searches bodies, and `topic_key` is reachable as text. |
| "upserts (same `sync_id`, bumped `revision_count`)" | "replaces the title and content" | The observed behaviour, stated without our internal columns. |
| `"Where": internal/storage/storage.go` | A path in *your* project | The example pointed into this repository's tree. |
| `tl_session_start` to obtain the id for the close call | Nothing — `mem_session_summary` takes only `content` | Engram's close call needs no session id from the agent. |

The original page is in this file's git history (`git log -p -- docs/integrations/claude-code-protocol.md`), including the stray `MC` that had been prefixed to its H1 since the page was first written — and that this revision removes.
