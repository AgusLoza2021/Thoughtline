# passive-capture Spec Delta

> Change: `tui-memory-workspace`
> Status: draft
> Operation: MODIFIED (delta against `openspec/specs/passive-capture/spec.md`)

## Purpose

Extends the existing `passive-capture` capability with an in-TUI pre-promotion edit path. The Inbox tab in the redesigned TUI lets a user accept a pending capture unchanged (`[A]`) or override `type`, `title`, and `content` before the underlying `MarkPromoted` write happens (`[E]`). The CLI/MCP capture path is unmodified. The `pending_events` row's original `payload` MUST remain untouched for audit fidelity — the edit only changes the values that flow into the newly created `memories` row.

---

## Modified Requirements

### Requirement 9 (NEW): Inbox-Mediated Pre-Promotion Edit

The TUI Inbox tab MUST support promoting a pending capture with either of two paths:

1. **Passthrough accept (`[A]`)**: The promotion writes a memory using the proposed values from `pending_events.payload` unchanged.
2. **Edited accept (`[E]`)**: Before the promote call, the user MAY override `type` (constrained to the 11-type taxonomy), `title`, and `content`. The promote then writes a memory using the edited values.

In both paths, the underlying storage write MUST go through the existing `MarkPromoted` code path. The `pending_events` row's `payload` column MUST NOT be mutated by the TUI — it remains the captured source of truth for audit. The created memory MUST inherit `project`, `session_id` (when present), and `captured_at` (where applicable) from the original `pending_events` row, regardless of which path was taken.

The TUI MUST NOT introduce a new MCP tool or new storage entry point for this flow — it reuses the existing promote write path, only the input values differ between `[A]` and `[E]`.

#### Scenario: Passthrough accept preserves proposed values

- GIVEN a `pending_event` with `status = 'pending'`, `project = "game-x"`, `payload` proposing type `decision`, title `Use WAL mode`, content `Switch SQLite to WAL`
- WHEN the user presses `[A]` on that row in the Inbox tab
- THEN the existing promote write path is invoked
- AND a new memory is created with type `decision`, title `Use WAL mode`, content `Switch SQLite to WAL`, project `game-x`
- AND the `pending_events` row's `status` becomes `'promoted'` and `promoted_memory_id` points at the new memory
- AND the `pending_events` row's `payload` column value is byte-for-byte unchanged from before the action

#### Scenario: Edited accept overrides type, title, and content

- GIVEN a `pending_event` with `status = 'pending'`, `project = "game-x"`, `payload` proposing type `bug`, title `crash on startup`, content `seg fault on init`
- WHEN the user presses `[E]`, changes type to `decision`, title to `Switch to WAL`, content to `Use WAL for SQLite concurrency`, and submits the form
- THEN the existing promote write path is invoked
- AND a new memory is created with type `decision`, title `Switch to WAL`, content `Use WAL for SQLite concurrency`, project `game-x`
- AND the `pending_events` row's `status` becomes `'promoted'` and `promoted_memory_id` points at the new memory
- AND the `pending_events` row's `payload` column value is byte-for-byte unchanged from before the action (the original `bug` / `crash on startup` / `seg fault on init` payload remains intact for audit)
