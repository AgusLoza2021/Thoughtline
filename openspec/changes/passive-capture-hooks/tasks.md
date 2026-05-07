# Tasks: passive-capture-hooks

Status: complete
Phase: tasks
Predecessor artifacts: proposal.md, specs/, design.md

## Reconciliation Notes (spec vs design)

| Point | Resolution |
|---|---|
| Schema columns (`event_name` vs `event_type`, `event_seq`, UNIQUE key) | Design wins — DDL in §3.1 is authoritative. `event_seq` dropped. UNIQUE on `(project, event_hash)`. |
| `session_id` nullability | Design wins — nullable; SessionStart/Stop/End may lack it. |
| `promoted_memory_id` FK | Design wins — soft pointer only, NOT a real FK. |
| `tl_promote` input shape | Design wins — per-item `{pending_event_id, type, topic_key?, title, content, scope?, tags?}`. |
| `tl_promote` transaction model | **Resolved here**: per-item transactions (spec partial-success scenario is the observable requirement). Design §5.2 confirms retryability. |
| New MCP tools `tl_pending_list` / `tl_pending_get` | Design adds both. Tasks include all 3 tools and all 3 added to allow-list (spec claude-code-integration only named `tl_promote` — gap filled here). |
| Schema version | 2 → 3 (design §3.2). |

---

## Group A — Foundation: Schema + Migration

**[x] A1** · TEST · `internal/storage/migration_test.go` · Size: S
- Write failing test: v2→v3 migration creates `pending_events` with exact DDL (all columns, CHECK constraint, 3 indexes, UNIQUE constraint on `(project, event_hash)`)
- Covers spec Req 1 scenario "Schema migration creates table"
- Depends on: nothing

**[x] ACODE · `internal/storage/schema.go` · Size: M
- Append `pending_events` DDL + 3 indexes to `schemaSQL` using `IF NOT EXISTS`
- Bump `currentSchemaVersion` constant from `2` to `3`
- Depends on: A1 (RED)

**[x] ATEST · `internal/storage/migration_test.go` · Size: S
- Write failing test: running migration twice against same DB is a no-op (no error, no data loss)
- Covers spec Req 1 scenario "Re-running migration is safe"
- Depends on: A2 (GREEN)

**[x] ACODE · `internal/storage/schema.go` (already modified in A2) · Size: S
- Confirm idempotency is satisfied by `IF NOT EXISTS` (no extra code needed); test A3 should pass after A2
- Depends on: A3 (RED → verify GREEN)

---

## Group B — Domain Package (`internal/pending/`)

**[x] BTEST · `internal/pending/hash_test.go` · Size: S
- Same input tuple → same hash; field order in canonical form doesn't affect result
- Covers design §3.1 hash composition: `sha256(event_type || "\n" || session_id_or_empty || "\n" || tool_use_id_or_empty || "\n" || captured_at_floor_to_second || "\n" || canonical_payload)`
- Depends on: nothing (pure function, no storage needed)

**[x] BTEST · `internal/pending/hash_test.go` · Size: S
- Different input → different hash; same-type same-session different payload → different hash
- Depends on: B1

**[x] BCODE · `internal/pending/hash.go` · Size: S
- Implement `ComputeHash(eventType, sessionID, toolUseID string, capturedAt time.Time, payload []byte) string`
- JSON-canonicalize payload (sorted keys, no whitespace) before hashing
- Depends on: B1, B2 (RED)

**[x] BTEST · `internal/pending/pending_test.go` · Size: S
- `Validate` rejects event with empty `event_type`; accepts all 6 known event types
- Covers spec Req 3 (6 accepted event names)
- Depends on: nothing

**[x] BCODE · `internal/pending/pending.go` · Size: S
- Define `Event` struct (fields per design §3.3), `Status` type + constants, `Validate(Event) error`
- Depends on: B4 (RED)

---

## Group C — Storage Layer (`internal/storage/pending.go`)

**[x] CTEST · `internal/storage/pending_test.go` · Size: S
- `InsertPending`: happy path inserts row with `status = 'pending'`, `promoted_memory_id` NULL
- Covers spec Req 2 scenario "Newly inserted event is pending"
- Depends on: A2

