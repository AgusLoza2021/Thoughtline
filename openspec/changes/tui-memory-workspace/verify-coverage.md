# Verify Coverage — tui-memory-workspace

Mapping of every spec Requirement to the test file/function that exercises it.
Generated as part of S4 (Verify gate) in batch 5 commit 15.

Test command: `go test ./...`

---

## Primary spec: `specs/tui-memory-workspace/spec.md` (25 requirements)

| Req | Title | Covered by |
|-----|-------|------------|
| 1   | Tab Navigation Contract | `flat_model_test.go` — `TestFlatModel_TabDigitJump` (F1), `TestFlatModel_TabCycle` (F2), `TestFlatModel_TabStateAcrossStack` (F3) |
| 2   | Home Tab Layout | `home_screen_test.go` — `TestHomeScreen_BrandAndQuickActionsRender`, `TestHomeScreen_ProjectHealthAndCardsRender` (I1, I2) |
| 3   | Latest Memories Card Interactivity | `home_screen_test.go` — `TestHomeScreen_CursorAndEnterPushesDetail` (I3) |
| 4   | Empty State | `home_screen_test.go` — `TestHomeScreen_EmptyStateRenders` (I4); golden `home_empty.golden` |
| 5   | Memories Tab Layout | `memories_screen_test.go` — `TestMemoriesScreen_PageOneRenders` (J1); golden `memories_page1.golden` |
| 6   | Memories Tab Filters | `memories_screen_test.go` — `TestMemoriesScreen_TypeFilter` (J2), `TestMemoriesScreen_TagFilter` (J3); golden `memories_filtered.golden` |
| 7   | Memories Tab Keybindings | `memories_screen_test.go` — `TestMemoriesScreen_Pagination` (J4), `TestMemoriesScreen_FilterFocusCycle` (J5) |
| 8   | Search Tab | `search_screen_test.go` (carried over from predecessor) + `flat_model_test.go` Quick Action `[/]` test (F5) |
| 9   | Inbox Tab Layout | `inbox_screen_test.go` — `TestInboxScreen_ListRenders` (K1); golden `inbox_three.golden` |
| 10  | Inbox Accept Action | `inbox_screen_test.go` — `TestInboxScreen_AcceptPromotes` (K2) |
| 11  | Inbox Edit Action | `inbox_edit_screen_test.go` — `TestInboxEditScreen_EditThenPromote` (L1-L8); golden `inbox_edit_form.golden` |
| 12  | Inbox Reject Action | `inbox_screen_test.go` — `TestInboxScreen_RejectMarksRejected` (K3); storage layer at `internal/storage/pending_test.go` `TestMarkRejected*` (A4-A5) |
| 13  | Sessions Tab | `sessions_screen_test.go` — `TestSessionsScreen_EmptyState`, `TestSessionsScreen_ListAndCursor` (M1, M2); golden `sessions.golden` |
| 14  | Help Tab | `keybindings_test.go` — `TestKeybindings_RegistryStructure`, `TestHelpScreen_RendersAllGroupTitles`, `TestHelpScreen_RendersRoadmap`, `TestKeybindings_DriftAgainstWiredHotkeys` (N1-N6); golden `help.golden` |
| 15  | Global Quick Actions Hotkeys | `flat_model_test.go` — `TestFlatModel_QuickActions_*` (F5-F6) |
| 16  | Input-Focus Suppression | `flat_model_test.go` — `TestFlatModel_InputFocusSuppression*` (F4, F8 q-literal-in-input) |
| 17  | Memory Detail Full-Screen View | `detail_screen_test.go` — `TestDetailScreen_FullScreenLayout`; golden `detail.golden` |
| 18  | Read-Only Contract for Saved Memories | `memories_screen_test.go` — `TestMemoriesScreen_NoMutationKeys` |
| 19  | Clipboard Copy | `detail_screen_test.go` — `TestDetailScreen_CopyAction`; platform backends compiled via build tags (`clipboard_{windows,darwin,linux}.go`) |
| 20  | Semantic Palette | `theme_test.go` — `TestPalette_TokenStructure`, `TestPalette_NoLegacyHex` (B1) |
| 21  | Brand Logo | `brand_test.go` — `TestRenderBrand_TextNotASCII` (E1); golden `brand.golden` |
| 22  | Minimum Viewport | `flat_model_test.go` — `TestFlatModel_MinViewportFreeze` (F7) with 6 boundary sub-cases (80×24 ok, 79×24 freeze, 80×23 freeze, etc.) |
| 23  | Quit Semantics | `flat_model_test.go` — `TestFlatModel_QuitSemantics_*` (F8: q-quit, q-literal-in-input, ctrl+c-always) |
| 24  | Project Health Card — Consistent Scoping | `home_screen_test.go` — `TestHomeScreen_GlobalCountsNotCwdScoped` (I-bug-1 fix verification) |
| 25  | Memory Detail — Wrap and Scroll | `detail_screen_test.go` — `TestDetailScreen_Wrap`, `TestDetailScreen_Scroll` |

