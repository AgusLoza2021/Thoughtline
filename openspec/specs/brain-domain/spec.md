# Brain Domain Specification

> Change: `brain-foundation`
> Status: shipped
> Operation: ADDED (new capability — no prior spec exists)

## Purpose

Defines what a `Brain` is, its valid kinds, slug rules, lifecycle (including archival), and the
isolation guarantee that every memory belongs to exactly one brain. Also covers the v3→v4
schema migration that backfills brains from existing `memories.project` values.

---

## Requirements

### Requirement: Brain Entity Shape

A `Brain` record MUST have the following fields:

| Field | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `slug` | TEXT | NOT NULL, UNIQUE among non-archived brains |
| `display_name` | TEXT | NOT NULL |
| `kind` | TEXT | NOT NULL — CHECK(`real`, `synthetic`, `sandbox`) |
| `description` | TEXT | NULLABLE |
| `config_json` | TEXT | NOT NULL DEFAULT `{}` — partial JSON overrides only |
| `created_at` | INTEGER | NOT NULL — unix epoch ms |
| `updated_at` | INTEGER | NOT NULL — unix epoch ms |
| `archived_at` | INTEGER | NULLABLE — NULL means active |

The database MUST enforce the `kind` CHECK constraint at the storage layer, not only in application code.

#### Scenario: Creating a brain with a valid kind succeeds

- GIVEN a `Brain` with `kind = "real"` and a valid slug
- WHEN the brain is saved
- THEN the row is persisted and the returned brain has an assigned `id`

#### Scenario: Creating a brain with an invalid kind is rejected

- GIVEN a `Brain` with `kind = "experimental"` (not in the closed set)
- WHEN the brain is saved
- THEN storage returns an error; no row is inserted

---

### Requirement: Slug Validation

A brain's `slug` MUST be a string that satisfies all of the following:

- Only lowercase ASCII letters (`a-z`), digits (`0-9`), and hyphens (`-`)
- Length between 1 and 64 characters (inclusive)
- Does NOT start or end with a hyphen

The system MUST reject any slug that violates these rules before attempting a database write.

#### Scenario: Valid slug accepted

- GIVEN `slug = "my-research-brain-2026"`
- WHEN the brain is created
- THEN creation succeeds

#### Scenario: Slug with uppercase rejected

- GIVEN `slug = "MyBrain"`
- WHEN the brain is created
- THEN an error is returned before any DB write; the error message references the invalid slug

#### Scenario: Slug longer than 64 chars rejected

- GIVEN a slug of 65 characters composed of valid chars
- WHEN the brain is created
- THEN an error is returned; no row is inserted

---

### Requirement: Slug Uniqueness Among Active Brains

The system MUST enforce that no two non-archived brains share the same slug. A slug MAY be reused once the original brain is archived.

#### Scenario: Duplicate slug on active brains fails

- GIVEN an active brain with `slug = "project-alpha"` already exists
- WHEN a second brain with `slug = "project-alpha"` is created
- THEN the operation fails with a uniqueness error; the second brain is not inserted

#### Scenario: Slug reuse after archival is allowed

- GIVEN a brain with `slug = "project-alpha"` has been archived (archived_at IS NOT NULL)
- WHEN a new brain with `slug = "project-alpha"` is created
- THEN creation succeeds; the new brain is active; the archived brain is unchanged

---

### Requirement: Brain Archival (Soft Delete)

A brain MUST be archivable by setting `archived_at` to the current timestamp. Hard deletion of a brain MUST NOT be performed by default. Archived brains MUST be excluded from default list queries. Memories belonging to an archived brain MUST remain accessible when queried directly by their `id` or `brain_id`.

#### Scenario: Archiving a brain hides it from default listings

- GIVEN an active brain `B` with 3 memories
- WHEN `B` is archived
- THEN a call to `ListBrains()` (default, no filter) does NOT include `B`
- AND the 3 memories are still retrievable by `brain_id = B.id`

#### Scenario: Archived brain is retrievable with explicit filter

- GIVEN brain `B` is archived
- WHEN `ListBrains(includeArchived: true)` is called
- THEN `B` appears in the result with a non-null `archived_at`

---

### Requirement: Memory Brain Membership

Every memory row MUST have a `brain_id` column that is a non-null foreign key to `brains.id`. A memory MUST belong to exactly one brain. The storage API MUST require `brain_id` as a parameter on every method that reads or writes memories — the signature MUST NOT allow a caller to omit it.

#### Scenario: Memory save requires brain_id

- GIVEN a valid `Memory` struct without `BrainID` set (zero value)
- WHEN `storage.Save()` is called
- THEN an error is returned; no row is inserted

#### Scenario: Memory query is scoped to its brain

- GIVEN brain A has memory M1 and brain B has memory M2
- WHEN `storage.Get(brainID=A)` is called for M2's id
- THEN the result is "not found" — M2 is invisible in brain A's scope

---

### Requirement: v3→v4 Migration — Brain Backfill

The v3→v4 migration MUST:

1. Create one `brains` row per `DISTINCT project` value in `memories`, with `kind = "real"` and `slug` derived from the project string (normalized to slug rules).
2. Backfill `memories.brain_id` for every row from its `project` value.
3. FAIL LOUDLY (non-zero exit, list offending memory ids) if any memory has a NULL or empty `project` value. The migration MUST NOT silently assign these to a default brain.

#### Scenario: Backfill creates one brain per distinct project

- GIVEN a v3 DB with memories across 3 distinct project values: `"alpha"`, `"beta"`, `"gamma"`
- WHEN the v3→v4 migration runs
- THEN exactly 3 rows exist in `brains`, each with `kind = "real"` and matching slugs; every memory has a non-null `brain_id`

#### Scenario: Migration fails loudly on empty project

- GIVEN a v3 DB where memory id=42 has `project = ""`
- WHEN the v3→v4 migration runs
- THEN the migration aborts with a non-zero exit code and logs memory id 42; no partial backfill is committed

#### Scenario: Migration is idempotent

- GIVEN the v3→v4 migration has already been applied
- WHEN the migration runs again
- THEN no duplicate brains are created; no error occurs; memory `brain_id` values are unchanged
