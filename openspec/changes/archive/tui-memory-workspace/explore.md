# Exploration: tui-memory-workspace

> Change: `tui-memory-workspace`
> Phase: explore
> Status: complete
> Supersedes: `tui-redesign` (archived as `tui-redesign-superseded`)

## Codebase snapshot (relevant files)

### internal/dashboard/*

| File | Purpose |
|------|---------|
| `flat_model.go` | LIVE entry point. `flatModel` owns `[]Screen` stack, dispatches push/pop/focus. This is what `Run()` uses. |
| `model.go` | Legacy tab-based `Model` — kept only for its test suite. `tabKey` enum, `New()` constructor. NOT used by the binary. |
| `view.go` | Legacy `Model.View()` — renders 6-tab layout with cube hero, roadmap panel. |
| `update.go` | Legacy `Model.Update()` — routes keys to per-tab logic, cube tick, splash timer. |
| `screen.go` | `Screen` interface (Init/Update/View/Title/OnFocus), `pushScreenCmd`, `popScreenCmd`. Final, no changes needed. |
| `screen_stubs.go` | Constructor shims so `DashboardScreen` compiles without Group G screens. |
| `dashboard_screen.go` | Current root screen: block-letter logo, stat card, 5-item menu. Pushes `WorkstationScreen` for Recent/Projects/Pending. |
| `workstation_screen.go` | 3-pane LazyGit-style layout (sidebar/center/right). NOT in original spec — added as bridge design. 800+ lines. |
| `search_screen.go` | Search tab — textinput + results list. Implements Screen. Reusable. |
| `recent_screen.go` | Recent memories list. Implements Screen. Adaptable to Memories tab. |
| `projects_screen.go` | Alphabetical project list → drill-in to RecentScreen. Likely absorbed into Memories tab filters. |
| `pending_screen.go` | Pending events list + detail push. Basis for Inbox tab. Needs accept/reject actions. |
| `detail_screen.go` | Scrollable viewport for memory content. Needs `[C]copy`, remove edit/delete. |
| `logo.go` | Block-letter ASCII art for "THOUGHTLINE" + gradient renderer. DELETE and replace with text brand. |
| `theme.go` | Single `palette` struct + `defaultPalette` (Rose-Pine-Moon hex). `statusLevel`/`statusStyle` helpers. MODIFY hex values only. |
| `themes.go` | Multi-theme infrastructure: ThemeBrand/ZBrush/Mono, ApplyTheme, nextTheme. DELETE — still referenced by styles.go/model.go. |
| `styles.go` | Package-level lipgloss vars rebuilt by `ApplyTheme`. Coupled to `themes.go`. MODIFY to decouple. |
| `commands.go` | `tickMsg`, `tickCmd`, `loadStatsCmd` for legacy Model. SIMPLIFY after legacy removal. |
| `status.go` | `diskStatusLevel`, `formatStatusLine`. KEEP — used by new Home tab. |
| `cube.go` | Animated ASCII cube. DELETE — referenced only by legacy `model.go`. |
| `splash.go` | Splash screen animation. DELETE — referenced only by legacy `model.go`/`update.go`. |
| `roadmap.go` + `roadmap.yaml` | Roadmap data and `StatusGlyph`. KEEP — moves to Help tab. |
| `items.go` | `sessionItem`, `tagItem`, `memoryItem` for old `bubbles/list`. STRIP down or remove. |
| `tabs_test.go` | 8 tests for legacy `tabKey` cycling on legacy `Model`. DELETE. |
| `model_test.go` | Mixed: some test legacy `Model`, some test roadmap/status. SPLIT — keep roadmap/status tests, delete legacy Model tests. |

### internal/storage/ (relevant)

| File | Status | Key symbols |
|------|--------|-------------|
| `pending.go` | COMPLETE | `CountPending`, `ListPending`, `GetPendingByID`, `MarkPromoted`, `SweepPending` |
| `stats.go` | COMPLETE | `Stats`, `MostRecentProjects []string`, `RecentMemories`, `RecentSessions`, `ByProject map[string]int` |
| `recent.go` | COMPLETE | `RecentAll`, `Recent` (project-scoped) |
| `sessions.go` | COMPLETE | `RecentSessions` |

