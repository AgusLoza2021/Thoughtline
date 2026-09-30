# Thoughtline Persistent Memory — Protocol

> For the vocabulary the protocol refers to, see the
> [memory domain](../design/memory-domain.md) and the
> [tag conventions](../design/tag-conventions.md). For the tools themselves —
> every argument, every default, every return shape — see the
> [tool catalogue](../TOOLS.md).

This page is the **Thoughtline layer** for Claude Code and compatible clients: the vocabulary, the save-and-recall behaviour, and the habits that keep a *local* memory base worth searching. It answers *what to save* and *how to keep it good*.

It is not the server's tool reference, and it does not try to be. That reference is [`docs/TOOLS.md`](../TOOLS.md), and it is authoritative for arguments, defaults and return shapes. This page stays deliberately short so it can stay true — and so that a change to a tool's arguments has exactly one place to land.

Copy the block below into your `CLAUDE.md` as the `## Thoughtline Persistent Memory — Protocol` section.

---

## Thoughtline Persistent Memory — Protocol

You have Thoughtline's memory tools available.
This protocol is MANDATORY and ALWAYS ACTIVE while this block is in your project instructions — not something you activate on demand.

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
// Required: title, content, type
// Optional: scope, topic_key, project, tags, session_id
{
  "title": "Use WAL journal mode for all SQLite connections",
  "type": "decision",                  // one of the fourteen catalogued types — always pass it
  "scope": "project",                  // "project" (default) or "personal"
  "topic_key": "decision/sqlite-wal",  // stable key → replaces the stored memory on re-save
  "project": "my-game",
  "tags": ["platform:windows", "tool:claude-code"],
  "content": "**What**: Enable WAL via the connection string.\n**Why**: Concurrent tool calls need readers and a writer at the same time.\n**Where**: src/storage/db.ts\n**Learned**: busy_timeout=5000 prevents lock errors under load."
}
```

Three things about this payload are easy to get wrong:

1. **Always pass `type`, and pass it from the catalogue.** The server checks the field against the fourteen types in [`memory-domain.md`](../design/memory-domain.md) and refuses anything else. An invented type is a failed save, not a memory quietly filed under a default. The set is closed.
2. **`scope` has two values.** `project` is the default — shared knowledge for the project the working directory belongs to. `personal` is your own notes. `preference` must be `personal` and everything else must be `project`; a save asking for `global` is refused.
3. **Tags go in `tags`**, a list of lowercase `key:value` strings. `tl_search` filters on the field and `tl_get_observation` prints it back. A `**Tags**:` first line in `content` still works and is what a human reading raw markdown sees — but it is indexed as body text, so it is searchable and not filterable. Full vocabulary: [tag conventions](../design/tag-conventions.md).

**Recommended content structure** (use **What** / **Why** / **Where** / **Learned**):

```
**What**: [one sentence — what was done or decided]
**Why**: [the reason — user request, bug, performance, etc.]
**Where**: [files or paths affected]
**Learned**: [gotchas, edge cases, surprises — omit if none]
```

**Two saves that look alike are not the same save.** A byte-identical save carries no `topic_key`: inside the dedupe window it is a noop, and you get the existing id back unchanged. A save that reuses a `topic_key` **replaces** the stored title, content and tags. That is the intended way to evolve a topic — and the reason a new topic needs a new key.

#### EVOLVING AND CORRECTING A MEMORY

A wrong memory is worse than a missing one, and re-saving is not always the fix.

- **Existing memory is now wrong or incomplete** → `tl_update`. Pass `id` plus the fields to change: `title`, `content`, `tags`. There is no find-and-replace — send the whole new body. An empty `tags` array clears them.
- **Memory is obsolete and should not be recalled** → `tl_delete`. A soft delete by id: the row leaves search and context, and its `topic_key` is freed for reuse.
- **Two memories are entangled** → `tl_link` records the edge and `tl_related` reads it back in either direction. The relations are `supersedes`, `contradicts`, `refines`, `depends_on`, `references`, `related` and `derived_from`. Nothing detects the relationship for you; recording it is the judgement.

Prefer `tl_update` over a second `tl_save` when you are correcting the same topic, and prefer `tl_delete` over leaving a claim you know is false in the index.

### WHEN TO SEARCH MEMORY

On any variation of "remember", "recall", "what did we do", "how did we solve", "recordar", "acordate", or references to past work:

1. Call `tl_context` **first** — the recent memories of this project, cheapest and usually enough to answer "what were we doing".
2. If that does not answer it, call `tl_search` with relevant keywords.
3. If a result looks promising, call `tl_get_observation` for the full untruncated content.

Also search PROACTIVELY when:

- Starting work on something that might have been done before
- User mentions a topic you have no context on
- User's FIRST message references the project, a feature, or a problem

#### `tl_search` usage

```jsonc
// Keyword search — default limit 10, server cap 50
{ "query": "WAL journal mode sqlite", "project": "my-game", "limit": 5 }

