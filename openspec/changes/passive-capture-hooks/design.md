# Design: Passive Capture Hooks

Status: ready
Phase: design
Companion artifacts: `proposal.md`, `spec.md` (parallel), `tasks.md` (next)

## 1. Architecture at a glance

Two-stage local pipeline, both stages synchronous, both stages talking to the existing SQLite database via `internal/storage`:

```
Claude Code
    │  spawns per event
    ▼
thoughtline hook <event-name>            (capture stage — fast, fail-silent)
    │  reads JSON on stdin
    │  fast-path: opt-in env check FIRST
    │  decode → validate → dedup hash → INSERT OR IGNORE
    ▼
pending_events  (new SQLite table, same DB file)
    │
    │  reviewed by the model via tl_search_pending / tl_get_pending
    │  curated by the model via tl_promote
    ▼
memories       (existing table, unchanged shape)

thoughtline worker                       (janitor stage — invoked, not daemon)
    │  archives stale rows past retention
    │  prunes archived rows past hard-delete window
```

Design boundaries:

- **Capture stage owns ONE responsibility**: turn a hook payload into a row. No semantics, no triage.
- **Promote stage owns curation**: the model decides which raw events become memories, and what `type` / `topic_key` / curated content they carry.
- **Worker stage owns retention only** (v1). LLM-based compression is explicitly out of scope and lives in a future ADR.

The hook contract is the stable seam — anything we add later (compression, embeddings, summarization) plugs in behind `pending_events` without changing what Claude Code sees.

## 2. Decisions table (resolves the 6 open questions)

| # | Question | Decision | One-line rationale |
|---|---|---|---|
| 1 | Config surface | `THOUGHTLINE_PASSIVE_CAPTURE=1` env var, read **once at process start** (per-call, since `thoughtline hook` is one-shot). No config file in v1. | Zero parser surface, zero new dependencies, trivial to enable per-shell or per-Claude-instance, easy to verify in tests. |
| 2 | `pending_events` schema | Dedicated table with full JSON payload as TEXT + extracted index columns; UNIQUE on `(project, event_hash)`; NO foreign key to `memories` (use `promoted_memory_id` nullable column). | Hash dedup works for ALL event types (incl. SessionStart/Stop with no `tool_use_id`); promotion is informational, not relational, so we don't want a FK that constrains delete behaviour on `memories`. |
| 3 | Worker shape | Foreground one-shot CLI (`thoughtline worker`), advisory lock via SQLite `BEGIN EXCLUSIVE`, exit on completion. User schedules via cron / Task Scheduler / manual run. | Same lifecycle model the rest of the binary uses; no daemon supervision; matches the proposal's "invoked, not daemon" lean. |
| 4 | `tl_promote` signature | Batch-by-default. Caller passes `{ items: [{pending_event_id, type, topic_key?, title, content, scope?, tags?}, ...] }`. Each item carries its own curated payload — content is NOT verbatim. | Single round-trip for typical session-end triage; model-curated content matches Thoughtline's "explicit, structured memory" thesis. |
| 5 | SessionEnd auto-prompt | NO auto-prompt in v1. Add a discovery tool `tl_pending_list` (filters: project, since, status, event_type, limit). Model calls it when it wants to triage. | Auto-prompts couple the queue to model behaviour we can't predict; an explicit discovery tool keeps the contract clean and testable. |
| 6 | Config file location | Defer. v1 = env-var only. Document future location as `os.UserConfigDir()/thoughtline/config.toml` once we need it. | Don't ship a parser we can't justify yet; YAGNI. |

## 3. Data model

### 3.1 `pending_events` DDL

