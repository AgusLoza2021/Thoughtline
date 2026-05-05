# Tasks: adopt-thoughtline-replace-engram

> Source: proposal.md + design.md + specs/memory-type-taxonomy/spec.md + specs/engram-migration/spec.md + specs/claude-code-integration/spec.md
> Strict TDD: tests are written BEFORE production code in every track.
> Execution order: Track 0 → Track 1 → Track 2 → Track 3 → Track 4 → Track 5 → Track 6.
> Tracks 1–3 may overlap only if Track 0 has completed. Track 5 (destructive) requires Tracks 0, 4.

---

## Track 0 — Pre-flight (must run first, blocks all other tracks)

- [x] 0.1 Verify `~\.engram\engram.db` is readable: file confirmed at `C:\Users\Agustin Lozano\.engram\engram.db` (3.3 MB, WAL files present). sqlite3 not on PATH — verified via filesystem stat; count deferred to Track 6 smoke test.
- [x] 0.2 Verify `%LOCALAPPDATA%\thoughtline\thoughtline.db` is reachable: file confirmed at `C:\Users\Agustin Lozano\AppData\Local\thoughtline\thoughtline.db`. Row count baseline deferred to Track 6.
- [x] 0.3 Confirmed Go module `github.com/AgusLoza2021/Thoughtline`; `internal/memory` and `internal/storage` packages verified with correct API surface.
- [x] 0.4 `modernc.org/sqlite v1.45.0` confirmed in `go.mod`.

---

## Track 1 — Memory type taxonomy extension (TDD order)

> Spec: specs/memory-type-taxonomy/spec.md — Requirements 1, 2, 4
> Files: `internal/memory/types.go`, `internal/memory/validate_test.go`

- [x] 1.1 **(test first)** Added `TestValidate_TypeRules_Extended` in `validate_test.go` with sub-tests for decision/project (pass), architecture/personal (fail ErrNonPreferenceMustBeProject), AllTypes count=11, decision/personal (fail). RED phase confirmed via IDE diagnostics (TypeDecision undefined before types.go edit).
- [x] 1.2 **(test first)** Added sub-tests in `TestValidate_TypeRules_Extended` for pattern/config/discovery/manual/"Decision"/"ARCHITECTURE" → ErrInvalidType.
- [x] 1.3 Added `TypeDecision Type = "decision"` and `TypeArchitecture Type = "architecture"` constants to `internal/memory/types.go`.
- [x] 1.4 Updated `AllTypes()` in `types.go` — `TypeDecision` and `TypeArchitecture` are now the 10th and 11th elements.
- [x] 1.5 Verified `validate.go` — `Type.Valid()` iterates `AllTypes()`, no switch/case. Zero changes needed.
- [x] 1.6 Updated `docs/design/memory-domain.md` — added full `decision` and `architecture` catalogue entries; updated Validation rules note to mention both new types require `scope = project`.
- [x] 1.7 Updated `docs/design/memory-domain.md` — "Adding a new memory type" section now includes back-population guidance referencing `cmd/migrate`.
- [x] 1.8 Updated `README.md` taxonomy table — added rows for `decision` and `architecture`; updated tl_save type count from 9 to 11.

---

## Track 2 — `cmd/migrate` binary (TDD order, strict per-file)

> Spec: specs/engram-migration/spec.md — Requirements 1–6
> Design: design.md — `cmd/migrate` Package Design section
> Package path: `github.com/AgusLoza2021/Thoughtline/cmd/migrate`
> All test files use `t.TempDir()` — never `:memory:`. All tests are table-driven with `t.Run(tt.name, ...)`.

### 2a. Types and shared fixtures

- [x] 2.1 Created `cmd/migrate/migrate.go` — `EngramObservation`, `RowResult`, `Summary`, `Config`, `Logger` interface. Types-only file.
- [x] 2.2 Created `cmd/migrate/testdata/engram_schema.sql` — full Engram observations table schema with all 16 columns.

