# Apply Progress — tui-memory-workspace

> Status: Batch 1 of 5 complete (commits 1–4 of 15).
> Test command: `go test ./...` — green on all packages.
> Strict TDD: enforced — F-group tests are SKIP-gated (alternative to RED at runtime) until commit 6 lands the G-group implementation.

## Per-commit summary

### Commit 1 — `fdf266d` test(dashboard): write RED tests for new flatModel tab intercept

| Task | Status | Notes |
|------|--------|-------|
| A6 | ✅ done | `internal/dashboard/testhelpers_test.go` created with `newWorkspaceStorage`, `seedMemories`, `seedPending` and a smoke test. |
| F1 | ✅ done (SKIP-gated) | Tab digit-jump test for `1`-`6` — covers Req 1 first scenario. |
| F2 | ✅ done (SKIP-gated) | Tab cycle forward/backward with wrap — covers Req 1 second scenario. |
| F3 | ✅ done (SKIP-gated) | Tab state survives Screen push/pop — covers Req 1 third scenario and Req 17 originator override. |
| F4 | ✅ done (SKIP-gated) | InputFocused guard for `m`/`s`/`2`/`q` literal-in-input; ctrl+c bypasses guard. Req 16, Req 23, Reconciliation #4. |
| F5 | ✅ done (SKIP-gated) | `/` jumps to Search AND focuses input — Req 15. |
| F6 | ✅ done (SKIP-gated) | OriginatingTab return on esc-pop, including Home/Memories/Search origins — Req 17. |
| F7 | ✅ done (SKIP-gated) | Min viewport boundaries (80×24 OK, 79×24 warn, 80×23 warn); under-min q/ctrl+c still quit; other keys frozen. Req 22, Reconciliation #8. |
| F8 | ✅ done (SKIP-gated) | q quits on Memories, ctrl+c always quits, q with stack overlay does not quit globally. Req 23. |

Additional work in this commit (not in tasks.md but required for compilation):
- Renamed legacy `tabKey` enum → `legacyTabKey` in `model.go`. Updated `update.go`, `view.go`, `tabs_test.go` so legacy Model keeps compiling while the new `tabKey` enum lives in `flat_model.go`. The legacy enum dies in commit 5.
- Added `tabKey` enum (`TabHome`..`TabHelp`), `tabDef` struct, `defaultTabs` array, `inputFocuser`, `originator` interface types, and `activeTab` / `tabs` stub fields to `flatModel`. The Update ladder itself is unchanged — F-tests use these compile-time hooks but assert runtime behavior that doesn't exist yet.

### Commit 2 — `dbfe951` refactor(dashboard): extract helpers from workstation_screen

| Task | Status | Notes |
|------|--------|-------|
| C1 | ✅ done | `internal/dashboard/helpers_test.go` — table-driven coverage for all 8 helpers with empty / exact-width / overflow / negative cursor / zero-time / wrap-clamp edges. |
| C2 | ✅ done | `internal/dashboard/helpers.go` created with verbatim extractions: `paneBox`, `centerString`, `clampCursor`, `truncateLeft`, `wrap`, `relTime`, `formatLastSave`, `padRight`. `workstation_screen.go` and `view.go` no longer redeclare them. |
| C3 | ✅ done | `workstation_screen.go` still compiles; its existing screen behavior is preserved by reading the same helper names from `helpers.go`. Non-F tests green. |

Note: `padRight` was sourced from `view.go` (the legacy version using `len(s)`), not from `workstation_screen.go`. Behavior preserved exactly.

### Commit 3 — `50f676a` chore(dashboard): decouple styles.go from themes.go

