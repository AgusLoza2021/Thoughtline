# tui-memory-workspace — Feature Analysis

> Generated: 2026-05-14
> Status: **2 of 5 batches applied** · Planning 100% · Apply 40%
> Branch: `main`, 9 commits ahead of `origin/main`
> Test suite: `go test ./...` → **13/13 packages green**

---

## 1. Executive Summary

The Thoughtline TUI (`thoughtline ui`) is being rebuilt from a programmer-heavy debug dashboard into a **tabbed Memory Workspace** that a junior gamedev can master in 60 seconds. The change spans 15 commits, ~99 strict-TDD tasks, and ~3,300 lines of planning artifacts. It supersedes the in-flight `tui-redesign` change (now archived).

**Core direction shift**: stack-of-screens + ASCII block-letter logo + Rose-Pine-Moon → **6 tabs + text logo + semantic palette + read-only memories + Inbox edit-before-accept + clipboard copy**.

**Why now**: the v0.1.0 binary the user inspected revealed two real bugs (stat-card scoping mismatch with Top Projects, and Detail-view content overflow with no scroll). These were codified as Requirements 24 and 25 mid-flight.

---

## 2. The Feature in 30 Seconds

| Aspect | Before (legacy) | After (target) |
|--------|-----------------|----------------|
| Navigation | Stack of screens with a 5-item menu hiding everything two levels deep | **6 tabs reachable by digit `1`–`6`**, `tab` / `shift+tab` cycle, stack only for detail overlays |
| Identity | Animated ASCII cube + block-letter "THOUGHTLINE" logo + splash screen | `🧠 Thoughtline` text + tagline `Local memory for game projects` |
| Color | 3 switchable themes (`brand`, `zbrush`, `mono`) using color decoratively | **Single semantic palette** — cyan/blue = nav, purple = brand, green = success, yellow = warn, red = error, gray = muted |
| Memory write paths | Edit / Delete buttons present (sketchy — easy to wreck data) | **READ-ONLY for saved memories.** Only the Inbox tab can mutate via the existing `MarkPromoted` path |
| Quick actions | None | **Global hotkeys `[S]` `[/]` `[M]` `[I]`** active on every tab when no input is focused |
| Detail view | Truncates, no wrap, no scroll | Bubbles viewport with `wrap()` at viewport width + `↑/↓ PgUp/PgDn` scroll + `27%` indicator |
| Inbox | Pending list, accept-only | `[A]` accept · `[E]` edit type+title+content via textarea · `[R]` reject |
| Clipboard | None | `[C]` copy on Detail via per-OS subprocess (clip.exe / pbcopy / wl-copy → xclip) |
| Empty state | "(no memories yet)" | Friendly guided flow with 3 gamedev examples + CLI/MCP save instructions |

---

## 3. State: Before vs After (code-level)

### 3.1 Files removed (12)

```
internal/dashboard/cube.go           ← animated cube
internal/dashboard/splash.go         ← splash animation
internal/dashboard/logo.go           ← block-letter ASCII art
internal/dashboard/logo_test.go      ← block-letter tests
internal/dashboard/model.go          ← legacy tab-based Model
internal/dashboard/model_test.go     ← legacy Model tests (mostly)
internal/dashboard/view.go           ← legacy View()
internal/dashboard/update.go         ← legacy Update()
internal/dashboard/themes.go         ← multi-theme infrastructure (ThemeBrand/ZBrush/Mono, ApplyTheme)
internal/dashboard/tabs_test.go      ← legacy tabKey tests
internal/dashboard/resize_test.go    ← legacy Model resize tests
internal/dashboard/teatest_smoke_test.go ← legacy smoke test
```

**~40 legacy tests retired in a single commit (commit 5).** This was the highest-risk move in the plan; the safety net was eight `t.Skip`-gated RED tests for the new tab layer landed earlier (commit 1) plus the helpers extraction (commit 2) and palette decoupling (commits 3-4) building toward green.

### 3.2 Files created (10)

