# Tasks: brain-foundation

> Strict TDD — every implementation task is preceded by a failing test task in the same numbered group.
> Test runner: `go test ./...` | `-race` required for Phase 6 | No `go build`.

---

## Phase 1 — Schema v4 + Migration

- [x] 1.1 Write failing test: `TestMigrateV4_EmptyProject` in `internal/storage/migration_v4_test.go` — seed a v3 DB with one memory where `project = ""`, call `Open`, assert returned error contains the offending memory id and no partial write is committed
- [x] 1.2 Implement `migrateV4` in `internal/storage/storage.go`: transaction-wrapped DDL exec, `columnExists` helper, `ALTER TABLE memories ADD COLUMN brain_id`, orphan-check step with `collectOrphanIDs`; bump `currentSchemaVersion = 4` in `schema.go`
- [x] 1.3 Write failing test: `TestMigrateV4_Backfill` — seed v3 DB with memories across 3 distinct projects, call `Open`, assert exactly 3 rows in `brains` (kind='real'), every memory has non-null `brain_id` matching its project slug
- [x] 1.4 Implement backfill steps in `migrateV4`: `INSERT OR IGNORE INTO brains SELECT DISTINCT project…`, `UPDATE memories SET brain_id = (SELECT id FROM brains WHERE slug = project)`; add global_config singleton seed via `config.DefaultGlobalJSON()`
- [x] 1.5 Write failing test: `TestMigrateV4_Idempotent` — run `Open` twice on the same DB, assert no duplicate brains, no error, memory `brain_id` unchanged
- [x] 1.6 Implement idempotency: `INSERT OR IGNORE` in all backfill steps; `columnExists` guard before `ALTER TABLE`; `INSERT OR IGNORE INTO schema_version(4, …)`
- [x] 1.7 Append v4 DDL to `schemaSQL` in `internal/storage/schema.go`: `brains`, `global_config`, `memory_links` tables + all indexes; `CREATE TABLE IF NOT EXISTS` throughout

---

## Phase 2 — `internal/config` Package

- [x] 2.1 Write failing test: `TestConfig_Defaults` in `internal/config/config_test.go` — call `DefaultGlobal()`, assert `Ranking.BM25 = 0.40`, `Decay.HalfLifeDays = 30`, `Simulation.Enabled = false`
- [x] 2.2 Create `internal/config/config.go` with `Config`, `Ranking`, `Decay`, `Graph`, `Simulation` structs; `internal/config/defaults.go` with `DefaultGlobal() Config` and `DefaultGlobalJSON() string`
- [x] 2.3 Write failing test: `TestEffective_EmptyOverride` — `Effective(global, "{}")` returns global unchanged
- [x] 2.4 Write failing test: `TestEffective_PartialOverride` — override with `{"decay":{"half_life_days":1}}`, assert `Decay.HalfLifeDays=1` and `Ranking.BM25=0.40` unchanged
- [x] 2.5 Implement `Effective(global Config, override json.RawMessage) Config` in `internal/config/resolve.go` using deep-merge map logic (patch wins on scalars, nested objects merge recursively)
- [x] 2.6 Write failing test: `TestEffective_UnknownField` — override with `{"experimental":{"foo":true}}`, assert no error returned and a warning is logged via `slog`
- [x] 2.7 Implement tolerant parsing in `resolve.go`: unmarshal into `map[string]any`, log unknown top-level keys via `slog.Warn`, then unmarshal merged map to `Config`
- [x] 2.8 Write failing test: `TestEffective_MalformedJSON` — override with `"not-json"`, assert error is returned with message referencing invalid JSON

---

## Phase 3 — `internal/brain` Package

