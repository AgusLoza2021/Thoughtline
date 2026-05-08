# Verify Report - brain-foundation

**Date**: 2026-05-07
**Verdict**: PASS_WITH_WARNINGS
**Mode**: Strict TDD

## Test results

| Command | Result |
|---------|--------|
| go test ./... | PASS - 355 tests, 0 failures, 0 skipped |
| go vet ./... | CLEAN - no warnings |

### Coverage per package

| Package | Coverage |
|---------|----------|
| internal/storage | 80.4% |
| internal/server | 82.2% |
| internal/brain | 91.0% |
| internal/links | 79.9% |
| internal/events | 75.9% |
| internal/config | 73.8% |
| internal/memory | 94.4% |
| cmd/migrate | 73.1% |

## Spec coverage matrix

| Capability | Reqs | Scenarios | Covered | Notes |
|---|---|---|---|---|
| brain-domain | 6 | 10 | 10/10 | All passing |
| memory-graph | 8 | 14 | 14/14 | Cascade via workaround (W1) |
| cognitive-config | 5 | 7 | 7/7 | Defaults match spec exactly |
| event-bus | 6 | 11 | 10/11 | No -race (INFO); no-network-listener untested (S1) |
| engram-migration (delta) | 1 | 2 | 1/2 | Scenario 2 is a spec deviation (W2) |
| passive-capture (delta) | 1 | 2 | 2/2 | All passing |

## Findings

### CRITICAL
None.

### WARNING

**W1 - memory_links cascade on brain delete requires FK-off workaround**
memories.brain_id has no ON DELETE CASCADE by design. When FKs are on, deleting a brain is blocked by orphaned memories rows. TestBrainIsolation_CASCADEDoesNotLeak tests the cascade contract via a two-step workaround: delete links first, then disable FK to delete brain. The ON DELETE CASCADE on memory_links.brain_id cannot fire end-to-end without either adding CASCADE to memories.brain_id or implementing HardDeleteBrain() with the correct deletion sequence. Risk: future HardDeleteBrain must account for this or the cascade contract could fail silently.

**W2 - Engram migration delta: no-matching-brain path auto-creates instead of failing row**
File: cmd/migrate/writer.go:67 - uses ResolveOrCreateBrainID() instead of ResolveBrainID().
Spec requires: fail the row, increment failed counter, log sync_id and project, continue to next row.
Implementation silently creates new brains for unknown project values. No test covers this negative path. Severity: WARNING (migration utility only; not a core isolation invariant; permissive deviation).

**W3 - Subgraph relation filter not implemented**
SubgraphOptions has no Relation field. The spec optional by-relation-type narrowing scenario is unimplemented and untested. All MUST requirements of Subgraph are met; this is an unimplemented MAY scenario.

### SUGGESTION

**S1 - TestBus_NoNetworkListener absent**: Event-bus spec has an explicit no-network-listener scenario. Property holds by construction but a structural assertion or comment would document the invariant.

**S2 - TestMigrateV4_EmptyProject error assertion is loose**: Test checks error contains any of brain_id, project, or backfill but does not assert the offending memory numeric ID appears in the message (spec says the error must list offending IDs).

### INFO

- HardDeleteBrain API: explicitly deferred per Phase 9 docs; archive review only.
- -race on Windows: documented in ADR 0005 D8; requires CGO unavailable with modernc.org/sqlite. CI on Linux should add -race.
- v5 migration (drop memories.project): deferred until all callers confirmed brain-aware.
- Stats per-brain breakdown: deferred; cross-brain aggregate intentional for single-user model (ADR D7).

## Non-spec requirement checks

- global_config seed uses config.DefaultGlobalJSON(): PASS. TestMigrateV4_GlobalConfigSeededWithDefaults verifies seeded JSON matches DefaultGlobalJSON() and is not bare {}.
- BrainConfigChanged emission wired in brain.UpdateConfig: PASS. Tests: TestBrain_BusEmitOnUpdateConfig, TestBrain_BusNilSafe.
- LinkCreated/LinkDeleted emission wired in links: PASS. Tests: TestLinks_BusEmitOnCreate, TestLinks_BusEmitOnDelete, TestLinks_BusNilSafe.
- Compile-time brainID enforcement: PASS. Every public per-brain method takes brainID int64 as explicit positional param.
- Isolation property test (1000-op random sequence): PASS. TestBrainIsolation_Property.
- Config defaults match spec: PASS. MaxNeighborsPerNode = 50, SyntheticDecay = 1.0 - exact match.

## Recommendation

PASS_WITH_WARNINGS - READY for sdd-archive.

All CRITICAL invariants are implemented, tested, and passing: brain isolation, slug uniqueness, cross-brain link rejection, memory scoping, orphan-safe migration, config deep-merge, and event emission. The 3 WARNINGs are non-blocking for archive. 355 tests pass, go vet is clean.
