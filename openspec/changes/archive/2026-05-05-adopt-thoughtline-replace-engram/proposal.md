# Proposal: Adopt Thoughtline, Replace Engram

> Phase: `sdd-propose`
> Change: `adopt-thoughtline-replace-engram`
> Date: 2026-05-04
> Inputs: `openspec/changes/adopt-thoughtline-replace-engram/exploration.md`

## Intent

Switch persistent AI memory from **Engram** (the vendored, soon-to-be-decommissioned tool) to **Thoughtline** (this repo) as the **primary and only** memory backend for Claude Code, while preserving every active observation (~291 rows across all projects) accumulated in `~\.engram\engram.db`.

Engram has served its purpose as a research scaffold; Thoughtline now has feature parity (9 MCP tools, FTS5 search, sessions, normalized hashing, TUI). Running both in parallel splits memory across two stores and forces dual upkeep on every save. We close that loop by migrating data once, repointing Claude Code at Thoughtline, and letting Engram go quiescent.

Success looks like: opening Claude Code in any project, calling `mem_search` (now backed by `tl_search`), and finding every memory that Engram used to return — same `sync_id`, same `topic_key`, same content — with zero loss of provenance.

## Scope

### In Scope

1. **Code — extend the type taxonomy** (`internal/memory/types.go`, `internal/memory/validate.go`)
   - Add `decision` and `architecture` to the closed type set (1:1 mapping from Engram).
   - Final type set (11 values): `game-design-decision | scene-pattern | asset-reference | perf-gotcha | pipeline-step | script-pattern | bugfix | convention | preference | decision | architecture`.
   - Update `Type.Valid()` and any switch statements that enumerate types.
   - Update colocated unit tests (`validate_test.go`).

2. **Code — migration binary** (`cmd/migrate/main.go`)
   - Standalone Go binary that reads `~\.engram\engram.db` (read-only, WAL-aware) and writes into `%LOCALAPPDATA%\thoughtline\thoughtline.db` via Thoughtline's `storage.Save()` (so FTS5 triggers, normalized hashing, and validation all fire correctly).
   - Skips soft-deleted rows (`WHERE deleted_at IS NULL`).
   - Maps Engram-only types `pattern | config | discovery | manual` to `convention` and stamps an `origin-type:<engram-type>` tag to preserve provenance.
   - Sets `session_id = NULL` (Engram session IDs are not UUIDv7 and would violate the FK).

3. **Docs**
   - Update `docs/design/memory-domain.md` — taxonomy table reflects 11 types.
   - Update `README.md` — taxonomy table + brief migration note + how to run `cmd/migrate`.

4. **Config — Claude Code only**, five files:
   | File | Action |
   |---|---|
   | `~\AppData\Roaming\Code\User\mcp.json` | Remove `engram` server entry (Thoughtline already configured alongside) |
   | `~\.claude\mcp\engram.json` | Replace with Thoughtline equivalent or delete |
   | `~\.claude\settings.json` | Replace 11 `mcp__plugin_engram_engram__*` allow-list entries with `mcp__thoughtline__tl_*` equivalents |
   | `~\.claude\CLAUDE.md` | Replace "Engram Persistent Memory — Protocol" block with Thoughtline equivalent |
   | `~\.claude\skills\_shared\engram-convention.md` | Replace with `thoughtline-convention.md` |

5. **Verification gate**
   - Post-migration smoke test via `thoughtline ui` Stats panel: memory count must equal pre-existing Thoughtline rows + active Engram rows.
   - Boot Claude Code in a project; confirm `tl_search` returns historical Engram memories via `sync_id` lookup.

6. **Rollback plan** — see dedicated section below.

### Out of Scope

- **Pre-publish TODOs** — Engram attribution audit, `_engram-research/` doc paths, `SECURITY.md`, GitHub issue templates, `CODE_OF_CONDUCT.md`, the missing `v0.0.1` git tag, and `CHANGELOG.md` `[Unreleased]` cleanup. Tracked separately under `publish-v0.0.1-prep`.
- **Cursor, Gemini, Codex, Copilot configs** — those clients keep their existing Engram wiring untouched in this change. They can migrate later under `adopt-thoughtline-other-clients`.
- **Engram binary uninstall** — left as a manual user step at the very end. The binary stays on disk, quiescent, until the user decides to remove it.
- **Storage migration v3** — the taxonomy extension is a domain-layer change (Go validator), NOT a SQL schema change. No new migration step required.