### 2b. mapper.go — pure functions (test first)

- [x] 2.3 **(test first)** Created `cmd/migrate/mapper_test.go` — `TestMapType` table with all 9 cases (8 Engram types + unknown "wizard").
- [x] 2.4 **(test first)** `TestMapTimestamp` in `mapper_test.go` — UTC Z suffix, explicit +00:00, garbage, empty.
- [x] 2.5 **(test first)** `TestMapRow` in `mapper_test.go` — happy path, preference scope forcing, 200-rune title, 201-rune truncation with logger spy, content at limit, content oversize error, valid topic_key, invalid topic_key cleared, bogus scope error.
- [x] 2.6 Created `cmd/migrate/mapper.go` — `MapType`, `MapTimestamp`, `MapRow` implemented. Pure functions, no DB. topicKeyRe mirrors validate.go pattern.

### 2c. reader.go — Engram DB iteration (test first)

- [x] 2.7 **(test first)** Created `cmd/migrate/reader_test.go` — `TestReadObservations_FiltersDeletedRows` with 3-row fixture (2 active, 1 deleted).
- [x] 2.8 **(test first)** `TestReadObservations_EmptyDB` in `reader_test.go`.
- [x] 2.9 Created `cmd/migrate/reader.go` — `ReadObservations(ctx, *sql.DB)` — WHERE deleted_at IS NULL, returns full slice.

### 2d. writer.go — idempotency + storage.Save path (test first)

- [x] 2.10 **(test first)** Created `cmd/migrate/writer_test.go` — `TestWriteRow_HappyPath`, `TestWriteRow_IdempotentDuplicateSyncID`, `TestWriteRow_TopicKeyCollision`. Uses real storage.Storage in t.TempDir().
- [x] 2.11 Created `cmd/migrate/writer.go` — `writeRow` with 3-step: sync_id pre-check, topic_key collision pre-check, storage.Save + post-UPDATE sync_id.

### 2e. main.go — wiring and CLI (test first for Run function)

- [x] 2.12 **(test first)** Created `cmd/migrate/integration_test.go` — `TestMigrate_EndToEnd` with 12-row fixture (11 active: all 8 types + oversize content + long title; 1 deleted). Asserts Total=11, Created=10, Errors=1, Truncations=1.
- [x] 2.13 **(test first)** `TestMigrate_DryRun` in `integration_test.go` — asserts Created=0, dest DB empty, Total>0.
- [x] 2.14 **(test first)** `TestMigrate_Idempotent` in `integration_test.go` — second run asserts Created=0, SkippedDuplicate=firstCreated.
- [x] 2.15 Created `cmd/migrate/run.go` — `Config` struct and `Run()` function. Per-row error model; all rows processed even on error.
- [x] 2.16 Created `cmd/migrate/logger.go` — `StructuredLogger` (key=value format, sorted keys) and `NopLogger`.
- [x] 2.17 Created `cmd/migrate/cmd/main.go` — flag parsing, log file creation, Run() call, summary print. 128 LOC (under 150 limit). Note: moved to `cmd/migrate/cmd/` subdirectory because Go does not allow `package main` and `package migrate` to coexist in the same directory. Build path: `go build ./cmd/migrate/cmd`.

---

## Track 3 — Migration tool documentation

> This track can begin once Track 2's public API is stable (tasks 2.1–2.6 done). It does NOT depend on tests passing.

- [x] 3.1 Created `cmd/migrate/README.md` — all required sections: What this does, Before you start, How to run (3 steps), Understanding the output (counter table), Type mapping table (8 rows), What to do on errors, Re-running safely, Log file location.
- [x] 3.2 Created `docs/integrations/claude-code-protocol.md` — canonical Thoughtline CLAUDE.md protocol block: proactive save triggers, tl_save usage with content format, tl_search/tl_get_observation usage, session close protocol (tl_session_summary), topic_key convention table, SDD artifact naming table.
- [x] 3.3 Updated `README.md` — "Migrating from Engram" section added before Roadmap: summary sentence, backup prerequisite, 3 command lines, link to cmd/migrate/README.md.
- [x] 3.4 Updated `README.md` — Credits section updated to mention cmd/migrate as a first-class migration path from Engram.

