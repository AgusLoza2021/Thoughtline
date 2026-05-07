# Passive Capture Specification

> Change: `passive-capture-hooks`
> Status: shipped
> Operation: ADDED (new capability — no prior spec exists)

## Capability Summary

Opt-in pipeline that captures raw Claude Code hook events into a local SQLite queue (`pending_events`) and exposes `tl_promote` to convert selected events into typed Thoughtline memories. Default OFF. The queue is a staging area, not a memory source — only promoted events become memories.

---

## Requirements

### Requirement 1: `pending_events` Table Schema

The system MUST maintain a `pending_events` table in the existing user-data SQLite database with the following columns:

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `sync_id` | TEXT | UNIQUE, uuidv7, mirrors memories.sync_id convention |
| `project` | TEXT | NOT NULL |
| `session_id` | TEXT | NULLABLE — not all events carry one |
| `event_type` | TEXT | NOT NULL — one of 6 known types |
| `tool_name` | TEXT | NULLABLE — extracted from PreToolUse/PostToolUse |
| `tool_use_id` | TEXT | NULLABLE — when present in payload |
| `payload` | TEXT | NOT NULL — raw JSON from stdin |
| `event_hash` | TEXT | NOT NULL — SHA-256 of normalized payload (sole source of dedup) |
| `status` | TEXT | NOT NULL — one of `pending`, `promoted`, `archived` |
| `promoted_memory_id` | INTEGER | NULLABLE — soft pointer to `memories.id` (NOT a FK) |
| `promoted_at` | INTEGER | NULLABLE — unix epoch ms |
| `archived_at` | INTEGER | NULLABLE — unix epoch ms |
| `created_at` | INTEGER | NOT NULL — unix epoch ms, server clock at insert |
| `captured_at` | INTEGER | NOT NULL — unix epoch ms, parsed from payload if present, else == created_at |

The table MUST have a UNIQUE constraint on `(project, event_hash)` to enforce idempotency. It MUST have an index on `status` for retention queries. It MUST have an index on `created_at` for time-based pruning. It MUST have an index on `session_id` for session-based triage.

Pending events are NOT memories. They MUST NOT appear in `tl_search` results. They live in a separate table with no shared FTS index.

#### Scenario: Schema migration creates table

- GIVEN a Thoughtline database without `pending_events`
- WHEN the application runs its migration
- THEN the `pending_events` table exists with all columns, the UNIQUE constraint, and all indexes

#### Scenario: Re-running migration is safe

- GIVEN the `pending_events` table already exists
- WHEN the migration runs again
- THEN no error occurs and no data is lost

---

### Requirement 2: Lifecycle States

A `pending_event` row MUST progress through exactly these states:

```
pending → promoted   (via tl_promote)
pending → archived   (via worker retention pass)
```

Terminal states are `promoted` and `archived`. A row in a terminal state MUST NOT be transitioned again. `promoted` rows MUST be preserved for linkage even after the retention window — they are excluded from the janitor sweep.

#### Scenario: Newly inserted event is pending

- GIVEN opt-in is ON and a valid hook event arrives
- WHEN the row is inserted
- THEN `status = 'pending'` and `promoted_memory_id` IS NULL

#### Scenario: Promoted row stays promoted

- GIVEN an event with `status = 'promoted'`
- WHEN `tl_promote` is called again for the same event id
- THEN the call returns a clear error; no second memory is created; `status` remains `'promoted'`

#### Scenario: Archived row is not re-archived

- GIVEN an event with `status = 'archived'`
- WHEN the worker retention pass runs
- THEN the row is not touched again (idempotent)

---

### Requirement 3: `thoughtline hook <event-name>` Command

The `hook` subcommand MUST accept exactly these event names: `session-start`, `user-prompt-submit`, `pre-tool-use`, `post-tool-use`, `stop`, `session-end`. Any other event name MUST be treated as an unknown event — exit 0, log to stderr, no DB write.

Input MUST be read from stdin as JSON. The following fields MUST be present in the JSON payload for all events:

| Field | Type | Description |
|-------|------|-------------|
| `session_id` | string | Claude Code session identifier |
| `hook_event_name` | string | Name of the event (mirrors `<event-name>`) |

Additional fields per event are passed through as raw payload without validation beyond syntactic JSON well-formedness.

Output behavior:

- On success: exit 0, stdout silent (no output).
- On any known error (opt-in OFF, malformed JSON, DB error, unknown event): exit 0, error message written to stderr only.
- Non-zero exit MUST only occur on programmer error (panic recovery fallback).

The command MUST complete in under 50ms median latency on a warm database.

#### Scenario: Opt-in OFF — no write

- GIVEN `THOUGHTLINE_PASSIVE_CAPTURE` is unset (or `0`) and no config-file flag enables it
- WHEN `thoughtline hook post-tool-use` is invoked with valid JSON on stdin
- THEN exit code is 0, stdout is empty, no row is inserted into `pending_events`