- [x] 3.1 Write failing test: `TestBrain_SlugValidation` in `internal/brain/brain_test.go` — table-driven: valid slugs pass `ValidateSlug`, invalid (uppercase, leading hyphen, >64 chars) return error with slug in message
- [x] 3.2 Create `internal/brain/types.go`: `Brain` struct, `Kind` type with `KindReal/Synthetic/Sandbox` consts, `ValidateSlug(slug string) error` using regex `^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$|^[a-z0-9]$`
- [x] 3.3 Write failing test: `TestBrain_Create_ValidKind` — create brain with `kind="real"`, assert row persisted with assigned id
- [x] 3.4 Write failing test: `TestBrain_Create_InvalidKind` — create brain with `kind="experimental"`, assert error, no row inserted
- [x] 3.5 Create `internal/brain/brain.go`: `Create(ctx, db, Brain) (Brain, error)`, `GetBySlug(ctx, db, slug string) (Brain, error)`, `GetByID(ctx, db, id int64) (Brain, error)`; validate slug and kind before INSERT
- [x] 3.6 Write failing test: `TestBrain_SlugUniqueness` — insert active brain with slug "alpha", insert second with same slug, assert uniqueness error
- [x] 3.7 Write failing test: `TestBrain_SlugReuseAfterArchive` — archive brain "alpha", create new brain "alpha", assert success
- [x] 3.8 Implement `Archive(ctx, db, id int64) error` in `brain.go`: sets `archived_at = now`; implement `ListActive(ctx, db) ([]Brain, error)` filtering `WHERE archived_at IS NULL`

---

## Phase 4 — Storage API Redesign

- [x] 4.1 Write failing test: `TestStorage_SaveRequiresBrainID` — call new `Save(ctx, 0, m)` (zero brainID), assert `ErrBrainRequired` returned, no row inserted
- [x] 4.2 Add `BrainID int64` to `memory.Memory` in `internal/memory/types.go`; define `ErrBrainRequired`, `ErrBrainMismatch`, `ErrMemoryNotFound` sentinel errors in `internal/storage/storage.go`
- [x] 4.3 Rewrite `Save(ctx, brainID int64, m memory.Memory)` signature in `internal/storage/storage.go` (and any file implementing it); add `brain_id = ?` to INSERT/UPDATE; emit bus event after commit (bus optional — nil-safe)
- [x] 4.4 Write failing test: `TestStorage_SearchScopedToBrain` — save memory to brainA and brainB, search with brainA id, assert only brainA memory returned
- [x] 4.5 Rewrite `Search(ctx, brainID int64, q SearchQuery)` in `internal/storage/search.go`; add `AND brain_id = ?` to all FTS and fallback queries
- [x] 4.6 Write failing test: `TestStorage_RecentScopedToBrain` — save memories to two brains, call `Recent(ctx, brainA, 10)`, assert no brainB rows
- [x] 4.7 Rewrite `Recent(ctx, brainID int64, limit int)` in `internal/storage/recent.go`; add `AND brain_id = ?`
- [x] 4.8 Write failing test: `TestStorage_GetByID_CrossBrain` — save memory to brainB, call `GetByID(ctx, brainA, memID)`, assert `ErrMemoryNotFound`
- [x] 4.9 Rewrite `GetByID(ctx, brainID int64, id int64)` and `UpdateByID(ctx, brainID int64, …)` and `SoftDelete(ctx, brainID int64, id int64)` in their respective files; all include `AND brain_id = ?`
- [x] 4.10 Create `internal/storage/brain_resolve.go`: `brainCache` struct with `sync.RWMutex`; `ResolveBrainID(ctx, slug) (int64, error)` with read-lock fast path, write-lock populate; `InvalidateBrainCache(slug)` method
- [x] 4.11 Write failing test: `TestResolveBrainID_Cache` — resolve same slug twice, assert single DB query (instrument with query counter or verify via sql.Stats)
- [x] 4.12 Write failing test: `TestResolveBrainID_Invalidate` — archive brain, invalidate cache, resolve again, assert error (archived brain not returned)

---

## Phase 5 — `internal/links` Package