// Filter by type
{ "query": "crash on startup", "type": "bugfix" }

// Filter by tag
{ "query": "export", "tags": ["engine:unity", "pipeline:fbx"] }

// Newest first, rather than by relevance
{ "query": "release checklist", "recent_first": true }

// A query containing "/" is tried against topic_key first: "design/auth"
// finds "design/auth/jwt" without needing a wildcard
{ "query": "design/auth" }
```

A snippet is a preview. When a result looks like the answer, read the whole of it with `tl_get_observation`.

#### Reading a result

A hit carries its `ID`, `Project`, `Type`, `Scope`, `Topic` when it is keyed, `Revision`, `Score`, `Updated`, `Tags` when it has any, and a `Snippet`. That is the whole set. There is no health state on a memory and no relation annotation in a result — if two memories contradict each other, that is a link you recorded, not a flag the server raises. Read the linked memory before you rely on either one.

#### `tl_get_observation` usage

```jsonc
// Call with the id from a tl_search result for full untruncated content
{ "id": 42 }
```

Always call `tl_get_observation` when a snippet is a preview and you need the full content.

### KEEPING THE BASE GOOD (local, cheap, do it without being asked)

A memory base rots in two ways: nothing is ever retired, and contradictions are never recorded. Both are yours to do — this engine implements no decay, no review horizon and no automatic re-surfacing, and says so in [`memory-domain.md`](../design/memory-domain.md).

- **Retire what has aged out.** There is no review queue and no `state` flag. If a memory is superseded, `tl_update` it to say so, `tl_link` it to its replacement, or `tl_delete` it. The judgement *is* the mechanism.
- **Record an entanglement when you notice one.** `tl_link(from_id, to_id, relation)` writes the edge; `tl_related` reads every edge on a memory. When the edge is `supersedes` or `contradicts` and the memory is typed `architecture` or `decision`, say so to the user rather than quietly rewriting history.
- **Judge a suspected duplicate without writing anything.** `tl_judge(existing_id, incoming_title, incoming_content, relation)` formats a comparison and a recommended action, and it does **not** touch the database. Its relation set is its own — `supersedes`, `compatible`, `conflicts_with`, `scoped`, `not_conflict` — and it persists nothing. Use `tl_link` when you want the decision to outlive the turn.
- **Look at what the store holds.** `tl_stats` reports counts by type, project and scope, and takes `"*"` for every project. It is how you notice that three months of saves all landed on one type.

### PASSIVE CAPTURE

The Claude Code plugin wires six hooks — `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `Stop` and `SessionEnd` — to `thoughtline hook <event>`. Each one stores the **raw event payload** as a pending event. Nothing is summarised, and nothing is extracted.

Pending events are invisible to search until you promote them. Read the queue with `tl_pending_list` (status `pending` by default), inspect one with `tl_pending_get`, and turn the ones worth keeping into memories with `tl_promote` — which takes the fields to file them under, because a raw payload does not know what a memory looks like.

Passive capture is opt-in and off by default: every hook no-ops unless `THOUGHTLINE_PASSIVE_CAPTURE=1` is set.

End a completed task with a `## Key Learnings:` section — numbered items, one learning each.

```
## Key Learnings:

1. bcrypt cost=12 is the right balance for our server performance
2. JWT refresh tokens need atomic rotation to prevent race conditions
```

Nothing parses that section for you, but it ends up inside the captured `Stop` payload, which is where a promoted memory tends to come from. Write it for the next reader, then promote it yourself.

### SESSION CLOSE PROTOCOL (mandatory)

Open a session with `tl_session_start` — it returns the UUIDv7 that `tl_save` accepts as `session_id` — and close it with `tl_session_summary`, which requires both `id` and `summary`:

```jsonc
{
  "id": "<the UUIDv7 tl_session_start returned>",
  "summary": "## Goal\n[What we were working on this session]\n\n## Instructions\n[User preferences or constraints discovered — skip if none]\n\n## Discoveries\n- [Technical findings, gotchas, non-obvious learnings]\n\n## Accomplished\n- [Completed items with key details]\n\n## Next Steps\n- [What remains to be done — for the next session]\n\n## Relevant Files\n- path/to/file — [what it does or what changed]"
}
```

This is NOT optional. If you skip this, the next session starts blind. After a compaction, pass the compacted text as `compaction_block` and the tool recovers the session id from it.

### AFTER COMPACTION

After a context compaction (or if you see "FIRST ACTION REQUIRED"):

1. Call `tl_session_summary` with `compaction_block` set to the compacted summary
2. Call `tl_context` to recover the recent state of this project
3. Call `tl_search` with keywords from the current task to recover prior decisions
4. Only THEN continue working