**[x] CTEST · `internal/storage/pending_test.go` · Size: S
- `InsertPending`: same hash twice → second call returns `ActionNoop`, row count stays 1
- Covers spec Req 3 scenario "Duplicate event — idempotent"
- Depends on: C1

**[x] CCODE · `internal/storage/pending.go` · Size: M
- Implement `InsertPending(ctx, Event) (ActionResult, error)` using `INSERT OR IGNORE`
- Depends on: C1, C2 (RED)

**[x] CTEST · `internal/storage/pending_test.go` · Size: S
- `ListPending`: filter by project, status, since-timestamp, event_type; pagination via limit+offset
- Covers spec Req 6 (tl_pending_list backend)
- Depends on: C3

**[x] CCODE · `internal/storage/pending.go` · Size: M
- Implement `ListPending(ctx, ListPendingParams) ([]pending.Event, error)`
- Depends on: C4 (RED)

**[x] CTEST · `internal/storage/pending_test.go` · Size: S
- `GetPendingByID`: fetches full event including payload; returns error on missing ID
- Depends on: C3

**[x] CCODE · `internal/storage/pending.go` · Size: S
- Implement `GetPendingByID(ctx, id int64) (pending.Event, error)`
- Depends on: C6 (RED)

**[x] CTEST · `internal/storage/pending_test.go` · Size: S
- `MarkPromoted`: sets `status='promoted'`, `promoted_memory_id`, `promoted_at`; second call returns error "already promoted"
- Covers spec Req 2 scenario "Promoted row stays promoted" and Req 6 scenario "Promote already-promoted"
- Depends on: C3

**[x] CCODE · `internal/storage/pending.go` · Size: S
- Implement `MarkPromoted(ctx, pendingID, memoryID int64) error`
- Depends on: C8 (RED)

**[x] CTEST · `internal/storage/pending_test.go` · Size: S
- `SweepPending`: archives `pending` rows older than retention window; leaves `promoted` rows untouched; DELETE archived rows past hard-delete window
- Covers spec Req 5 scenarios
- Depends on: C3

**[x] CCODE · `internal/storage/pending.go` · Size: S
- Implement `SweepPending(ctx, retentionDur, hardDeleteDur time.Duration) (SweepResult, error)`
- Depends on: C10 (RED)

---

## Group D — Hook CLI (`cmd/thoughtline/hook.go`)

**[x] DTEST · `cmd/thoughtline/hook_test.go` · Size: M
- Opt-in ON + valid JSON on stdin → 1 row inserted, exit 0, stdout empty
- Opt-in OFF → exit 0, no row, stdout empty
- Malformed JSON → exit 0, error on stderr, no row
- Unknown event name → exit 0, error on stderr, no row
- Oversized payload (>1 MiB) → exit 0, logged, no row
- Covers spec Req 3 scenarios, Req 4 scenarios, Req 8 scenario
- Depends on: A2, B3, B5, C3

**[x] DCODE · `cmd/thoughtline/hook.go` · Size: M
- Implement `runHook(ctx, args []string, stdin io.Reader) error`
- Fast-path: env check before DB open; `io.LimitReader` at 1 MiB; `defer recover` at boundary; log errors to `<dataDir>/thoughtline/hook.log` (append-only JSONL); exits 0 on all user-facing errors; exits 2 only on unknown subcommand
- Depends on: D1 (RED)

**[x] DTEST · `cmd/thoughtline/hook_test.go` · Size: S
- Two invocations with identical payload → only 1 DB row (dedup via INSERT OR IGNORE)
- Covers spec Req 3 scenario "Duplicate event — idempotent"
- Depends on: D2

**[x] DCODE · `cmd/thoughtline/main.go` (or dispatch file) · Size: S
- Wire `hook` subcommand into the CLI dispatch table
- Depends on: D2

---

## Group E — Worker CLI (`cmd/thoughtline/worker.go`)

**[x] ETEST · `cmd/thoughtline/worker_test.go` · Size: M
- Empty queue → exit 0, no errors
- Old pending rows (>7d) → archived; promoted rows untouched
- Two concurrent workers → second exits 0 with advisory lock note on stderr
- Covers spec Req 5 scenarios
- Depends on: A2, C11

**[x] ECODE · `cmd/thoughtline/worker.go` · Size: M
- Implement `runWorker(ctx, flags) error`
- Acquire advisory lock via `BEGIN EXCLUSIVE`; run `SweepPending`; print summary; exit 0
- Flags: `--retention` (default 7d), `--hard-delete` (default 30d), `--rotate-log` (size-based, default 10 MiB)
- Depends on: E1 (RED)