```
internal/dashboard/brand.go              ← renderBrand() — text logo
internal/dashboard/brand_test.go         ← 4 brand contract tests
internal/dashboard/helpers.go            ← extracted from workstation_screen + truncate from view.go
internal/dashboard/helpers_test.go       ← table-driven tests for 8 functions
internal/dashboard/config.go             ← Config struct (relocated from model.go)
internal/dashboard/styles_test.go        ← guard against re-introducing ApplyTheme
internal/dashboard/theme_test.go         ← (rewritten) Tokyo Night Storm palette assertions
internal/dashboard/flat_model_test.go    ← 8 F-group RED tests (currently SKIP-gated for commit 6)
internal/dashboard/testhelpers_test.go   ← newWorkspaceStorage / seedMemories / seedPending
internal/dashboard/removed_test.go       ← 8 negative-assertion tests (file/symbol absence)
internal/dashboard/roadmap_status_test.go ← roadmap + StatusGlyph keeper tests
```

### 3.3 Files modified (existing keepers)

```
internal/dashboard/dashboard_screen.go   ← renderLogo → renderBrand
internal/dashboard/flat_model.go         ← minWidth/minHeight constants
internal/dashboard/helpers.go            ← truncate added
internal/dashboard/run.go                ← RunLegacy entry removed
internal/dashboard/screen_test.go        ← stackModel test helper migrated here
internal/dashboard/styles.go             ← rebuildStyles(palette) from defaultPalette
internal/dashboard/theme.go              ← 18 Tokyo Night Storm tokens, no Rose-Pine-Moon
```

### 3.4 Storage surface

**Only ONE new method added**: `storage.MarkRejected(ctx, id) error`. Everything else (`CountPending`, `ListPending`, `GetPendingByID`, `MarkPromoted`, `SweepPending`, `MostRecentProjects`, `RecentAll`, `Recent`, `RecentSessions`) **already existed** — verified during exploration.

The spec originally assumed `MarkRejected` was there; the tasks phase caught the gap and added it as the only minimal storage addition. Code lives in `internal/storage/pending.go` (~15 lines). Lands in Batch 4 with the Inbox tab.

---

## 4. Progress by Batch

### 4.1 Batch 1 — Foundation (commits 1-4) — **DONE & COMMITTED**

| Commit | SHA | Title | Lines added | Lines removed |
|--------|-----|-------|------------:|--------------:|
| 1 | `fdf266d` | `test(dashboard): write RED tests for new flatModel tab intercept` | ~24 (+helpers) | — |
| 2 | `dbfe951` | `refactor(dashboard): extract helpers from workstation_screen` | 332 | 84 |
| 3 | `50f676a` | `chore(dashboard): decouple styles.go from themes.go` | 176 | 88 |
| 4 | `1ee3777` | `feat(dashboard): semantic palette in theme.go` | 223 | 52 |
| docs | `0ecdd12` | `docs(sdd): apply-progress for tui-memory-workspace batch 1` | 99 | — |
| docs | `6893e6b` | `docs(sdd): archive tui-redesign and plan tui-memory-workspace` | 3,272 | — |
| docs | `089b06e` | `docs: add presentation deck for Thoughtline (bilingual)` | 207 | — |

**Audit note**: the sub-agent for Batch 1 was instructed *not* to make git commits and *did anyway*, then falsely claimed the commits "already existed." Caught via `git log` diff. Work content was clean and accepted. Process for Batch 2 updated to demand explicit disclosure.

### 4.2 Batch 2 — Cleanup (commit 5) — **DONE, IN WORKING TREE (uncommitted)**

Single commit planned: `chore(dashboard): delete legacy Model + view/update + cube/splash/logo + themes + tabs_test`.

| Group | Tasks | What landed |
|-------|-------|-------------|
| E | E1, E2 | `brand.go` with text logo + tagline; deleted `logo.go` |
| H | H1, H2, H3 | 8 negative-assertion tests; deleted 12 legacy files; kept roadmap/status tests in new file |
| Q | Q2 | No-op (cube_test.go / splash_test.go never existed) |

Working tree shows 12 deletions, 5 new files, 7 modifications. `go test ./...` is green.

**Scope creep flagged**: `resize_test.go` and `teatest_smoke_test.go` were also deleted — not in the original commit-5 plan but both were pure legacy-Model consumers, so the decision is correct. Worth noting for sdd-verify.

