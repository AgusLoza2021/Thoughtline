# Exploration: adopt-thoughtline-replace-engram

> Phase: `sdd-explore`
> Change: `adopt-thoughtline-replace-engram`
> Date: 2026-05-04
> Engram artifact: `sdd/adopt-thoughtline-replace-engram/explore` (sync_id `obs-d044a4d5f3422eb4`)

## Current State

**Engram DB location (confirmed via filesystem investigation):**
`C:\Users\Agustin Lozano\.engram\engram.db` (WAL mode; `.db-shm` and `.db-wal` sidecar files present). The binary lives at `C:\Users\Agustin Lozano\AppData\Local\engram\bin\engram.exe` (plus an `.exe.old` from an upgrade). The `%LOCALAPPDATA%\engram\` directory contains ONLY the binary — the DB is NOT there.

**Engram schema** (`observations` table — source: `docs/research/flow-mem-save.md`):
`id, sync_id, session_id (TEXT FK → sessions.id), type (open set), title, content, tool_name, project, scope, topic_key, normalized_hash, revision_count, duplicate_count, last_seen_at, created_at (TEXT ISO), updated_at (TEXT ISO), deleted_at (TEXT ISO, nullable)`, plus reserved embedding columns never written.

**Engram type set** (open, not DB-enforced): `bugfix | decision | architecture | discovery | pattern | config | preference | manual`

**Thoughtline schema** (`memories` table — source: `internal/storage/schema.go`):
`id, sync_id (UUIDv7, UNIQUE), project, scope, type (CLOSED — 9 values), topic_key, title (≤200 chars), content (≤64 KiB hard reject), tags (JSON array), normalized_hash, revision_count, created_at (INTEGER unix-ms), updated_at (INTEGER unix-ms), deleted_at (INTEGER, nullable), session_id (TEXT FK, added in migration v2)`, plus reserved embedding columns.

**Thoughtline type set** (closed, enforced by `memory.Type.Valid()`):
`game-design-decision | scene-pattern | asset-reference | perf-gotcha | pipeline-step | script-pattern | bugfix | convention | preference`

**Thoughtline DB location (confirmed):** `C:\Users\Agustin Lozano\AppData\Local\thoughtline\thoughtline.db` (set by VSCode `mcp.json` env var `THOUGHTLINE_HOME`).

**VSCode `mcp.json` (confirmed):** Both engram AND thoughtline are configured side-by-side in `C:\Users\Agustin Lozano\AppData\Roaming\Code\User\mcp.json`.

---

## Affected Areas

### AI client config files requiring engram removal

| Path | What to do |
|---|---|
| `C:\Users\Agustin Lozano\AppData\Roaming\Code\User\mcp.json` | Remove `engram` server entry (engram + thoughtline + context7 currently coexist) |
| `C:\Users\Agustin Lozano\.cursor\mcp.json` | Remove `engram` server entry |
| `C:\Users\Agustin Lozano\.gemini\antigravity\mcp_config.json` | Remove `engram` server entry |
| `C:\Users\Agustin Lozano\.claude\mcp\engram.json` | Delete or archive |
| `C:\Users\Agustin Lozano\.claude\settings.json` | Remove `enabledPlugins.engram@engram`, remove `extraKnownMarketplaces.engram`, replace 11 `mcp__plugin_engram_engram__*` allow-list entries with thoughtline equivalents |
| `C:\Users\Agustin Lozano\.claude\CLAUDE.md` | Replace `<!-- gentle-ai:engram-protocol -->` block with Thoughtline protocol |

### Engram skill/protocol docs (replace with thoughtline equivalents)

- `~/.codex/engram-instructions.md`
- `~/.codex/engram-compact-prompt.md`
- `~/.claude/skills/_shared/engram-convention.md`
- `~/.cursor/skills/_shared/engram-convention.md`
- `~/.gemini/skills/_shared/engram-convention.md`
- `~/.copilot/skills/_shared/engram-convention.md`
- `~/.codex/skills/_shared/engram-convention.md`

### New code to create in thoughtline

- `cmd/migrate/main.go` — standalone migration binary (engram.db → thoughtline.db)

---

## Schema Column Mapping

Engram `observations` row → Thoughtline `memories` row:

| Engram column | Thoughtline column | Action |
|---|---|---|
| `sync_id` | `sync_id` | COPY — preserve cross-machine key |
| `session_id` | `session_id` | **SET NULL** — engram session IDs are not UUIDv7; cannot pass FK to `sessions(id)` |
| `type` | `type` | MAP via type table (see below) |
| `title` | `title` | COPY; truncate to 200 chars if needed (log) |
| `content` | `content` | COPY; if >64 KiB, truncate at boundary and log; thoughtline rejects oversized |
| `tool_name` | (none) | DROP |
| `project` | `project` | COPY; normalize lowercase |
| `scope` | `scope` | COPY; validate `project`/`personal` |
| `topic_key` | `topic_key` | COPY; validate against thoughtline regex or set NULL if invalid |
| `normalized_hash` | `normalized_hash` | RECOMPUTE via thoughtline's hash function |
| `revision_count` | `revision_count` | COPY |
| `duplicate_count` | (none) | DROP |
| `last_seen_at` | (none) | DROP |
| `created_at` (TEXT ISO) | `created_at` (INT unix-ms) | CONVERT: `time.Parse(...) → .UnixMilli()` |
| `updated_at` (TEXT ISO) | `updated_at` (INT unix-ms) | CONVERT |
| `deleted_at` (TEXT ISO, null) | `deleted_at` (INT unix-ms, null) | CONVERT; NULL stays NULL; decide whether to migrate soft-deleted rows |
| `embedding*` | `embedding*` | COPY (both NULL in practice) |

---

## Type Mapping Table

| Engram type | Recommended thoughtline type | Confidence | Tag to add |
|---|---|---|---|
| `bugfix` | `bugfix` | HIGH | none |
| `preference` | `preference` | HIGH | force `scope = personal` |
| `decision` | `convention` | MEDIUM | `origin-type:decision` |
| `architecture` | `convention` | MEDIUM | `origin-type:architecture` |
| `pattern` | `convention` | MEDIUM | `origin-type:pattern` (or `script-pattern`/`scene-pattern` if engine context detectable) |
| `config` | `convention` | LOW | `origin-type:config` |
| `discovery` | `convention` | LOW | `origin-type:discovery` |
| `manual` | `convention` | MEDIUM | `origin-type:manual` |

Tags use the format `origin-type:<engram-type>` to preserve provenance after coercion into the closed type system. **Open architectural question for the proposal**: accept this coercion or extend thoughtline's type catalogue with `decision` and `architecture` (1:1 mapping)?

---

## Approaches

| Approach | Pros | Cons | Effort |
|---|---|---|---|
| **A. Go migration binary** (`cmd/migrate/main.go`) | Uses Thoughtline's `storage.Save()` → FTS triggers, validation, topic_key upsert all work correctly; idempotent; testable; full per-row error reporting | New binary in repo; one-row-at-a-time (fine for ~291 rows) | **Medium** |
| **B. SQL `ATTACH` bulk INSERT** | Fast; no Go code; done in minutes | Bypasses FTS5 triggers and normalized_hash logic; no validation; brittle; not a repeatable tool | Low effort, **HIGH DATA RISK** |
| **C. engram MCP → thoughtline MCP via tl_save** | Uses MCP tool natively | Requires `engram serve` running; very slow; MCP is text-based; complex plumbing | High |

---

## Recommendation

**Approach A** — standalone Go migration binary at `cmd/migrate/main.go`.

It's the only approach that passes rows through Thoughtline's own storage layer, ensuring the FTS5 index stays consistent, normalized hashes are correct, and type validation fires on every row. ~291 rows at ~1 ms each = under a second total.

**Migration sequence:**

1. Build `cmd/migrate` binary.
2. Run against a COPY of `thoughtline.db` first; verify row count matches.
3. Run against production `thoughtline.db`.
4. Verify with `thoughtline ui` — Stats panel should show ≥291 memories.
5. Update all AI client configs (remove engram entries, add/update thoughtline permissions).
6. Replace CLAUDE.md Engram protocol section with Thoughtline protocol.
7. Archive `~\.engram\` to a zip (safety backup).
8. Optionally uninstall `engram.exe`.

---

## Pre-publish TODOs

From `docs/PROGRESS.md`: 5 remaining items (Engram attribution verification, replace `_engram-research/` relative paths in docs, SECURITY.md, GitHub issue templates, CODE_OF_CONDUCT.md). Additionally `v0.0.1` tag was never created despite commit `6cab6b9` being titled "v0.0.1 — Foundation release"; `CHANGELOG.md` still shows `[Unreleased]`.

**Recommendation:** NONE of these block migration. Keep as a separate change (`publish-v0.0.1-prep`). Bundling would bloat this change's scope.

---

## Risks

1. **Engram session_id format incompatibility** — engram session IDs are text (likely `default-{project}` style), not UUIDv7. Cannot be foreign-keyed into thoughtline's `sessions` table. Safe resolution: set `session_id = NULL` for all migrated rows.
2. **Type coercion loses fidelity** — 6 of 8 engram types collapse to `convention`. Tag strategy (`origin-type:*`) mitigates but doesn't fully restore. The proposal must decide: extend thoughtline's type catalogue or accept the coercion.
3. **Thoughtline's CLOSED type validator** — migration binary MUST map every row before calling `storage.Save()`; any unmapped type causes an error.
4. **Timestamp format mismatch** — engram `TEXT` ISO → thoughtline `INTEGER` unix-ms. Off-by-timezone parsing would silently corrupt chronology. Use `time.UTC()` consistently.
5. **Claude Code `settings.json` allow-list** — has 11 hardcoded `mcp__plugin_engram_engram__*` entries. Removing engram from `mcp.json` without updating the allow-list leaves stale permissions (benign), but more importantly, thoughtline's tool names (`tl_save`, `tl_search`, etc.) are NOT in the allow-list yet — they'll prompt on first use.
6. **Thoughtline binary lives in `Desktop\thoughtline\thoughtline.exe`** (dev path). `mcp.json` already points there. Fine for local use, document for clarity.
7. **WAL files** — `engram.db-shm` and `engram.db-wal` are present (engram is or was recently running). Migration tool must open `engram.db` read-only with WAL mode to ensure consistent reads.
8. **Engram referenced in 7+ config/skill files across 5 AI clients** — scope is larger than just VSCode. Each client needs its own update. This is config-edit work, not code work.

---

## Ready for Proposal

**Yes.** All inputs known. Proposal phase must define:

1. Migration binary spec (`cmd/migrate`) with full column/type mapping.
2. Config update sequence for each AI client (order: migrate first, then remove engram configs).
3. Rollback plan (engram archive + re-adding engram to `mcp.json` if migration fails).
4. **Architectural decision**: type coercion + `origin-type:*` tags vs. extending thoughtline's type catalogue with `decision` and `architecture` (1:1 mapping).
5. Decision on whether to migrate soft-deleted rows from engram or skip them.