#### Scenario: Opt-in ON — happy path insert

- GIVEN opt-in is enabled, `pending_events` table exists, and no row exists for this `(project, event_hash)`
- WHEN `thoughtline hook pre-tool-use` is invoked with valid JSON
- THEN exit code is 0, exactly one row is inserted with `status = 'pending'`

#### Scenario: Duplicate event — idempotent

- GIVEN opt-in is ON and a row already exists for `(project, event_hash)`
- WHEN `thoughtline hook pre-tool-use` is invoked with the identical JSON payload a second time
- THEN exit code is 0, no new row is inserted (INSERT OR IGNORE), total row count unchanged

#### Scenario: Malformed JSON input

- GIVEN opt-in is ON
- WHEN `thoughtline hook post-tool-use` is invoked with `{bad json` on stdin
- THEN exit code is 0, an error is written to stderr, no row is inserted, Claude Code session is unaffected

#### Scenario: Unknown event name

- GIVEN opt-in is ON
- WHEN `thoughtline hook unknown-event` is invoked
- THEN exit code is 0, an error is written to stderr, no row is inserted

---

### Requirement 4: Opt-in Mechanism

Passive capture MUST be disabled by default. The system MUST be activated by setting `THOUGHTLINE_PASSIVE_CAPTURE=1` (env var) OR by setting an equivalent flag in the Thoughtline config file. The env var MUST take precedence over the config file when both are set.

Toggling the opt-in MUST NOT require any database migration. Existing memories are unaffected. The only effect of toggling is whether `thoughtline hook` writes rows.

#### Scenario: Default OFF without any config

- GIVEN a fresh Thoughtline installation with no env vars set and no config file
- WHEN `thoughtline hook session-start` is called
- THEN it exits 0 with no DB writes

#### Scenario: Env var enables capture

- GIVEN `THOUGHTLINE_PASSIVE_CAPTURE=1` is set in the environment
- WHEN `thoughtline hook session-start` is called with valid JSON
- THEN a row is inserted into `pending_events`

#### Scenario: Env var overrides config-file

- GIVEN the config file has passive capture disabled AND `THOUGHTLINE_PASSIVE_CAPTURE=1` is set
- WHEN `thoughtline hook session-start` is called
- THEN capture is active (env var wins)

---

### Requirement 5: `thoughtline worker` Command

The `worker` subcommand MUST implement a retention janitor in v1. It MUST transition `pending` rows whose `created_at` is older than the retention window (default: 7 days) to `archived` status. It MUST NOT delete or modify rows with `status = 'promoted'`. It MUST exit 0 on success and be safe to invoke repeatedly (idempotent).

The command MUST run in foreground mode only in v1 (no daemon, no background process). It MUST be safe to invoke from a cron job or scheduled task.

#### Scenario: Retention pass archives old pending rows

- GIVEN there are `pending` rows older than 7 days and `promoted` rows of any age
- WHEN `thoughtline worker` is invoked
- THEN only `pending` rows older than 7 days have their status set to `archived`; `promoted` rows are untouched

#### Scenario: No eligible rows — no-op

- GIVEN all `pending` rows are less than 7 days old
- WHEN `thoughtline worker` is invoked
- THEN exit code is 0, no rows are modified

#### Scenario: Cron-safe invocation

- GIVEN `thoughtline worker` completes its retention pass
- WHEN invoked again immediately
- THEN exit code is 0, behavior is idempotent, no duplicate operations

---

### Requirement 6: `tl_promote` MCP Tool

The `tl_promote` tool MUST accept a batch of one or more `pending_event` IDs plus target memory metadata. For each event ID in the batch, it MUST:

1. Verify the event exists and has `status = 'pending'`.
2. Create a new memory using the existing 11-type taxonomy with the supplied fields.
3. Set the event's `status` to `'promoted'` and `promoted_memory_id` to the new memory's ID.

Each event MUST be processed in its own transaction. Partial success is allowed — failures in one event do NOT roll back others. The tool MUST return a result per event ID indicating success or failure.

Input shape:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `event_ids` | []int | Yes | One or more pending_event IDs |
| `type` | string | Yes | One of the 11 taxonomy types |
| `title` | string | Yes | Memory title |
| `content` | string | Yes | Memory content |
| `topic_key` | string | No | Stable topic key for upsert |

Output shape: array of `{ event_id, memory_id, status: "promoted" | "error", error?: string }` — one entry per input `event_id`. Consistent with the response envelope shape of `tl_save`.

#### Scenario: Single event promoted successfully

- GIVEN a `pending_event` with `id = 42` and `status = 'pending'`
- WHEN `tl_promote` is called with `event_ids: [42]`, valid `type`, `title`, and `content`
- THEN a new memory is created, `pending_events.status` for row 42 becomes `'promoted'`, `promoted_memory_id` is set to the new memory ID, and the response contains `{ event_id: 42, memory_id: <new_id>, status: "promoted" }`