```sql
CREATE TABLE IF NOT EXISTS pending_events (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id             TEXT    NOT NULL UNIQUE,         -- uuidv7, mirrors memories.sync_id convention
    project             TEXT    NOT NULL,
    session_id          TEXT,                            -- nullable: not all events carry one
    event_type          TEXT    NOT NULL,                -- SessionStart | UserPromptSubmit | PreToolUse | PostToolUse | Stop | SessionEnd
    tool_name           TEXT,                            -- extracted from PreToolUse/PostToolUse, NULL otherwise
    tool_use_id         TEXT,                            -- when present in payload
    payload             TEXT    NOT NULL,                -- full JSON as received on stdin (post UTF-8 validation)
    event_hash          TEXT    NOT NULL,                -- sha256 over (event_type|session_id|tool_use_id|created_at_bucket|payload-canonical)
    status              TEXT    NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending','promoted','archived')),
    promoted_memory_id  INTEGER,                         -- soft pointer to memories.id at promote time; NOT a FK
    promoted_at         INTEGER,
    archived_at         INTEGER,
    created_at          INTEGER NOT NULL,                -- unix epoch ms, server clock at insert
    captured_at         INTEGER NOT NULL                 -- unix epoch ms, parsed from payload if present, else == created_at
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_events_dedup
    ON pending_events(project, event_hash);

CREATE INDEX IF NOT EXISTS idx_pending_events_triage
    ON pending_events(project, status, captured_at DESC);

CREATE INDEX IF NOT EXISTS idx_pending_events_session
    ON pending_events(session_id, captured_at DESC)
    WHERE session_id IS NOT NULL;
```

Key choices:

- **`event_hash` as the dedup primary**, not `(session_id, tool_use_id)`. Hash composition: `sha256(event_type || "\n" || session_id_or_empty || "\n" || tool_use_id_or_empty || "\n" || captured_at_floor_to_second || "\n" || canonical_payload)`. This works uniformly across ALL six event types — SessionStart and SessionEnd dedup on (type+session+second+payload), tool events dedup on (type+session+tool_use_id+payload).
- **No FK on `promoted_memory_id`**. If a memory is hard-deleted later, we don't want pending_events to cascade or block. The pointer is informational ("we did promote this once").
- **`status` as a string check constraint** mirrors the existing `scope` pattern in `memories`. No enum tables.
- **`payload` stored verbatim as TEXT** (UTF-8 validated, JSON parse validated, but not normalized). Keeping raw payload preserves audit fidelity; we extract `event_type`, `session_id`, `tool_name`, `tool_use_id`, `captured_at` into columns for indexing.

### 3.2 Migration

Existing migration mechanism (confirmed by reading `internal/storage/storage.go` and `schema.go`):

- Append the new `CREATE TABLE` + indexes to `schemaSQL` (idempotent — uses `IF NOT EXISTS`).
- Bump `currentSchemaVersion` from `2` to `3`.
- The `INSERT OR IGNORE INTO schema_version` line in `migrate()` records the new version with its applied_at.

No ALTER TABLE needed — pending_events is a brand-new table. Existing DBs pick it up on next `Open`. Backward compat: a binary built without this change ignores the table; a binary built with it tolerates a DB that doesn't yet have the row in `schema_version`.

### 3.3 Domain package

Add `internal/pending/` (new package):

```go
package pending

type Event struct {
    ID                int64
    SyncID            string
    Project           string
    SessionID         string  // "" when absent
    EventType         string
    ToolName          string  // "" when absent
    ToolUseID         string  // "" when absent
    Payload           string  // raw JSON
    Hash              string
    Status            Status
    PromotedMemoryID  int64   // 0 when not promoted
    PromotedAt        time.Time
    ArchivedAt        time.Time
    CreatedAt         time.Time
    CapturedAt        time.Time
}

type Status string
const (
    StatusPending  Status = "pending"
    StatusPromoted Status = "promoted"
    StatusArchived Status = "archived"
)
```

Mirrors `internal/memory` shape. Validation lives here (`Validate(Event) error`) so storage stays a transport.

## 4. Component map

