# Delta for Engram Migration

> Change: `brain-foundation`
> Base spec: `openspec/specs/engram-migration/spec.md`

## ADDED Requirements

### Requirement: project Field Is a Transitional Mirror

After the v3→v4 migration, `memories.project` MUST continue to be populated by the engram
migration binary exactly as before (Requirements 3 and 5 of the base spec are unchanged).
The field is now a **transitional mirror** of the brain's slug — it will be dropped in v5.

The engram migration binary MUST also backfill `memories.brain_id` for each row it writes,
resolving the brain by matching the normalized `project` value to `brains.slug`. If no
matching brain exists for a given `project` value, the migration MUST fail that row (increment
`failed` counter, log the sync_id and project value) and continue to the next row.

#### Scenario: Migrated row gets brain_id resolved from project slug

- GIVEN a brain with `slug = "my-project"` exists in v4 DB
- AND an Engram row has `project = "my-project"`
- WHEN the migration processes that row
- THEN the written memory has both `project = "my-project"` AND `brain_id` set to the matching brain's id

#### Scenario: No matching brain — row fails with log

- GIVEN no brain with `slug = "orphan-project"` exists in the v4 DB
- AND an Engram row has `project = "orphan-project"`
- WHEN the migration processes that row
- THEN the row is NOT written; the failed counter increments; the log includes the sync_id and project value