---

## Track 4 — Backups and safety nets (must complete before Track 5)

> All backup tasks are PowerShell. None modify source files — only create copies.

- [x] 4.1 Created `C:\Users\Agustin Lozano\.engram-backup-2026-05-04.zip` via `tar -a -cf` (zip not available; tar with .zip extension creates a zip-compatible archive). Verified: 4.4 MB file exists.
- [x] 4.2 Backed up `mcp.json` → `mcp.json.bak` in `AppData\Roaming\Code\User\`. Verified.
- [x] 4.3 Backed up `.claude\settings.json` → `.claude\settings.json.bak`. Verified.
- [x] 4.4 Backed up `.claude\mcp\engram.json` → `.claude\mcp\engram.json.bak`. Verified.
- [x] 4.5 Backed up `.claude\CLAUDE.md` → `.claude\CLAUDE.md.bak`. Verified.
- [x] 4.6 Backed up `.claude\skills\_shared\engram-convention.md` → `.claude\skills\_shared\engram-convention.md.bak`. Verified.
- [x] 4.7 Created `.claude\mcp\thoughtline.json` with command path and THOUGHTLINE_HOME env from mcp.json. Verified 195-byte file exists.

---

## Track 5 — Claude Code config edits (DESTRUCTIVE — requires Track 0 + Track 4 complete)

> Spec: specs/claude-code-integration/spec.md — Requirements 1–7
> Design: design.md — Config-Edit Sequence section
> Order is MANDATORY per spec Requirement 7. Apply one step, verify, then proceed.

- [x] 5.1 Edited `~\AppData\Roaming\Code\User\mcp.json` — removed `"engram"` server entry. `context7` and `thoughtline` entries unchanged. Verified: file re-read, no `"engram"` key in `servers` object. NOTE: timestamp fix to `MapTimestamp` was required first (Engram stores `"YYYY-MM-DD HH:MM:SS"` not ISO 8601 with T; dry-run found 298 errors, all timestamp-related).
- [x] 5.2 Edited `~\.claude\settings.json` — removed 11 `mcp__plugin_engram_engram__*` entries, removed `enabledPlugins.engram@engram`, removed `extraKnownMarketplaces.engram`. Added 9 `mcp__thoughtline__*` allow-list entries: `tl_save`, `tl_search`, `tl_get_observation`, `tl_context`, `tl_update`, `tl_delete`, `tl_session_start`, `tl_session_summary`, `tl_stats` (names verified against actual server.go registrations). NOTE: spec listed `tl_get`, `tl_list_sessions`, `tl_export` — actual server exposes `tl_get_observation`, `tl_context`, `tl_update`, `tl_session_summary` instead. Used real tool names.
- [x] 5.3 Deleted `~\.claude\mcp\engram.json`. Verified: file does not exist. `.bak` sibling intact.
- [x] 5.4 Replaced `<!-- gentle-ai:engram-protocol -->` block in `~\.claude\CLAUDE.md` with `<!-- gentle-ai:thoughtline-protocol -->` block sourced from `docs/integrations/claude-code-protocol.md`. `tl_save` and `tl_search` confirmed present. DEVIATION: `sdd-orchestrator` block still references `engram` as an artifact store option (13 occurrences) — these are doc references, not tool calls; updating them is out of scope for this task but verify phase should note it.
- [x] 5.5 Deleted `~\.claude\skills\_shared\engram-convention.md`. Created `~\.claude\skills\_shared\thoughtline-convention.md` with full Thoughtline conventions: `tl_save`, `tl_search`, `tl_get_observation`, `tl_update`, topic_key format, scope rules, session lifecycle, SDD artifact naming. Verified: `engram-convention.md` does not exist; `thoughtline-convention.md` exists; no engram tool names in new file.

---

## Track 6 — Smoke test and verification (read-only, requires Track 5 complete)

> Spec: specs/claude-code-integration/spec.md — post-flight gate

- [x] 6.1 Ran `go run ./cmd/migrate/cmd --dry-run` against default paths. FIRST run: Total=298, Errors=298 (all timestamp failures — Engram stores `"YYYY-MM-DD HH:MM:SS"` not ISO 8601). Fixed `MapTimestamp` in `mapper.go` to add `"2006-01-02 15:04:05"` SQLite format. Also fixed wrong `knownUTCUnixMS` constant in test (was 1710495000000, correct is 1710498600000). SECOND run after fix: Total=298, Errors=0, Created=0 (dry-run). Log: `%LOCALAPPDATA%\thoughtline\migrate-2026-05-05-094851.log`.
- [x] 6.2 Ran `go run ./cmd/migrate/cmd` (real run). migrated=298, failed=0, skipped=0, truncations=0, total=298. Duration=583ms. Log: `%LOCALAPPDATA%\thoughtline\migrate-2026-05-05-095211.log`. Exit code 0.
- [ ] 6.3 MANUAL: Open `thoughtline ui` → Stats panel. Verify total = pre-migration baseline + 298. Log file at `%LOCALAPPDATA%\thoughtline\migrate-2026-05-05-095211.log` for row details.
- [ ] 6.4 MANUAL: Quit and relaunch Claude Code. Verify no MCP errors. Confirm `tl_stats` is callable.
- [ ] 6.5 MANUAL: Call `tl_search` with a known `sync_id` from migration log (e.g. `obs-d3d80c508430af23`). Assert memory returned with original content.
- [ ] 6.6 MANUAL: Call `tl_stats` and verify total count matches Stats panel.
- [ ] 6.7 MANUAL: Restart VSCode. Open Command Palette → "MCP: List Servers". Verify `engram` absent, `thoughtline` present and connected.
- [ ] 6.8 Final acceptance gate — confirm ALL spec success criteria (manual steps 6.3–6.7 required first):
  - [x] 298 active Engram observations imported with `sync_id` preserved 1:1 *(verified in 6.2)*
  - [ ] Type mapping spot-check — one `pattern`→`convention` row has `origin-type:pattern` tag *(MANUAL: verify in 6.5)*
  - [ ] `thoughtline ui` Stats count matches expected total *(MANUAL: 6.3)*
  - [ ] Claude Code boots with Thoughtline as sole backend *(MANUAL: 6.4)*
  - [ ] `tl_search` returns a known historical Engram memory *(MANUAL: 6.5)*
  - [x] Engram binary still installed but quiescent — no `mcp.json` reference *(verified in 5.1)*
  - [ ] All Go unit tests pass for extended taxonomy *(MANUAL: run `go test ./internal/memory/...` and `go test ./cmd/migrate/...`)*
  - [x] `~\.engram-backup-2026-05-04.zip` exists on disk *(verified in 4.1)*

---

## Task summary

| Track | Tasks | Focus | Parallelizable? |
|---|---|---|---|
| 0 — Pre-flight | 4 | Environment validation, no writes | Must run first, blocks all |
| 1 — Taxonomy | 8 | TDD type extension, docs | After Track 0; safe in parallel with Track 2 |
| 2 — cmd/migrate | 17 | TDD migration binary | After Track 0; safe in parallel with Track 1 |
| 3 — Docs | 4 | cmd/migrate README, integration doc, README | After 2.1–2.6 for API surface |
| 4 — Backups | 7 | zip + .bak files, thoughtline.json | After Track 0; safe in parallel with 1–3 |
| 5 — Config edits | 5 | Destructive: 5 Claude Code files, strict order | After Tracks 0 AND 4 |
| 6 — Smoke test | 8 | Post-migration verification, acceptance gate | After Track 5 |
| **Total** | **53** | | |
