MC# Thoughtline Persistent Memory — Protocol

This document is the canonical Thoughtline memory protocol block for AI assistants (Claude Code and compatible clients). It defines **when** to save, **when** to search, **how** to use each tool, and **what** the session close protocol looks like.

Copy this block into your `CLAUDE.md` as the `## Thoughtline Persistent Memory — Protocol` section.

---

## Thoughtline Persistent Memory — Protocol

You have access to Thoughtline, a persistent memory system that survives across sessions and compactions.
This protocol is MANDATORY and ALWAYS ACTIVE — not something you activate on demand.

### PROACTIVE SAVE TRIGGERS (mandatory — do NOT wait for user to ask)

Call `tl_save` IMMEDIATELY and WITHOUT BEING ASKED after any of these:
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

Self-check after EVERY task: "Did I make a decision, fix a bug, learn something non-obvious, or establish a convention? If yes, call `tl_save` NOW."

#### `tl_save` usage

```jsonc
// Required: title, content, type, scope
// Optional: topic_key (for evolving topics), project, tags, session_id
{
  "title": "Use WAL journal mode for all SQLite connections",
  "type": "decision",                  // one of the 11 catalogued types
  "scope": "project",                  // "project" (default) or "personal"
  "topic_key": "decision/sqlite-wal",  // stable key → upserts on re-save
  "project": "thoughtline",
  "tags": ["platform:windows"],
  "content": "**What**: Enable WAL via DSN pragma.\n**Why**: Concurrent MCP tool calls need readers + writer simultaneously.\n**Where**: internal/storage/storage.go\n**Learned**: busy_timeout=5000 prevents lock errors under load."
}
```

**Recommended content structure** (use **What** / **Why** / **Where** / **Learned**):
```
**What**: [one sentence — what was done or decided]
**Why**: [the reason — user request, bug, performance, etc.]
**Where**: [files or paths affected]
**Learned**: [gotchas, edge cases, surprises — omit if none]
```

**`topic_key` naming convention**: `category/subject` or `category/subcategory/subject`.
- `architecture/storage-layer`
- `decision/auth/jwt-vs-session`
- `bugfix/fts5-shared-cache`
- `convention/script-naming`
- `preference/keybindings`

Re-saving the same `topic_key` upserts (same `sync_id`, bumped `revision_count`) — no duplicates.

### WHEN TO SEARCH MEMORY

On any variation of "remember", "recall", "what did we do", "how did we solve", "recordar", "qué hicimos", or references to past work:

1. Call `tl_search` with relevant keywords
2. If a result looks promising, call `tl_get_observation` for the full untruncated content

Also search PROACTIVELY when:
- Starting work on something that might have been done before
- User mentions a topic you have no context on
- User's FIRST message references the project, a feature, or a problem

#### `tl_search` usage

```jsonc
// Keyword search (FTS5 + BM25 ranking)
{ "query": "WAL journal mode sqlite", "project": "thoughtline", "limit": 5 }

// Filter by type
{ "query": "crash on startup", "type": "bugfix" }

// Topic-key shortcut — a query containing "/" matches topic_key first
{ "query": "decision/sqlite-wal" }

// GLOB shortcut — all decisions
{ "query": "decision/*" }
```

#### `tl_get_observation` usage

```jsonc
// Call with the id from a tl_search result for full untruncated content
{ "id": 42 }
```

Search results contain `id`, `sync_id`, `title`, `snippet` (≤ 300 chars), `score`, and metadata. Always call `tl_get_observation` when a snippet is a preview and you need the full content.

### SESSION CLOSE PROTOCOL (mandatory)

Before ending a session or saying "done" / "listo" / "that's it", call `tl_session_summary`:

```jsonc
{
  "id": "<session-id-from-tl_session_start>",
  "summary": "## Goal\n[What we were working on this session]\n\n## Instructions\n[User preferences or constraints discovered — skip if none]\n\n## Discoveries\n- [Technical findings, gotchas, non-obvious learnings]\n\n## Accomplished\n- [Completed items with key details]\n\n## Next Steps\n- [What remains to be done — for the next session]\n\n## Relevant Files\n- path/to/file — [what it does or what changed]"
}
```

This is NOT optional. If you skip this, the next session starts blind.

### AFTER COMPACTION

After a context compaction (or if you see "FIRST ACTION REQUIRED"):
1. Call `tl_search` with keywords from the current task to recover prior decisions
2. Call `tl_session_summary` if you were mid-session and lost that state
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
