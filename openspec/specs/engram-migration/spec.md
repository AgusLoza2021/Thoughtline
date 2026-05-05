# Engram Migration Specification

> Change: `adopt-thoughtline-replace-engram`
> Status: proposed
> Operation: ADDED (new `cmd/migrate` binary — no prior spec exists)

## Capability Summary

Defines the contract for `cmd/migrate`: the standalone Go binary that reads active Engram observations from `~\.engram\engram.db` (read-only) and writes them into Thoughtline via `storage.Save()`, applying the column mapping and type coercion rules from `exploration.md`. The spec covers input contract, output contract, type mapping, idempotency, error handling, and per-field transformation rules. Implementation details (flags, log format) are deferred to `design.md`.

## Requirements

### Requirement 1: Input Contract — Engram DB Read-Only

The binary MUST open `~\.engram\engram.db` in read-only mode with WAL awareness (URI param `mode=ro&_journal_mode=WAL`). It MUST NOT lock, write, or checkpoint the Engram DB at any point. It MUST select only rows where `deleted_at IS NULL`.

#### Scenario: Soft-deleted row skipped

- GIVEN an Engram DB containing a row with a non-null `deleted_at`
- WHEN the migration runs
- THEN that row is not written to Thoughtline and does not appear in the success count

#### Scenario: WAL-mode DB opened without write lock

- GIVEN `engram.db-shm` and `engram.db-wal` sidecar files exist (Engram process was or is running)
- WHEN the migration opens `engram.db`
- THEN it reads all committed rows without error and without blocking the Engram process

---

### Requirement 2: Output Contract — Storage.Save() Path

The binary MUST write every selected row through Thoughtline's `storage.Save()` function, not via raw SQL. This guarantees FTS5 index consistency, normalized-hash computation, and domain validation on every row.

#### Scenario: FTS5 index populated

- GIVEN 5 active Engram rows that migrate successfully
- WHEN the migration completes
- THEN `tl_search` can find those rows by title keywords

---

### Requirement 3: Column Mapping

Each Engram `observations` row MUST be transformed to a `memory.Memory` struct according to the following table before being passed to `storage.Save()`:

| Engram column | Thoughtline field | Rule |
|---|---|---|
| `sync_id` | `SyncID` | COPY verbatim |
| `session_id` | `SessionID` | SET `""` (Engram IDs are not UUIDv7; FK violation risk) |
| `type` | `Type` | MAP via type mapping table (Requirement 4) |
| `title` | `Title` | COPY; truncate at rune boundary to 200 chars if longer — log truncation |
| `content` | `Content` | COPY; if `len(content) > 64 KiB`, skip row and log — do NOT truncate content |
| `project` | `Project` | COPY; normalize to lowercase |
| `scope` | `Scope` | COPY; validate `"project"` or `"personal"` — reject row if neither |
| `topic_key` | `TopicKey` | COPY if valid per Thoughtline regex; set `""` if invalid |
| `normalized_hash` | `NormalizedHash` | RECOMPUTE via Thoughtline's hash function (do not copy Engram's value) |
| `revision_count` | `RevisionCount` | COPY |
| `created_at` (TEXT ISO UTC) | `CreatedAt` | CONVERT: `time.Parse(time.RFC3339, ...)` → `time.Time`; use UTC |
| `updated_at` (TEXT ISO UTC) | `UpdatedAt` | Same conversion |
| `tool_name` | — | DROP |
| `duplicate_count` | — | DROP |
| `last_seen_at` | — | DROP |
| `embedding*` | — | COPY (both NULL in practice) |

#### Scenario: Happy path — 5 rows including each Engram type

- GIVEN an Engram DB with one row each for types `bugfix`, `preference`, `decision`, `architecture`, `pattern`
- WHEN the migration runs
- THEN all 5 rows are written to Thoughtline; types map to `bugfix`, `preference`, `decision`, `architecture`, `convention` respectively; the `pattern` row carries tag `origin-type:pattern`

#### Scenario: Title over 200 chars truncated and logged

- GIVEN an Engram row whose `title` is 250 characters
- WHEN the migration processes that row
- THEN the row is saved with `Title` truncated to 200 characters at a valid rune boundary; a log entry records the original `sync_id` and truncated length; the migration continues

#### Scenario: Content over 64 KiB rejected and logged

- GIVEN an Engram row whose `content` exceeds 64 KiB
- WHEN the migration processes that row
- THEN the row is NOT written to Thoughtline; a log entry records the `sync_id` and the actual byte count; the failure count increments; the migration continues to the next row

---

### Requirement 4: Type Mapping Rules

The binary MUST apply the following type mapping to every row. Types not in the Engram set must be treated as unknown and coerced to `convention` with tag `origin-type:<raw-value>`.

| Engram type | Thoughtline type | Tag added |
|---|---|---|
| `bugfix` | `bugfix` | none |
| `preference` | `preference` | none (scope forced to `personal`) |
| `decision` | `decision` | none |
| `architecture` | `architecture` | none |
| `pattern` | `convention` | `origin-type:pattern` |
| `config` | `convention` | `origin-type:config` |
| `discovery` | `convention` | `origin-type:discovery` |
| `manual` | `convention` | `origin-type:manual` |
| *(unknown)* | `convention` | `origin-type:<raw-value>` |

The `preference` type MUST also have its `Scope` forced to `personal` regardless of the value in the Engram row, because Thoughtline's `Validate()` enforces `ErrPreferenceMustBePersonal`.

#### Scenario: Preference row scope forced to personal

- GIVEN an Engram `preference` row with `scope = "project"`
- WHEN the migration maps the row
- THEN the resulting `memory.Memory` has `Scope = "personal"` before being passed to `storage.Save()`

---

### Requirement 5: Idempotency

Re-running the migration against the same Engram DB MUST NOT create duplicate rows in Thoughtline. The binary MUST detect existing rows by `sync_id` before writing. If a row with the same `sync_id` already exists, the binary MUST skip it and increment a "skipped (already exists)" counter. It MUST NOT overwrite or update the existing row.

#### Scenario: Empty Engram DB — 0 rows migrated

- GIVEN an Engram DB with no rows (or all rows soft-deleted)
- WHEN the migration runs
- THEN 0 rows are written; the final report shows `migrated: 0, skipped: 0, failed: 0`

#### Scenario: Idempotent re-run — 0 new rows on second execution

- GIVEN a migration has already run successfully and imported N rows
- WHEN the migration runs again against the same Engram DB
- THEN 0 new rows are written; the report shows `migrated: 0, skipped: N, failed: 0`

---

### Requirement 6: Per-Row Error Handling and Summary Report

Per-row failures (type mapping error, validation rejection, storage error) MUST be logged individually with `sync_id` and error message. A single row failure MUST NOT abort the migration. The binary MUST print a final summary to stdout with counts: `migrated`, `skipped (already exists)`, and `failed`. Exit code MUST be `0` if `failed == 0`, non-zero otherwise.

#### Scenario: One invalid row does not abort migration

- GIVEN an Engram DB with 10 rows where 1 row has content exceeding 64 KiB
- WHEN the migration runs
- THEN 9 rows are written successfully, 1 is logged as failed, and the binary exits with a non-zero code; the summary shows `migrated: 9, skipped: 0, failed: 1`
