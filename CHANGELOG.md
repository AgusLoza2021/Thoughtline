# Changelog

All notable changes to Thoughtline will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) starting at the M1 release.

## [Unreleased]

> **This is not a release plan.** Thoughtline is retired; no release will ever carry any of this.
> The entries below separate three different things: changes that landed on `main` and were
> never released, work that was **parked mid-feature** before it was finished, and the
> decisions that ended the project. Every parked entry says so in its own heading. Nothing
> here is a promise, and no date below is a release date.

### Docs — the adoption path stops being homework (2026-09-29)

The README's step 3 told the reader to "put the type list into whatever your editor calls
project instructions", and the repository shipped no such file. `docs/design/memory-domain.md`
went further and named that step as *the* thing that makes the vocabulary real — "these rules
are a contract with your agent, not a gate" — so the enforcement mechanism the design doc
depended on was a document the user was expected to write from scratch. The retired v0.1.0
server did carry one: `serverInstructions` in `internal/server/server.go` primes the model on
the vocabulary, the `topic_key` patterns and proactive saving. The successor dropped it.

[`presets/AGENTS.md`](presets/AGENTS.md) is that artifact, restored as a file you copy into your
own project: when to save, the eleven types with their `topic_key` patterns, the four-line
content shape, where tags actually live now that Engram has no tags field, and the hard rules
with what each one costs when it drifts. It is what README step 3 links to, and what
`memory-domain.md` points at.

Three live claims were corrected alongside it: the README said the retired server shipped
"twelve `tl_*` tools" when `internal/server/server.go` registers fifteen; `CONTRIBUTING.md` sent
contributors to `docs/PROGRESS.md`, which carries the retirement marker; and its "Local setup"
built the archived engine with no note that it is archived.

### Docs — the memory protocol stops restating Engram, which is why it was wrong (2026-09-28)

`docs/integrations/claude-code-protocol.md` opened by claiming to be "the canonical Thoughtline
memory protocol block for AI assistants" and then quoted another project's tool mechanics from
memory. That is a structural problem rather than a proofreading one: a hand-maintained copy of
someone else's interface has no mechanism behind it, so it drifted. Four of its mechanics were
wrong against Engram `3ba7df6` (2026-09-28): `scope` was described as having two values when
`normalizeScope` accepts `personal` and `global` and folds everything else to `project`; `type`
was written up as a constraint Engram enforces nowhere; search was taught as `mem_search` first
instead of Engram's `mem_context` → `mem_search` → `mem_get_observation`; and results were
undersold as "snippet and metadata" when `state` and the relation annotations are meant to be
acted on. The same stale `scope` clause sat byte-identical in seven files.

The page now teaches what this repository owns — the vocabulary and the behaviour that follows
from it — and points at Engram's canonical protocol for the mechanics. Behaviour is stated once:
`docs/AGENT-SETUP.md` holds the canonical drop-in block, and the six per-editor guides keep a
paste-ready block of vocabulary only, each one naming the canonical block and why it is not
repeated there.

- Capabilities that were absent or unmentioned are now documented: `mem_update` and
  `mem_suggest_topic_key` (correcting and evolving a memory instead of saving it twice),
  `mem_delete`, `mem_pin`/`mem_unpin`, `mem_review`, `mem_doctor`, conflict resolution through
  `mem_judge`/`mem_compare`, passive capture via a `## Key Learnings:` section, and
  `capture_prompt: false` for automated saves.
- **Decided, and it does not change the vocabulary**: `decayReviewAfterMonths`
  (`internal/store/store.go:380`) assigns a review horizon to exactly `decision`, `policy` and
  `preference`. Nine of this vocabulary's eleven types therefore have no review horizon and
  never appear in `mem_review`. The fix is not to reshape the type list to fit someone else's
  provisional map — the request goes upstream, to extend that map or make the decay policy
  configurable.
- [ADR 0007 — this repository owns the vocabulary, not the tool mechanics](docs/decisions/0007-vocabulary-not-mechanics.md)
  writes the ownership boundary down, pins the Engram revision its claims were read from, and
  records that decision with its verified costs: option (b) spends the differentiator, option
  (c) rests on `mem_doctor`, whose ten checks cannot see a stale memory at all.
- **Scope**: local memory only. Nothing networked or multi-user is documented, and every
  capability named here is local to one machine.
- **Why**: the vocabulary is the product; the mechanics belong to Engram. A page that prints no
  mechanics cannot disagree with them, so the class of drift that produced these four wrong
  claims is closed rather than re-corrected.

