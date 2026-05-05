# Example — what a saved memory looks like

This is the JSON shape `tl_save` receives. Use it as a reference when teaching new contributors what "good" looks like.

## A decision

```json
{
  "title": "Use WAL journal mode for all SQLite connections",
  "type": "decision",
  "topic_key": "decision/sqlite-wal",
  "project": "thoughtline",
  "scope": "project",
  "tags": ["platform:windows", "storage"],
  "content": "**What**: Enable WAL via DSN pragma for every storage.Open call.\n\n**Why**: Concurrent MCP tool calls need readers + writer simultaneously. Default rollback journal serializes everyone behind a single mutex.\n\n**Where**: internal/storage/storage.go — Open() appends `?_journal_mode=WAL&_busy_timeout=5000` to the DSN.\n\n**Learned**: busy_timeout=5000 is the sweet spot — lower causes spurious BUSY under load, higher just hides slow queries."
}
```

## A bugfix

```json
{
  "title": "FTS5 contentless table breaks UPDATE; switch to non-contentless",
  "type": "bugfix",
  "topic_key": "bug/storage/fts5-contentless",
  "project": "thoughtline",
  "scope": "project",
  "content": "**What**: Replaced FTS5 contentless table with a managed contentful index synced via triggers.\n\n**Why**: Contentless FTS5 doesn't allow UPDATE — re-indexing a memory required DELETE+INSERT, which lost the rowid. tl_update was effectively broken.\n\n**Where**: migrations/0003_fts_managed.sql, internal/storage/search.go.\n\n**Learned**: SQLite docs are explicit about this but easy to miss — `external content` tables are the safer default."
}
```

## A convention

```json
{
  "title": "Topic-key naming: category/area/subject lowercased",
  "type": "convention",
  "topic_key": "convention/topic-keys",
  "project": "thoughtline",
  "scope": "project",
  "content": "**What**: All topic_keys lowercased, slash-separated, no spaces. Pattern: `<category>/<subject>` for top-level, `<category>/<area>/<subject>` when scoping is useful.\n\n**Why**: GLOB lookups (`decision/*`, `bug/storage/*`) only work when keys follow a stable shape.\n\n**Where**: enforced in code at internal/memory/topickey.go; documented in plugin SKILL.md and main README.\n\n**Learned**: kebab-case wins over snake_case here — the URL feel matches the GLOB feel."
}
```

## What NOT to save

A memory is not a journal. These examples are anti-patterns:

```jsonc
// ❌ Too vague
{ "title": "Worked on the dashboard", "content": "Made changes" }

// ❌ Restating the diff
{ "title": "Renamed foo to bar", "content": "Renamed foo to bar in 12 files" }

// ❌ Ephemeral state
{ "title": "Test passing", "content": "All tests are passing right now" }
```

Memories should answer: *what would Future Me thank Past Me for writing down?*
