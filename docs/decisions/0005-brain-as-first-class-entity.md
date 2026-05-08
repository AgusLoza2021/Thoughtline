# ADR 0005 — Brain as First-Class Entity

**Status**: Accepted
**Date**: 2026-05-07

---

## Context

Thoughtline's original schema (v1–v3) used a plain `project` text column on
`memories` as the only isolation boundary. Every storage operation filtered by
project string. This worked for a single developer with a small number of
projects, but created several structural problems:

- No typed metadata per project (kind, config overrides, description).
- No lifecycle management: projects could not be archived or renamed safely.
- Stats and search operated on ad-hoc string comparison — vulnerable to
  typos producing orphaned rows.
- No hooks for multi-brain simulation or synthetic workload testing.
- The `project` column could not enforce referential integrity (no FK target).

The `brain-foundation` change introduced `brains` as a first-class table and
migrated all operations to be brain-scoped.

---

## Decision

### D1 — Expand-contract migration (v4)

Schema v4 adds `memories.brain_id INTEGER REFERENCES brains(id)` alongside the
existing `project` column. Both columns coexist. The v4 migration backfills
`brain_id` from `project` (one brain row per distinct project slug, `kind=real`).
The `project` column is kept for denorm compat and will be dropped in a future
v5 migration once all callers are brain-aware.

### D2 — Brain kind enum

Three kinds: `real` (production data), `synthetic` (AI-generated or simulated),
`sandbox` (ephemeral test workloads). Kind is a CHECK constraint on the `brains`
table. The distinction drives delete policy: real brains are archived, not
hard-deleted; synthetic/sandbox brains support hard delete with full cascade.

### D3 — Brain-scoped query API

Every storage method that reads or writes memories takes `brainID int64` as its
first argument. Cross-brain access returns `ErrMemoryNotFound` (not a permission
error — existence is not leaked). The server layer resolves `project` → `brainID`
via `ResolveBrainID` / `ResolveOrCreateBrainID` before calling storage.

### D4 — Config overrides

Two-level config: a singleton `global_config` row holds system-wide defaults
(seeded with `config.DefaultGlobalJSON()` on first migration); each brain has
`config_json` with field overrides. `config.Effective(global, brainOverride)`
deep-merges them at read time. Unknown keys are tolerated with a `slog.Warn`.

### D5 — In-process event bus

`internal/events.Bus` is a typed pub/sub with per-brain subscription fanout and
a non-blocking deliver (slow subscribers drop events with an atomic counter).
Storage, links, and brain packages emit events after successful commits. Callers
wire the bus via `SetBus(b)` — nil bus is always safe.

### D6 — Memory link isolation

`memory_links` has `brain_id NOT NULL REFERENCES brains(id) ON DELETE CASCADE`.
All link queries include `brain_id = ?`. Cross-brain link creation returns
`ErrCrossBrainLink`. The cascade on `brain_id` fires when a brain row is
deleted — `from_id`/`to_id` cascades fire on memory hard-delete.

### D7 — Stats remain cross-brain aggregate

`Stats()` and `tl_stats` query by project string rather than brainID. This is
intentional: the dashboard and MCP tool serve a global overview. Per-brain stats
breakdown is deferred to a future change when multi-brain UI is needed.

### D8 — `-race` flag unavailable on Windows (no CGO)

`modernc.org/sqlite` is a pure-Go SQLite driver with no CGO. The Go race
detector requires CGO on Windows. All concurrent tests pass without `-race`
locally; CI on Linux should add `-race` to the test matrix.

---

## Consequences

### Positive

- Every memory, link, and event is now provably scoped to a single brain.
- Brain lifecycle (archive, unarchive, config override) is a first-class API.
- Isolation is enforced in storage — the server layer cannot accidentally
  cross brains.
- The event bus enables future real-time features (live dashboard, webhooks)
  without polling.

### Negative / trade-offs

- Storage API is a breaking change: all callers must pass `brainID`.
- The `project` column is now denorm — it must be kept in sync with
  `brains.slug` by convention until v5 drops it.
- `memories.brain_id` has no `ON DELETE CASCADE` (intentional — memories
  outlive their brain row for audit purposes). Hard-deleting a real brain
  requires explicit opt-in via kind check.

---

## Alternatives considered

**Replace `project` with `brain_id` directly (no backfill period)**
Rejected: too disruptive for existing databases. The expand-contract pattern
lets v3 databases migrate in place without data loss.

**Parallel tables (memories_v2, links_v2)**
Rejected: ambiguity about which table is authoritative; doubles migration
complexity; ruins query plan caching.

**Single global brain (no multi-brain)**
Rejected: synthetic/sandbox kind is required for simulation tests and the
future smarts milestone (M6) needs isolated vector spaces per brain.

---

## References

- `openspec/changes/brain-foundation/proposal.md` — original proposal
- `openspec/changes/brain-foundation/design.md` — detailed design with ADR-1 through ADR-8
- `internal/storage/schema.go` — schemaV4SQL, schemaV4MemoriesBrainIndexSQL
- `internal/config/` — Config, DefaultGlobal, Effective
- `internal/brain/` — Store, Kind, ValidateSlug
- `internal/links/` — Store, ErrCrossBrainLink, ErrSelfLoop
- `internal/events/` — Bus, Event interface, concrete event types
- `internal/storage/isolation_test.go` — isolation invariant test suite (Phase 7)
