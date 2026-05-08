# Proposal: brain-foundation

## Why

Thoughtline today treats memories as a flat collection scoped by a free-form `project` string. That model has carried us far, but the next chapter — **brAIn**, a visible cognitive editor that lets a human SEE and SHAPE the graph behind their assistant — needs a backend that understands more than rows of text.

Three forces converge:

1. **brAIn needs a graph-aware backend.** brAIn (a separate desktop client: Tauri + React + Cytoscape) renders memory as a living graph: nodes connect, ideas supersede each other, contradictions surface visually. That requires first-class typed links, not implicit semantic neighborhood.

2. **Multi-brain unlocks synthetic cognition.** A `brain` should be a first-class entity, not a string. Once it is, we can spin up *synthetic* brains — for simulation, regression tests, onboarding demos, A/B experiments on ranking — without contaminating the user's real cognitive state. `kind: real | synthetic | sandbox` makes the trust boundary explicit.

3. **Different brains deserve different personalities.** A research brain decays slowly and prefers depth; a daily-journal brain decays fast and prefers recency. Per-brain configuration (Ranking, Decay, Graph, Simulation) over a global default lets each brain behave like itself, not a clone.

This change is the foundation. It does NOT ship brAIn, new MCP tools, decay formulas, or a synthetic generator — it ships the *substrate* those features will stand on. Thoughtline core stays **headless**: brAIn will consume Thoughtline as a dependency, never the other way around.

## What Changes

### Schema (v3 → v4, additive)

- **NEW** `brains` — first-class entity. Columns: `id`, `name`, `kind` (real | synthetic | sandbox), `config_json` (partial overrides), `created_at`, `updated_at`.
- **NEW** `global_config` — singleton row holding the default Ranking / Decay / Graph / Simulation config. Per-brain `config_json` overlays this.
- **NEW** `memory_links` — typed graph edges. Columns: `from_id`, `to_id`, `relation` (closed enum), `weight`, `source`, `brain_id` (FK). Brain-scoped — links never cross brains.
- **MODIFIED** `memories` — add `brain_id` column (FK to `brains.id`). Existing `project` column is kept as a **mirror** for the transition; v5 will drop it (out of scope for this change).
- **Backfill**: one row in `brains` per `DISTINCT project` value, `kind = real`. Memories backfill `brain_id` from their `project`. Memories with NULL/empty `project` fail the migration loudly — we do not silently bucket.

Closed relation enum: `supersedes`, `contradicts`, `refines`, `depends_on`, `references`, `related`, `derived_from`.

### New Go packages

- `internal/brain` — CRUD for brains, kind validation, name uniqueness.
- `internal/config` — global_config + per-brain resolution. Tolerant JSON parsing (unknown fields warn, never panic). Versioned config schema.
- `internal/links` — typed edge operations, relation enum guard, brain-scoped queries.
- `internal/events` — in-process events bus (publish/subscribe in the same process). No WebSocket, no IPC. The bus exists so future code can react to "memory created" / "link added" without coupling.

### Modified packages

- `internal/storage` — schema v4 migration, all query sites extended to require `brainID` at the API boundary. This is the **isolation contract**: a caller cannot accidentally query across brains because there is no signature that lets them.

### Tests

- **Isolation invariant suite**: for every storage method that touches memories or links, prove that data from brain A is invisible when querying brain B. Table-driven, exhaustive, brutal.
- Migration tests with seeded v3 fixtures.
- Config resolution tests (global only, override only, both, malformed JSON).

## Impact

### Affected specs (new capabilities)

- `brain-domain` — what a brain IS, kinds, lifecycle, name uniqueness, isolation guarantee.
- `memory-graph` — typed links, closed relation enum, brain-scoping rule, weight semantics.
- `cognitive-config` — two-level resolution (global + per-brain), tolerant parsing, versioning.
- `event-bus` — in-process publish/subscribe contract, event taxonomy seed.

Existing specs that mention `project` semantically may need a delta to clarify that `project` is now a transitional mirror of `brain.name` and will be removed in v5.

### Affected code

- `internal/storage/*` — schema, every query site, every method signature touching memories or links.
- **NEW** `internal/brain/`, `internal/config/`, `internal/links/`, `internal/events/`.
- **NO changes** to `cmd/thoughtline/` MCP server in this phase — only a compatibility shim so existing tools keep working against the new schema by resolving `project` → `brain_id` internally.
- **NO changes** to the TUI dashboard.

### Migration

- v3 → v4 is purely additive: new tables + new column with backfill. No destructive operations.
- v5 (future, separate change) will drop `memories.project` once all read paths use `brain_id`.

## Out of Scope (Phase 0)

Explicitly NOT in this change — these come later, on top of the foundation:

- WebSocket / external event transport (only in-process bus ships).
- New MCP tools for brains, links, or config (compat shim only — existing tools keep working).
- Synthetic data generator (the `kind=synthetic` value exists; the generator does not).
- brAIn desktop UI (separate product, separate repo).
- Importance and decay formula wiring (config sections exist; the math that consumes them lands later).
- Any change to ranking behavior visible to MCP callers.

## Rollback Plan

Schema v4 is purely additive, so rollback is mechanical:

```sql
DROP TABLE memory_links;
DROP TABLE brains;
DROP TABLE global_config;
ALTER TABLE memories DROP COLUMN brain_id;
```

Plus reverting the Go code to the previous tag. **No data loss** because `memories.project` remains the source of truth throughout the transition — `brain_id` is a derived mirror until v5.

## Risks

- **Cross-brain leak via a forgotten `WHERE brain_id` clause.** A single missed predicate would silently expose another brain's memories. *Mitigation*: `brainID` is a **required parameter** at the storage API boundary — the type system makes the mistake unrepresentable. Backed by an exhaustive isolation invariant test suite that runs against every query method.

- **Backfill ambiguity for memories with no project.** Silently bucketing them into a "default" brain would corrupt the trust boundary forever. *Mitigation*: migration **fails loudly** with the offending memory IDs and refuses to proceed. Operator must clean the data first.

- **Config drift between `brains.config_json` and the Go struct.** As config evolves, old rows will have unknown fields and new code will expect new fields. *Mitigation*: versioned config schema (`version` field on every config blob) + tolerant parsing (unknown fields warn to log, never panic; missing fields fall back to global, then to compiled default).

- **Required `brainID` is a breaking API change inside the codebase.** Every storage call site must be touched in this change. *Mitigation*: this is intentional — the compiler becomes the enforcer of isolation. The blast radius is contained to `internal/storage` callers, all in our own tree.
