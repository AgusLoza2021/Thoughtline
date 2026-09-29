# Thoughtline Persistent Memory — Protocol

<!-- retired-v0.1.0 -->
> **Updated for Engram.** This page was written for the v0.1.0 MCP server. The
> protocol below is live — what changed is the tool names (`tl_*` back then,
> `mem_*` now), where tags go, and a few claims we can no longer make now that the
> engine is not ours. The exact list is at the bottom, under
> [What changed from the v0.1.0 version](#what-changed-from-the-v010-version).
>
> For the vocabulary the protocol refers to, see the
> [memory domain](../design/memory-domain.md) and the
> [tag conventions](../design/tag-conventions.md). For the server itself, see
> [Engram](https://github.com/Gentleman-Programming/engram).

This page is the **Thoughtline layer** for Claude Code and compatible clients: the vocabulary, the save-and-recall behaviour, and the habits that keep a *local* memory base worth searching. It answers *what to save* and *how to keep it good*.

It is not Engram's tool reference, and it does not try to be. Engram owns its tools and documents them in [its memory protocol](https://github.com/Gentleman-Programming/engram/blob/main/DOCS.md#memory-protocol-full-text) — that page is authoritative for tool mechanics and argument details, and it is the one to re-read when Engram ships. This page stays deliberately short so it can stay true; Engram's own documentation authority note is explicit that related surfaces "must change together" without copying identical text into every host.

Copy the block below into your `CLAUDE.md` as the `## Thoughtline Persistent Memory — Protocol` section.

---

## Thoughtline Persistent Memory — Protocol

You have Engram's memory tools available.
This protocol is MANDATORY and ALWAYS ACTIVE while this block is in your project instructions — not something you activate on demand.

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
// Required: title, type, content
// Optional: scope, topic_key, project, capture_prompt
{
  "title": "Use WAL journal mode for all SQLite connections",
  "type": "decision",                  // one of this vocabulary's 11 types — always pass it
  "scope": "project",                  // "project" (default) or "personal" in this vocabulary
  "topic_key": "decision/sqlite-wal",  // stable key → replaces the stored memory on re-save
  "project": "my-game",
  "capture_prompt": false,             // false for automated/artifact saves, true (default) otherwise
  "content": "**Tags**: platform:windows, tool:claude-code\n\n**What**: Enable WAL via DSN pragma.\n**Why**: Concurrent tool calls need readers + writer simultaneously.\n**Where**: src/storage/db.ts\n**Learned**: busy_timeout=5000 prevents lock errors under load."
}
```

Three things about this payload are easy to get wrong:

1. **Always pass `type`.** It is a free-form string — nothing validates it, and Engram's own tool description defaults it to `manual`. A memory typed `manual` is not in this vocabulary and will never be found by a type filter. The vocabulary is an argument for consistency, not a constraint the server enforces.
2. **`scope` is not a filter you have much use for here.** This vocabulary uses two values: `project` (default — shared project knowledge) and `personal` (your own notes). Engram additionally accepts `global`; it is not part of this vocabulary, which is local-first by design.
3. **There is no `tags` field.** Tags live on a `**Tags**:` line as the first line of `content`, comma-separated, in `key:value` form. Engram indexes the body, so the line stays findable — but it is not a filter, so treat it as an aid to recall rather than a schema. Full vocabulary: [tag conventions](../design/tag-conventions.md).

**Recommended content structure** (use **What** / **Why** / **Where** / **Learned**):

```
**What**: [one sentence — what was done or decided]
**Why**: [the reason — user request, bug, performance, etc.]
**Where**: [files or paths affected]
**Learned**: [gotchas, edge cases, surprises — omit if none]
```

**Two saves that look alike are not the same save.** Re-saving a byte-identical memory inside Engram's rolling dedupe window is folded into the existing one rather than appended; re-saving with a `topic_key` you already used **replaces** the stored title and content. That is the intended way to evolve a topic — and the reason a new topic needs a new key.

#### EVOLVING AND CORRECTING A MEMORY

A wrong memory is worse than a missing one, and re-saving is not always the fix.

- **New topic, unsure of the key** → `mem_suggest_topic_key` derives a stable `topic_key` from `type` + `title`. Use it *before* the first save when the topic is one you expect to revisit.
- **Existing memory is now wrong or incomplete** → `mem_update`. Pass `id` plus the fields to change. For a surgical edit use the paired `find` / `replace` inputs: both literal, both case-sensitive, both global, and neither can be combined with `content`.
- **Memory is obsolete and should not be recalled** → `mem_delete`. Soft delete is the default; hard delete is opt-in and permanent.

Prefer `mem_update` over a second `mem_save` when you are correcting the same topic, and prefer `mem_delete` over leaving a claim you know is false in the index.

### WHEN TO SEARCH MEMORY

On any variation of "remember", "recall", "what did we do", "how did we solve", "recordar", "acordate", or references to past work:

1. Call `mem_context` **first** — it is the cheap check of recent sessions, prompts and observations, and it usually answers "what were we doing".
2. If that does not answer it, call `mem_search` with relevant keywords.
3. If a result looks promising, call `mem_get_observation` for the full untruncated content.

Also search PROACTIVELY when:

- Starting work on something that might have been done before
- User mentions a topic you have no context on
- User's FIRST message references the project, a feature, or a problem

#### `mem_search` usage

```jsonc
// Keyword search
{ "query": "WAL journal mode sqlite", "project": "my-game", "limit": 5 }

// Filter by type — one of this vocabulary's 11 values
{ "query": "crash on startup", "type": "bugfix" }

// Broader recall: "all" (default) is AND across tokens, "any" is OR
{ "query": "asset bundle android apk", "match_mode": "any" }

// Across every project on this machine, when you are not sure where it was saved
{ "query": "blender export", "all_projects": true }
```

#### Reading a result

Do not read a result as a bare title and snippet. Each entry carries state you are expected to act on:

- **`state`** — `active`, or `needs_review` when the memory has outlived the review horizon for its `type` (see below). A `needs_review` hit is a candidate for `mem_update`, `mem_delete`, or `mem_review` — not a fact to trust unchanged.
- **Relation annotations** — `supersedes:`, `superseded_by:`, `conflicts:` and `conflict: contested by #N (pending)` appear directly under a result when the memory is entangled with another one. When you see one, read the other memory before you rely on either.

#### `mem_get_observation` usage

```jsonc
// Call with the id from a mem_search result for full untruncated content
{ "id": 42 }
```

Always call `mem_get_observation` when a snippet is a preview and you need the full content.

### KEEPING THE BASE GOOD (local, cheap, do it without being asked)

A memory base rots in three ways: nothing is pinned, nothing is retired, and contradictions are never resolved. All three have local tools.

- **Pin what a future session must not miss.** `mem_pin` puts an observation ahead of recent ones in `mem_context`; `mem_unpin` undoes it. Pinned state is local to this machine. Pin the handful of memories that define a project — the architecture decision, the convention everything else follows.
- **Retire what has aged out.** `mem_review` with `action: "list"` returns observations whose review horizon has passed; `action: "mark_reviewed"` resets one using its type's decay policy. Review state is local too.
- **Resolve contradictions when a save surfaces one.** `mem_save` can come back with `candidates[]` and `judgment_required: true` — it detected an existing memory that may contradict what you just saved. Inspect the candidates and call `mem_judge` with `judgment_id` and a `relation` of `related`, `compatible`, `scoped`, `conflicts_with`, `supersedes`, or `not_conflict`. If the relation is `supersedes` or `conflicts_with` **and** the memory is typed `architecture`, `policy` or `decision`, ask the user before recording it — and always ask when your confidence is below 0.7. `mem_compare` judges a pair you picked yourself.
- **Diagnose before you guess.** `mem_doctor` runs a read-only diagnostic report and takes an optional `project` or `check` filter. Reach for it when a save or a search behaves unexpectedly rather than inventing a workaround.

> **The one gap worth knowing.** Engram's review horizon is keyed on the exact `type` string, and it currently exists for only three types: `decision` (six months), `policy` (twelve) and `preference` (three). Everything typed `bugfix`, `architecture`, `convention`, `perf-gotcha`, `scene-pattern`, `asset-reference`, `pipeline-step`, `script-pattern` or `game-design-decision` is treated as permanently `active` and will never appear in `mem_review` — Engram says so in the comment above the map itself: *"Types absent from this map get `review_after` = NULL (Phase 1 behavior)."* Use the review tools where they apply, and treat staleness for the other types as your own judgement — `memory-domain.md` records the same limitation.

### PASSIVE CAPTURE

End a completed task with a `## Key Learnings:` section — numbered items, one learning each. Engram extracts each item into its own observation and skips duplicates.

```
## Key Learnings:

1. bcrypt cost=12 is the right balance for our server performance
2. JWT refresh tokens need atomic rotation to prevent race conditions
```

Use this for the small, real learnings that would not justify a `mem_save` payload of their own. `mem_capture_passive` does the same thing explicitly, on any text that contains such a section.

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

1. Call `mem_session_summary` with the compacted summary content
2. Call `mem_context` to recover the recent state of this project
3. Call `mem_search` with keywords from the current task to recover prior decisions
4. Only THEN continue working

### DELIVERY GUARANTEE

Memory operations are internal bookkeeping, not the answer. Finish the memory work, then send the complete reply as the final message of the turn — do not narrate a save. If a memory call fails, say so in the reply and send it anyway.

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

## What this revision changed (2026-09-28, against Engram `3ba7df6`)

Every fact below was read out of Engram's source or docs at that revision. The engine moves; that is exactly why this page no longer restates its mechanics.

| Then | Now | Why |
|------|-----|-----|
| "`scope`: `project` (default) or `personal`" | Two-value *vocabulary*, three accepted *values*: Engram also accepts `global` | `normalizeScope` honours `personal` and `global` and folds everything else to `project`. Engram's own docs disagree with each other on this — its canonical protocol says three, an earlier guide says two. State what the code does. |
| `"type": "decision" // one of the 11 catalogued types` | An argument for consistency, not a constraint | Nothing in Engram validates `type`. Its tool description suggests a *different* seven values and defaults to `manual`. Our claim that the field is closed by the server was never true. |
| Search started at `mem_search` | Search starts at `mem_context` | Engram's canonical order is `mem_context` → `mem_search` → `mem_get_observation`. `mem_context` is the cheap one. |
| "Search results contain `id`, `title`, `snippet`, and metadata" | `state` and relation annotations are part of the result, and are meant to be acted on | `state` reports `active` vs `needs_review`; `supersedes:` / `superseded_by:` / `conflicts:` / `conflict: contested by #N (pending)` mark entangled memories. |
| Nothing about correcting, pinning, reviewing or resolving | `EVOLVING AND CORRECTING`, `KEEPING THE BASE GOOD`, `PASSIVE CAPTURE`, `DELIVERY GUARANTEE` | Six Engram capabilities with real local value were never mentioned: `mem_update`, `mem_suggest_topic_key`, `mem_delete`, `mem_pin`/`mem_unpin`, `mem_review`, `mem_doctor`, plus conflict resolution via `mem_judge`/`mem_compare` and passive capture via `## Key Learnings:`. |
| `capture_prompt` unmentioned | Documented, with the "`false` for automated saves" rule | The default is `true`; Engram's own guidance is that automated artifact saves should opt out. |
| "This document is the canonical Thoughtline memory protocol block" | This page owns the vocabulary and the behaviour; Engram's protocol is authoritative for tool mechanics | Two pages claiming canonicality for the same tools is how the page above went stale without anyone noticing. |