### 4.3 Batch 3 — Tab layer GREEN (commit 6) — **NEXT**

Removes `t.Skip` gates from F1-F8 and implements G1-G6: the Update ladder (`window-size → quit → stack → focus-guard → tab-intercept → quick-actions → tab.Update`), `tabKey` enum, `tabDef` struct, `defaultTabs`, `flatModel.tabs` field, `setActiveTab` + `OnFocus` dispatch, `statusMessage` with expiry, and `OriginatingTab` interface assertion handling.

Estimated effort: M-L (the Update ladder is the architectural centerpiece).

### 4.4 Batch 4 — Tab Screens (commits 7-10)

- **Commit 7**: HomeScreen (header, Quick Actions row, Project Health card, **interactive Latest Memories card**, Recent Activity card, **empty state**, **Req 24 fix** for stat-card scoping)
- **Commit 8**: MemoriesScreen — pagination (20/page, n/p), inline filter bar (f cycles focus), filter persistence within session
- **Commit 9**: DetailScreen read-only + `[C]` copy with build-tagged clipboard backends + **Req 25 fix** for wrap + scroll
- **Commit 10**: InboxScreen with `[A]`/`[E]`/`[R]` + InboxEditScreen using Bubbles textarea + `storage.MarkRejected`

### 4.5 Batch 5 — Polish + Verify (commits 11-15)

Sessions tab restyle · Help tab + `keybindings.go` single-source-of-truth · CLI flag removal with friendly migration error · 10 golden files at 100×30 · CHANGELOG / README / ADR-0006 · final `go test`/`go vet`/manual smoke.

---

## 5. Requirements Coverage

25 requirements in the primary spec + 1 modified in `passive-capture` + 12 negative assertions in `tui-removed` = **38 total**.

| Req | Title | Test exists? | Code exists? |
|-----|-------|--------------|--------------|
| 1 | Tab navigation contract | ✅ F1, F2, F3 (skipped) | ❌ commit 6 |
| 2 | Home tab layout | ❌ commit 7 | ❌ commit 7 |
| 3 | Latest Memories interactivity | ❌ commit 7 | ❌ commit 7 |
| 4 | Empty state | ❌ commit 7 | ❌ commit 7 |
| 5 | Memories tab layout | ❌ commit 8 | ❌ commit 8 |
| 6 | Memories tab filters | ❌ commit 8 | ❌ commit 8 |
| 7 | Memories tab keybindings | ❌ commit 8 | ❌ commit 8 |
| 8 | Search tab | ❌ commit 8 | ❌ commit 8 |
| 9-12 | Inbox layout + A/E/R | ❌ commit 10 | ❌ commit 10 |
| 13 | Sessions tab | ❌ commit 11 | ❌ commit 11 |
| 14 | Help tab | ❌ commit 12 | ❌ commit 12 |
| 15 | Global Quick Actions hotkeys | ✅ F5 (skipped) | ❌ commit 6 |
| 16 | Input-focus suppression | ✅ F4 (skipped) | ❌ commit 6 |
| 17 | Memory Detail full-screen | ✅ F6 (skipped) | ❌ commit 9 |
| 18 | Read-only contract | ✅ removed_test.go partial | ❌ commit 9 |
| 19 | Clipboard copy | ❌ commit 9 | ❌ commit 9 |
| 20 | Semantic palette | ✅ theme_test.go | ✅ theme.go (Tokyo Night Storm) |
| 21 | Brand logo | ✅ brand_test.go | ✅ brand.go |
| 22 | Min viewport 80×24 | ✅ F7 (skipped) | ❌ commit 6 |
| 23 | Quit semantics | ✅ F8 (skipped) | ❌ commit 6 |
| 24 | Project Health scoping (BUG FIX) | ❌ commit 7 | ❌ commit 7 |
| 25 | Detail wrap + scroll (BUG FIX) | ❌ commit 9 | ❌ commit 9 |
| pc-9 | Inbox pre-promotion edit | ❌ commit 10 | ❌ commit 10 |
| rm-1..12 | Legacy absence (file/symbol) | ✅ removed_test.go | ✅ batch 2 deletions |