## Capabilities

### New Capabilities

- `memory-type-taxonomy`: the closed set of memory `type` values, the rules for adding/removing values, and the validator behavior. This change introduces the formal spec for the 11-value taxonomy.
- `engram-migration`: the `cmd/migrate` binary — its inputs (Engram DB path), outputs (Thoughtline DB rows), column mapping, type coercion rules, and idempotency guarantees.
- `claude-code-integration`: the contract between Thoughtline (MCP server) and Claude Code (allow-list, MCP config, CLAUDE.md protocol block, skills convention). Spec captures which files participate and what they must contain.

### Modified Capabilities

None. There are no existing `openspec/specs/` to delta against — this is the first change to populate that tree.

## Approach

**Migration (recommended Approach A from exploration):** Standalone Go binary at `cmd/migrate/main.go` that opens Engram's SQLite DB read-only, iterates `observations WHERE deleted_at IS NULL`, applies the column + type mapping table from `exploration.md`, and writes through Thoughtline's `storage.Save()` API. Approach B (raw SQL `ATTACH` + `INSERT`) is rejected because it bypasses FTS5 triggers and the normalized-hash logic — high data risk. Approach C (MCP-to-MCP) is rejected for being slow and brittle.

**Taxonomy extension (locked):** Adding `decision` and `architecture` to Thoughtline's closed type set is a simpler, more honest preservation strategy than coercing them to `convention` with origin tags. The other four Engram-only types (`pattern | config | discovery | manual`) still coerce to `convention` with `origin-type:*` tags — they're vague enough that forcing them into Thoughtline's domain-specific types would lie about the data.

**Sequencing (single-session execution):**