#### Scenario: Batch promote — partial failure

- GIVEN event 42 is `pending` and event 99 does not exist
- WHEN `tl_promote` is called with `event_ids: [42, 99]`
- THEN event 42 is promoted (committed), event 99 returns `{ event_id: 99, status: "error", error: "not found" }` — the batch is NOT rolled back wholesale

#### Scenario: Promote already-promoted event

- GIVEN event 42 has `status = 'promoted'`
- WHEN `tl_promote` is called with `event_ids: [42]`
- THEN no new memory is created; response contains `{ event_id: 42, status: "error", error: "already promoted" }`

#### Scenario: Promoted memory is findable via tl_search

- GIVEN event 42 was promoted into memory with `type = 'decision'` and `title = "use WAL mode"`
- WHEN `tl_search` is called with query `"WAL mode"`
- THEN the promoted memory appears in results (same search behavior as a directly saved memory)

#### Scenario: Crash mid-write atomicity

- GIVEN a `tl_promote` call is in progress for event 42 inside a transaction
- WHEN the process crashes before commit
- THEN on next startup, event 42 is still `pending` and no orphaned memory row exists (at-least-once delivery — incomplete promotes are safe to retry)

---

### Requirement 7: Memory Type Taxonomy — Unchanged

This change MUST NOT introduce new values into the 11-type memory taxonomy. Pending events are raw, untyped records. They MUST NOT be stored in the `memories` table. Only `tl_promote` creates memories, and it MUST use one of the existing 11 types. `AllTypes()` MUST still return exactly 11 values after this change is applied.

#### Scenario: tl_search does not surface pending events

- GIVEN 10 `pending` rows exist in `pending_events`
- WHEN `tl_search` is called with any query
- THEN zero results come from `pending_events`; only `memories` rows are returned

#### Scenario: Taxonomy count is unchanged

- GIVEN the passive-capture change is fully applied
- WHEN `AllTypes()` is called
- THEN it returns exactly 11 types (no new types added)

---

### Requirement 8: Failure Isolation

Any error inside `thoughtline hook` (DB unavailable, disk full, lock timeout, unknown event, invalid JSON) MUST NOT propagate as a non-zero exit code. The command MUST always exit 0 for any user-facing error condition. This guarantees Claude Code sessions are never broken by hook failures.

#### Scenario: DB unavailable — session survives

- GIVEN the SQLite database file is locked by another process
- WHEN `thoughtline hook post-tool-use` is invoked
- THEN exit code is 0, error is logged to stderr, the Claude Code session continues uninterrupted

---

## Resolved Decisions (2026-05-07 — user confirmed)

1. ✅ Opt-in surface: **env var `THOUGHTLINE_PASSIVE_CAPTURE=1`** for v1. Config-file deferred. (design §1)
2. ✅ Worker execution model: **foreground only**, advisory SQLite lock, cron-driven. (design §3)
3. ✅ `tl_promote` batch semantics: **per-event transactions, partial success**. Each event is committed in its own transaction; failures in one event do NOT roll back others. Response shape per-id. (user decision 2026-05-07)
4. ✅ Retention default window: **7 days**. (design §3)
5. ✅ `SessionEnd` hook: **manual v1** — no auto-prompt. Model uses `tl_pending_list` to triage. (design §5)
6. ✅ Config-file location: **deferred** (env-var only in v1). Future: `os.UserConfigDir() + /thoughtline/config.toml`. (design §6)
7. ✅ Hash-based dedup: **sole source of truth**, using `sha256(event_type || "\n" || session_id_or_empty || "\n" || tool_use_id_or_empty || "\n" || captured_at_floor_to_second || "\n" || canonical_payload)`. UNIQUE constraint on `(project, event_hash)`.

---

## Known v1 Limitations

These items are acceptable for v1 and are tracked for future work:

1. **No hook log file** (WARNING-2 from verify): Implementation uses stderr-only error reporting instead of the promised `<dataDir>/thoughtline/hook.log` JSONL file. Error diagnostics require tailing stderr or system journal. Future ADR 0005 candidate.
2. **Advisory lock simplified** (WARNING-4 from verify): Worker uses `busy_timeout=5000ms` instead of explicit `BEGIN EXCLUSIVE` lock pattern. Concurrent workers will serialize via timeout rather than friendly advisory lock. Still idempotent per spec.
3. **Orphan-memory race on MarkPromoted failure** (design §5.2): Between memory save and event status update, a failure in the second step leaves a memory without its source event linked. Mitigated by recommending callers always supply `topic_key` for upsert idempotence. v1.1 candidate.

---

## v1 Accepted Deviations from Design

- Hook logging: implementation uses stderr only; design promised `hook.log` JSONL file. Spec Req 8 still satisfied (exit 0 always).
- Advisory lock: implementation uses `busy_timeout` only; design promised explicit `BEGIN EXCLUSIVE` pattern. Spec Req 5 still satisfied (idempotent, cron-safe).