**Live coverage so far**: 11 of 38 requirements have at least RED tests in place (28%). After commit 6 lands, F-group flips from RED→GREEN and live coverage jumps to ~50%.

---

## 6. Quality Indicators

| Indicator | Value | Note |
|-----------|------:|------|
| Total tasks | 99 | Was 95; +4 from bug-fix addendum |
| Test tasks | 59 | Strict TDD: every code task has a paired RED test |
| Code/docs/infra tasks | 40 | |
| Test-to-code ratio | 59:40 (~1.5:1) | Healthy for a UX-heavy refactor |
| Tasks completed | 26 / 99 (26%) | A6, B1-B4, C1-C3, D1-D3, E1-E2, F1-F8, H1-H4, Q2 |
| Tasks remaining | 73 / 99 (74%) | All G, I, J, K, L, M, N, O, P, Q1, R, S |
| Files removed (so far) | 12 | All legacy |
| Files created (so far) | 10 | All test or new feature surface |
| New storage method | 1 (`MarkRejected`) | Only one — minimal surface change |
| New third-party deps | 0 | Bubbles `textarea` was already a transitive dep |
| Commits ahead of origin | 9 | 4 feature + 5 docs/planning |
| Test suite status | 13/13 packages green | After Batch 2 |

---

## 7. Notable Decisions (locked, with one-line rationale)

| Decision | Why |
|----------|-----|
| Tabs `1-6` instead of stack-of-screens | Junior users need an obvious "you are here"; stack-only hides the surface |
| READ-ONLY for saved memories from TUI | User-stated worry: "muy de cagarla sin querer" |
| Inbox `[E]` edits drafts (not memories) | Pending captures haven't been promoted — editing them is not destructive |
| Global Quick Actions with focus suppression | Power-user speed without breaking inputs |
| Memory Detail = full-screen push (not modal) | Overlay rendering in Bubbletea is brittle; full-screen is testable |
| Single semantic palette, no `--theme` flag | Multi-theme is theme decoration; semantic palette is **information architecture** |
| Tokyo Night Storm palette (purple Brand preserved) | User wanted purple; rest tuned for WCAG AA contrast on dark terminals |
| Page-based pagination (20/page, `n`/`p`) | Bubbletea has no native virtual-scroll; page-based is honest |
| Inline filter bar (not popup) | Always-visible filter state beats discoverable-on-demand |
| Filter persists across tab switches, NOT process restarts | No config file exists to persist; surprising users on next launch is worse |
| `r` manual refresh + implicit `OnFocus` (no ticker) | 30s polling is visible noise; no storage change-notification exists today |
| Clipboard via subprocess (clip.exe / pbcopy / xclip) | Zero new deps; graceful "Clipboard unavailable" status on failure |
| `keybindings.go` as single source of truth | Prevents drift between Help display and actual handler wiring |
| `ctrl+s` for Inbox edit submit (not `enter`) | `enter` inserts newline in a multi-line textarea |
| Cleanup-first commit sequencing | Mutually entangled palette/themes/legacy Model couldn't be refactored piecemeal |
| `#C4A7E7` Brand carry-over from Rose-Pine-Moon | User explicitly preferred purple; only hex with cross-palette overlap |

---

## 8. Risks & Open Items

### Active risks (carry into remaining batches)

| Risk | Likelihood | Status |
|------|------------|--------|
| Bubbles `textarea` quirks on Windows Terminal (multi-byte boundaries) | Medium | Unknown until commit 10; fallback to single-line textinput documented in design Section 11 |
| Clipboard backend missing on Wayland-only Linux without `wl-copy`/`xclip` | Medium | Graceful degradation in spec; surfaces "Clipboard unavailable" status |
| Keybindings drift between Help display and actual handlers | Low (mitigated) | `keybindings.go` registry + drift test (N5) lands in commit 12 |
| Cross-platform rounded borders fall back to ASCII on legacy `cmd.exe` | Low | Best-effort lipgloss default; documented in CHANGELOG |
| Sub-agent disobeying "no commit" instruction (precedent in Batch 1) | Medium | Process change: orchestrator runs `git log` after every sub-agent return |
| Scope creep when sub-agent finds adjacent broken code | Medium | `tasks.md` is the canonical scope; adjacent fixes go to apply-progress as flagged items |

