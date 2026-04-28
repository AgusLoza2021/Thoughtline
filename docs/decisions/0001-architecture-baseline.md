# ADR 0001 — Architecture baseline: copy Engram's pattern set, customize taxonomy

- **Status**: Accepted
- **Date**: 2026-04-28
- **Deciders**: Project bootstrap session

## Context

We are building Thoughtline, a local-first MCP memory server tailored to game-development workflows. Before writing any persistence or protocol code, we asked: *should we fork [Engram](https://github.com/Gentleman-Programming/engram), build something net-new, or build something parallel that intentionally borrows Engram's shape?*

Engram is mature, MIT-licensed, well-tested, and solves 80% of the problem we want to solve. A focused architectural reconnaissance (see [`research/engram-anatomy.md`](../research/engram-anatomy.md), [`research/flow-mem-save.md`](../research/flow-mem-save.md), [`research/flow-mem-search.md`](../research/flow-mem-search.md)) confirmed:

- Engram's MCP surface is clean and well-designed (`mark3labs/mcp-go`, stdio).
- Storage layer is SQLite with FTS5 + BM25 — not embedding-based, contrary to common assumption.
- The `topic_key` upsert pattern in `AddObservation` (engram `internal/store/store.go:1913-1958`) is genuinely clever and worth adopting verbatim.
- Cross-cutting concerns (cloud sync, TUI, conflict resolution, obsidian export) are bolt-on and unrelated to the core memory experience.

The remaining 20% — gamedev-specific memory types, vocabulary tuned for asset/scene/perf concerns — is exactly where Engram's generic positioning leaves room.

## Decision

**Build Thoughtline as a parallel project, deliberately architecture-compatible with Engram, but not a fork.**

Specifically:

1. **Reuse the same MCP framework** (`github.com/mark3labs/mcp-go v0.44.0`) so tool registration, request/response shapes, and transport behavior match Engram's exactly. This makes future cross-pollination of tooling (e.g. an MCP debugger) trivial.
2. **Reuse the same storage primitives**: SQLite via `modernc.org/sqlite` (no CGO), FTS5 + BM25 for search, contentless virtual tables kept in sync via triggers, the `topic_key` upsert pattern. Schema column names follow Engram's where they overlap.
3. **Reuse Engram's tool naming convention** with a different prefix: `tl_save`, `tl_search`, `tl_context`, `tl_get_observation`, `tl_session_summary`, etc. mirror Engram's `mem_*` family. This makes prompts and skill files written for one mostly-compatible with the other.
4. **Diverge on memory taxonomy and schema fields**. Thoughtline ships gamedev-first types (`game-design-decision`, `scene-pattern`, `asset-reference`, `perf-gotcha`, `pipeline-step`, ...) and may add gamedev-specific fields per type (e.g. `engine`, `platform`, `scene_path`). Full taxonomy in [`design/memory-domain.md`](../design/memory-domain.md).
5. **Skip Engram's bolt-ons**: no cloud sync, no TUI, no obsidian export, no plugin glue, no autosync UI. v1 is one binary, one database, one MCP transport.

Why parallel and not a fork:

- Forking commits us to either (a) merge upstream forever, or (b) diverge and lose updates. We do not have the maintainer bandwidth for (a), and (b) is a strictly worse Engram.
- The gamedev taxonomy is a schema-level concern. Adding it as a fork creates merge conflicts on every Engram release. Building it parallel keeps the boundary clean.
- The MIT license lets us re-implement freely while crediting Engram in the LICENSE file and documentation.

## Consequences

### Positive

- **Fast bootstrap**. Instead of designing from scratch, we lifted decisions Engram already validated (search engine, sqlite driver, MCP library, upsert pattern).
- **Smaller surface area**. By skipping bolt-ons, M1 is "tool registration + storage + one tool" — achievable in a single session.
- **Future cross-pollination is cheap**. Either project can borrow improvements (a better search ranker, a smarter session digest) by reading the other's source. We are kindred projects, not competitors.
- **Clear positioning for users**. Generic gigs go to Engram; gamedev gigs come to Thoughtline. The README makes this explicit.

### Negative / risks

- **Code duplication**. Two implementations of nearly the same SQL schema is real cost. If Engram fixes a bug in a trigger, we have to notice and port it.
  - *Mitigation*: keep `internal/storage/schema.go` small, reference Engram source in comments, watch their CHANGELOG.
- **Visual divergence over time**. As Thoughtline's taxonomy grows, the storage layer may need extra fields Engram doesn't have. This is fine but must be ADR-gated to prevent drift.
- **Branding confusion**. Two projects in the same niche may confuse first-time users. The README addresses this by openly recommending Engram for non-gamedev use cases.

### Neutral

- We adopt Engram's wisdom that **embeddings are not required for a credible memory experience** (see [ADR 0002](0002-search-strategy-fts5-first.md)). Our v1 ships FTS5 + BM25 only.

## Alternatives considered

### A — Fork Engram, add gamedev types

Rejected. Adds maintenance burden in exchange for negligible code reuse savings. Every Engram release would create merge work on Thoughtline-specific schema columns.

### B — Build from scratch on a different stack (TypeScript MCP SDK)

Rejected. Loses architectural compatibility with Engram. The Anthropic TS SDK is excellent for new MCP projects but here we are deliberately mirroring an existing Go project.

### C — Contribute gamedev taxonomy upstream to Engram

Considered seriously. Rejected because the taxonomy is opinionated enough that it would either dilute Engram's general-purpose stance or end up gated behind config. Cleaner to ship a sibling project.

## References

- Engram repo: <https://github.com/Gentleman-Programming/engram>
- Reconnaissance docs: [`docs/research/`](../research/)
- ADR 0002 (search strategy): [`0002-search-strategy-fts5-first.md`](0002-search-strategy-fts5-first.md)