- [x] 5.1 Write failing test: `TestLinks_CreateValidLink` in `internal/links/links_test.go` — create memories M1 M2 in same brain, call `Create(ctx, db, brainID, M1, M2, "supersedes")`, assert row persisted with weight=1.0
- [x] 5.2 Create `internal/links/queries.go` with raw SQL constants; create `internal/links/links.go` with `Create(ctx, db, brainID int64, fromID, toID int64, relation string, …) (MemoryLink, error)` including self-loop check and cross-brain validation inside transaction
- [x] 5.3 Write failing test: `TestLinks_CrossBrainRejected` — M1 in brainA, M2 in brainB, call `Create`, assert `ErrCrossBrainLink`
- [x] 5.4 Write failing test: `TestLinks_SelfLoopRejected` — call `Create` with `fromID == toID`, assert `ErrSelfLoop`
- [x] 5.5 Write failing test: `TestLinks_DuplicateTripleRejected` — create (M1→M2,"refines") twice, assert uniqueness error on second
- [x] 5.6 Write failing test: `TestLinks_SamePairDifferentRelation` — create (M1→M2,"refines") then (M1→M2,"related"), assert both persist
- [x] 5.7 Write failing test: `TestLinks_Neighbors` — M1 with 2 outbound + 1 inbound link, call `Neighbors(ctx, db, M1, "")`, assert 3 links; filter by relation returns subset
- [x] 5.8 Implement `Neighbors(ctx, db, memID int64, relation string) ([]MemoryLink, error)` in `links.go`; SQL uses `WHERE (from_id = ? OR to_id = ?)` with optional `AND relation = ?`
- [x] 5.9 Write failing test: `TestLinks_Subgraph` — brainA has 4 memories + 3 links; brainB has 2+1; call `Subgraph(ctx, db, brainA, SubgraphFilters{})`, assert 4 nodes 3 edges; no brainB data
- [x] 5.10 Implement `Subgraph(ctx, db, brainID int64, f SubgraphFilters) (SubgraphResult, error)` in `links.go`; `SubgraphResult` holds `Nodes []memory.Memory` and `Edges []MemoryLink`
- [x] 5.11 Write failing test: `TestLinks_CascadeOnBrainDelete` — 5 links in brain, hard-delete brain row, assert 0 rows in `memory_links` for that brain (verifies ON DELETE CASCADE)
- [x] 5.12 Write failing test: `TestLinks_CascadeOnMemoryDelete` — M1 has 3 outbound + 2 inbound; hard-delete M1, assert 0 incident link rows

---

## Phase 6 — `internal/events` Package + Storage Hooks

- [x] 6.1 Write failing test: `TestBus_Subscribe_ReceivesMatchingBrain` in `internal/events/bus_test.go` (run with `-race`) — subscribe to brainA, publish `MemoryCreated{BrainID:brainA}`, assert channel receives event
- [x] 6.2 Create `internal/events/events.go`: `Event` interface with `BrainID()`, `Timestamp()`, `Kind()`; concrete structs `MemoryCreated`, `MemoryUpdated`, `MemoryDeleted`, `LinkCreated`, `LinkDeleted`, `BrainConfigChanged`
- [x] 6.3 Create `internal/events/bus.go`: `Bus` struct with `sync.RWMutex`, `subs map[int64][]subscription`, `bufSize int`, `drops atomic.Uint64`; `New(log) *Bus`; `Subscribe(brainID) (<-chan Event, func())`; `Publish(e Event)`
- [x] 6.4 Write failing test: `TestBus_BrainIsolation` — subscribe S1 to brainA, S2 to brainB; publish to brainA; assert S1 receives, S2 receives nothing
- [x] 6.5 Write failing test: `TestBus_SlowSubscriberDrop` — subscribe and never read; publish 100 events (buffer=64); assert `bus.Drops() >= 36`; assert a second (draining) subscriber received all 100
- [x] 6.6 Implement non-blocking `deliver` in `bus.go`: `select { case ch <- e: default: drops.Add(1); log.Warn(...) }`; brainID=0 subscriptions receive all events
- [x] 6.7 Write failing test: `TestBus_Unsubscribe` — subscribe, receive 3 events, call unsub, publish 3 more; assert channel is closed; assert `Drops()` unchanged for other subscriber
- [x] 6.8 Implement `unsub` closure in `Subscribe`: sets `closed` atomic, removes from `subs`, closes channel; calling unsub twice is a no-op
- [x] 6.9 Write failing test (race): `TestBus_ConcurrentPublishSubscribe` — 10 subscribers brainA, 100 goroutines publish; run with `-race`; assert no panic
- [x] 6.10 Add `bus *events.Bus` optional field to `Storage`; add `SetBus(b *events.Bus)` method in `internal/storage/storage.go`; emit `MemoryCreated/Updated/Deleted` from `Save`/`UpdateByID`/`SoftDelete` after commit; emit `LinkCreated/Deleted` from links package after commit; emit `BrainConfigChanged` from brain config update
- [x] 6.11 Write failing test: `TestStorage_EmitsMemoryCreated` — attach bus, subscribe, call `Save`, assert `MemoryCreated` received with correct `memory_id`
- [x] 6.12 Write failing test: `TestStorage_NoEventOnFailedSave` — attach bus, subscribe, force a constraint violation on `Save`, assert no event received