### Removed — the retired server is no longer distributed (2026-09-28)

- `.github/workflows/release.yml` — the only publishing path (a `v*.*.*` tag running
  GoReleaser) — is gone, so no further release can be cut.
- The Claude Code plugin manifests are gone: `plugin/claude-code/.claude-plugin/plugin.json`
  (what made the plugin installable at all), `.claude-plugin/marketplace.json` (the listing)
  and `plugin/claude-code/.mcp.json` (which registered `thoughtline serve` in a client).
- `SECURITY.md` no longer describes a live local-first server, nor promises an
  acknowledgement within 5 business days and a fix within 30 days. It states the retired
  status and keeps only the two threat classes still worth reporting here: the frozen code,
  and the automation around it.
- **Why**: the repository declared the v0.1.0 MCP server retired while still publishing,
  listing and supporting it. Those four things were the only parts that actively did so.
- **Deliberately kept**: `scripts/install.{ps1,cmd}` and `.goreleaser.yaml` still exist and are
  still linked from the docs — including from inside the archived legacy blocks in
  `docs/integrations/`. The installers run `go install ...@latest` against a module whose code
  is still present, so their reach is unchanged: what changed is publication, not reach.
- `ci.yml` and `plugin.yml` also stay — a build check and a protocol-contract check, not
  distribution. They keep the archived code compiling and its contract tested.

### Docs — the retired server is marked consistently (2026-09-28)

Sixteen Markdown pages carried a `<!-- retired-v0.1.0 -->` banner, but the repository read as
if the v0.1.0 MCP server were still current: six of those banners did not fix the page body,
three pages the audit never covered had no banner at all, and three claims were simply false.

The banner now names what replaced the server and each page whose body *is* the old
documentation closes with a `Legacy` notice saying that what follows is a record, not steps to
follow. Pages that keep a live half — `docs/AGENT-SETUP.md`, `docs/integrations/*` — put that
half in front of the notice. Three pages the audit had missed are now covered:
`docs/media/README.md`, `cmd/migrate/README.md` and `docs/pitch/why-thoughtline.md` (which
keeps its Spanish, as that page always has).

The ten specs under `openspec/specs/` — the project's current contract — carry the marker too.
They needed three different notices rather than one. Seven describe the retired engine outright
(`brain-domain`, `claude-code-integration`, `cognitive-config`, `engram-migration`,
`event-bus`, `memory-graph`, `passive-capture`). `memory-type-taxonomy` is half current: the 11
type values are the product this repository still owns, and only the `Type.Valid()` enforcement
belongs to the retired server. `tui-memory-workspace` and `tui-removed` are neither: they
describe code that landed on `main` and was never shipped — `v0.1.0` predates it.

Three false claims are corrected in the same pass:

- `docs/COMPARISON.md` no longer says the `check-no-claude-mem` scripts "still run in CI".
  They were deleted with the AGPL firewall; the page says so and points at `CONTRIBUTING.md`.
- `docs/decisions/0002-search-strategy-fts5-first.md` cited three `_engram-research` line
  numbers that no longer pointed where the ADR said they did, and called one of them "the
  FTS5 virtual table" when the line is `return nil, err`. The claims are corrected in an
  appended `## Amendment`, each scoped to the revision it was verified at — including the
  `bm25()` citation, which was true of that snapshot and is not true of Engram today.
- `docs/research/*.md` keep their bodies: they are dated snapshots, not current
  documentation. Each now ends with a `## What changed since this snapshot` note against
  Engram's current `main`, which also fixes an undercount ("up to 16 tools") that was already
  wrong when the snapshot was taken.

### Removed — the AGPL firewall (2026-09-24)

- `scripts/check-no-claude-mem.{sh,ps1}` and their two CI steps are gone, along with the
  CONTRIBUTING section and the mandatory PR affirmation they backed.
- **Why**: the apparatus existed to protect against claude-mem's AGPL-3.0 licence, which
  does not exist. That project's first licence was a custom permissive one with MIT terms
  for `/hooks`, and it is Apache-2.0 today. Neither has ever mentioned the AGPL.
- The `0.1.0` entry below is left exactly as written — it describes a guard that did exist
  on that date. This entry records the guard's removal.

### Changed — the project is repositioned (2026-09-24)