| Component | Lives in | Responsibility | Reads | Writes |
|---|---|---|---|---|
| Hook CLI | `cmd/thoughtline/hook.go` (new) | One-shot subcommand; reads stdin, builds Event, calls storage; ALWAYS exits 0 unless usage error | env, stdin | log file, `pending_events` |
| Worker CLI | `cmd/thoughtline/worker.go` (new) | One-shot subcommand; archives + prunes; advisory-locked | flag args | `pending_events` |
| Pending domain | `internal/pending/` (new) | `Event`, `Validate`, hash, JSON helpers | — | — |
| Pending storage | `internal/storage/pending.go` (new) | `InsertPending`, `ListPending`, `GetPendingByID`, `MarkPromoted`, `MarkArchived`, `PrunePending` | sqlite | sqlite |
| `tl_pending_list` MCP tool | `internal/server/tl_pending_list.go` (new) | Filter pending events for the model | storage | — |
| `tl_pending_get` MCP tool | `internal/server/tl_pending_get.go` (new) | Fetch full payload of one pending event | storage | — |
| `tl_promote` MCP tool | `internal/server/tl_promote.go` (new) | Batch promote: insert memories + mark events | storage | `memories`, `pending_events` |
| Hooks manifest | `plugin/claude-code/hooks/hooks.json` (modified) | Register the 6 hook events → `thoughtline hook <name>` | — | — |
| Docs | `docs/integrations/claude-code-passive-capture.md` (new); `docs/COMPARISON.md` (modified) | User-facing opt-in + privacy + license-hygiene clause | — | — |

## 5. Data flow per event

### 5.1 Capture path (Claude Code → DB)

```
Claude spawns: thoughtline hook PostToolUse < payload.json
  1. main.go switches on cmd == "hook" → runHook(ctx, args, os.Stdin)
  2. runHook checks os.Getenv("THOUGHTLINE_PASSIVE_CAPTURE") == "1"
       false → return nil  (exit 0, no DB open, fast-path well under 10ms)
  3. Resolve project (THOUGHTLINE_PROJECT or cwd basename)
  4. Read stdin, cap at 1 MiB to defend against runaways
  5. json.Unmarshal into a generic map; pull session_id, tool_use_id, tool_name,
     event-type-specific timestamp (default to now if absent)
  6. Build pending.Event; pending.Validate
  7. Compute event_hash
  8. storage.InsertPending — INSERT OR IGNORE; ignored hits return ActionNoop
  9. On ANY error after step 2: write to log file (see §7), exit 0.
     Step 1 errors (unknown event name) exit 2 — they're a misconfiguration, not a runtime fault.
```

### 5.2 Promote path (model → DB)

```
Model calls tl_pending_list project=foo status=pending limit=50
  → returns [{id, event_type, captured_at, snippet}, ...]
Model calls tl_pending_get id=42 (optional, for full payload review)
Model calls tl_promote items=[
    {pending_event_id: 42, type: "bugfix", topic_key: "bugfix/foo-crash",
     title: "...", content: "...", tags: ["x","y"]},
    ...
]
  For each item INDEPENDENTLY:
    1. Validate item (type ∈ taxonomy, content non-empty, etc.)
    2. Lookup pending event by id; reject if missing or already promoted
    3. storage.Save(memory) → returns memory.ID
    4. storage.MarkPromoted(pending_event_id, memory.ID)
    5. Append per-item result {pending_event_id, memory_id, status, error?}
  Return aggregated results array.
```

**Per-item transaction semantics** (resolved by user 2026-05-07, supersedes earlier "all-or-nothing" wording): each item commits in its own transaction. A failure in item N does NOT roll back items 1..N-1. The model receives a per-id outcome and can retry only the failed entries. Rationale: matches the spec's partial-success scenario, gives the model granular retry, avoids whole-batch retries that re-pay validation cost.

**Known v1 limitation — orphan memory on `MarkPromoted` failure**: between step 3 (memory saved) and step 4 (event marked promoted) there is a non-atomic seam. If step 4 fails (e.g. DB lock race), the memory exists but the event remains `pending`. A naive retry would call `Save` again and create a *second* memory.

Mitigations available without a v1.1 fix:
1. The `topic_key` upsert in `Save` makes the second save a no-op when the caller passes the same `topic_key` — recommend in docs that callers always supply `topic_key` when promoting.
2. `MarkPromoted` is a single-row UPDATE on a row we just successfully INSERT-ed; lock contention is the only realistic failure path, and SQLite WAL + `busy_timeout` make it rare.
3. The orphan memory is harmless content — it just isn't linked back to the source event.