**[x] ECODE · `cmd/thoughtline/main.go` · Size: S
- Wire `worker` subcommand into CLI dispatch table
- Depends on: E2

---

## Group F — MCP Tools

**[x] FTEST · `internal/server/tl_pending_list_test.go` · Size: S
- Returns paginated list for project + status filters; empty result when queue empty
- Covers spec Req 5 / design §5.2 (model discovery)
- Depends on: C5

**[x] FCODE · `internal/server/tl_pending_list.go` · Size: S
- Implement MCP tool handler calling `storage.ListPending`; return snippet (id, event_type, captured_at, first 200 chars of payload)
- Depends on: F1 (RED)

**[x] FTEST · `internal/server/tl_pending_get_test.go` · Size: S
- Returns full payload for valid ID; returns error for missing ID
- Depends on: C7

**[x] FCODE · `internal/server/tl_pending_get.go` · Size: S
- Implement MCP tool handler calling `storage.GetPendingByID`
- Depends on: F3 (RED)

**[x] FTEST · `internal/server/tl_promote_test.go` · Size: M
- Single item: pending row → memory created, status=promoted, promoted_memory_id set
- Batch: item 42 (pending) + item 99 (not found) → 42 promoted, 99 errors, no rollback of 42
- Already-promoted: returns `{status: "error", error: "already promoted"}`, no duplicate memory
- Promoted memory findable via tl_search
- Covers spec Req 6 all scenarios
- Depends on: C9, storage.Save (existing)

**[x] FCODE · `internal/server/tl_promote.go` · Size: M
- Implement MCP tool: for each item open a separate transaction (`Save` + `MarkPromoted`); collect results; partial failure does not roll back committed items
- Input: `{items: [{pending_event_id, type, topic_key?, title, content, scope?, tags?}]}`
- Output: `[{pending_event_id, memory_id, sync_id, status, error?}]`
- Depends on: F5 (RED)

**[x] FTEST · `internal/server/server_test.go` · Size: S
- Server registers exactly 3 new tools: `tl_pending_list`, `tl_pending_get`, `tl_promote`
- Depends on: F2, F4, F6

**[x] FCODE · `internal/server/server.go` · Size: S
- Register the 3 new tool handlers in server initialization
- Depends on: F7 (RED)

---

## Group G — Plugin Integration

**[x] GCODE · `plugin/claude-code/hooks/hooks.json` · Size: S
- Add 6 hook entries (SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop, SessionEnd); cross-platform command form; `timeout: 10`; `on_failure: continue`
- Covers claude-code-integration spec Req 8
- Depends on: D2 (hook CLI must exist)

**[x] GTEST · `plugin/claude-code/hooks/hooks_test.go` (Go test) · Size: S
- Parse hooks.json; assert 6 entries exist with correct event names and binary references
- Covers claude-code-integration spec Req 8 scenario "Hooks file contains all 6 entries"
- Depends on: G1

**[x] GCODE · `~/.claude/settings.json` (user config, documented in docs) · Size: S
- Add `mcp__thoughtline__tl_promote`, `mcp__thoughtline__tl_pending_list`, `mcp__thoughtline__tl_pending_get` to allow-list (brings total to 12 Thoughtline tools)
- NOTE: spec says 10 (only `tl_promote`), but design adds 3 tools — all 3 must be allow-listed; document discrepancy in task note
- Covers claude-code-integration spec Req 9
- Depends on: F8

---

## Group H — License Hygiene

**[x] HCODE · `scripts/check-no-claude-mem.sh` (+ `scripts/check-no-claude-mem.ps1`) · Size: S
- Grep for forbidden strings (claude-mem table names, CLI flag spellings, prompt strings); exit 1 on match
- Depends on: nothing (can run in parallel)

**[x] HCODE · `.github/workflows/ci.yml` · Size: S
- Add fast pre-test step running `check-no-claude-mem.sh` (unix) / `.ps1` (windows)
- Depends on: H1

**[x] HCODE · `CONTRIBUTING.md` · Size: S
- Add contributor checklist note: "no AGPL source/prompt/schema text copied from claude-mem"
- Depends on: nothing