### cmd/thoughtline/main.go

Current flags registered for `thoughtline ui`:
- `--theme {brand|zbrush|mono}` — DELETE (migration error message on use)
- `--no-splash` — DELETE
- `--splash-ms N` — DELETE
- `--no-update-check` — KEEP

`dashboard.Config` fields to remove: `ThemeName`, `Splash`, `SplashDuration`.

---

## Salvage / Delete / Build Matrix

| File or concept | Status | Rationale |
|---|---|---|
| `flat_model.go` | KEEP | Live entry point, push/pop correct. Add tabs[] + activeTab. |
| `screen.go` | KEEP | Interface already final. |
| `run.go` | KEEP | `Run()` wires flatModel correctly. |
| `theme.go` (struct + helpers) | MODIFY | Struct shape reusable. Replace 14 hex values with new semantic palette. |
| `themes.go` | DELETE | Multi-theme dead. Blocked on decoupling `styles.go` and `model.go` first. |
| `styles.go` | MODIFY | Decouple from `Theme` struct, rebuild against single `palette`. |
| `model.go` (legacy) | DELETE | Once legacy tests rewritten. `stackModel` test helper preserved separately. |
| `view.go` (legacy) | DELETE | Drives only legacy `Model`. |
| `update.go` (legacy) | DELETE | Drives only legacy `Model`. |
| `commands.go` | MODIFY | Simplify after legacy removal. |
| `logo.go` | DELETE/REPLACE | Block-letter art out. New: emoji + text brand `renderBrand()`. |
| `logo_test.go` | MODIFY | Rewrite for new text logo tests. |
| `DashboardScreen` | MODIFY (major) | Home tab: Quick Actions, Project Health card, Latest Memories card, Recent Activity card. No 5-item menu. |
| `WorkstationScreen` | DELETE | Bridge design not in new spec. Extract helpers first. |
| `SearchScreen` | KEEP | Tab 3 Search. Restyling only. |
| `RecentScreen` | MODIFY | Adapt to Memories tab (add pagination, filters). |
| `BrowseProjectsScreen` | DELETE or MERGE | Project filtering becomes a filter in Memories tab, not a screen. |
| `PendingScreen` | MODIFY | Tab 4 Inbox. Add [A]accept / [E]edit-before-accept / [R]reject. |
| `DetailScreen` | MODIFY | Remove edit/delete. Add [C]copy. Better metadata rendering. |
| `screen_stubs.go` | MODIFY | Remove WorkstationScreen stub. Add stubs for SessionsTab, HelpTab. |
| `cube.go` | DELETE | After legacy Model removal. |
| `splash.go` | DELETE | After legacy Model removal. |
| `roadmap.go` + `roadmap.yaml` | KEEP | Move rendering to Help tab. |
| `status.go` + tests | KEEP | Reused in Project Health card. |
| `diskfree_*.go` + tests | KEEP | Already implemented, cross-platform. |
| `updatecheck.go` | KEEP | Orthogonal. |
| `items.go` | DELETE or STRIP | No bubbles/list in new design. |
| `tabs_test.go` | DELETE | Tests dead legacy tabKey. |
| `model_test.go` (legacy tests) | MODIFY | Keep roadmap/status tests; delete legacy Model tests. |
| `teatest_smoke_test.go` | KEEP | Infrastructure. |
| Tab switcher in flatModel | BUILD | New: `tabs []tabDef` + `activeTab tabKey`, keys 1-6, tab/shift+tab. |
| Home tab (new root screen) | BUILD | Quick Actions row, Project Health card, Latest Memories card, Recent Activity card, empty state. |
| Memories tab | BUILD | Paginated memory list, type/tag/scope/sort filters. |
| Memory detail overlay | BUILD | Full-screen stack push OR true modal — TBD per open question. |
| Inbox tab with accept/reject | BUILD | Adapt PendingScreen + promote/reject write path. |
| Sessions tab (restyled) | BUILD (small) | Extract session rendering from WorkstationScreen. |
| Help tab | BUILD (small) | Keybindings + roadmap. |
| Empty state (no memories) | BUILD | Friendly copy + 3 gamedev examples. |
| `helpers.go` | BUILD | Extract relTime, truncateLeft, paneBox, clampCursor, wrap, padRight, centerString from WorkstationScreen before deletion. |
| `--theme/--no-splash/--splash-ms` removal | BUILD | main.go flag cleanup + migration error message. |

