# Design: Adopt Thoughtline, Replace Engram

> Phase: `sdd-design`
> Change: `adopt-thoughtline-replace-engram`
> Date: 2026-05-04
> Inputs: `proposal.md`, `exploration.md`, `specs/memory-type-taxonomy/spec.md`

## Technical Approach

Two parallel tracks of work, sequenced strictly:

1. **Domain track** — extend `memory.Type` taxonomy from 9 → 11 values (`+decision +architecture`) by editing two source files (`types.go`, `validate.go`) plus colocated tests. The validator is loop-based over `AllTypes()`, so the only switch-like surfaces (`tl_stats.go:96`, `view.go:120`, `tl_save.go:118`) already iterate or compare against constants — no fan-out edits needed.
2. **Migration track** — new `cmd/migrate` binary that opens the Engram DB read-only (WAL-aware), iterates `observations WHERE deleted_at IS NULL`, applies a deterministic mapper (`MapRow`), and writes through Thoughtline's `storage.Save()` so FTS5 triggers, normalized hashes, and validation all fire correctly. Domain track ships first so the migrator can rely on `decision`/`architecture` being valid types.

Then a single-session config-edit sequence rewires Claude Code from Engram MCP entries to Thoughtline equivalents.

---

## Architecture Decisions

| # | Decision | Choice | Rejected | Rationale |
|---|---|---|---|---|
| Q1 | `cmd/migrate` CLI surface | **Flags with sensible defaults**: `--source` (default `~\.engram\engram.db`), `--dest` (default reads `THOUGHTLINE_HOME` env, falls back to `%LOCALAPPDATA%\thoughtline\thoughtline.db`), `--dry-run` (default false), `--verbose` (default false). Zero-arg invocation works. | Hardcoded paths only | Flags cost ~30 LOC of `flag.Parse` and let us run `--dry-run` against a DB copy without symlinks. The `--dest` flag is essential because production Thoughtline points at `%LOCALAPPDATA%` while `--dry-run` testing should hit a temp-dir copy. |
| Q2 | Migration logging | **Both stdout AND a structured log file** at `%LOCALAPPDATA%\thoughtline\migrate-YYYY-MM-DD-HHMMSS.log` (not `~\.thoughtline\` — that path doesn't exist; `LOCALAPPDATA\thoughtline\` is where the DB lives). Format: one line per row (`level=INFO sync_id=... action=created` / `action=skipped reason=duplicate-sync-id`), final summary block. | stdout only | One-shot tools that touch user data MUST leave an audit trail. Stdout is for humans during the run; the log file is for forensics if rows look wrong later. ~291 rows × ~100 bytes = trivial disk cost. |
| Q3 | Engine-context type upgrade | **Strict mapping** — `pattern → convention` always, never opportunistically promote to `scene-pattern`/`script-pattern` based on tags. | Opportunistic: if `engine:playcanvas` tag present, upgrade `pattern → scene-pattern` | Predictable migrations are debuggable migrations. Opportunistic upgrades create two classes of `pattern` row (some upgraded, some not) with subtle rules that are impossible to audit at row-count level. The `origin-type:pattern` tag preserves provenance — operators can run a follow-up grooming pass once data is in Thoughtline if they want to re-classify. **Aligns with orchestrator recommendation.** |
| Q4 | Idempotency strategy | **Skip + log warning** — on re-run, if `sync_id` already exists in the destination, skip the row, emit `level=WARN sync_id=... action=skipped reason=duplicate-sync-id`, increment a `skipped` counter, do NOT abort. | Overwrite (data loss risk); error-abort (re-runs become impossible) | Re-runs MUST be safe — the user might re-run after partial failure, after backup-restore, or just to verify counts. Skip is the only choice that preserves Thoughtline's existing data while allowing the operator to keep going. The orchestrator-level recommendation matches. Implementation: pre-check via `SELECT 1 FROM memories WHERE sync_id = ?` before calling `storage.Save()`. (We can't rely on `storage.Save()` to detect this because it generates fresh `sync_id`s — the migrator must inject the Engram `sync_id` itself; see "Idempotency mechanism" below.) |
| Q5 | Config-edit safety | **`.bak` sibling files** for every Claude Code config touched. Pattern: `settings.json` → `settings.json.bak` written *before* edit. Each step verified by reading the file post-edit. | Git commit per file (no git repo at `~\.claude\`); atomic write-then-verify (no rollback if a *later* file edit fails) | The user's `~\.claude\` and VSCode user dir aren't git-tracked, so file-system-level snapshotting is the only rollback path. `.bak` siblings are dead simple, atomic per-file (write `.bak`, then write the new content), and trivially reversible (`move .bak → original`). Aligns with orchestrator recommendation. The pre-flight zip backup of `~\.engram\` covers Engram-side rollback; `.bak` files cover Claude-config rollback. |

---

## `cmd/migrate` Package Design

### Directory layout

```
cmd/migrate/
├── main.go              ← flag parsing, DB open/close, top-level orchestration, summary print
├── mapper.go            ← pure functions: MapType, MapTimestamp, MapRow, MapTags
├── mapper_test.go       ← table tests for every mapper (NO DB)
├── reader.go            ← engram-side: opens engram.db read-only, iterates observations
├── reader_test.go       ← table test using sqlite-on-disk fixture in t.TempDir
├── writer.go            ← thoughtline-side: idempotency check + storage.Save() call
├── writer_test.go       ← integration with real *storage.Storage in t.TempDir
└── integration_test.go  ← end-to-end: fake engram.db → fake thoughtline.db, assert row equality
```

### Public API surface (test-facing)

```go
package migrate // (internal helpers; main.go is its own package main)

// EngramObservation is the raw row read from engram.observations.
type EngramObservation struct {
    SyncID         string
    Type           string   // open set: bugfix|decision|architecture|pattern|config|preference|discovery|manual
    Title          string
    Content        string
    Project        string
    Scope          string
    TopicKey       string
    Tags           []string // engram has no tags column; always nil. Reserved for future.
    NormalizedHash string
    RevisionCount  int
    CreatedAt      string   // ISO 8601
    UpdatedAt      string
    DeletedAt      *string  // nil = active row
}

// MapType returns the thoughtline type and any tags to add (e.g. "origin-type:pattern").
// Returns ErrUnknownType if the engram type isn't in the mapping table.
func MapType(engramType string) (memory.Type, []string, error)

// MapTimestamp parses an ISO-8601 string and returns unix-ms. Always interprets
// in UTC to avoid silent timezone corruption.
func MapTimestamp(iso string) (int64, error)

// MapRow applies all column-and-type mappings, returning a memory.Memory ready
// for storage.Save(). Truncates oversize title (>200 runes) and content (>64 KiB)
// with a logged warning; never errors solely on size.
func MapRow(o EngramObservation, logger Logger) (memory.Memory, error)
```

`main.go` stays under 150 LOC — it's wiring, not logic.

### Reuse vs duplicate

| Thoughtline package | Imported? | Why |
|---|---|---|
| `internal/memory` | YES | `memory.Memory`, `memory.Type` constants, `memory.Validate` |
| `internal/storage` | YES | `storage.Open`, `*Storage.Save`, `*Storage.Close` |
| `internal/server` | NO | MCP transport — irrelevant here |
| `internal/dashboard` | NO | TUI — irrelevant |

We do NOT duplicate `Validate()` logic, hash computation, or FTS5 indexing — calling `storage.Save()` is the whole point of Approach A from the proposal.

### DB connection lifecycle

**Engram (source, read-only):**
```go
dsn := "file:" + escapedPath + "?mode=ro&_pragma=journal_mode(WAL)&_pragma=query_only(1)"
engramDB, err := sql.Open("sqlite", dsn)
defer engramDB.Close()
```
`mode=ro` plus `query_only=1` means we cannot accidentally write — even WAL checkpoints are read-only-safe. Critical because Engram's `.db-shm`/`.db-wal` files imply a live or recently-live process.

**Thoughtline (destination):**
```go
ctx := context.Background()
tlStore, err := storage.Open(ctx, destPath)  // existing API; runs migrations idempotently
defer tlStore.Close()
```

**Cleanup on error:** wrap both opens in a function that returns both handles plus a single `cleanup()` closure; defer that closure in `main()`. Any error during open closes whatever was already opened.

### Error model

Per-row error, never whole-batch abort. Rationale: with ~291 rows, finding 1 bad row should not waste a successful migration of the other 290.

```go
type RowResult struct {
    SyncID string
    Action string  // "created", "skipped-deleted", "skipped-duplicate", "skipped-truncated", "error"
    Reason string  // populated when Action contains "error" or "skipped"
}

type Summary struct {
    Total       int
    Created     int
    SkippedDeleted    int  // soft-deleted rows are not migrated
    SkippedDuplicate  int  // already in destination by sync_id
    Errors      int
    Truncations int
    Duration    time.Duration
    Rows        []RowResult  // full per-row trail; written to log file, NOT stdout
}
```

Stdout shows aggregate counters at the end; the log file holds the full `Rows` slice as one-line-per-row. Errors are reported as `Summary.Errors > 0 → exit code 1`, but the migration still processes every row.

### Idempotency mechanism (Q4 detail)

`storage.Save()` always generates a fresh UUIDv7 — it doesn't accept caller-supplied `sync_id`. So the migrator must:

1. **Pre-check**: `SELECT 1 FROM memories WHERE sync_id = ?` against the destination. If hit → skip, log `action=skipped-duplicate`, increment counter.
2. **Insert via Save**: call `storage.Save(ctx, mappedMemory)` — this generates a NEW UUIDv7 sync_id. We DON'T want that for migrated rows; we want to preserve Engram's sync_id 1:1.
3. **Post-update**: immediately after a successful `Save`, run `UPDATE memories SET sync_id = ? WHERE id = ?` to overwrite the auto-generated UUIDv7 with the Engram sync_id. The `memories.sync_id` column has a UNIQUE constraint, so a duplicate from a re-run would fail at this UPDATE — but the pre-check above prevents reaching this point.

**Alternative considered**: extend `storage.Save()` to accept an optional sync_id override. Rejected — pollutes the production API surface for a one-shot migration tool. The 3-step pattern is contained inside `cmd/migrate/writer.go` and disappears when the binary is archived.

A small helper exposed for testing:
```go
func writeRow(ctx context.Context, st *storage.Storage, db *sql.DB, m memory.Memory, engramSyncID string) (RowResult, error)
```
where `db` is the raw `*sql.DB` for the post-update — `storage.Storage` does not expose its `*sql.DB`, so the migrator opens a second handle to the same destination DB (read/write) for the UPDATE step. This is safe with WAL — SQLite handles concurrent connections to the same file.

---

## Type Taxonomy Extension Diff

### `internal/memory/types.go`

Add two constants and append them to `AllTypes()`:

```go
const (
    TypeGameDesignDecision Type = "game-design-decision"
    TypeScenePattern       Type = "scene-pattern"
    TypeAssetReference     Type = "asset-reference"
    TypePerfGotcha         Type = "perf-gotcha"
    TypePipelineStep       Type = "pipeline-step"
    TypeScriptPattern      Type = "script-pattern"
    TypeBugfix             Type = "bugfix"
    TypeConvention         Type = "convention"
    TypePreference         Type = "preference"
    TypeDecision           Type = "decision"      // ← NEW
    TypeArchitecture       Type = "architecture"  // ← NEW
)

func AllTypes() []Type {
    return []Type{
        TypeGameDesignDecision,
        TypeScenePattern,
        TypeAssetReference,
        TypePerfGotcha,
        TypePipelineStep,
        TypeScriptPattern,
        TypeBugfix,
        TypeConvention,
        TypePreference,
        TypeDecision,      // ← NEW
        TypeArchitecture,  // ← NEW
    }
}
```

### `internal/memory/validate.go`

**No changes.** `Type.Valid()` iterates `AllTypes()`; the type/scope coupling rule (`Type != TypePreference → ScopeProject`) automatically applies to the two new types because they're not `TypePreference`. This is the design discipline the spec's Requirement 2 already locks in.

### `internal/memory/validate_test.go`

Append explicit table cases (the existing `TestValidate_TypeRules` already covers them via `AllTypes()` enumeration, but we add named cases for clarity and to make the spec scenarios self-documenting):

```go
// New cases inside TestValidate_TypeRules (after the existing "every catalogued type" loop):

t.Run("decision type with project scope passes", func(t *testing.T) {
    m := validMemory()
    m.Type = TypeDecision
    m.Scope = ScopeProject
    if err := Validate(m); err != nil {
        t.Fatalf("decision/project should be valid; got %v", err)
    }
})

t.Run("architecture type with personal scope rejected", func(t *testing.T) {
    m := validMemory()
    m.Type = TypeArchitecture
    m.Scope = ScopePersonal
    if err := Validate(m); !errors.Is(err, ErrNonPreferenceMustBeProject) {
        t.Fatalf("architecture/personal should fail; got %v", err)
    }
})

t.Run("AllTypes returns 11 values including decision and architecture", func(t *testing.T) {
    types := AllTypes()
    if len(types) != 11 {
        t.Fatalf("expected 11 types, got %d", len(types))
    }
    have := make(map[Type]bool)
    for _, t := range types { have[t] = true }
    if !have[TypeDecision] || !have[TypeArchitecture] {
        t.Fatalf("AllTypes missing decision or architecture: %v", types)
    }
})
```

### Impact on other handlers (verified by reading the code)

| File | Line | Pattern | Impact |
|---|---|---|---|
| `internal/server/tl_save.go` | 118 | `args.Type == string(memory.TypePreference)` | NONE — single equality check, not an enum |
| `internal/server/tl_stats.go` | 96 | `for _, t := range memory.AllTypes()` | NONE — automatically picks up new types |
| `internal/dashboard/view.go` | 120 | `for _, t := range memory.AllTypes()` | NONE — automatically picks up new types |

No handler enumerates types via hardcoded switch; the discipline of "iterate `AllTypes()`" pays off here.

### Impact on `docs/design/memory-domain.md`

Sections requiring update (orchestrator-level edits, but flagged here for the tasks phase):
- Type catalogue table → 9 entries → 11 entries (add `decision`, `architecture` rows with description)
- Validation rules section → mention `decision` and `architecture` are project-scoped (no preference exemption)
- Migration note → reference `cmd/migrate` in the "history" section

`README.md` taxonomy table follows the same edit.

---

## Config-Edit Sequence (atomic & rollback-friendly)

### Pre-flight (MUST complete before any edit)

1. **Zip backup** of `~\.engram\` to `~\.engram-backup-2026-05-04.zip` using PowerShell `Compress-Archive`. Verify the zip exists and is non-zero. **Abort the entire change if this fails.**
2. **Run `cmd/migrate --dry-run`** against a temp-dir copy of `thoughtline.db`. Verify summary shows `Total ~= 291, Errors = 0, Truncations = 0` (or document accepted truncations).
3. **Run `cmd/migrate`** for real against production `thoughtline.db`. Verify final row count matches expectation via `thoughtline ui` Stats.

### Config edits (in strict order — each step verified before proceeding)

| Step | File | Backup | Edit | Verification |
|---|---|---|---|---|
| 1 | `~\AppData\Roaming\Code\User\mcp.json` | Copy → `mcp.json.bak` | Remove `engram` server entry; keep `thoughtline` and `context7` | Read file post-edit; assert no string `"engram"` appears as a top-level server key. |
| 2 | `~\.claude\settings.json` | Copy → `settings.json.bak` | Remove 11 `mcp__plugin_engram_engram__*` allow-list entries; remove `enabledPlugins.engram@engram`; remove `extraKnownMarketplaces.engram`; ADD 9 `mcp__thoughtline__tl_*` allow-list entries | Read file; count `mcp__thoughtline__` entries == 9; count `mcp__plugin_engram_engram__` == 0 |
| 3 | `~\.claude\mcp\engram.json` | Copy → `engram.json.bak` (file then deleted) | Delete the file | `Test-Path` returns `$false` |
| 4 | `~\.claude\CLAUDE.md` | Copy → `CLAUDE.md.bak` | Replace `<!-- gentle-ai:engram-protocol -->` block with Thoughtline equivalent (block content TBD in tasks phase, body lives in Thoughtline repo's `docs/integrations/claude-code-protocol.md`) | Read file; assert `engram-protocol` marker absent, `thoughtline-protocol` marker present |
| 5 | `~\.claude\skills\_shared\engram-convention.md` | Copy → `engram-convention.md.bak` (then file renamed) | Rename to `thoughtline-convention.md`; rewrite tool references | `Test-Path engram-convention.md` is `$false`; `Test-Path thoughtline-convention.md` is `$true` |

### Post-flight (manual but documented)

1. Quit Claude Code completely (close all windows).
2. Relaunch Claude Code in the Thoughtline project.
3. Verify boot is clean (no MCP errors in the developer console).
4. Run `tl_search` on a known historical Engram memory by `sync_id` (e.g. `obs-d044a4d5f3422eb4` from the exploration artifact). Expect a hit with the migrated content.
5. Run `tl_stats` and verify `total = pre_existing_thoughtline + migrated_count`.
6. Quit and relaunch VSCode (separately) to verify `mcp.json` change took effect.

### Rollback drill (documented but not automated)

If post-flight fails:
1. Move `*.bak` siblings back over their originals (5 files): `Move-Item -Force settings.json.bak settings.json`, etc.
2. Restore `engram.json` from its `.bak`.
3. Re-run Claude Code; Engram MCP resumes.
4. Thoughtline DB still contains migrated rows; they're benign — they don't double-up because Engram writes to its own DB. Optional: run `cmd/migrate --rollback-tag` (NOT in this change's scope) to soft-delete migrated rows by tag presence.

---

## Test Strategy

Strict TDD applies. The repo has no `:memory:` tests — every storage test uses `t.TempDir()` because FTS5 edge cases differ in shared-cache mode. We follow that pattern.

| Layer | What | Where | Approach |
|---|---|---|---|
| Unit | `MapType` covers all 8 Engram types → correct Thoughtline type + tag list, plus rejects unknown | `cmd/migrate/mapper_test.go` | Table-driven, no DB |
| Unit | `MapTimestamp` parses ISO with/without `Z`, returns unix-ms in UTC; rejects garbage | `cmd/migrate/mapper_test.go` | Table-driven, includes a known fixture date |
| Unit | `MapRow` applies all column transforms; truncates oversize content; preserves sync_id | `cmd/migrate/mapper_test.go` | Table-driven over `EngramObservation` fixtures |
| Unit | Soft-delete filter — rows with `deleted_at NOT NULL` are skipped at the reader level | `cmd/migrate/reader_test.go` | Build a tiny SQLite fixture in `t.TempDir()` with 3 rows (1 deleted), assert iterator yields 2 |
| Unit | Idempotency pre-check — `writeRow` skips when sync_id exists | `cmd/migrate/writer_test.go` | Open real `*storage.Storage` in `t.TempDir()`, insert one fake row, call `writeRow` with same sync_id, assert `Action=skipped-duplicate` |
| Unit | `memory.Type` taxonomy extension | `internal/memory/validate_test.go` | Append cases per "Type Taxonomy Extension Diff" above |
| Integration | End-to-end migration | `cmd/migrate/integration_test.go` | (1) Build a fake `engram.db` in `t.TempDir()` matching Engram's schema (a small static SQL fixture in `testdata/`); (2) seed 5–10 rows covering every Engram type plus a soft-deleted row plus an oversize content row; (3) build a fresh `thoughtline.db` in `t.TempDir()`; (4) run the migrator's `Run()` function; (5) query the destination and assert: row counts match, sync_ids preserved, types mapped per table, soft-deleted skipped, oversize truncated and logged. |

### Coverage targets

- `cmd/migrate/mapper.go` — **>90%** (pure logic, easy)
- `cmd/migrate/writer.go` — **>80%** (DB interaction)
- `cmd/migrate/reader.go` — **>80%**
- Integration test is the system-level guarantee; coverage is incidental there.

### What we do NOT test

- The user's actual `~\.engram\engram.db` contents — they vary, they're personal data, they're not in the repo.
- File operations on `~\.claude\` and `~\AppData\Roaming\Code\User\` — those are humans-and-PowerShell territory. The tasks phase will produce a verification checklist; the binary itself never touches those paths.
- `storage.Open` re-validation — already covered by `internal/storage/storage_test.go`.

---

## Open Questions Remaining

- [ ] **CLAUDE.md Thoughtline protocol body** — the design says "replace the engram-protocol block with a Thoughtline equivalent"; the actual body text is a docs deliverable for the tasks phase, not architectural. (Likely sourced from a new `docs/integrations/claude-code-protocol.md` so it can be versioned with the binary.)
- [ ] **Allow-list tool name spelling** — the proposal says `mcp__thoughtline__tl_*` but the exact prefix depends on how Claude Code namespaces non-plugin MCP servers vs plugin servers. Verify against a live `settings.json` during tasks phase. If wrong, rename uniformly across all 9 entries.

Neither blocks design — both are content-level decisions that surface when the tasks phase reads the actual current `settings.json` and `CLAUDE.md` contents.

---

## Risks Surfaced During Design

| Risk | Severity | Mitigation |
|---|---|---|
| The `sync_id` post-update step opens a *second* DB handle to thoughtline.db while `*storage.Storage` already holds one. SQLite WAL mode handles this fine, but it's worth a smoke test before relying on it in the integration test. | Low | Integration test exercises the dual-handle path with real WAL |
| `storage.Save()` runs `topic_key` upsert when `TopicKey != ""`. For migrated rows we want INSERT semantics, not upsert. If a migrated row has the same `(project, topic_key)` as a pre-existing Thoughtline row, Save will UPSERT silently and we'll lose the Engram content under a Thoughtline `topic_key` collision. | **Medium** | Migrator does its own pre-check on `(project, topic_key)` before calling Save; on collision, log `action=skipped-topic-collision reason="conflicts with existing thoughtline row"` and skip. Add this to the writer logic. |
| ISO 8601 parsing accepting non-UTC strings would shift timestamps. | Low | `MapTimestamp` enforces `time.UTC()` and rejects strings without a parseable timezone marker; table tests pin this. |