A v1.1 follow-up could either (a) wrap `Save + MarkPromoted` in a single SQL transaction (requires `Storage.Save` to accept an existing tx handle), or (b) make `tl_promote` idempotent by checking `pending_events.promoted_memory_id` first and returning the linked memory instead of saving again. Tracked as a known issue in `docs/decisions/0004-passive-capture-via-hooks.md`.

### 5.3 Worker path (cron → DB)

```
thoughtline worker --retention 7d --hard-delete 30d
  1. Acquire advisory lock: BEGIN EXCLUSIVE; SELECT 1 FROM pending_events LIMIT 0; COMMIT;
     If another worker holds it, exit 0 with a stderr note.
  2. UPDATE pending_events SET status='archived', archived_at=? WHERE status='pending' AND captured_at < ?
  3. DELETE FROM pending_events WHERE status='archived' AND archived_at < ?
  4. Print one-line summary; exit 0.
```

Defaults: retention `7d`, hard-delete `30d` (configurable via flags). Soft-then-hard, never one-shot delete: gives operators a window to inspect what was archived.

## 6. Hook payload assumptions (Claude Code contract)

Anthropic Claude Code documents the hook events publicly. Working assumption (to be confirmed in the spec/verification phase):

| Event | Stable fields we extract | Notes |
|---|---|---|
| `SessionStart` | `session_id`, `cwd`, `transcript_path`, `hook_event_name` | Fires on startup/clear/resume — already used by current `protocol` hook. |
| `UserPromptSubmit` | `session_id`, `prompt`, `hook_event_name` | Privacy-sensitive (contains user input verbatim). |
| `PreToolUse` | `session_id`, `tool_use_id`, `tool_name`, `tool_input`, `hook_event_name` | Contains tool args — may include secrets. |
| `PostToolUse` | `session_id`, `tool_use_id`, `tool_name`, `tool_response`, `hook_event_name` | Contains tool output. |
| `Stop` | `session_id`, `hook_event_name` | End of model turn. |
| `SessionEnd` | `session_id`, `reason`, `hook_event_name` | True session boundary. |

If any field is absent in a real payload, capture still succeeds (we store the raw blob; missing extracted columns become NULL). The contract is "store everything, extract what you can".

**Assumption flag for spec phase**: confirm against current Claude Code docs that the field names above are stable. If the docs disagree, update the extractor only — the `payload` blob means nothing is lost.

## 7. Failure isolation

The hook command's #1 invariant: **never break the host Claude session**. Implementation rules:

1. The `runHook` function returns `error` only for unknown subcommand-name. EVERY runtime error (DB locked, disk full, malformed JSON, oversized payload) is caught, written to a log file, and yields `os.Exit(0)`.
2. Default panics are recovered at the `runHook` boundary with `defer recover` and logged.
3. Log file location: `<dataDir>/thoughtline/hook.log`, append-only, line-delimited JSON `{ts, event_type, err}`. Manual rotation via worker's `--rotate-log` flag (size-based, default 10 MiB → keep one `.log.1`).
4. Stdin read is bounded by `io.LimitReader(stdin, 1<<20)` — 1 MiB cap is well above any realistic Claude Code payload.
5. Hook timeout: Claude Code already accepts a `timeout` field per hook in `hooks.json`. We register all six events with `timeout: 10` seconds, an order of magnitude over our p99 budget.

## 8. Performance budget

| Path | Budget | How we achieve it |
|---|---|---|
| Hook (opt-in OFF) | < 5 ms | Single env read; no DB open. |
| Hook (opt-in ON, dedup hit) | < 25 ms p99 | Single `INSERT OR IGNORE` against an indexed unique key; WAL already enabled. |
| Hook (opt-in ON, new row) | < 50 ms p99 | Same, plus index update. |
| Worker pass on 10k rows | streaming, < 100 MiB RSS | Statement-level UPDATE/DELETE; never SELECT-then-loop. |

Justification: the existing `tl_save` integration tests show a single insert + FTS5 trigger fan-out completes in single-digit ms on Windows under WAL; pending_events has no FTS5 trigger, so the budget is comfortable.

## 9. License hygiene plan

