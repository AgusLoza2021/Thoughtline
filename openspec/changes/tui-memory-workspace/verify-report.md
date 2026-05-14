# Verify Report — tui-memory-workspace

> Phase: verify
> Date: 2026-05-14
> Verifier: sdd-verify (opus 4.7)
> Subject: HEAD 464c503
> Spec set: 25 primary + 1 passive-capture delta + 12 tui-removed = 38 requirements

## Executive Summary

Verdict: ACCEPT-WITH-WARNINGS with 3 CRITICAL defects in the Inbox promotion paths that block archive until either remediated or formally accepted as known issues. go test ./... is green across all 13 packages and go vet ./... is clean. The 15-commit series matches the design plan in order. Negative-assertion deletion contract holds for cube/splash/logo/themes/legacy-Model/projects-screen/tabs-test, and the Rose-Pine-Moon hex purge is complete except for the pre-disclosed #C4A7E7 Brand carry-over (plus two additional #C4A7E7 assignments to legacy alias slots Tag and StatNumber that exceed the spec scenario allow-list). The one pre-disclosed Req 7 drift (workstation_screen.go still on disk) is confirmed. The CRITICAL findings are all in inbox_screen.go and inbox_edit_screen.go.

Recommendation: block sdd-archive until the user explicitly accepts the 3 CRITICAL findings as deferred follow-up or until they are remediated.

## Build and Lint Health

| Check | Status | Detail |
|-------|--------|--------|
| go test ./... | PASS | 13 packages cached green, 0 failures |
| go vet ./... | PASS | no diagnostics |
| Commit series matches design (15 commits) | PASS | fdf266d through 4f27030 in order |
| Working tree clean | PASS | git status reports no modifications |
| Schema v5 (rejected status) | PASS | schema.go:177 declares schemaV5SQL with CHECK constraint including rejected |
| Only new storage method is MarkRejected | PASS | internal/storage/pending.go:233-261 |
| --no-update-check only retained UI flag | PASS | cmd/thoughtline/main.go:117 |
| README no longer mentions removed flags/cube/splash | PASS | grep finds zero matches |
| CHANGELOG has BREAKING/Added/Changed/Storage sections | PASS | lines 9, 19, 29, 34 |

## Requirement Verification

### Primary spec (Reqs 1-25)