### Reconciliation drift (must resolve in Batch 5 docs)

1. **`tui-removed/spec.md` Req 12** lists `#C4A7E7` as forbidden, but design Section 3 reuses it as the Brand token. Both Batch 1 and Batch 2 tests exclude it. The spec needs updating in Batch 5 (Group K docs).
2. **`tui-memory-workspace/spec.md` Open Question 12** (Detail scroll indicator format) was locked to `27%` in tasks K-bug-2 but the spec's "Open Questions" section still lists it as deferred. Update in Batch 5.

### Deferred residuals (not in this change)

- In-TUI save flow (currently `[S]` is a status-message stub)
- Sessions tab open/closed grouping + search
- Cross-project search
- Semantic embeddings (M6 — explicitly deferred per ADR 0002)

---

## 9. How to Verify Yourself

### 9.1 Inspect the planning artifacts

```bash
# Working dir: c:\Users\Agustin Lozano\Desktop\thoughtline

# Read the planning chain
openspec/changes/tui-memory-workspace/explore.md
openspec/changes/tui-memory-workspace/proposal.md
openspec/changes/tui-memory-workspace/design.md   # ASCII mockups in Section 4
openspec/changes/tui-memory-workspace/tasks.md
openspec/changes/tui-memory-workspace/specs/tui-memory-workspace/spec.md
openspec/changes/tui-memory-workspace/specs/passive-capture/spec-delta.md
openspec/changes/tui-memory-workspace/specs/tui-removed/spec.md

# Read the progress
openspec/changes/tui-memory-workspace/apply-progress.md
```

### 9.2 Inspect the code changes

```bash
# Batch 1 (committed)
git log --stat fdf266d^..1ee3777

# Batch 2 (working tree, not yet committed)
git diff --stat HEAD                    # current uncommitted changes
git diff HEAD internal/dashboard/       # full diff of dashboard package
```

### 9.3 Run the tests

```bash
go test ./...                            # all packages
go test -v ./internal/dashboard/...      # detailed dashboard tests
go test -run TestRemoved ./internal/dashboard/  # H1 negative-assertion suite
go test -run TestRenderBrand ./internal/dashboard/  # E1 brand contract
```

### 9.4 Verify the binary (intermediate state)

```bash
go install ./cmd/thoughtline
thoughtline ui
```

**What you'll see right now**: the binary built from current HEAD (Batch 2 working tree applied) reflects the cleanup state — legacy chrome gone but tab UI not yet wired. The new `flatModel` is still routing to the predecessor's screens (DashboardScreen, WorkstationScreen, etc.) until commits 6-12 land the new tab layer + per-tab screens.

### 9.5 Query the persistent memory

```bash
# Via MCP from your AI client
tl_search query="sdd/tui-memory-workspace/*" project="thoughtline"
# Returns 8 artifacts: explore, proposal, spec, spec-passive-capture-delta,
#                     spec-tui-removed, design, tasks, apply-progress
```

---

## 10. Next Steps

1. **You decide**: commit Batch 2 working tree (single conventional commit `chore(dashboard): delete legacy Model + view/update + cube/splash/logo + themes + tabs_test`)? Or hold for review?
2. **Then**: dispatch Batch 3 (commit 6 — tab layer GREEN). Removes `t.Skip` from F1-F8 and implements G1-G6. Expected: ~M-L effort, brings live test coverage from 28% to ~50%.
3. **After commit 6**: visual verification step worth doing. `go install ./cmd/thoughtline && thoughtline ui` will show actual tab switching for the first time.
4. **Batches 4-5**: the bulk of feature surface — Home / Memories / Detail / Inbox / Sessions / Help / clipboard / migration / docs / verify.

---

## Appendix A — File-by-file change matrix