### DELIVERY GUARANTEE

Memory operations are internal bookkeeping, not the answer. Finish the memory work, then send the complete reply as the final message of the turn — do not narrate a save. If a memory call fails, say so in the reply and send it anyway.

### Topic Key Format (reference)

Every one of the fourteen types has a documented key pattern, listed per type in [`memory-domain.md`](../design/memory-domain.md) — `decision/<area>/<choice>`, `architecture/<area>`, `perf/<platform>/<area>`, `convention/<area>`, and so on. A key is `category/subject`, lowercase and slash-separated, and it is what an upsert matches on, so re-reading the same key is how a topic evolves. A query containing `/` is tried against `topic_key` before full-text search runs, which makes the key you choose the cheapest way to find it again.

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

Both tables above are accurate history and stay as they are. What follows is what changed when the server came back.

## What this revision changed (2026-09-29)

The retirement was reversed: this repository ships the memory server again, so the protocol speaks `tl_*` once more. [ADR 0008](../decisions/0008-the-engine-is-the-product.md) records why, and supersedes [ADR 0007](../decisions/0007-vocabulary-not-mechanics.md) without editing it. Every claim below was read out of this repository's own source, which is the point — the mechanics and the page that describes them now live in the same place.

| Then (2026-09-28) | Now | Why |
|------|-----|-----|
| `mem_save`, `mem_search`, `mem_get_observation`, `mem_context`, `mem_update`, `mem_delete`, `mem_session_summary` | `tl_save`, `tl_search`, `tl_get_observation`, `tl_context`, `tl_update`, `tl_delete`, `tl_session_summary` | The tools are this project's again. |
| "It is not Engram's tool reference" | [`TOOLS.md`](../TOOLS.md) owns the arguments; this page owns the vocabulary and the habits | One reference, not two, and not a pointer into another project. |
| `"type": "decision" // one of this vocabulary's 11 types` | Fourteen, checked by the server | `Validate` rejects an unknown type outright. Eleven was undercounting; the catalogue is 7 core plus 7 gamedev extensions. |
| "`scope`: `project`, `personal` or `global`" | Two: `project` or `personal`. `preference` must be `personal` | `global` was Engram's. The column has a two-value CHECK, and a request for `global` is refused with a message that names the divergence. |
| `"capture_prompt": false` | Removed | No counterpart: there is no prompt-capture path to opt out of. |
| `mem_suggest_topic_key` derives a key from `type` + `title` | No counterpart | A query containing `/` is matched against `topic_key` as a GLOB, retried as a prefix, and only then falls through to full-text search. `mem_pin` has no counterpart either: there is no pinning, so a memory that must not be missed has to be findable by its content. |
| "Pin what a future session must not miss" (`mem_pin`/`mem_unpin`) | Nothing pins. `tl_context` orders by `updated_at` | The ordering is a column, not an operator. |
| "Retire what has aged out" (`mem_review` with its review horizon) | No review horizon is implemented here | `tl_stats` shows what the store holds; retiring a memory is `tl_update`, `tl_link` or `tl_delete`, and the judgement is the whole mechanism. |
| "`state` is `active` or `needs_review`; relation annotations appear under a result" | A hit has no health state and no relation annotation | Those fields were Engram's. The fields a hit actually carries are the ones listed above. |
| "`mem_save` can come back with `candidates[]` and `judgment_required: true`" | `tl_save` returns the saved memory and nothing else | Nothing in this server detects a contradiction. `tl_judge` compares what *you* hand it, and writes nothing. |
| "`mem_judge` records the relation; `mem_compare` judges a pair" | `tl_judge` formats a comparison, `tl_link` writes the edge | The two relation sets are different and neither is a schema enum. |
| `mem_doctor` runs a diagnostic report | No counterpart | A failed call returns the error that caused it. |
| "`mem_capture_passive` extracts a `## Key Learnings:` section" | Six Claude Code hooks store the raw event payload in `pending_events`; `tl_promote` files the ones you keep | Nothing parses a section. Capture is opt-in behind `THOUGHTLINE_PASSIVE_CAPTURE=1`. |
| "`mem_session_summary` takes only `content`" | `tl_session_summary` requires `id` and `summary`, and recovers `id` from `compaction_block` after a compaction | `tl_session_start` returns the id, so the agent has it. |
| `match_mode: "any"` and `all_projects: true` on `mem_search` | `tl_search` has neither; it filters on `type`, `scope`, `project`, `topic_key`, `tags`, and can sort with `recent_first` | Different server, different arguments. Only `project` scopes a search, and `tl_stats` is where `"*"` spans every project. |
| An 11-row topic-key table duplicated from the domain page | A pointer to the per-type patterns, which live in one place | The table was a second copy of a catalogue. Second copies drift, and this one had already lost three types. |