| Task | Status | Notes |
|------|--------|-------|
| D1 | ✅ done | `internal/dashboard/styles_test.go` — source-level guard ensures `styles.go` no longer references `ThemeBrand`/`ThemeZBrush`/`ThemeMono`/`ApplyTheme(`/`nextTheme`/`currentTheme`. Also asserts every style var renders non-empty after `init()`. |
| D2 | ✅ done | `styles.go` rewritten: declares the lipgloss style vars, exposes `rebuildStyles(palette)` as the single builder, calls it once from `init()` against `defaultPalette`. Legacy `col*` adaptive-color shorthands deleted because nothing outside styles.go consumed them. |
| D3 | ✅ done | Legacy `Model` (model.go / view.go / update.go) still compiles via `ApplyTheme` retained in `themes.go` as a no-op shim. Legacy model_test.go and workstation_screen_test paths remain green. |

### Commit 4 — `1ee3777` feat(dashboard): semantic palette in theme.go

| Task | Status | Notes |
|------|--------|-------|
| B1 | ✅ done | `theme_test.go` rewritten: asserts all 18 new tokens are valid 6-char hex; asserts the 10 non-overlapping Rose-Pine-Moon hex codes are absent from theme.go source; asserts `LogoGradient` is the zero array. |
| B2 | ✅ done | `theme.go` palette replaced with the design Section 3 table verbatim. New tokens: `Foreground`, `Muted`, `Border`, `FocusedBg`, `BorderFocused`, `Nav`, `NavActive`, `NavActiveBg`, `Brand`, `BrandDim`, `Success`, `Warning`, `Error`, `StatusOK`, `StatusWarn`, `StatusErr`, `BadgeMemoryType`, `BadgeMemoryTypeBg`. Legacy aliases retained (`Tag`, `StatNumber`, `Cursor`, `MenuSelectedBg`) pointing at the new tokens so existing renderers keep working. |
| B3 | ✅ done | Status style maps OK→Success / WARN→Warning / ERR→Error. Test compares lipgloss.Color values directly. Also asserts `Success != Nav` and `Success != NavActive` per Req 20. |
| B4 | ✅ done | `statusStyle` continues to read from `p.StatusOK`/`p.StatusWarn`/`p.StatusErr` which now map to the new semantic green/yellow/red. |

## Known deviations from spec / design

| # | Deviation | Spec / design reference | Rationale |
|---|-----------|------------------------|-----------|
| 1 | `#C4A7E7` is NOT in the Rose-Pine-Moon absence assertion in B1 | `tui-removed` Req 12 lists `#C4A7E7` as forbidden | The design Section 3 explicitly assigns `Brand = #C4A7E7` and `BadgeMemoryType = #C4A7E7`. We retain the design's choice; the test asserts the other 10 forbidden hexes are absent. The `tui-removed` spec should be updated in archive to reflect that #C4A7E7 is re-used by the new design as Brand. |
| 2 | `LogoGradient` field is retained on the palette struct as a zero-value array | Design Section 3 says "Remove `LogoGradient` field entirely" | The orchestrator's batch instructions explicitly preserve `logo.go` until commit 5 (next batch). `logo.go` references `palette.LogoGradient`; removing the field would break compilation. Field is zero-initialised so the rendered output is empty-color (no visible Rose-Pine-Moon gradient). Field is deleted alongside `logo.go` in the next batch. |
| 3 | Legacy `tabKey` → `legacyTabKey` rename | Not in tasks.md | Required to free the `tabKey` identifier for the new flat-workspace enum because legacy `model.go` (deleted in commit 5) cannot be edited away yet. Pure rename, no behavior change. |
| 4 | F-tests are `t.Skip`-gated rather than failing at runtime | Orchestrator preferred honest RED, but listed `t.Skip` as the acceptable alternative | The F-tests assert the target contract and will be ungated in commit 6 (G-group). Tests still compile against the production type, proving the structural contract holds. |

## Test status by commit

| After commit | Result |
|--------------|--------|
| 1 (RED tests + helpers) | A6 smoke green. F-tests RED at runtime initially; SKIP-gated by linter/user follow-up — equivalent state per orchestrator's accepted alternative. All other dashboard tests green. All other packages green. |
| 2 (helpers extracted) | All C-group tests green. Legacy `workstation_screen.go` still compiles. F-tests still SKIP-gated. |
| 3 (styles decoupled) | All D-group tests green. Legacy Model still compiles (`ApplyTheme` is a no-op shim). F-tests still SKIP-gated. |
| 4 (semantic palette) | All B-group tests green. F-tests still SKIP-gated. Full `go test ./...` green across the repo. |