## Passive-capture delta: `specs/passive-capture/spec-delta.md`

| Req | Title | Covered by |
|-----|-------|------------|
| 9 (NEW) | Inbox-Mediated Pre-Promotion Edit | `inbox_edit_screen_test.go` — `TestInboxEditScreen_FormFields` (L7), `TestInboxEditScreen_PromoteWritesEdits` (L8) |

## tui-removed delta: `specs/tui-removed/spec.md`

| Req | Title | Covered by | Notes |
|-----|-------|------------|-------|
| 1   | `--theme` CLI Flag Is Absent | `cmd/thoughtline/main_test.go` — `TestCheckRemovedUIFlags` cases `--theme bare token` / `--theme=value` (O1) | exact-stderr-format assertion + exit-2 contract |
| 2   | `--no-splash` CLI Flag Is Absent | same test — `--no-splash bare` / `--no-splash=false` cases | false-positive guard for `--no-splashy` |
| 3   | `--splash-ms` CLI Flag Is Absent | same test — `--splash-ms bare` / `--splash-ms=value` cases | |
| 4   | `cube.go` Does Not Exist | `removed_test.go` — `TestRemovedFile_cube_go` | |
| 5   | `splash.go` Does Not Exist | `removed_test.go` — `TestRemovedFile_splash_go` | |
| 6   | `logo.go` (Block-Letter) Does Not Exist | `removed_test.go` — `TestRemovedFile_logo_go` | |
| 7   | `workstation_screen.go` Does Not Exist | **NOT COVERED — drift** | File still present; legacy code is dead (not wired into `Run`) but full removal cascades through `dashboard_screen.go` + `recent_screen.go` + their tests. Tracked as follow-up cleanup. Documented in apply-progress batch 5. |
| 8   | `projects_screen.go` Does Not Exist | `removed_test.go` — `TestRemovedFile_projects_screen_go` (added or existing) | Deleted in commit 15 of this batch. |
| 9   | `tabs_test.go` Does Not Exist | `removed_test.go` — `TestRemovedFile_tabs_test_go` | |
| 10  | `themes.go` Does Not Exist | `removed_test.go` — `TestRemovedFile_themes_go` | |
| 11  | Legacy `Model` Type Is Absent | `removed_test.go` — `TestRemovedType_Model` (or compile-time absence — verified by file absence) | |
| 12  | Rose-Pine-Moon Hex Values Are Absent (with one intentional carry-over) | `theme_test.go` — palette structure test confirms only `#C4A7E7` Brand carry-over | ADR 0006 documents the exception |

---

## Notes and drift surfacing

- **Req 7 drift**: `workstation_screen.go`, `dashboard_screen.go`, and the legacy `recent_screen.go` still exist as zombie files (no path from `Run`). They were not deleted in this change because the scope of commit 15's Q1 task was explicitly `projects_screen.go` + `items.go`. A follow-up cleanup change should delete them.
- **Req 12 carry-over**: the Brand color `#C4A7E7` is preserved from Rose-Pine-Moon at the user's explicit request. ADR 0006 documents this as the single carry-over exception to the "no legacy hex" rule.
- **Some Req scenarios verified by goldens, not assertions**: the 10 golden files at `internal/dashboard/testdata/*.golden` are the visual contract for layout-level requirements (2, 4, 5, 6, 9, 11, 13, 14, 17, 21). Regenerate with `go test ./internal/dashboard/ -run "Golden" -update` after intentional UI changes.
- **Clipboard cross-platform smoke (S3)**: only the Windows backend (`clip.exe`) was verified on the dev host. Linux (`wl-copy` / `xclip`) and macOS (`pbcopy`) backends compile via build tags but have not been exercised on real hardware in this batch. Recommended manual test: `thoughtline ui` → tab to Memories → enter on a row → press `C` → verify clipboard contents.

## Verify-gate summary (S1-S4)

| Step | Result |
|------|--------|
| S1 `go test ./...` | 13 packages green, 0 failures |
| S2 `go vet ./...` | clean (no diagnostics) |
| S3 cross-platform clipboard smoke | Windows verified locally; Linux/macOS compile-only |
| S4 spec scenario-to-test audit | This file. 25 + 1 + 12 = 38 requirements mapped; 1 drift surfaced (Req 7) |