1. **No claude-mem source files are opened during the implementation phase.** Only proposal-level architectural notes (already captured in this design) inform the work.
2. **Forbidden-strings grep in CI**: maintain a small list (e.g. claude-mem-specific table names, CLI flag spellings, prompt strings) in `scripts/check-no-claude-mem.sh`; add a CI step that fails the PR if any match appears in the diff.
3. **PR checklist**: a one-line affirmation in `docs/COMPARISON.md` and in the PR template — "no AGPL source/prompt/schema text copied".
4. **Reviewer item**: any new file under `internal/pending/`, `cmd/thoughtline/hook.go`, or `cmd/thoughtline/worker.go` must be reviewed against the checklist before merge.

## 10. Tests strategy (Strict TDD)

Strict TDD is enabled for this project. Every code change ships with a failing test first. Test taxonomy:

- **Unit (`internal/pending`)**: `Validate`, hash determinism, hash differs across event types, hash collision avoidance for same-type same-session.
- **Unit (`internal/storage/pending_test.go`)**: insert, dedup (insert-twice = single row + Noop), list filters (status, project, since), MarkPromoted, MarkArchived, PrunePending.
- **Integration (`cmd/thoughtline/hook_test.go`)**: spawn the binary with a temp DB and crafted stdin; assert opt-in OFF = no row, opt-in ON = one row, malformed JSON = exit 0 + log line, oversized payload = exit 0 + log line.
- **Integration (`cmd/thoughtline/worker_test.go`)**: seed pending+old rows, run worker, assert status transitions and pruning.
- **MCP integration (`internal/server/tl_promote_test.go`)**: end-to-end through mcp-go server harness — pending row → tl_promote → memory row + status=promoted + promoted_memory_id set.
- **Cross-platform CI**: existing `.github/workflows/ci.yml` runs ubuntu/macos/windows. Confirmed Windows-safe by using `os/exec` directly from Go tests rather than shell strings; `hooks.json` invokes the binary directly (no shell).
- **Negative tests**: malformed JSON, missing `event_type`, capture-disabled fast-path benchmark (< 5 ms).

## 11. Privacy posture

- User prompts (`UserPromptSubmit.prompt`) and tool inputs (`PreToolUse.tool_input`) may contain secrets, paths, or sensitive data.
- v1 stores raw payloads unencrypted on disk, in the same DB file as `memories`. **This must be stated explicitly** in `docs/integrations/claude-code-passive-capture.md` and on the README pointer to that doc.
- Default-OFF posture is the primary mitigation; users opting in are signing up for local persistence of their session content.
- Future work (separate ADR): encryption-at-rest, sensitive-field redaction filters at capture time, allow/deny lists on `tool_name`.

## 12. Risks (architectural)

| Risk | Likelihood | Mitigation in this design |
|---|---|---|
| Hook payload schema drifts in a future Claude Code release | Med | We store the raw payload. Extractors degrade gracefully (NULL columns); backfill possible. |
| `event_hash` collision causes a real event to be dropped | Very Low | sha256 over a tuple including raw payload makes collisions cryptographically unrealistic. |
| Worker advisory lock deadlocks on a stale BEGIN EXCLUSIVE | Low | Worker is one-shot; OS process exit releases the lock. Add `--force-unlock` escape hatch only if a real incident appears. |
| Pending queue grows unbounded if user never runs worker | Med | Document the cron/Task Scheduler step prominently; ship a Task Scheduler XML and a sample `crontab` line; consider in M+1 a "self-prune on capture" guard with a high threshold. |
| Privacy: user enables capture, forgets, ships DB to a teammate | Med | README + integration doc + first-run stderr line stating capture is ON when env is set. |

## 13. What this design intentionally does NOT decide

- LLM compression of pending events into typed memories (separate ADR).
- Multi-provider hook adapters (Cursor, Zed, etc.) — the hook CLI stays Claude-Code-shaped in v1.
- A web/TUI surface for triaging pending events — out of scope; model-driven triage via MCP tools is sufficient for v1.
- Encryption-at-rest — separate change; same DB as `memories` so any future change covers both.
