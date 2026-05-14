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

## Next batch (commits 5–6)

- Commit 5: delete legacy mass (`model.go`, `view.go`, `update.go`, `cube.go`, `splash.go`, `themes.go`, `tabs_test.go`, plus `logo.go` and its remaining `LogoGradient` field reference). Slim `model_test.go` to roadmap/status only. Rewrite `logo_test.go` for the new `renderBrand()`.
- Commit 6: implement G-group (`tabKey` enum / `defaultTabs` array already exist; add Update ladder ordering, OriginatingTab handling, statusMessage with TTL). F-tests get their `t.Skip` removed and turn GREEN.

## Risks carried forward

| Risk | Mitigation |
|------|------------|
| Removing `LogoGradient` in commit 5 will require simultaneously deleting `logo.go` | Already in the batch-2 plan; reviewed. |
| `#C4A7E7` overlap may surface in `sdd-verify` against tui-removed Req 12 | Spec deviation documented above; verifier should treat as an accepted deviation. |
| Legacy `legacyTabKey` rename ripples through any not-yet-discovered consumer | grep already covered all 5 source files; if more references appear they will surface at commit 5 deletion time. |