---

## Phase 7 — Isolation Invariant Test Suite

- [x] 7.1 Create `internal/storage/isolation_test.go`: `setupTwoBrains` helper — create brainA and brainB via `brain.Create`, insert 5 memories each (using new `Save(ctx, brainID, m)`), create 3 intra-brain links each
- [x] 7.2 Implement `TestBrainIsolation` table-driven suite: for ops `{Search, Recent, GetByID-foreign, ListLinks, Neighbors, Subgraph, UpdateByID-foreign, SoftDelete-foreign}` — query brainA, assert zero brainB rows returned; GetByID/Update/Delete with foreign id returns `ErrMemoryNotFound`
- [x] 7.3 Implement `TestBrainIsolation_Property`: random 1000-step sequence of `{Save, Update, Delete, Link, Unlink}` split across brainA and brainB; after every step assert per-brain counts match in-memory shadow model; run with `-race`

---

## Phase 8 — Server Compat Layer

- [x] 8.1 Write failing test: `TestTlSave_ResolvesProject` in `internal/server/brain_autocreate_test.go` — call `tl_save` with `project="test-proj"` on a DB with a matching brain, assert memory is created with `brain_id` set
- [x] 8.2 Update `internal/server/tl_save.go`: call `storage.ResolveOrCreateBrainID(ctx, project)` before `storage.Save(ctx, brainID, m)`; auto-creates brain on first encounter
- [x] 8.3 Write failing test: `TestTlSave_AutoCreatesBrain` in `internal/server/brain_autocreate_test.go` — call `tl_save` with a `project` value that has no corresponding brain, assert brain is created and memory has valid `brain_id`
- [x] 8.4 Update `internal/server/tl_search.go`, `tl_context.go`, `tl_get_observation.go`: resolve `project` → `brainID` via `ResolveBrainID` before calling storage methods; on ErrBrainNotFound return empty (no auto-create)
- [x] 8.5 Update `internal/server/tl_delete.go`, `tl_update.go`: use `GetByIDUnscoped` to discover brain_id, then call `SoftDelete`/`UpdateByID` with it
- [x] 8.6 `tl_stats.go` uses project-string queries (Stats/recentMemoriesForStats) — no brainID needed; no change required
- [x] 8.7 Write failing test: `TestTlPromote_SetsBrainID` in `internal/server/brain_autocreate_test.go` — pending event with `project="my-game"`, matching brain exists, promote, assert created memory has `brain_id` set
- [x] 8.8 Update `internal/server/tl_promote.go`: resolve `brainID` from `pending_event.project`; if no matching brain, return `{ status: "error", error: "no brain found for project: <value>" }`
- [x] 8.9 Write failing test: `TestTlPromote_FailsIfNoBrain` in `internal/server/brain_autocreate_test.go` — pending event with `project="orphan"`, no matching brain, assert promotion fails with expected error message
- [x] 8.10 Verify existing server integration tests still compile and pass after signature changes; fixed `seedMemory` helper, all promote tests got `ensureBrain` calls; `go test ./...` GREEN

---

## Phase 9 — Hardening + Verify Gate

- [x] 9.1 Run `go vet ./...` — fix any vet warnings introduced by new packages; commit clean
- [x] 9.2 Run `go test -race ./...` — NOTE: `-race` requires CGO; unavailable on Windows with modernc.org/sqlite. Documented in ADR 0005 (D8). CI on Linux should add `-race`. All tests pass without `-race`.
- [x] 9.3 Run `go test -cover ./...` — baseline: storage 80.4%, server 82.2%, brain 91.0%, links 79.9%, events 75.9%, config 73.8%. No coverage drop vs pre-change.
- [x] 9.4 Create `docs/decisions/0005-brain-as-first-class-entity.md` — ADR capturing D1–D8 from design.md
- [x] 9.5 Update `docs/PROGRESS.md` — M-brain-foundation marked done 2026-05-07; ARCHITECTURE.md updated with multi-brain section
- [x] 9.6 Run `/sdd-verify brain-foundation` and address any CRITICAL findings before `/sdd-archive`
