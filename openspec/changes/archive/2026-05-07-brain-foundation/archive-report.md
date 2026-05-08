# Archive Report — brain-foundation

**Archived**: 2026-05-07
**Verify Verdict**: PASS_WITH_WARNINGS

## Summary

Schema v4 (brains, global_config, memory_links, memories.brain_id), 4 new packages (config, brain, links, events), storage API redesign with required brainID for compile-time isolation, server compat layer, isolation invariant test suite, ADR 0005.

## Stats

- 78 tasks completed across 9 phases
- 355 tests passing
- 12 packages green, go vet clean
- Coverage: storage 80.4%, server 82.2%, brain 91.0%, links 79.9%, events 75.9%, config 73.8%

## Specs promoted to source-of-truth

- brain-domain (NEW)
- memory-graph (NEW)
- cognitive-config (NEW)
- event-bus (NEW)
- engram-migration (delta merged)
- passive-capture (delta merged)

## Open warnings carried forward (NOT blocking)

- W1: memories.brain_id has no ON DELETE CASCADE by design; HardDeleteBrain API deferred to a future change
- W2: cmd/migrate auto-creates brains instead of failing-row per engram-migration delta — permissive deviation, follow-up change recommended to align
- W3: Subgraph lacks Relation filter (MAY scenario in spec, not MUST)

## Suggestions for follow-up

- Add Relation filter to SubgraphOptions (W3)
- Implement HardDeleteBrain API for synthetic/sandbox cleanup (W1 + Phase 9 deferral)
- Schema v5 migration to drop memories.project column (deferred per design)
- CI matrix on Linux with `-race` for concurrency tests (per ADR 0005 D8)
- Per-brain stats breakdown (currently cross-brain aggregate)

## References

- Proposal: archive/2026-05-07-brain-foundation/proposal.md
- Design: archive/2026-05-07-brain-foundation/design.md
- Tasks: archive/2026-05-07-brain-foundation/tasks.md
- Verify report: archive/2026-05-07-brain-foundation/verify-report.md
- ADR: docs/decisions/0005-brain-as-first-class-entity.md
- Engram topic_keys: sdd/brain-foundation/{proposal,spec,design,tasks,apply-progress,verify-report}
- Architecture decisions in Thoughtline: architecture/brain-{product-split,entity-model,config-resolution,memory-links,foundation-phase0-scope}