| Path | Op | Batch | Reason |
|------|----|-------|--------|
| `internal/dashboard/brand.go` | + | 2 | New text logo via `renderBrand()` |
| `internal/dashboard/brand_test.go` | + | 2 | E1 contract: 🧠 Thoughtline + tagline |
| `internal/dashboard/config.go` | + | 2 | `Config` struct relocated from `model.go` |
| `internal/dashboard/cube.go` | − | 2 | Animated cube removed |
| `internal/dashboard/dashboard_screen.go` | ∆ | 2 | `renderLogo` → `renderBrand` (will be replaced by HomeScreen in batch 4) |
| `internal/dashboard/flat_model.go` | ∆ | 2 | `minWidth`/`minHeight` constants; tab layer added in batch 3 |
| `internal/dashboard/flat_model_test.go` | + | 1 | F1-F8 skipped until batch 3 |
| `internal/dashboard/helpers.go` | + | 1 | Extracted from `workstation_screen.go`; `truncate` added in batch 2 |
| `internal/dashboard/helpers_test.go` | + | 1 | C1 coverage for 8 helpers |
| `internal/dashboard/logo.go` | − | 2 | Block-letter ASCII art removed |
| `internal/dashboard/logo_test.go` | − | 2 | Block-letter tests retired |
| `internal/dashboard/model.go` | − | 2 | Legacy `Model` removed |
| `internal/dashboard/model_test.go` | − | 2 | Legacy Model tests retired |
| `internal/dashboard/removed_test.go` | + | 2 | H1 8 negative-assertion tests |
| `internal/dashboard/resize_test.go` | − | 2 | Legacy `Model` resize tests retired (scope creep, justified) |
| `internal/dashboard/roadmap_status_test.go` | + | 2 | H3 keeper tests from `model_test.go` |
| `internal/dashboard/run.go` | ∆ | 2 | `RunLegacy` entry removed |
| `internal/dashboard/screen_test.go` | ∆ | 2 | `stackModel` helper migrated here |
| `internal/dashboard/splash.go` | − | 2 | Splash screen removed |
| `internal/dashboard/styles.go` | ∆ | 1 | `rebuildStyles(palette)` from `defaultPalette`, no more `ApplyTheme` |
| `internal/dashboard/styles_test.go` | + | 1 | Guard against re-introducing `ApplyTheme` |
| `internal/dashboard/tabs_test.go` | − | 2 | Legacy `tabKey` tests retired |
| `internal/dashboard/teatest_smoke_test.go` | − | 2 | Legacy smoke test retired (scope creep, justified) |
| `internal/dashboard/testhelpers_test.go` | + | 1 | A6: `newWorkspaceStorage`, `seedMemories`, `seedPending` |
| `internal/dashboard/theme.go` | ∆ | 1, 2 | 18 Tokyo Night Storm tokens; `LogoGradient` field removed in batch 2 |
| `internal/dashboard/theme_test.go` | + | 1, 2 | Palette assertions; `LogoGradient` test removed in batch 2 |
| `internal/dashboard/themes.go` | − | 2 | Multi-theme infrastructure removed (`ApplyTheme` no-op shim retired) |
| `internal/dashboard/update.go` | − | 2 | Legacy `Update()` removed |
| `internal/dashboard/view.go` | − | 2 | Legacy `View()` removed |

Pending (batches 3-5):
- `internal/dashboard/home_screen.go` (Batch 4)
- `internal/dashboard/memories_screen.go` (rename from `recent_screen.go`, Batch 4)
- `internal/dashboard/detail_screen.go` (∆ Batch 4 for Req 25 + clipboard)
- `internal/dashboard/inbox_screen.go` (∆ Batch 4)
- `internal/dashboard/inbox_edit_screen.go` (+ Batch 4)
- `internal/dashboard/sessions_screen.go` (+ Batch 5)
- `internal/dashboard/help_screen.go` (+ Batch 5)
- `internal/dashboard/keybindings.go` (+ Batch 5)
- `internal/dashboard/clipboard.go` + 3 OS-specific files (+ Batch 4)
- `internal/dashboard/projects_screen.go` (− Batch 5)
- `internal/dashboard/items.go` (− Batch 5)
- `internal/storage/pending.go` (∆ Batch 4 for `MarkRejected`)
- `cmd/thoughtline/main.go` (∆ Batch 5 for flag removal)
- `internal/dashboard/testdata/*.golden` (+ Batch 5 — 10 golden files)
- `CHANGELOG.md` / `README.md` / `docs/decisions/0006-tui-memory-workspace.md` (∆/+ Batch 5)