---

## Group I — Documentation

**[x] ICODE · `docs/integrations/claude-code-passive-capture.md` · Size: M
- Opt-in flow, env var, privacy statement (plaintext storage), sample cron line + Task Scheduler XML
- Depends on: D2, G1 (content references real commands)

**[x] ICODE · `docs/COMPARISON.md` · Size: S
- Update "auto-capture" row to ✅ opt-in; add license-hygiene affirmation clause
- Depends on: nothing

**[x] ICODE · `README.md` · Size: S
- One-line mention of passive capture + link to I1 doc
- Depends on: I1

**[x] ICODE · `docs/decisions/0004-passive-capture-via-hooks.md` · Size: S
- ADR: problem, options, decision, consequences
- Depends on: nothing

**[x] ICODE · `docs/PROGRESS.md` · Size: S
- Add this milestone entry
- Depends on: nothing (write last, after all groups GREEN)

---

## Group J — Integration + E2E

**[x] JTEST · `internal/server/e2e_test.go` (or new `cmd/thoughtline/e2e_test.go`) · Size: M
- Full happy path: `thoughtline hook post-tool-use` → `tl_pending_list` returns 1 row → `tl_promote` → `tl_search` finds the promoted memory
- Depends on: D2, F2, F6, storage (all groups A–F complete)

**[x] JTEST · `internal/storage/pending_test.go` · Size: S
- DB file permissions: assert file mode is not world-readable on Unix (os.Stat + mode check)
- Documents v1 privacy posture (design §11)
- Depends on: A2

---

## Group K — Verification Gate

**[x] KCODE · CI (no new files) · Size: S
- Run `go test ./...` — full suite must be green
- Depends on: all groups A–J

**[x] KCODE · CI · Size: S
- Run `go vet ./...` — no vet errors
- Depends on: K1

**[x] KCODE · CI · Size: S
- Verify cross-platform build matrix (ubuntu/macos/windows) passes — already wired in `.github/workflows/ci.yml`
- Depends on: K1

**[x] KMANUAL · local · Size: S
- Smoke test: set `THOUGHTLINE_PASSIVE_CAPTURE=1`, open Claude Code session, verify `tl_pending_list` returns events, `tl_promote` creates a memory findable via `tl_search`
- Depends on: G1, G3

---

## Dependency / Parallel Execution Map

```
A1→A2→A3→A4                    (sequential — foundation)
B1→B2→B3   B4→B5               (can run in parallel with A)
C1→C2→C3→C4→C5
         →C6→C7
         →C8→C9
         →C10→C11              (C branches after C3; all depend on A2)
D1→D2→D3→D4                    (depends on A2, B3, B5, C3)
E1→E2→E3                       (depends on A2, C11)
F1→F2   F3→F4   F5→F6→F7→F8   (depends on C5, C7, C9)
G1→G2   G3                     (G1 depends on D2; G3 depends on F8)
H1→H2   H3                     (parallel with everything)
I1→I3   I2   I4   I5           (I1 depends on D2, G1; rest parallel)
J1                              (depends on D2, F2, F6 — all A–F green)
J2                              (depends on A2 — can run early)
K1→K2→K3   K4                  (gate tasks, sequential at end)
```

Critical path: A1 → A2 → C3 → [C9 + C5] → F6 → J1 → K1 → K2 → K3

---

## Task Count Summary

| Group | Tasks | Type split |
|---|---|---|
| A (Schema/Migration) | 4 | 2 test, 2 code |
| B (Domain package) | 5 | 3 test, 2 code |
| C (Storage layer) | 11 | 6 test, 5 code |
| D (Hook CLI) | 4 | 2 test, 2 code |
| E (Worker CLI) | 3 | 1 test, 2 code |
| F (MCP tools) | 8 | 4 test, 4 code |
| G (Plugin integration) | 3 | 1 test, 2 code |
| H (License hygiene) | 3 | 0 test, 3 code |
| I (Documentation) | 5 | 0 test, 5 docs |
| J (E2E + privacy) | 2 | 2 test |
| K (Verification gate) | 4 | 0 test, 4 gate |
| **Total** | **52** | **21 test, 31 code/docs/gate** |

Effort estimate: 23× S + 17× M + 0× L ≈ 3–5 dev-days at Strict TDD pace.