| Req | Title | Test | Status | Note |
|-----|-------|------|--------|------|
| 1 | Tab Navigation Contract | TestFlatModel_F1_TabDigitJump, F2_TabCycle, F3_TabSurvivesPushPop | PASS | All three scenarios covered |
| 2 | Home Tab Layout | TestHomeScreen_I1_HeaderPresent, I1_QuickActionsRow, I2_ProjectHealthCard, I6_RecentActivityCard | PASS | Test names diverge from verify-coverage.md but coverage equivalent |
| 3 | Latest Memories Card Interactivity | TestHomeScreen_I3_CursorMovesOnJK, I3_EnterPushesDetail | PASS | Cursor preservation across detail push/pop covered via F3 |
| 4 | Empty State | TestHomeScreen_I4_EmptyState, I4_EmptyStateNotShownWhenMemoriesExist | PASS | home_empty.golden asserts scene-pattern / perf-gotcha / pipeline-step / tl save / tl_save |
| 5 | Memories Tab Layout | TestMemoriesScreen_J1_Pagination_*, J6_DefaultSort | PASS | Default sort = updated_at DESC verified |
| 6 | Memories Tab Filters | TestMemoriesScreen_J2_FilterCycling, J3_FilterClear, J9_FilterBarVisible | PASS | All four filter dimensions reachable |
| 7 | Memories Tab Keybindings | TestMemoriesScreen_J7_CursorNoWrapAtEdges, J8_EnterPushesDetail | PASS | No-wrap, enter pushes detail verified |
| 8 | Search Tab | search_screen_test.go + TestFlatModel_F5_SlashJumpsAndFocuses | PASS | |
| 9 | Inbox Tab Layout | TestInboxScreen_L1_ListsPendingRows, L1_EmptyState | PASS | A/E/R affordances rendered |
| 10 | Inbox Accept Action | TestInboxScreen_L2_AKeyAcceptsPassthrough | PARTIAL | Test asserts row removal + count decrement, NOT that created memory title/content match proposed values. See CRITICAL C1. |
| 11 | Inbox Edit Action | TestInboxEditScreen_L5_FieldCycling, L7_EscCancels, L8_InputFocusedAlwaysTrue | PARTIAL | Cycle/cancel/focus covered, no L6 test for ctrl+s submit-and-assert-edited-memory. See CRITICAL C2. |
| 12 | Inbox Reject Action | TestInboxScreen_L4_RKeyRejects + pending_test.go TestMarkRejected* | PASS | Two-layer coverage |
| 13 | Sessions Tab | sessions_screen_test.go | PASS | Read-only contract verified |
| 14 | Help Tab | TestKeybindings_RegistryStructure, TestHelpScreen_RendersAllGroupTitles, TestHelpScreen_RendersRoadmap, TestKeybindings_DriftAgainstWiredHotkeys | PASS | Drift test is a strong guard |
| 15 | Global Quick Actions Hotkeys | TestFlatModel_F5_SlashJumpsAndFocuses + F4 family | PASS | |
| 16 | Input-Focus Suppression | TestFlatModel_F4_InputFocusGuard | PASS | Covers m, s, digit, q literal-in-input |
| 17 | Memory Detail Full-Screen View | TestDetailScreen_K1_RendersContent, TestDetailScreen_OriginatingTab* | PASS | All 8 metadata fields rendered; esc returns to originator |
| 18 | Read-Only Contract | TestDetailScreen_K3_EKeyNoEffect, K3_DKeyNoEffect | PASS | Source-level absence guarded |
| 19 | Clipboard Copy | TestDetailScreen_K2_CKeyTriggersClipboard, K2_ClipboardSuccessStatus, K2_ClipboardFailureStatus | PASS | Failure path surfaces status, does not crash |
| 20 | Semantic Palette | theme_test.go TestPalette_* | PASS | Six token roles verified |
| 21 | Brand Logo | brand_test.go TestRenderBrand_* + brand.golden | PASS | No ASCII block letters |
| 22 | Minimum Viewport | TestFlatModel_F7_MinViewport | PASS | 6 boundary sub-cases including q-quit-under-min |
| 23 | Quit Semantics | TestFlatModel_F8_QuitSemantics | PASS | q-quit, q-literal-in-input, ctrl+c-always |
| 24 | Project Health Consistent Scoping | TestHomeScreen_IBug1_GlobalScoping | PASS | All-projects totals confirmed |
| 25 | Memory Detail Wrap and Scroll | TestDetailScreen_KBug1_LongLineWraps, KBug1_ContentWrapsOnWindowResize, KBug2_ScrollWithArrowKeys, KBug2_FooterScrollPercent | PASS | Wrap + scroll + footer indicator |

### passive-capture delta (Req 9)

| Req | Title | Test | Status | Note |
|-----|-------|------|--------|------|
| 9 (NEW) | Inbox-Mediated Pre-Promotion Edit | TestInboxEditScreen_L7_EscCancels (cancel only) | FAIL | Edited-accept scenario untested; production code reveals project hardcoding violating the spec. See CRITICAL C2 and C3. |

### tui-removed (Reqs 1-12)

