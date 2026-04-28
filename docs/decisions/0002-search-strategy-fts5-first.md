# ADR 0002 — Search strategy: FTS5 + BM25 in v1, embeddings deferred (additive in M5)

- **Status**: Accepted
- **Date**: 2026-04-28
- **Supersedes**: —
- **Related**: [ADR 0001](0001-architecture-baseline.md)

## Context

The search experience is the single most user-facing decision of a memory system. A user calls `tl_search "lantern bloom"` and either gets the answer they need or doesn't — that determines whether they trust the tool.

Three candidate strategies:

| Strategy        | What it is                                           | Cost                                          | Recall ceiling                              |
| --------------- | ---------------------------------------------------- | --------------------------------------------- | -------------------------------------------- |
| **Lexical (FTS5 + BM25)** | SQLite's built-in full-text engine                  | Free, zero deps, instant                      | Limited to lexical overlap (synonyms miss)  |
| **Semantic (embeddings)** | Vectorize content, cosine similarity                 | API cost or local model, vector index, schema | High (if model is good)                      |
| **Hybrid**       | Both, with fusion ranking                            | All of the above + fusion logic                | Best, theoretically                          |

Engram's reconnaissance ([`research/flow-mem-search.md`](../research/flow-mem-search.md)) revealed something important: **Engram has reserved schema columns for embeddings (`embedding BLOB`, `embedding_model TEXT`, `embedding_created_at INTEGER` at `internal/store/store.go:789-791`) but never writes to them.** The actual search code is 100% FTS5 + BM25 (`store.go:2630-2656`), with one clever shortcut: queries containing `/` match against `topic_key` first (`store.go:2584-2625`).

Engram has been used in production by its community for months without embeddings, and the feedback has been positive. This is informative: **lexical search is sufficient for the dominant use case** (a developer recalling something they recently wrote, with overlapping vocabulary).

## Decision

**Ship Thoughtline v1 with FTS5 + BM25 only. Reserve schema columns for embeddings (mirroring Engram). Add a semantic layer in milestone M5, additively, behind a feature flag.**

Concretely:

1. **`tl_search` (M2)** uses FTS5 with BM25 ranking. The query string is passed through with minimal preprocessing (handle reserved chars, but expose advanced FTS5 syntax only behind a `--raw` flag if asked).
2. **Topic-key shortcut**: queries containing `/` are first matched against `topic_key` via GLOB before falling through to FTS5. Identical to Engram's behavior.
3. **Filters**: `type`, `scope`, `project`, optional `topic_key` glob — all applied as SQL WHERE predicates after the FTS5 match.
4. **Result preview**: 300 chars max, generated via SQLite's `snippet()` function. Full content via the companion `tl_get_observation(id)` tool.
5. **Schema reservation**: include `embedding`, `embedding_model`, `embedding_created_at` columns from day one (M1). Nullable. Never written by v1.
6. **M5 plan**: when we add embeddings, the design must satisfy:
   - additive — no rewrite of existing rows;
   - opt-in — controlled by config / env var, off by default;
   - provider-agnostic — model name and dimensionality stored per row, so multiple providers can coexist during a transition;
   - hybrid ranking — FTS5 score and vector score combined via reciprocal rank fusion (RRF) or a tunable linear blend.

## Consequences

### Positive

- **Zero external dependencies for v1.** No API keys, no local model downloads, no GPU concerns. `go build` and run.
- **Predictable, auditable results.** A user can read the FTS5 query, look at the SQL, and reason about why a result ranked where it did. Embeddings are far more opaque.
- **Cost: $0**. No per-token billing, no rate limits.
- **Fast.** FTS5 over a few thousand memories is sub-millisecond.
- **Shippable in days, not weeks.** M2 is realistic with this scope.

### Negative

- **Synonym blindness.** A search for `"lighting"` will not find a memory titled `"illumination tweaks"`. Mitigations:
  - `topic_key` discipline in user prompts (we'll document patterns).
  - The `tl_save` skill prompt will guide the AI to use varied vocabulary in titles.
  - M5 fixes this for users who opt in.
- **Acronym fragility.** Searching `"GPU"` won't find content that says only `"graphics card"`. Same mitigation set.
- **Non-English content.** FTS5 with `unicode61 remove_diacritics 2` handles diacritics but not stemming for non-English. Acceptable tradeoff for v1.

### Neutral

- We are knowingly choosing the same compromise Engram made. If user feedback shows lexical recall failures dominate complaints, M5 moves up in priority. If not, M5 stays deferred indefinitely.

## Alternatives considered

### A — Embeddings-first (no FTS5)

Rejected. Forces a model/provider commitment we can't responsibly make in M1. Locks users into either external API spend or local inference setup. Removes the "just runs" property we are protecting.

### B — Hybrid in v1

Rejected. Doubles the design surface area before we know it's needed. Hybrid ranking (RRF, linear, learned-to-rank) is itself a research problem.

### C — FTS5 only, never embeddings

Rejected. The reserved schema columns are cheap insurance. Refusing to allow embeddings forever would be ideological, not engineering.

### D — Use `sqlite-vec` extension for vectors

Considered for M5. Attractive because it keeps everything in one SQLite file. Decision deferred — see M5 design when it's drafted. Constraints we know now: must remain pure-Go (no CGO), so any extension we use needs a Go-native equivalent or modernc-compatible build.

## Migration path to M5 (sketch)

```sql
-- M5 migration is purely additive:
ALTER TABLE memories ADD COLUMN embedding_dim INTEGER;  -- only new column needed

-- New companion table for vector index (if we use sqlite-vec or a Go-native lib):
CREATE TABLE IF NOT EXISTS memory_embeddings (
    sync_id     TEXT PRIMARY KEY REFERENCES memories(sync_id) ON DELETE CASCADE,
    embedding   BLOB    NOT NULL,
    model       TEXT    NOT NULL,
    dim         INTEGER NOT NULL,
    created_at  INTEGER NOT NULL
);
```

Existing rows remain untouched. A background `tl_reindex` tool can backfill embeddings for memories worth re-embedding.

## References

- Engram FTS5 schema: `_engram-research/engram/internal/store/store.go:789` (reserved columns), `:1832` (FTS5 virtual table)
- BM25 ranking call: `_engram-research/engram/internal/store/store.go:2630`
- Topic-key shortcut: `_engram-research/engram/internal/store/store.go:2584`
- SQLite FTS5 docs: <https://www.sqlite.org/fts5.html>