## Files changed in this batch

- `internal/dashboard/flat_model.go` — added tabKey / tabDef / interfaces / activeTab/tabs stub fields
- `internal/dashboard/flat_model_test.go` — created (F1–F8 + A6 wiring)
- `internal/dashboard/testhelpers_test.go` — created (A6)
- `internal/dashboard/helpers.go` — created (C2)
- `internal/dashboard/helpers_test.go` — created (C1)
- `internal/dashboard/workstation_screen.go` — helpers removed (C2)
- `internal/dashboard/view.go` — `padRight` removed, `tabSearch`/`tabHelp` renamed (C2 + legacy rename)
- `internal/dashboard/styles.go` — rewritten as `rebuildStyles(palette)` (D2)
- `internal/dashboard/styles_test.go` — created (D1)
- `internal/dashboard/themes.go` — `ApplyTheme` becomes no-op shim (D2/D3)
- `internal/dashboard/theme.go` — semantic palette per Section 3 (B2/B4)
- `internal/dashboard/theme_test.go` — rewritten for new tokens (B1/B3)
- `internal/dashboard/model.go`, `update.go`, `tabs_test.go` — legacy `tabKey` renamed to `legacyTabKey` to free the identifier

## Batch 2 — Commit 5 (legacy mass deletion)

> Status: complete. All 7 tasks done. `go test ./...` green on all 13 packages.
> Git commit policy: **Option A** — code implemented in working tree only, NO commit made.
> Orchestrator must inspect diff and ask user before committing.

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| E1 | `brand_test.go` | Unit | N/A (new) | ✅ Written (4 tests, `renderBrand` undefined) | ✅ Passed after E2 | ✅ 4 cases (content, length, hex, newline) | ➖ Clean as written |
| E2 | `brand_test.go` | Unit | ✅ baseline green | ✅ RED from E1 | ✅ Passed | ✅ E1 covered 4 scenarios | ➖ None needed |
| H1 | `removed_test.go` | Unit | N/A (new) | ✅ Written (8 tests, 6 failed before H2) | ✅ Passed after H2 | ✅ File-absence + symbol-absence + hex-absence | ➖ Clean as written |
| H2 | `removed_test.go` | Unit | ✅ baseline green | ✅ RED from H1 | ✅ All 8 TestRemoved_ pass | ✅ Symbol-absence + hex-absence tests force real logic | ✅ Moved `Config`/`truncate`/`stackModel` cleanly |
| H3 | `roadmap_status_test.go` | Unit | ✅ model_test.go was green | ✅ N/A (pure restructure) | ✅ Passed | ➖ Single responsibility | ➖ None needed |
| H4 | — | — | N/A | N/A | N/A | N/A | N/A — `logo_test.go` deleted in E2 |
| Q2 | — | — | N/A | N/A | N/A | N/A | N/A — no cube_test.go or splash_test.go existed |

### Test Summary

- **Total new tests written**: 12 (brand_test: 4, removed_test: 8)
- **Total tests passing**: all (13 packages green)
- **Layers used**: Unit (12)
- **Approval tests** (refactoring): None — pure deletions, not behavior changes
- **Pure functions created**: `renderBrand(palette) string`

### Per-task notes