| Req | Title | Verification | Status | Note |
|-----|-------|--------------|--------|------|
| 1 | --theme flag absent | grep cmd/thoughtline/main.go | PASS | 0 matches as flag name; appears only in migration-error helper string list |
| 2 | --no-splash flag absent | grep same | PASS | only in removal helper list |
| 3 | --splash-ms flag absent | grep same | PASS | only in removal helper list |
| 4 | cube.go absent | Glob | PASS | file does not exist |
| 5 | splash.go absent | Glob | PASS | |
| 6 | logo.go absent | Glob | PASS | |
| 7 | workstation_screen.go absent | Glob | FAIL pre-disclosed drift | File still present on disk. Treated as WARNING per pre-disclosure agreement. |
| 8 | projects_screen.go absent | Glob | PASS | deleted in commit 15 |
| 9 | tabs_test.go absent | Glob | PASS | |
| 10 | themes.go absent + multi-theme symbols absent | Glob + grep | PASS | symbol matches only in test files asserting absence |
| 11 | Legacy Model type absent | Glob + grep | PASS-WITH-TEST-DEVIATION | model.go/view.go/update.go confirmed absent. Spec scenario greps for literal tabKey; implementation kept that name for the NEW flatModel tab enum and renamed the assertion target to legacyTabKey. See WARNING W1. |
| 12 | Rose-Pine-Moon hex absent (except #C4A7E7 carry-over) | grep | PASS-WITH-SCOPE-CREEP | 10 forbidden hex codes return zero production matches. #C4A7E7 appears 4 times in theme.go: lines 77 (Brand allowed), 91 (BadgeMemoryType allowed), 95 (Tag NOT allowed), 96 (StatNumber NOT allowed). See WARNING W2. |

## Findings

### CRITICAL (3 items)

C1 — Inbox [A] accept hardcodes title and dumps raw payload as content (Req 10)
- File: internal/dashboard/inbox_screen.go:232-239
- Defect: the accept-as-is path builds the memory with Title set to literal "Inbox capture" and Content set to ev.Payload (raw JSON). Spec scenario "Accept promotes unchanged" says proposed type decision, title T, content C flow into the new memory. The implementation cannot satisfy that contract because it never extracts proposed title or content from the payload JSON.
- Why tests miss it: TestInboxScreen_L2_AKeyAcceptsPassthrough only asserts the row disappears and CountPending decremented. It does NOT load the new memory and assert its title/content.
- Remediation: parse ev.Payload JSON, extract title and content, pass them through to memory.Memory. Add a test that loads the saved memory after [A] and asserts title/content match.

C2 — Inbox [E] edit submit hardcodes Project to "test-workspace" (passive-capture Req 9)
- File: internal/dashboard/inbox_edit_screen.go:222, 228, 233
- Defect: promoteCmd calls ResolveOrCreateBrainID(ctx, "test-workspace") and builds the memory with Project: "test-workspace". This is a test fixture name leaking into production. The passive-capture spec delta Req 9 explicitly requires: the created memory MUST inherit project, session_id (when present), and captured_at from the original pending_events row. The scenario shows project = game-x flowing through the edited path.
- Impact: any user editing a pending capture in production gets their memory filed under "test-workspace" instead of the captured project. Session linkage and captured_at also dropped.
- Why tests miss it: no L6 test submits the edit form and inspects saved memory project. Existing L5/L7/L8 cover only field cycling, esc cancel, and InputFocused.
- Remediation: thread pending.Event (or at least Project, SessionID, CapturedAt) into NewInboxEditScreen and use those values in promoteCmd. Add TestInboxEditScreen_L6_SubmitPromotesWithEdits that asserts new memory project equals ev.Project and type/title/content equal edited values.

C3 — Inbox loadCmd hardcodes Project filter to "test-workspace" (Req 9 layout)
- File: internal/dashboard/inbox_screen.go:200-218
- Defect: ListPending is called with Project: "test-workspace". On failure it falls back to a global query. In production: if any project named test-workspace exists with pending rows, those alone are shown; otherwise the fallback path silently shows ALL pending rows globally. Neither matches Req 9 for the current project context.
- Why tests miss it: tests seed pending events under "test-workspace" (the fixture), so the hardcode appears to work.
- Remediation: derive the project filter from Config.Project (or unset for all-projects, per Req 24 default policy) — never from a string literal. Add a test that seeds under a non-fixture project and asserts they render.

### WARNING (5 items)

W1 — Test asserts legacyTabKey instead of literal tabKey (tui-removed Req 11)
- File: internal/dashboard/removed_test.go:123
- Detail: spec scenario greps for literal tabKey. Implementation kept that name for the NEW flatModel tab enum (flat_model.go:29) and renamed the assertion target to legacyTabKey. Legacy type is genuinely gone, but the test deviates from the spec literal grep.
- Recommendation: update the spec wording, or rename the new tabKey to workspaceTab and restore the literal tabKey grep target.

W2 — #C4A7E7 appears on Tag and StatNumber legacy alias slots (tui-removed Req 12 scenario 2)
- File: internal/dashboard/theme.go:95-96
- Detail: spec Req 12 scenario 2 says #C4A7E7 MAY appear "but ONLY assigned to the Brand and/or BadgeMemoryType palette tokens." Implementation also assigns it to Tag and StatNumber as legacy aliases (line 94 comment: Legacy aliases map onto the new tokens). Semantically these slots collapsed onto the same purple, intentional bridges from the pre-rename palette.
- Recommendation: delete Tag and StatNumber from palette and migrate call sites to Brand / BadgeMemoryType, or amend the spec scenario to allow legacy alias slots.

W3 — workstation_screen.go still on disk (tui-removed Req 7) pre-disclosed
- File: internal/dashboard/workstation_screen.go
- Detail: file is dead (no path from Run) but not deleted. Documented in verify-coverage.md and apply-progress.md batch 5.
- Recommendation: open a follow-up cleanup change titled tui-zombie-cleanup to delete workstation_screen.go plus the legacy dashboard_screen.go and recent_screen.go.

W4 — Cross-platform clipboard smoke not verified (Req 19, deferred S3)
- Detail: only Windows clip.exe was runtime-verified. macOS pbcopy and Linux wl-copy/xclip backends compile via build tags and have unit tests but lack runtime verification on real hardware. Pre-disclosed in verify-coverage.md.
- Recommendation: run manual smoke on CI runners or by a platform-equipped contributor before tagging a release.

W5 — Schema v5 migration was not in the original design (pre-disclosed)
- File: internal/storage/schema.go:177-204
- Detail: schema v5 added during batch 4 to support MarkRejected rejected status. Idempotent, data-preserving, properly versioned. Spec did not anticipate this, but it is a necessary corollary of Req 12. Pre-disclosed in batch 4 apply-progress.
- Recommendation: amend the spec or accept the deviation in the archive report.

### SUGGESTION (4 items)

S1 — Test names diverge from verify-coverage.md. Map cites TestFlatModel_TabDigitJump; actual is TestFlatModel_F1_TabDigitJump. Regenerate verify-coverage.md to match.

S2 — inbox_edit_screen_test.go does not exist as a standalone file; tests live inside inbox_screen_test.go. Rename the file or update the coverage map.

S3 — acceptCmd builds TopicKey as inbox/EventType/ID, which may collide with user-authored topic keys. Consider scoping under inbox-promoted/EventType/ID or omitting.

S4 — removed_test.go and styles_test.go define independent forbidden-symbol lists that will drift. Consolidate.

## Recommendation

Verdict: ACCEPT-WITH-WARNINGS conditional on user acknowledgment of the 3 CRITICAL findings as deferred items.

If the user agrees to defer C1, C2, C3 to a follow-up change (e.g., tui-inbox-promotion-fidelity), proceed to sdd-archive and record the deferrals in the archive report. If the user wants the contract fully honored at this change boundary, return to sdd-apply to remediate the three Inbox project/title/content bugs and add the L6 submit-edited test.

The build, lint, deletion contract, and 22 of 25 primary requirements are solid. The defects concentrate in the Inbox promotion subsystem and share a single root cause: existing tests assert lifecycle (row removed, count decremented) but never load the saved memory and assert its fields. Three field-level assertion tests would catch all three CRITICALs.
