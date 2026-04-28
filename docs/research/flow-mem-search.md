# Flow: `mem_search` — End-to-End Trace

> The "intelligence" of Engram lives here. Citations are `file:line` relative to `C:\Users\Agustin Lozano\Desktop\_engram-research\engram`.

## 1. Tool registration

`mem_search` is registered at [internal/mcp/mcp.go:240-269](../../../_engram-research/engram/internal/mcp/mcp.go#L240):

```go
// ─── mem_search (profile: agent, core — always in context) ─────────
if shouldRegister("mem_search", allowlist) {
    srv.AddTool(
        mcp.NewTool("mem_search",
            mcp.WithDescription("Search your persistent memory across all sessions. ..."),
            mcp.WithTitleAnnotation("Search Memory"),
            mcp.WithReadOnlyHintAnnotation(true),
            mcp.WithDestructiveHintAnnotation(false),
            mcp.WithIdempotentHintAnnotation(true),
            mcp.WithOpenWorldHintAnnotation(false),
            mcp.WithString("query",   mcp.Required(), ...),
            mcp.WithString("type",    ...),
            mcp.WithString("project", ...),
            mcp.WithString("scope",   ...),
            mcp.WithNumber("limit",   ...),
        ),
        handleSearch(s, cfg, activity),
    )
}
```

Registration is reached the same way as `mem_save`: `cmdMCP` ([main.go:765](../../../_engram-research/engram/cmd/engram/main.go#L765)) → `mcp.NewServerWithConfig` ([mcp.go:214](../../../_engram-research/engram/internal/mcp/mcp.go#L214)) → `registerTools` ([mcp.go:239](../../../_engram-research/engram/internal/mcp/mcp.go#L239)).

The tool is marked **read-only + idempotent** via the `WithReadOnlyHintAnnotation` / `WithIdempotentHintAnnotation` annotations, so MCP clients can cache safely.

## 2. Handler signature and accepted parameters

`func handleSearch(s *store.Store, cfg MCPConfig, activity *SessionActivity) server.ToolHandlerFunc` ([mcp.go:776](../../../_engram-research/engram/internal/mcp/mcp.go#L776)).

Inputs read from `req.GetArguments()`:

| Argument | Type | Required | Default | Notes |
|---|---|---|---|---|
| `query` | string | yes | — | Natural-language or keyword string |
| `type` | string | no | `""` (no filter) | One of `tool_use`, `file_change`, `decision`, `bugfix`, `architecture`, `pattern`, etc. |
| `project` | string | no | auto-detect from cwd | If supplied and unknown, returns `unknown_project` envelope |
| `scope` | string | no | `""` (no filter) | `project` or `personal` |
| `limit` | number | no | 10, hard-capped at `MaxSearchResults` (default 20) | |

Resolution flow:
1. `resolveReadProject(s, projectOverride)` ([mcp.go:785](../../../_engram-research/engram/internal/mcp/mcp.go#L785) → [mcp.go:1659](../../../_engram-research/engram/internal/mcp/mcp.go#L1659)) validates the override against the store, OR auto-detects from cwd. Unknown override returns an `unknown_project` MCP envelope ([mcp.go:789](../../../_engram-research/engram/internal/mcp/mcp.go#L789)) with a list of available project names.
2. Project is normalized via `store.NormalizeProject` ([mcp.go:797](../../../_engram-research/engram/internal/mcp/mcp.go#L797)).
3. Activity is recorded for nudge hints: `activity.RecordToolCall(sessionID)` ([mcp.go:801](../../../_engram-research/engram/internal/mcp/mcp.go#L801)).
4. The handler delegates to `s.Search(query, store.SearchOptions{...})` ([mcp.go:803-808](../../../_engram-research/engram/internal/mcp/mcp.go#L803)).

## 3. Search strategy — pure FTS5/BM25 with a topic-key shortcut

Implementation: `func (s *Store) Search(query string, opts SearchOptions)` at [internal/store/store.go:2571-2693](../../../_engram-research/engram/internal/store/store.go#L2571).

There is **no semantic / embedding search**. There is **no LIKE fallback**. The two paths are:

### 3a — Topic-key direct lookup (only if query contains `/`)

If the raw query string contains a `/` ([store.go:2584](../../../_engram-research/engram/internal/store/store.go#L2584)) it is treated as a literal `topic_key` lookup first:

```sql
SELECT id, ifnull(sync_id, '') AS sync_id, session_id, type, title, content, tool_name,
       project, scope, topic_key, revision_count, duplicate_count,
       last_seen_at, created_at, updated_at, deleted_at
FROM observations
WHERE topic_key = ?       -- the raw query, e.g. "architecture/auth-model"
  AND deleted_at IS NULL
[ AND type     = ? ]      -- optional
[ AND project  = ? ]      -- optional
[ AND scope    = ? ]      -- optional
ORDER BY updated_at DESC
LIMIT ?
```

Hits get a synthetic rank of `-1000` ([store.go:2621](../../../_engram-research/engram/internal/store/store.go#L2621)) so they sort first when merged with FTS hits.

This is a small but smart UX trick: agents that already know a topic key get O(1) retrieval without paying FTS5 cost.

### 3b — FTS5 / BM25 path (always runs)

The query is sanitized by `sanitizeFTS` ([store.go:5553-5561](../../../_engram-research/engram/internal/store/store.go#L5553)):

```go
func sanitizeFTS(query string) string {
    words := strings.Fields(query)
    for i, w := range words {
        w = strings.Trim(w, `"`)        // strip existing quotes
        words[i] = `"` + w + `"`        // wrap each token in literal quotes
    }
    return strings.Join(words, " ")
}
```

So `fix auth bug` becomes `"fix" "auth" "bug"`. This protects FTS5 from query operator characters in user input (FTS5 treats `:` `*` `^` etc. as operators; literal-quoted phrases are matched verbatim). It also means **no implicit OR/NEAR/AND combinators are usable** — the user cannot write FTS5 syntax. There is no MATCH-rewrite for stemming or fuzzy matching.

The actual search SQL ([store.go:2630-2656](../../../_engram-research/engram/internal/store/store.go#L2630)):

```sql
SELECT o.id, ifnull(o.sync_id, '') AS sync_id, o.session_id, o.type, o.title, o.content,
       o.tool_name, o.project, o.scope, o.topic_key,
       o.revision_count, o.duplicate_count, o.last_seen_at,
       o.created_at, o.updated_at, o.deleted_at,
       fts.rank
FROM observations_fts fts
JOIN observations o ON o.id = fts.rowid
WHERE observations_fts MATCH ?
  AND o.deleted_at IS NULL
[ AND o.type    = ? ]
[ AND o.project = ? ]
[ AND o.scope   = ? ]
ORDER BY fts.rank
LIMIT ?
```

`fts.rank` is SQLite FTS5's built-in **BM25** score (negative; values closer to 0 = better match — that's why it sorts ascending). The `observations_fts` virtual table is declared at [store.go:633-642](../../../_engram-research/engram/internal/store/store.go#L633) with columns `(title, content, tool_name, type, project, topic_key)` — meaning the search corpus is title + content + tool_name + type + project + topic_key, weighted equally by BM25.

The FTS table is kept in sync by triggers `obs_fts_insert`, `obs_fts_delete`, `obs_fts_update` ([store.go:914-929](../../../_engram-research/engram/internal/store/store.go#L914)).

### 3c — Result merge

Topic-key direct hits go first; FTS5 hits follow, deduped by `id` against the direct hits ([store.go:2664-2684](../../../_engram-research/engram/internal/store/store.go#L2664)). Total truncated to `limit` ([store.go:2689](../../../_engram-research/engram/internal/store/store.go#L2689)).

## 4. Embeddings — not generated, not queried

Confirmed: `embedding`, `embedding_model`, `embedding_created_at` exist as **reserved BLOB/TEXT columns** ([store.go:789-791](../../../_engram-research/engram/internal/store/store.go#L789)) and nothing else. No vector-similarity SQL, no embedding library import, no model field populated. A grep for `embedding|vector|cosine` across `internal/` returns only the schema reservation, an unrelated setup file, and a conflict-loop test. The search path is 100% FTS5/BM25 today.

## 5. Ranking and scoring

Ordering is **purely BM25** as computed by SQLite FTS5 (`ORDER BY fts.rank`). There is:

- **No recency boost.**
- **No type-weighted boost** (a `decision` is not promoted over a `tool_use`).
- **No scope boost** (project vs personal are filtered, not scored).
- **No length penalty/normalization beyond what BM25 already does.**
- **No popularity/`duplicate_count` boost** even though the field exists.
- **One synthetic rank**: topic-key direct hits get `-1000` to float to the top.

The `MCPConfig.BM25Floor` knob ([mcp.go:35-44](../../../_engram-research/engram/internal/mcp/mcp.go#L35)) used in `FindCandidates` ([relations.go:156](../../../_engram-research/engram/internal/store/relations.go#L156)) is **NOT** applied in `Store.Search` — only conflict-candidate detection thresholds on it. Plain `mem_search` returns whatever BM25 says.

Annotations in the formatted response ([mcp.go:851-869](../../../_engram-research/engram/internal/mcp/mcp.go#L851)) decorate each result with `supersedes`, `superseded_by`, and `conflict: contested by` lines using a batch-loaded relation map ([mcp.go:826-829](../../../_engram-research/engram/internal/mcp/mcp.go#L826)) — but this is decoration, not scoring.

## 6. Pagination, truncation, and the `mem_get_observation` companion

There is **no real pagination** — no offset, cursor, or `from_id` parameter. The handler just clamps `limit` to `min(limit, MaxSearchResults)` (default cap 20 in `Store.Search` at [store.go:2579-2581](../../../_engram-research/engram/internal/store/store.go#L2579)).

Each result's `content` is truncated to **300 chars** with the suffix ` [preview]` ([mcp.go:841-844](../../../_engram-research/engram/internal/mcp/mcp.go#L841)). When any result was truncated, the response footer instructs the agent to call `mem_get_observation`:

```
---
Results above are previews (300 chars). To read the full content of a specific memory,
call mem_get_observation(id: <ID>).
```

`handleGetObservation` ([mcp.go:1321-1368](../../../_engram-research/engram/internal/mcp/mcp.go#L1321)) is a trivial `SELECT … FROM observations WHERE id = ? AND deleted_at IS NULL` ([store.go:2316](../../../_engram-research/engram/internal/store/store.go#L2316)). It returns the **untruncated** `content` plus metadata (project, scope, topic, tool, duplicate_count, revision_count, created_at). It does **not** filter by project — any agent can fetch any observation by ID.

This is the "progressive disclosure" pattern: search returns shallow previews, and the agent calls `mem_get_observation` to drill in. It keeps tool-call payloads small.

## 7. Sequence diagram

```mermaid
sequenceDiagram
    autonumber
    participant C as MCP Client (agent)
    participant T as mcp-go ServeStdio
    participant H as handleSearch
    participant P as resolveReadProject
    participant S as Store.Search
    participant FTS as observations_fts (FTS5 BM25)
    participant DB as observations (SQLite)
    participant R as GetRelationsForObservations

    C->>T: tools/call mem_search {query, type?, project?, scope?, limit?}
    T->>H: ctx, req
    H->>P: resolve project (override or cwd auto-detect)
    P-->>H: DetectionResult{project} or unknown_project err
    H->>S: Search(query, SearchOptions{type, project, scope, limit})

    alt query contains "/"
        S->>DB: SELECT ... WHERE topic_key = ? AND deleted_at IS NULL ORDER BY updated_at DESC LIMIT ?
        DB-->>S: direct topic_key hits (rank = -1000)
    end

    S->>S: sanitizeFTS(query) → wrap each word in quotes
    S->>FTS: SELECT o.*, fts.rank FROM observations_fts fts JOIN observations o ON o.id = fts.rowid<br/>WHERE observations_fts MATCH ? [+ filters] ORDER BY fts.rank LIMIT ?
    FTS-->>S: rows scored by BM25 (negative; closer to 0 = better)
    S->>S: merge direct + FTS hits, dedupe by id, truncate to limit
    S-->>H: []SearchResult

    H->>R: GetRelationsForObservations(syncIDs) batch (avoid N+1)
    R-->>H: relationsMap
    H->>H: build text response: per-row preview (300 chars) + supersedes/conflict annotations
    H-->>T: respondWithProject(detRes, formattedText, _meta)
    T-->>C: tools/call result (text content; agent may follow up with mem_get_observation)
```

## 8. Decisions we must make BEFORE we implement `tl_search`

These are the architectural choices this flow forces. Each one shapes the data model and downstream tooling.

| # | Decision | Recommendation (one line) |
|---|---|---|
| D1 | **Search backend: FTS5-only, embeddings-only, or hybrid?** | Start FTS5-only (mirror Engram). Add embeddings later behind a flag. |
| D2 | **If embeddings: provider?** (local ONNX / `gguf`, OpenAI, Voyage, Cohere, etc.) | If we add them, prefer local (e.g. `bge-small`/`gte-small` via `onnx-runtime` Go bindings) to avoid network/lock-in. |
| D3 | **Vector storage: BLOB column + in-Go cosine, sqlite-vec extension, sqlite-vss, or a separate index?** | If we ship vectors, use `sqlite-vec` — pure C extension, no external service. |
| D4 | **Embedding dimensionality and storage shape** | Pick once and pin (e.g. 384 dims as float32 → 1.5 KB per row). Store as `BLOB` with explicit `embedding_model` discriminator so we can re-embed on model change. |
| D5 | **Sync trigger pattern: triggers + `content=` table (Engram's choice) vs manual writes?** | Reuse Engram's pattern — triggers + contentless FTS5 table is the lowest-maintenance shape. |
| D6 | **FTS5 query sanitization strategy** | Mirror `sanitizeFTS` exactly — wrap each token in quotes. Cheap, prevents 100% of FTS5 syntax injection. |
| D7 | **Should `tl_search` accept structured FTS5 syntax (NEAR, AND, OR) for power users?** | No. Keep query a flat string; expose advanced ops only behind a `--raw` flag if ever. |
| D8 | **Ranking: pure BM25 or apply boosts (recency, type, duplicate_count)?** | Pure BM25 in v1. Document the limitation. Boosts are a v2 lever. |
| D9 | **Pagination model** | Add `offset` + `total` from day 1 — Engram skipped this and we'll regret it the first time we search 1000+ rows. |
| D10 | **Result preview length and "expand by id" companion tool** | Mirror Engram: 300-char preview + `tl_get_observation(id)`. Tweak length only with telemetry. |
| D11 | **Project / scope semantics during search** | Same as Engram: hard filter, never a soft boost. Agents shouldn't see other projects' memory. |
| D12 | **Topic-key shortcut on `/`** | Keep it (reuse Engram). It's a 5-line win for agent UX. |
| D13 | **Should the response be plaintext (Engram), structured JSON, or both?** | Return both — `mcp.NewToolResultText` for the human-readable summary AND a structured JSON `_meta.results` array for tools that want to parse without regex. |
| D14 | **Soft-delete semantics in search** | Mirror Engram: `deleted_at IS NULL` filter. Hard delete only when explicitly requested. |
| D15 | **Conflict / supersedes annotations on results** | Defer (separate from search). Ship plain results in v1; add the relations decoration when we ship `tl_judge`. |
| D16 | **Concurrency / connection pool** | Single-writer SQLite via `db.SetMaxOpenConns(1)` + WAL is Engram's choice ([store.go:528](../../../_engram-research/engram/internal/store/store.go#L528)). Reuse. |
| D17 | **Max query length / abuse protection** | Add an explicit `MaxQueryLength` (e.g. 512 chars) and reject longer queries. Engram has none. |
| D18 | **Should results include the project name unconditionally, or only on cross-project search?** | Include unconditionally — disambiguates personal vs project scope at a glance. |
| D19 | **Telemetry on search** | Log query, hit-count, top rank locally (debug-only). Do NOT phone home. |
| D20 | **Result limit cap** | Mirror Engram's hard cap (20). Anything more bloats agent context. |