Thoughtline is no longer a memory server. It is now the **gamedev memory vocabulary** for [Engram](https://github.com/Gentleman-Programming/engram): the types, the tags, and the engine-specific guidance for adopting them.

The v0.1.0 Go MCP server — twelve `tl_*` tools, the SQLite schema, the Bubbletea dashboard and the `cmd/migrate` migrator — is **retired and unmaintained**. Nothing has been deleted: the code and its documentation remain in the repository, with the original README preserved verbatim in an appendix.

**Why.** Every differentiator this project claimed turned out to be either absorbed upstream or never an engine feature at all. Engram now ships the architectural pieces Thoughtline built (a TUI, an HTTP API, sessions), and it stores an observation's `type` as a **free-form string** — so this taxonomy runs on Engram unmodified, with no code to maintain. Competing on the engine meant one developer racing a team that ships continuously. Owning the vocabulary competes with nobody.

**Impact.** Your data is unaffected — and do **not** migrate memories *into* the retired server; keep them in Engram. The in-flight `storage-caps` and TUI work below is parked, and `main` is not a supported upgrade path.

### Storage caps (`storage-caps`) — **parked mid-feature, never released**

The first of two changes in the `architecture/token-economy` goal, which set out to cut the
number of tokens a session spends on memory. This half bounded how much a single save can
write and how much a single read can return. Phases 0–3 landed on `main` on 2026-05-15;
**phase 4 — the integration test and the closing commit — never landed**, so the change was
never verified, never archived, and produced no `openspec/` artifacts. Surviving `go test`
coverage and the three commits below are the entire record of it.

- **Observation cap** — `memory.MaxObservationChars = 50000`, enforced in `internal/server`
  before validation: content over the limit is cut on a rune boundary and gets a
  `…[truncated by Thoughtline at 50000 chars]` marker, so stored content is always valid
  UTF-8 and exactly the cap in runes. It coexists with the pre-existing 64 KiB
  `memory.MaxContentBytes` byte ceiling rather than replacing it.
- **Dedupe window** — `storage.DedupeWindow = 15 * time.Minute` plus `Storage.DedupeCheck`:
  a save that carries no `topic_key` and whose SHA-256 over `trim(title) + NUL + trim(content)`
  already exists for the same brain returns the existing row instead of inserting a second one.
  It runs only when `topic_key == ""`, so upsert semantics are untouched.
- **Schema v6** — partial covering index `idx_memories_hash` on
  `(brain_id, normalized_hash, created_at) WHERE deleted_at IS NULL`. Additive and idempotent;
  applied automatically on first open.
- **Context budget** — `server.ContextResponseMaxChars = 4000` for `tl_context`. Snippets
  strip oldest-first, then whole entries drop oldest-first, and the newest entry is never
  dropped (the newest-entry floor), so its ID always survives for post-compaction recovery.

**Why it stopped.** The work was sound but half-shipped; it stopped because the project did.
`main` is not an upgrade path, so finishing phase 4 would have added tests for a feature of a
retired server. The resume checkpoint survives in Engram at `sdd/storage-caps/state`; the
second change, `slim-inject`, was never started.

### Removed (BREAKING) — TUI flags (`tui-memory-workspace`) — **parked, never released**
- `--theme {brand|zbrush|mono}` CLI flag — single semantic palette replaces the multi-theme system
- `--no-splash` CLI flag — splash screen removed
- `--splash-ms N` CLI flag — splash duration setting removed

Invoking any of the removed flags prints a friendly migration error to stderr and exits with code 2:
```
thoughtline ui: --theme was removed in v0.2 (single semantic palette). See CHANGELOG.md.
```

### Added — Tabbed Memory Workspace TUI (`tui-memory-workspace`) — **parked, never released**
- 6 tabs (Home, Memories, Search, Inbox, Sessions, Help) reachable with digit keys 1-6 or `tab` / `shift+tab`
- Global Quick Actions hotkeys: `[S]` save (CLI/MCP hint), `[/]` search, `[M]` memories, `[I]` inbox
- Inbox tab with `[A]` accept, `[E]` edit-then-promote, `[R]` reject for pending captures from `tl_capture`
- Memory Detail content now wraps at viewport width and scrolls with `↑/↓` + `PgUp/PgDn`
- `[C]` copy-to-clipboard from Memory Detail (cross-platform: `clip.exe` on Windows, `pbcopy` on macOS, `wl-copy` / `xclip` on Linux)
- Friendly empty state with gamedev-type examples (scene-pattern, perf-gotcha, pipeline-step) when no memories exist
- Help tab with a single-source-of-truth keybindings registry and roadmap rendered from `roadmap.yaml`
- Drift test guards `keybindings.go` against silent divergence from `flat_model.go`'s Update ladder

### Changed — TUI (`tui-memory-workspace`) — **parked, never released**
- Single semantic palette replaces the multi-theme system: cyan/blue = navigation, purple `#C4A7E7` = brand, green = success, yellow = warn, red = error, gray = meta. Carry-over: the gamedev-purple Brand color is preserved from the predecessor Rose-Pine-Moon palette.
- Brand surface: text `🧠 Thoughtline` (with the muted tagline `Local memory for game projects`) replaces the block-letter ASCII art
- Project Health card on the Home tab now shows GLOBAL counts across all projects (previously scoped to the cwd basename which caused 0-count bugs in v0.1.0)

### Storage (`tui-memory-workspace`) — **parked, never released**
- New `storage.MarkRejected(ctx, id)` method backing the inbox `[R]` reject action
- Schema migrated to v5 (adds `rejected` to the `pending_events` status CHECK constraint) — runs automatically on first start and preserves all existing data

## [0.1.0] - 2026-05-07

This is the first numbered release. It folds in the M5 launch-readiness work (TUI overhaul, distribution, plugin) plus two structured changes built under SDD: passive capture from Claude Code hooks, and the v2 workstation-style TUI.

### Added — Passive capture (`passive-capture-hooks`)
- New SQLite table `pending_events` (schema v3) with UNIQUE on `(project, event_hash)` for idempotent inserts
- `thoughtline hook <event-name>` subcommand — reads JSON from stdin, fail-silent, never breaks the host Claude Code session
- `thoughtline worker` subcommand — retention janitor with archive + hard-delete windows (default 7d / 30d)
- 3 new MCP tools: `tl_pending_list`, `tl_pending_get`, `tl_promote` (per-event partial-success batch)
- Plugin integration — `plugin/claude-code/hooks.json` registers all 6 Claude Code hook events
- Opt-in only via `THOUGHTLINE_PASSIVE_CAPTURE=1`; default OFF
- License hygiene: `scripts/check-no-claude-mem.{sh,ps1}` blocks accidental copy of AGPL strings
- ADR 0004 documents the design and the known v1 limitation around the `tl_promote` non-atomic seam

### Added — TUI v2 workstation (`tui-redesign`)
- Welcome screen (engram-inspired) — ASCII logo, stat card, 5-action menu
- Workstation screen — 3-pane layout (sidebar / center / right detail) with operational top bar and bottom status line
- 4 new drill-in screens: SearchScreen, RecentScreen, BrowseProjectsScreen / Projects, PendingScreen, DetailScreen
- Screen-stack navigation (`Screen` interface, push/pop semantics) with vim-style keys (`hjkl`, `gg`, `G`, `/`, `r`)
- Single Rose-Pine-Moon adapted palette; multi-theme infrastructure deprecated
- Cross-platform disk-free measurement (`diskfree_unix.go` / `diskfree_windows.go`)
- New storage methods backing the workstation: `CountPending`, `MostRecentProjects`, `RecentAll`, project-scoped recent

### Fixed
- `q` no longer pops out of `SearchScreen` while typing — queries containing the letter `q` now work; `esc` is the universal back
- `tl_search` license hygiene script no longer trips on legitimate doc references

### Added — Public-launch readiness

This batch lands the work needed to flip the repo public: cross-platform plugin, Claude Code marketplace metadata, brand polish, distribution, and a Bubbletea TUI restyle.

#### Distribution & repo hygiene
- **GoReleaser config** (`.goreleaser.yaml`) — cross-compile for linux / darwin / windows × amd64 / arm64, tar.gz / zip archives, checksums, version + commit + date injected via ldflags.
- **Release workflow** (`.github/workflows/release.yml`) — fires on `v*.*.*` tags, runs GoReleaser, publishes artifacts.
- **`.gitignore` tightened** — explicit ignores for bare-name binaries (`/thoughtline`, `/thoughtline-exe`). The tracked Linux binary was untracked in the same change.
- **Community files** — `SECURITY.md` (private vulnerability reporting), `.github/ISSUE_TEMPLATE/{bug_report.yml,feature_request.yml,config.yml}`, `.github/PULL_REQUEST_TEMPLATE.md`.

#### Dashboard TUI overhaul
- **Theme system** (`internal/dashboard/themes.go`) — three palettes shippable on day one:
  - `brand` — violet + cyan, the original tech-SaaS look
  - `zbrush` — warm tactile palette inspired by Pixologic ZBrush (`#D68A3C` amber on `#2D2A26` warm-dark, `#E8DCC4` cream text)
  - `mono` — minimalist grayscale for screenshots and slides
  Switchable at runtime with the `[t]` hotkey or via `--theme {brand|zbrush|mono}`. `ApplyTheme(t)` rebuilds every package-level style on swap.
- **Cockpit status bar** (`renderStatusBar`) — single line at the top of every tab: `◆ THOUGHTLINE ONLINE · MEM N · SESSIONS N · vX.Y.Z`. When an update is available, an amber pill appears: `↑ vX.Y.Z available · [u] open release`.
- **Background update check** (`internal/dashboard/updatecheck.go`) — non-blocking GitHub releases lookup with 3s timeout. `[u]` opens the release URL in the user's browser cross-platform (`rundll32` / `open` / `xdg-open`). Toggleable via `--no-update-check`.
- **Engram-style stats** — right-aligned numerals in `renderStatsPanel`. Bold brand-colored numbers, muted labels.
- **Animated header cube** (`internal/dashboard/cube.go`) — 3D wireframe ASCII cube that oscillates through 5 frames. Plain ASCII (CMD-safe) and "fancy" box-drawing modes. 220ms per frame default.
- **Splash screen** (`internal/dashboard/splash.go`) — cube + ASCII wordmark + tagline, vertically centered with `lipgloss.Place`. Auto-dismiss after 1.5s (configurable). Skip with `--no-splash`.
- **CLI flags** — `--theme`, `--no-splash`, `--no-update-check`, `--splash-ms N` on `thoughtline ui`.
- **Help tab** — new rows for `[t]` (cycle theme) and `[u]` (open release).
- **Hero composition** — Overview tab uses `lipgloss.JoinHorizontal` to stack the cube next to the brand pill, tagline, and breadcrumb metadata.

#### Claude Code plugin
- **`thoughtline protocol` subcommand** (`cmd/thoughtline/protocol.go`) — single source of truth for the active-protocol markdown. Replaces the duplicated heredocs in three different files. Cross-platform (Go binary, no shell needed). Flags: `--event {session-start|post-compaction}`, `--project NAME`, `-o FILE`. Versioned via `Protocol-Version: 1` header so plugins can detect drift.
- **`plugin/claude-code/` overhaul** — the plugin now ships:
  - `hooks/hooks.json` — calls `thoughtline protocol` directly, no bash. Adds `resume` to the matcher so re-opening a project re-injects the protocol.
  - `commands/{tl-recent, tl-search, tl-stats, tl-ui, tl-export}.md` — slash commands surfaced in autocomplete.
  - `agents/tl-archivist.md` — specialist subagent for memory hygiene (dedup, prune, audit). Read-mostly, never deletes without explicit confirmation.
  - `examples/{saved-memory.md, session-transcript.md}` — calibration set for what "good" memories look like and what a real session flow looks like.
  - `skills/memory/SKILL.md` — extended with `## Examples` block (save triggers, search triggers, NOT-a-trigger).
  - `.claude-plugin/plugin.json` — bumped with `repository`, `homepage`, `keywords`, `protocolVersion`. Version synced to the binary's (`0.0.1`).
  - `.claude-plugin/marketplace.json` — listing manifest for Claude Code plugin marketplaces (publisher, displayName, summary, tags, requirements).
  - `LICENSE` — MIT, copied from the repo root so the plugin directory is legally self-contained.
  - `README.md` — install, prerequisites, file layout, configuration, uninstall.
- **Bash scripts removed** — `plugin/claude-code/scripts/{session-start.sh, post-compaction.sh, _helpers.sh}`. The binary subcommand replaces them; Windows users no longer need WSL or Git Bash.

#### Plugin CI
- **`.github/workflows/plugin.yml`** — JSON validation (`jq -e .` on every plugin JSON file), front-matter check on commands and agents, build of the binary, smoke test of `thoughtline protocol` for both events, and a guard that fails CI if `hooks.json` ever references a `.sh` script again.

#### Repo presentation
- **README.md hook section** — `Quick start` block at the top with `go install`, MCP JSON config, `thoughtline ui`, and a Claude Code plugin install one-liner.

#### Game-dev launch polish
- **"For game devs, in 60 seconds"** hook section above the fold — pain-point list (re-explaining hierarchies, import settings, batching gotchas) followed by the value prop in three lines.
- **Comparison table** — Thoughtline vs Engram vs Cursor memories vs ChatGPT memory vs manual notes. Honest tradeoffs; explicit "use Engram if you don't ship games" callout.
- **`docs/TOOLS.md`** — full parameter reference + 10 end-to-end examples extracted from README (the README dropped from 704 → ~330 lines).
- **`docs/design/tag-conventions.md`** — canonical tag vocabulary: engine, platform, pipeline, asset, phase, tooling, performance buckets. Examples per category.
- **`docs/integrations/cursor.md`** — wiring Thoughtline into Cursor (MCP config + `.cursorrules` snippet).
- **`docs/integrations/zed.md`** — wiring into Zed's assistant context server config.
- **`docs/integrations/rider-unity.md`** — JetBrains Rider for Unity, with a "day 1 saves" cheat sheet (folder layout, MonoBehaviour conventions, Android asset bundles).
- **`docs/media/`** — screenshots directory with capture-tip README; placeholder image references in main README.
- **README integrations directory** — every README install section links the per-IDE guide instead of dumping every JSON config inline.

### Added — M5 (Dashboard)
- **`thoughtline ui` subcommand** — opens an interactive Bubbletea TUI with four panels: header (version + DB path + active project), stats (counts by type / project / scope), recent activity (last 10 memories + 5 sessions), and roadmap (M0–M6 status). Keys: `r` refresh, `q` / `ctrl+c` / `esc` quit. Reads from the same SQLite store the MCP server uses.
- **`tl_stats` MCP tool** — programmatic access to the same stats snapshot. Optional `project` argument; pass `*` to see counts across all projects. Returns text-formatted breakdown the AI can read aloud or summarize.
- **`storage.Stats(ctx, opts) (Stats, error)`** — single query interface returning total memories (active + soft-deleted), counts grouped by type / project / scope, open + closed session counts, and the most recent N memories + sessions. Supports project filter and configurable RecentLimit (default 10, capped at 50).
- **CLI subcommands** — `thoughtline help`, `thoughtline version`, `thoughtline serve` (default), `thoughtline ui`. Friendly error message + usage on unknown subcommand.
- **`internal/dashboard` package** — Bubbletea Model / Update / View split, lipgloss styling, hardcoded `Roadmap()` so PRs review milestone status changes alongside the corresponding code change.
- **Tests** — `internal/storage/stats_test.go` covers every aggregation (totals, by type, by project, by scope, open/closed sessions, recent memories with limit + default, recent sessions, project filter). `internal/server/tl_stats_test.go` covers happy path, empty DB, project filter, default-project fallback, `*` wildcard, type breakdown, recent activity inclusion. `internal/dashboard/model_test.go` follows the SKILL.md patterns: direct `Model.Update()` tests for state transitions (q / ctrl+c / esc / r / window resize / stats loaded), and a basic View test pinning the panels render their headers + roadmap entries.
- **Roadmap renumbered** — M5 is now Dashboard (this release). Smarts (semantic embeddings) moves to M6 and remains deferred per [ADR 0002](docs/decisions/0002-search-strategy-fts5-first.md). Schema reservation for embeddings is still in place from M1.

### Added — M4 (Sessions)
- **`tl_session_start` MCP tool** — opens a session, returns its UUIDv7 id. Optional `agent_label` for cross-session forensics ("claude-code", "cursor", "zed", ...). Project defaults to working directory basename.
- **`tl_session_summary` MCP tool** — closes a session, persists the structured digest. Sessions are append-once: a second call returns "already ended". Summary required, ≤ 64 KB.
- **`tl_save` learns an optional `session_id` argument** — when present, attaches the memory to that session. Empty = unattached (preserves M1 default behaviour exactly).
- **Domain `Session` type** (`internal/memory/session.go`) — UUIDv7 id, project, optional agent_label (≤ 64 chars), started_at, optional ended_at, summary. `IsOpen()` and `Duration()` helpers. `ValidateSession` enforces UUIDv7 format, project required, label/summary size caps, ended_at >= started_at.
- **`memory.SessionID` field** added to `Memory` with UUIDv7 format validation in `memory.Validate`.
- **Schema v2 migration** — bumped `currentSchemaVersion` from 1 to 2. Adds `sessions` table + `idx_sessions_recent` (CREATE IF NOT EXISTS) and adds `memories.session_id TEXT REFERENCES sessions(id) ON DELETE SET NULL` via idempotent `ALTER TABLE` guarded by a `PRAGMA table_info` check. Fresh DBs and existing v1 DBs both migrate cleanly on first Open.
- **Storage CRUD for sessions** (`internal/storage/sessions.go`):
  - `StartSession(ctx, project, agentLabel)` — assigns UUIDv7, persists, returns the Session.
  - `EndSession(ctx, id, summary)` — sets ended_at and summary; rejects already-ended.
  - `GetSession(ctx, id)` / `RecentSessions(ctx, project, limit)`.
  - `ErrSessionNotFound`, `ErrSessionAlreadyEnded`, `ErrSessionProjectMismatch` exported sentinels.
  - **Cross-table integrity**: `Save` now calls `validateSessionLink` before inserting. A non-empty `m.SessionID` must point to an existing session in the same project, otherwise `ErrSessionNotFound` or `ErrSessionProjectMismatch`.
  - **Sticky session_id on upsert**: re-saving with the same `topic_key` and an empty `SessionID` PRESERVES the prior session linkage (`COALESCE(?, session_id)` in the UPDATE). To overwrite, pass an explicit `SessionID`. This prevents accidental session detachment.
- **Tests added**:
  - `internal/memory/session_test.go` — every validation rule.
  - `internal/storage/sessions_test.go` — full CRUD coverage, regression guards for upsert-clobber-session, cross-project rejection, unknown-session rejection, schema v2 idempotency on reopen.
  - `internal/server/tl_session_start_test.go`, `tl_session_summary_test.go`, `tl_save_session_test.go` — handler-level coverage for happy path, validation errors, not-found, already-closed.
  - `internal/server/integration_session_test.go` — end-to-end: start → save (× 2) → upsert preserves session → close → second close rejected → post-mortem save allowed → cross-project session rejected.

### Added — M3 (Context, Update, Delete)
- **`tl_context` MCP tool** — recent memories for the active project, ordered by `updated_at DESC`, soft-deleted rows excluded. Returns the same per-result envelope as `tl_search` so an AI parses both with one parser.
- **`tl_update` MCP tool** — patch a memory by id. Mutable fields: `title`, `content`, `tags`. Identity-defining fields (`type`, `topic_key`, `project`, `scope`) are intentionally NOT mutable. Empty patch = noop. Real changes bump `revision_count`, refresh `updated_at`, preserve `id`/`sync_id`/`created_at`. Re-validates the merged memory before persisting.
- **`tl_delete` MCP tool** — soft-delete by id. Sets `deleted_at`, hides the row from search/context/get_observation, frees the `topic_key` for a fresh `tl_save`. Repeating delete on an already-deleted row returns "not found".
- **`storage.Recent(ctx, project, limit)`** — uses the existing `idx_memories_recent` index. Project parameter is required (never silently spans projects). Limit clamped to `[1, 50]`, default 10.
- **`storage.UpdateByID(ctx, id, UpdatePatch)`** — `UpdatePatch` uses pointer fields (`*string`, `*[]string`) so callers can distinguish "leave unchanged" (nil) from "set to zero value" (e.g. `&[]string{}` to clear all tags).
- **`storage.SoftDelete(ctx, id)`** — partial unique index on `(project, topic_key)` where `deleted_at IS NULL` means the topic_key is automatically freed by soft-delete.
- **`storage.ErrMemoryNotFound`** — exported sentinel for `errors.Is` in callers when an id doesn't exist or has been soft-deleted.
- **Soft-delete regression guard**: M2's `Search` already filtered `deleted_at IS NULL` on the FTS path; pinned that behaviour in `TestSearch_ExcludesSoftDeletedRegression` plus the topic-key shortcut path.
- **Integration scenario test** (`internal/server/integration_test.go`): exercises every `tl_*` tool end-to-end through real `mcp.CallToolRequest` decoding — save → search (FTS + topic-key shortcut) → get_observation → context → update → search post-update → delete → search post-delete → context post-delete → get/update on deleted id (both not-found) → re-save reusing the freed topic_key.

### Added — M2 (Search)
- **`tl_search` MCP tool** — keyword search backed by SQLite FTS5 + BM25 ranking. Parameters: `query` (required), optional `type`, `scope`, `project`, `topic_key` (GLOB filter), `limit` (default 10, hard cap 50), `offset`.
- **Topic-key shortcut** — queries containing `/` are first matched against `topic_key` as a GLOB pattern (so `design/auth/*` or an exact `scene/playcanvas/inn-cellar` lookup is O(1)). If any rows match the shortcut, FTS does not run. Misses fall through to FTS cleanly.
- **FTS5 query sanitization** — every whitespace-delimited token is wrapped in literal quotes; FTS5 special operators (`:`, `*`, `^`, `(`, `)`, `NEAR`, `OR`, ...) in user input become inert. No way to crash the engine with malformed input.
- **`tl_get_observation` MCP tool** — fetch the full untruncated content + metadata of a single memory by `id`. Companion to `tl_search`'s 300-char snippets. Soft-deleted rows are filtered out.
- **`storage.Search`** — returns `[]SearchResult` with `id`, `sync_id`, `title`, `snippet` (≤300 chars from FTS5 `snippet()`), `score` (BM25 rank), `topic_key`, `tags`, `revision_count`, `updated_at`. Filters: `project`, `scope`, `type`, `topic_key` GLOB. Limit/offset pagination.
- **Tests** — storage layer covers basic FTS5 match, BM25 ordering, every filter, both topic-key shortcut paths (hit and fall-through-on-miss), GLOB wildcards, snippet truncation, FTS operator sanitization, default + clamped limits, offset pagination, empty result, revision count round-trip. Server layer covers happy path, missing query, default-project scoping, explicit project override, type filter, topic-key shortcut end-to-end, get-observation hint, limit override.

### Added — M1 (Save)
- **`tl_save` MCP tool** — first working tool. Persists a memory with `title`, `content`, `type`, optional `scope`/`topic_key`/`project`/`tags`. Returns `id`, `sync_id`, `action` (`created`/`updated`/`noop`).
- **Storage layer** (`internal/storage`) — SQLite via `modernc.org/sqlite`, FTS5 contentless virtual table kept in sync via three triggers, `(project, topic_key)` unique index for upserts, reserved `embedding*` columns for M5.
- **Domain layer** (`internal/memory`) — `Memory`, `Type`, `Scope` types and `Validate(Memory) error` with all rules from `docs/design/memory-domain.md`.
- **Topic-key upsert semantics** (mirrors Engram): same project + same topic_key reuses `id`, `sync_id`, `created_at` and bumps `revision_count`. Identical re-saves are noops.
- **CLI / runtime** — `cmd/thoughtline` resolves `THOUGHTLINE_DB` / `THOUGHTLINE_HOME` env vars, opens the database (creating the parent dir if missing), boots the MCP stdio server, exits cleanly on EOF / SIGINT / SIGTERM.
- **Tests** — `go test ./...` covers domain validation (every rule), storage (insert, upsert, noop, FTS sync, persistence across reopen), and the `tl_save` handler (happy path, validation errors, scope auto-defaults, end-to-end upsert flow).

### Added — M0 (Bootstrap)
- Project skeleton: `cmd/thoughtline`, `internal/{server,storage,memory}` with package docs.
- README, LICENSE (MIT, with attribution to Engram), `.gitignore`, `go.mod` targeting Go 1.25.
- Architectural reconnaissance of [Engram](https://github.com/Gentleman-Programming/engram) — three deep-dive docs in `docs/research/`.
- ADR 0001: Architecture baseline — copy Engram's pattern set, customize taxonomy.
- ADR 0002: Search strategy — FTS5 + BM25 in v1, embeddings reserved for M5.
- Memory taxonomy design (`docs/design/memory-domain.md`) with gamedev-first types.
- Architecture overview (`docs/ARCHITECTURE.md`) with mermaid diagrams.
- Example MCP client config in `examples/mcp-config.example.json`.
- Basic CI workflow: `go vet` + `go build` + `go test` on Linux/macOS/Windows.
- CONTRIBUTING.md with PR and ADR conventions.

### Status
- M0–M5 complete. Nine MCP tools live: `tl_save`, `tl_search`, `tl_get_observation`, `tl_context`, `tl_update`, `tl_delete`, `tl_session_start`, `tl_session_summary`, `tl_stats`. Plus a `thoughtline ui` interactive dashboard. M6 (semantic embeddings) remains deferred per ADR 0002 — schema reserved, opt-in if/when needed.

[Unreleased]: https://github.com/AgusLoza2021/Thoughtline/compare/HEAD...HEAD