1. Build `cmd/migrate` against Thoughtline's local source.
2. Zip-backup `~\.engram\` to `~\.engram-backup-2026-05-04.zip`.
3. Run `cmd/migrate --dry-run` (flag TBD in design phase) against a copy of the Thoughtline DB; verify row counts.
4. Run `cmd/migrate` for real against production `thoughtline.db`.
5. Verify via `thoughtline ui` Stats panel.
6. Update the 5 Claude Code config files in order: `mcp.json` first (so the MCP server actually disappears from the client), then `settings.json` allow-list, then `engram.json` deletion, then `CLAUDE.md` protocol block, then `engram-convention.md` → `thoughtline-convention.md`.
7. Restart Claude Code; confirm boot is clean and `tl_search` finds historical memories.

Engram coexists with Thoughtline throughout — both `mcp.json` entries stay live until step 6 — so there is always a working memory backend during the transition.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `internal/memory/types.go` | Modified | Add `decision` and `architecture` to the closed type constants |
| `internal/memory/validate.go` | Modified | Update `Type.Valid()` switch to accept the two new values |
| `internal/memory/validate_test.go` | Modified | Add table cases for the two new types |
| `cmd/migrate/main.go` | New | Standalone migration binary |
| `cmd/migrate/*_test.go` | New | Unit tests for column mapper, type mapper, timestamp converter |
| `docs/design/memory-domain.md` | Modified | Taxonomy table updated to 11 values |
| `README.md` | Modified | Taxonomy table + migration usage section |
| `~\AppData\Roaming\Code\User\mcp.json` | Modified | Remove `engram` server entry |
| `~\.claude\mcp\engram.json` | Removed | Delete or rename |
| `~\.claude\settings.json` | Modified | Swap 11 `mcp__plugin_engram_engram__*` allow-list entries for `tl_*` equivalents |
| `~\.claude\CLAUDE.md` | Modified | Replace Engram protocol block with Thoughtline protocol |
| `~\.claude\skills\_shared\engram-convention.md` | Removed → Replaced | Becomes `thoughtline-convention.md` |
| `~\.engram\engram.db` | Read-only | Source of migration; never written |
| `%LOCALAPPDATA%\thoughtline\thoughtline.db` | Modified | Receives migrated rows via `storage.Save()` |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Type catalogue extension is a public taxonomy change | Med | Bump README, `docs/design/memory-domain.md`, and (later) `CHANGELOG.md`; document the addition explicitly so consumers see it |
| Coordinated edits across 4 Claude Code files drift if done piecemeal | Med | Tasks phase produces a single explicit checklist; verify-phase confirms each file post-edit |
| Engram WAL files (`.db-shm`, `.db-wal`) imply an active engram process | Med | Open Engram DB read-only with `_journal_mode=WAL` URI param; never lock the file |
| Claude Code allow-list rebuild prompts user once per new tool name on first use | Low | Pre-populate `settings.json` with all 9 `tl_*` tools so first run is silent |
| Type-coercion via `origin-type:*` tag loses some semantic richness | Low | Accepted tradeoff — locked decision; future change can extend taxonomy further if needed |
| Migration is non-idempotent if `sync_id` collisions are mishandled | Low | Use Thoughtline's `topic_key` upsert path or `sync_id` UNIQUE constraint; surface conflicts as warnings, not aborts |

## Rollback Plan

Before any destructive step, take a **zip backup** of `~\.engram\` to `~\.engram-backup-YYYY-MM-DD.zip`. This freezes the source-of-truth Engram state.

If migration or config swap fails:

1. **Data rollback:** Thoughtline's DB is unaffected by Engram migration if `cmd/migrate` errors out before writing (it processes rows in a single transaction per row, but failures abort cleanly). If partial writes happened, restore Thoughtline's DB from its own backup (the user is expected to have one) or re-run `cmd/migrate` — it must be idempotent on `sync_id`.
2. **Config rollback:** revert the 5 Claude Code config edits via git (`~\.claude\` and `~\AppData\Roaming\Code\User\` are user-managed and not in the Thoughtline repo, so the user keeps their own backup before edits — design phase will define the exact backup convention, e.g. `.bak` siblings).
3. **Engram resurrection:** unzip the backup, re-add the `engram` entry to `mcp.json`, restart Claude Code. Engram resumes as primary backend; Thoughtline migration data remains in `thoughtline.db` but is simply unused.

The point of no return is **deleting the Engram binary** — and that is explicitly out of scope for this change.

## Dependencies

- Thoughtline source builds locally (`go build ./cmd/thoughtline` and `go build ./cmd/migrate` must both work — but per project rule we do NOT actually build during development).
- Engram DB at `~\.engram\engram.db` is readable.
- Claude Code is installed and launched at least once so the config files exist.

## Success Criteria

- [ ] All ~291 active (`deleted_at IS NULL`) Engram observations are imported into `thoughtline.db` with `sync_id` preserved 1:1.
- [ ] Type mapping rules applied: `bugfix → bugfix`, `preference → preference` (forced `scope = personal`), `decision → decision`, `architecture → architecture`, `pattern | config | discovery | manual → convention` with `origin-type:<engram-type>` tag.
- [ ] `thoughtline ui` Stats panel shows memory count = pre-existing Thoughtline count + active Engram count.
- [ ] Claude Code session boots successfully with Thoughtline as the only memory backend (Engram entry removed from `mcp.json`).
- [ ] `tl_search` returns a known historical Engram memory (spot-check by `sync_id` or `topic_key`).
- [ ] Engram binary is still installed but quiescent — no `mcp.json` reference, no allow-list entry. User may uninstall at their discretion.
- [ ] All Go unit tests pass for the extended taxonomy (`internal/memory/`).
- [ ] `~\.engram-backup-YYYY-MM-DD.zip` exists on disk before any destructive config edit.

## Open Questions for Design Phase

These are deliberately deferred — `sdd-propose` decides WHAT, `sdd-design` decides HOW.

1. **`cmd/migrate` CLI surface** — flags like `--dry-run`, `--source`, `--dest`, `--verbose`, or hardcode default paths and accept zero arguments?
2. **Migration logging** — stdout only, a structured log file in `~\.thoughtline\migrate-YYYY-MM-DD.log`, or both?
3. **Engine-context type upgrade** — for Engram rows tagged with `playcanvas` or similar, should the migrator opportunistically upgrade `pattern → scene-pattern` / `script-pattern` instead of falling back to `convention`? Or keep the simple "all unmapped types → convention" rule for predictability?
4. **Idempotency strategy** — on re-run, skip rows where `sync_id` already exists in Thoughtline, or treat as conflict and surface to user?
5. **Config-edit safety** — `.bak` sibling files, git commit per file, or atomic write-then-verify?
