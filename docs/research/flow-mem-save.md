# Flow: `mem_save` — End-to-End Trace

> Trace from MCP client request to persisted SQLite row. Citations are `file:line` relative to `C:\Users\Agustin Lozano\Desktop\_engram-research\engram`.

## 1. Tool registration

The `mem_save` tool is registered in [internal/mcp/mcp.go:271-329](../../../_engram-research/engram/internal/mcp/mcp.go#L271):

```go
// ─── mem_save (profile: agent, core — always in context) ───────────
if shouldRegister("mem_save", allowlist) {
    srv.AddTool(
        mcp.NewTool("mem_save",
            mcp.WithTitleAnnotation("Save Memory"),
            mcp.WithReadOnlyHintAnnotation(false),
            mcp.WithDestructiveHintAnnotation(false),
            mcp.WithIdempotentHintAnnotation(false),
            mcp.WithOpenWorldHintAnnotation(false),
            mcp.WithDescription(`Save an important observation to persistent memory ...`),
            mcp.WithString("title",     mcp.Required(), ...),
            mcp.WithString("content",   mcp.Required(), ...),
            mcp.WithString("type",      ...),
            mcp.WithString("session_id", ...),
            mcp.WithString("scope",     ...),
            mcp.WithString("topic_key", ...),
        ),
        handleSave(s, cfg, activity),
    )
}
```

`registerTools` is called from `newServerWithActivity` ([mcp.go:226](../../../_engram-research/engram/internal/mcp/mcp.go#L226)) which itself is wrapped by `NewServerWithConfig` ([mcp.go:214](../../../_engram-research/engram/internal/mcp/mcp.go#L214)) and reached from `cmd/engram/main.go cmdMCP` ([main.go:785](../../../_engram-research/engram/cmd/engram/main.go#L785)).

The tool advertises six MCP arguments: `title` (required), `content` (required), `type`, `session_id`, `scope`, `topic_key`. **There is no `project` argument by design** — the project is auto-detected from cwd (REQ-308 in their notes; see [mcp.go:893](../../../_engram-research/engram/internal/mcp/mcp.go#L893)).

## 2. Handler: `handleSave`

Defined at [internal/mcp/mcp.go:885](../../../_engram-research/engram/internal/mcp/mcp.go#L885).

```go
func handleSave(s *store.Store, cfg MCPConfig, activity *SessionActivity) server.ToolHandlerFunc {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        title, _    := req.GetArguments()["title"].(string)
        content, _  := req.GetArguments()["content"].(string)
        typ, _      := req.GetArguments()["type"].(string)
        sessionID,_ := req.GetArguments()["session_id"].(string)
        scope, _    := req.GetArguments()["scope"].(string)
        topicKey, _ := req.GetArguments()["topic_key"].(string)
        // project field intentionally not read — auto-detect only
        ...
    }
}
```

### Validation performed
1. **Project auto-detection**: `resolveWriteProject()` ([mcp.go:896](../../../_engram-research/engram/internal/mcp/mcp.go#L896) → [mcp.go:1643](../../../_engram-research/engram/internal/mcp/mcp.go#L1643)) walks the cwd. If multiple repos are detected, returns an `ambiguous_project` error envelope ([mcp.go:900-904](../../../_engram-research/engram/internal/mcp/mcp.go#L900)).
2. **Project normalization**: lower-cased + trimmed via `store.NormalizeProject` ([mcp.go:908](../../../_engram-research/engram/internal/mcp/mcp.go#L908)).
3. **Type default**: blank `type` becomes `"manual"` ([mcp.go:911](../../../_engram-research/engram/internal/mcp/mcp.go#L911)).
4. **Session id default**: blank session_id becomes `defaultSessionID(project)` ([mcp.go:914](../../../_engram-research/engram/internal/mcp/mcp.go#L914) → [mcp.go:1734](../../../_engram-research/engram/internal/mcp/mcp.go#L1734)).
5. **Topic-key suggestion** (informational only): `suggestTopicKey(typ, title, content)` ([mcp.go:917](../../../_engram-research/engram/internal/mcp/mcp.go#L917)) computes a recommended `topic_key` and surfaces it in the response message if the caller did not provide one.
6. **Similar-project warning**: if the resolved project is brand new, `projectpkg.FindSimilar` ([mcp.go:931](../../../_engram-research/engram/internal/mcp/mcp.go#L931)) prepares a "did you mean `XYZ`?" warning.
7. **Implicit session ensure**: `ensureImplicitSessionWithCWD` ([mcp.go:941](../../../_engram-research/engram/internal/mcp/mcp.go#L941) → [mcp.go:66](../../../_engram-research/engram/internal/mcp/mcp.go#L66)) creates the session row if missing — so saving never errors on missing session FK.

There is **no** explicit length validation on `title` or `content` at the handler layer. Truncation happens later in the store.

## 3. From request payload to stored record

The handler builds `store.AddObservationParams` ([store.go:127](../../../_engram-research/engram/internal/store/store.go#L127)) and calls `s.AddObservation(...)` ([mcp.go:945](../../../_engram-research/engram/internal/mcp/mcp.go#L945)):

```go
savedID, err := s.AddObservation(store.AddObservationParams{
    SessionID: sessionID,
    Type:      typ,
    Title:     title,
    Content:   content,
    Project:   project,
    Scope:     scope,
    TopicKey:  topicKey,
})
```

### Schema (the `observations` table)

From `migrate()` ([store.go:607-626](../../../_engram-research/engram/internal/store/store.go#L607)) plus the additive columns at [store.go:722-740](../../../_engram-research/engram/internal/store/store.go#L722) and [store.go:783-797](../../../_engram-research/engram/internal/store/store.go#L783):

```sql
CREATE TABLE IF NOT EXISTS observations (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    sync_id         TEXT,                  -- portable cross-machine id ("obs-<uuid>")
    session_id      TEXT    NOT NULL,
    type            TEXT    NOT NULL,      -- 'manual' | 'decision' | 'bugfix' | ...
    title           TEXT    NOT NULL,
    content         TEXT    NOT NULL,
    tool_name       TEXT,
    project         TEXT,
    scope           TEXT    NOT NULL DEFAULT 'project',  -- 'project' | 'personal'
    topic_key       TEXT,
    normalized_hash TEXT,                  -- sha-ish of normalized content for dedupe
    revision_count  INTEGER NOT NULL DEFAULT 1,
    duplicate_count INTEGER NOT NULL DEFAULT 1,
    last_seen_at    TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    deleted_at      TEXT,
    -- Reserved for future use (currently unwritten):
    review_after          TEXT,
    expires_at            TEXT,
    embedding             BLOB,
    embedding_model       TEXT,
    embedding_created_at  TEXT,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);
```

Indexes ([store.go:628-631](../../../_engram-research/engram/internal/store/store.go#L628), [store.go:751-755](../../../_engram-research/engram/internal/store/store.go#L751)):

```sql
CREATE INDEX idx_obs_session ON observations(session_id);
CREATE INDEX idx_obs_type    ON observations(type);
CREATE INDEX idx_obs_project ON observations(project);
CREATE INDEX idx_obs_created ON observations(created_at DESC);
CREATE INDEX idx_obs_scope   ON observations(scope);
CREATE INDEX idx_obs_sync_id ON observations(sync_id);
CREATE INDEX idx_obs_topic   ON observations(topic_key, project, scope, updated_at DESC);
CREATE INDEX idx_obs_deleted ON observations(deleted_at);
CREATE INDEX idx_obs_dedupe  ON observations(normalized_hash, project, scope, type, title, created_at DESC);
```

The companion FTS5 virtual table ([store.go:633-642](../../../_engram-research/engram/internal/store/store.go#L633)):

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(
    title, content, tool_name, type, project, topic_key,
    content='observations',
    content_rowid='id'
);
```

The FTS5 table is kept in sync by three triggers `obs_fts_insert`, `obs_fts_delete`, `obs_fts_update` ([store.go:914-929](../../../_engram-research/engram/internal/store/store.go#L914)). **Save never writes to `observations_fts` directly** — the trigger handles it.

## 4. Storage write — `AddObservation`

Function: [store.go:1895-2035](../../../_engram-research/engram/internal/store/store.go#L1895). The actual write decision tree is:

### Step 4a — Pre-processing ([store.go:1897-1908](../../../_engram-research/engram/internal/store/store.go#L1897))
```go
p.Project, _ = NormalizeProject(p.Project)             // lowercase + trim
title   := stripPrivateTags(p.Title)                   // remove <private>...</private>
content := stripPrivateTags(p.Content)
if len(content) > s.cfg.MaxObservationLength {         // hard truncation
    content = content[:s.cfg.MaxObservationLength] + "... [truncated]"
}
scope    := normalizeScope(p.Scope)
normHash := hashNormalized(content)
topicKey := normalizeTopicKey(p.TopicKey)
```

### Step 4b — Topic-key upsert path ([store.go:1913-1958](../../../_engram-research/engram/internal/store/store.go#L1913))

If a `topic_key` was supplied, the latest matching observation in the same `(project, scope)` is found and **updated in place**:

```sql
SELECT id FROM observations
 WHERE topic_key = ?
   AND ifnull(project, '') = ifnull(?, '')
   AND scope = ?
   AND deleted_at IS NULL
 ORDER BY datetime(updated_at) DESC, datetime(created_at) DESC
 LIMIT 1
```

If a row matches:

```sql
UPDATE observations
   SET type = ?, title = ?, content = ?, tool_name = ?,
       topic_key = ?, normalized_hash = ?,
       revision_count = revision_count + 1,
       last_seen_at  = datetime('now'),
       updated_at    = datetime('now')
 WHERE id = ?
```

(`revision_count` increments — the topic gets versioned via this counter, not a new row).

### Step 4c — Dedupe path ([store.go:1960-1995](../../../_engram-research/engram/internal/store/store.go#L1960))

If no topic match (or no topic key was given), the function looks for an exact-content duplicate within the configured dedupe window (`s.cfg.DedupeWindow`):

```sql
SELECT id FROM observations
 WHERE normalized_hash = ?
   AND ifnull(project, '') = ifnull(?, '')
   AND scope = ?
   AND type  = ?
   AND title = ?
   AND deleted_at IS NULL
   AND datetime(created_at) >= datetime('now', ?)
 ORDER BY created_at DESC
 LIMIT 1
```

On hit, only `duplicate_count`, `last_seen_at`, `updated_at` are bumped — the row is reused, not duplicated.

### Step 4d — Fresh insert ([store.go:1997-2010](../../../_engram-research/engram/internal/store/store.go#L1997))

If neither path matched, a new row is inserted with a freshly-minted `sync_id`:

```sql
INSERT INTO observations
    (sync_id, session_id, type, title, content, tool_name, project, scope,
     topic_key, normalized_hash, revision_count, duplicate_count,
     last_seen_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 1, datetime('now'), datetime('now'))
```

`sync_id` is generated by `newSyncID("obs")` ([store.go:1997](../../../_engram-research/engram/internal/store/store.go#L1997)).

After insert, `review_after` may be backfilled from `decayReviewAfterMonths[type]` ([store.go:2015-2023](../../../_engram-research/engram/internal/store/store.go#L2015)). `embedding`, `embedding_model`, `embedding_created_at` are **never set on the write path**.

### Step 4e — Sync journaling ([store.go:2029](../../../_engram-research/engram/internal/store/store.go#L2029))

Every write path ends with `s.enqueueSyncMutationTx(...)` which appends a row to `sync_mutations` for downstream cloud sync. Skipped if the project is not enrolled.

The whole sequence runs inside a single `withTx` ([store.go:1911](../../../_engram-research/engram/internal/store/store.go#L1911)) so the FTS triggers see a consistent snapshot.

## 5. Embeddings — does `mem_save` generate any?

**No.** The schema reserves `embedding BLOB`, `embedding_model TEXT`, `embedding_created_at TEXT` ([store.go:789-791](../../../_engram-research/engram/internal/store/store.go#L789)) but `AddObservation` never writes to them. A grep for `embedding|vector|cosine` across `internal/` returns matches only in the schema reservation, a setup test, and an unrelated conflict-loop test — there is **no embedding code anywhere**. Search is FTS5/BM25 only (see `flow-mem-search.md`). Vectors are a planned future feature, not a shipped one.

## 6. `topic_key` handling — the upsert mechanism (we will replicate this)

Three things make `topic_key` work:

1. **Index for fast lookup**: `idx_obs_topic ON observations(topic_key, project, scope, updated_at DESC)` ([store.go:753](../../../_engram-research/engram/internal/store/store.go#L753)) — guarantees the upsert lookup is O(log n).
2. **Selection rule**: latest-`updated_at` then latest-`created_at` row in the same `(project, scope)` wins ([store.go:1921](../../../_engram-research/engram/internal/store/store.go#L1921)). Soft-deleted rows are excluded.
3. **Update semantics**: only `type`, `title`, `content`, `tool_name`, `topic_key`, `normalized_hash`, `revision_count`, `last_seen_at`, `updated_at` change. **`session_id`, `created_at`, and `sync_id` are preserved** ([store.go:1927-1944](../../../_engram-research/engram/internal/store/store.go#L1927)). That last one is critical for cross-machine sync — the topic stays the "same row" globally.

`topic_key` is normalized via `normalizeTopicKey` (lower-case, trim) so `Architecture/Auth-Model` and `architecture/auth-model` collapse together.

The MCP layer also surfaces a *suggestion* via `suggestTopicKey(type, title, content)` ([mcp.go:917](../../../_engram-research/engram/internal/mcp/mcp.go#L917)) when the caller did not provide one — pure helper, never auto-applied.

## 7. Sequence diagram

```mermaid
sequenceDiagram
    autonumber
    participant C as MCP Client (agent)
    participant T as mcp-go ServeStdio
    participant H as handleSave
    participant P as resolveWriteProject
    participant S as store.AddObservation
    participant DB as SQLite (observations + FTS triggers)

    C->>T: tools/call mem_save {title, content, type?, topic_key?, scope?, session_id?}
    T->>H: ctx, req
    H->>H: extract args from req.GetArguments()
    H->>P: detect project from cwd
    P-->>H: DetectionResult{project} or ambiguous_project err
    H->>H: NormalizeProject; default type=manual; default session_id; suggest topic_key
    H->>S: AddObservation(params)
    S->>S: stripPrivateTags + truncate to MaxObservationLength
    alt topic_key present and matches existing row
        S->>DB: SELECT existing by topic_key+project+scope
        DB-->>S: existingID
        S->>DB: UPDATE observations SET type/title/content/...,<br/>revision_count = revision_count+1
        DB-->>S: ok (FTS triggers refresh row in observations_fts)
    else duplicate within DedupeWindow
        S->>DB: SELECT by normalized_hash+project+scope+type+title
        DB-->>S: existingID
        S->>DB: UPDATE duplicate_count = duplicate_count+1
    else fresh insert
        S->>DB: INSERT INTO observations (sync_id=newSyncID, ...)
        DB-->>S: lastInsertID
        S->>DB: enqueue sync_mutations row (best-effort)
    end
    S-->>H: savedID, nil
    H->>S: GetObservation(savedID) → sync_id
    H->>S: FindCandidates(savedID) for conflict surfacing (REQ-001)
    S-->>H: []Candidate (often empty)
    H-->>T: respondWithProject(detRes, msg, {id, sync_id, judgment_required, candidates?})
    T-->>C: tools/call result envelope
```

## 8. Gotchas, surprises, things we'd do differently

1. **Surprise: `embedding` columns exist but are never written.** This is dead schema. Either commit to embeddings or remove the columns — they create a false impression of capability.
2. **Surprise: `mem_save` ignores any client-supplied `project` argument** (the schema doesn't even advertise one). Auto-detect-only is opinionated and breaks scripted seeding. We should keep auto-detect as the default but accept an explicit override for tooling.
3. **Surprise: silent truncation at `MaxObservationLength` with a magic suffix `"... [truncated]"`** ([store.go:1904](../../../_engram-research/engram/internal/store/store.go#L1904)). Thoughtline should reject oversized content with a clear error, OR split into chunks — silent truncation is a footgun that corrupts conflict-detection (truncated `normalized_hash` is misleading).
4. **Surprise: dedupe collapses on `(normalized_hash, project, scope, type, title)`** but ignores `session_id`. Two unrelated sessions saving the same content collide. Probably intentional, but worth flagging.
5. **Surprise: conflict candidate detection (`FindCandidates`) runs on every save and re-queries FTS5** ([mcp.go:987](../../../_engram-research/engram/internal/mcp/mcp.go#L987)). On a hot save loop this doubles the FTS5 work. We should make it opt-in via a flag.
6. **Surprise: `topic_key` upsert preserves `created_at` and `sync_id`** but clobbers `session_id` (no — it actually does NOT update `session_id`; the original session keeps the row). That's correct, just not obvious.
7. **Surprise: response payload is text-formatted, not structured JSON.** Handlers build a `strings.Builder` and return `mcp.NewToolResultText(string)` ([mcp.go:1030](../../../_engram-research/engram/internal/mcp/mcp.go#L1030)). Structured metadata is squeezed in via the `extra` map / `_meta` envelope. Thoughtline should consider returning structured `content` (e.g. JSON `application/json` block) so agents don't parse natural-language strings.
8. **Surprise: no per-tool rate limiting / size caps on the MCP server**. A noisy agent can flood the DB. We may want a simple in-process throttle.
9. **Surprise: `suggestTopicKey` runs even when one was provided** ([mcp.go:917](../../../_engram-research/engram/internal/mcp/mcp.go#L917)) — wasted work; trivial fix.
10. **Things we'd do differently for Thoughtline**:
    - Make embeddings a first-class column **only if** we actually compute them on save.
    - Reject oversized content with a clear error (no silent truncation).
    - Keep `topic_key` semantics 1:1 with Engram — they're load-bearing.
    - Return structured tool results (JSON content block) alongside the human-readable summary.
    - Make conflict surfacing opt-in (config flag), not always-on.