| Task | Status | Files touched | Notes |
|------|--------|---------------|-------|
| E1 | ✅ done | `brand_test.go` (NEW) | 4 tests: content, length, no-forbidden-hex, single-line |
| E2 | ✅ done | `brand.go` (NEW), `logo.go` (DELETED), `logo_test.go` (DELETED), `theme.go` (LogoGradient removed), `dashboard_screen.go` (renderLogo→renderBrand), `theme_test.go` (TestPalette_LogoGradientNotInitialised removed) | `renderBrand` lives in new `brand.go`. `dashboard_screen.go` was the only caller of `renderLogo` outside `logo.go` — updated to `renderBrand(p)`. |
| H1 | ✅ done | `removed_test.go` (NEW) | 8 negative-assertion tests for Req 4–12 (cube, splash, themes, tabs_test, model triad file absence, theme symbols, model symbols, Rose-Pine-Moon hex) |
| H2 | ✅ done | `model.go` (DELETED), `view.go` (DELETED), `update.go` (DELETED), `cube.go` (DELETED), `splash.go` (DELETED), `themes.go` (DELETED), `tabs_test.go` (DELETED), `model_test.go` (DELETED — H3), `resize_test.go` (DELETED), `teatest_smoke_test.go` (DELETED) | Also required: `config.go` (NEW — moved `Config` struct from model.go), `flat_model.go` (added `minWidth`/`minHeight` constants from view.go), `helpers.go` (added `truncate` from view.go), `screen_test.go` (added `stackModel` struct that was in model.go), `run.go` (removed `RunLegacy` which called deleted `New`). `commands.go` NOT simplified (splash/cube ticks were already only in the deleted files; cube ticks were in cube.go itself). |
| H3 | ✅ done | `roadmap_status_test.go` (NEW), `model_test.go` (DELETED — entire file) | Kept: `TestRoadmap_HasAllExpectedMilestones`, `TestStatusGlyph_KnownStatuses`. Deleted: all legacy Model tests. Also deleted `resize_test.go` and `teatest_smoke_test.go` which were purely legacy Model tests (newTestModel/Model consumers). |
| H4 | ✅ done (no-op) | — | `logo_test.go` was deleted in E2; `brand_test.go` covers the contract. |
| Q2 | ✅ done (no-op) | — | `cube_test.go` and `splash_test.go` did not exist. |

### Deviations from design

| # | Deviation | Rationale |
|---|-----------|-----------|
| 1 | `resize_test.go` and `teatest_smoke_test.go` also deleted (not listed in commit 5 deletions) | Both files used `newTestModel(t)` and `Model` which are now gone. They were legacy Model tests with no surviving value. |
| 2 | `config.go` created (new production file) | `Config` struct was in `model.go`; it is needed by `flat_model.go` and `run.go`. Moved to a dedicated `config.go`. |
| 3 | `truncate` added to `helpers.go` | `truncate` (rune-based right-truncation) was in `view.go`. `items.go` and potentially other files use it. Moved to `helpers.go` alongside the other formatting helpers. |
| 4 | `stackModel` moved to `screen_test.go` | Was in `model.go` as a production symbol but used only by `screen_test.go`. Now a test-internal type. |
| 5 | `RunLegacy` removed from `run.go` | Called the now-deleted `New(st, cfg)` constructor. No external callers — safe to remove. |

### Spec drift

- `tui-removed Req 9` says `tabs_test.go` must not exist — SATISFIED (deleted).
- `tui-removed Req 12` (`#C4A7E7` carry-over): H1's `TestRemoved_ForbiddenRosePineMoonHex` correctly excludes `#C4A7E7` from the forbidden list, matching the accepted deviation from batch 1. This drift should be documented in archive.

## Next batch (commit 6)

- Commit 6: implement G-group — Update ladder ordering, OriginatingTab handling, statusMessage with TTL. F-tests get their `t.Skip` removed and turn GREEN.

## Risks carried forward

| Risk | Mitigation |
|------|------------|
| `#C4A7E7` overlap may surface in `sdd-verify` against tui-removed Req 12 | Spec deviation documented; verifier should treat as accepted deviation. |
| `workstation_screen.go` is still present — it has its own in-file usage of `truncate` (now moved to helpers.go) | Checked: workstation_screen.go does NOT define its own truncate post-batch-1 extraction; it uses the one from helpers.go. |