---

## Open Questions for the User

1. **Global hotkeys vs Home-only**: Are `[S]`save · `[/]`search · `[M]`memories · `[I]`inbox Quick Actions global hotkeys active on ALL tabs, or only on the Home tab?

2. **Detail view — overlay or full-screen push?**: The user said "overlay/modal on top of active tab." In Bubbletea, true overlays require the underlying tab to render and then be overdrawn. The simpler (and current) approach is a full-screen stack push. Which is preferred?

3. **Inbox `[E]` edit-before-accept scope**: Can the user edit the full content body, or only the proposed type and title before `MarkPromoted` is called?

4. **Inbox `[E]` confirmation step**: Preview/confirm step before promoting, or immediate submit after editing?

5. **Latest Memories card on Home — interactive?**: Can the user press `enter` on a memory in the Latest Memories card to open the detail view, or is it a read-only display list?

6. **Pagination size for Memories tab**: 20 per page? 50? Page indicator or infinite scroll/virtual scroll?

7. **Memories tab filter UX**: Inline filter bar (always visible) or popup triggered by `f`?

8. **Scroll position preservation**: On Memories tab → detail → `esc` back, does cursor return to same position?

9. **Live refresh on Home**: Auto-refresh every 30s (current behavior) or only on `r`?

10. **Filter persistence across tab switches**: Does the Memories tab filter reset when switching tabs?

11. **Sessions tab scope**: Just restyle, or add open/closed grouping + session search?

12. **`[C]copy` in scope for this change?**: Clipboard access is OS-specific (clip.exe/pbcopy/xclip). Recommend deferring to a small follow-up change.

13. **`--theme`/`--no-splash` removal**: Should there be a user-friendly migration error message on flag use, or silent removal?

14. **Rounded borders**: Required (with graceful fallback) or best-effort only?

---

## Constraints and Risks

- **~40 of 80+ dashboard tests target the legacy Model**. When legacy Model is deleted, they die. Strict TDD means new flatModel tests must be written first (RED), then legacy tests deleted in one commit.
- **`themes.go` + `theme.go` dual existence**: Both compile today. `styles.go` calls `ApplyTheme(ThemeBrand)` (from themes.go), `model.go` stores a `Theme` field. This coupling must be fully severed before `themes.go` can be deleted — this is the first prerequisite task, not an afterthought.
- **`workstation_screen.go` shared helpers**: `relTime`, `truncateLeft`, `wrap`, `paneBox`, `clampCursor`, `padRight`, `centerString`, `formatLastSave` are defined here. Extract to `helpers.go` before deletion.
- **`flatModel` tab adaptation**: Currently stack-only. Must add `tabs []tabDef` + `activeTab` + tab key intercept (1-6, tab/shift+tab) at the `flatModel` level. This is the architectural centerpiece.
- **Clipboard `[C]copy` risk**: OS-specific, brittle. Recommend scoping out of this change unless user confirms it's required.
- **`cube.go` / `splash.go` still referenced** by legacy `model.go`/`update.go`. Cannot delete until those files are gone or references removed.

---

## Recommended Next Step

The proposal phase should focus on:
1. **Resolve the 5 highest-priority open questions** (detail overlay vs push, global hotkeys, Inbox edit scope, Home Latest Memories interactivity, clipboard) — these determine interface contracts.
2. **Define the tab+stack coexistence architecture** in `flatModel` — tabs for main navigation, stack for detail overlays on any tab.
3. **Define the cleanup-first sequencing** — legacy Model + themes.go removal as commit #1 before any new UI work begins, to avoid the dual-palette conflict.
