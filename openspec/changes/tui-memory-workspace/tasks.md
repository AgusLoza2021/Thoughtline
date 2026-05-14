# Tasks: tui-memory-workspace

> Change: `tui-memory-workspace`
> Status: Batch 4 complete (commits 7–10). Remaining: commits 11–15 (batch 5).
> Strict TDD: enforced — every code task preceded by its test task in the same commit batch.
> Test command: `go test ./...`
> Total tasks: 96 across 19 groups, mapped to 15 commits.

---

## Reconciliation Notes (predecessor + spec/design contradictions resolved here)

| # | Drift | Spec / Design says | Reality verified | Resolution baked into tasks |
|---|-------|--------------------|------------------|-----------------------------|
| 1 | Storage surface for Inbox `[R]` reject | Spec Req 12: "MUST mark that row as rejected (no promotion)". Design (Section 6, `pending_screen.go` row): "calls `MarkRejected` if exists, else `SweepPending(id)`". | `grep` of `internal/storage/pending.go` shows only `CountPending`, `ListPending`, `GetPendingByID`, `MarkPromoted`, `SweepPending`. NO `MarkRejected`. `SweepPending` takes `(retentionDur, hardDeleteDur, now)` — it cannot reject a single row by id. | **Add a new `func (s *Storage) MarkRejected(ctx, id) error`** that updates the row's `status` from `'pending'` to `'rejected'`. This is the only minimal storage addition this change needs. Tracked as Group A (A4 RED, A5 GREEN). |
| 2 | `CountPending`, `ListPending`, `MostRecentProjects` existence | Explore says all exist (`pending.go`, `stats.go`). | Grep confirmed `CountPending` (pending.go:15), `ListPending` (pending.go:125), `MarkPromoted` (pending.go:192), `SweepPending` (pending.go:236), `MostRecentProjects` (stats.go:47/85). | No new storage helpers needed beyond `MarkRejected`. Group A tasks read-verify these via short table-driven tests, not re-implement. |
| 3 | `q` literal-in-input vs `q` quit (Spec Req 23 & 16) | Spec Req 23 (#3 scenario): pressing `q` while textarea focused → literal `q`. Design Section 5 / Section 2b(B): step (1) global quit handles `ctrl+c` always, `q` only when stack empty AND not `InputFocused`. | Update ladder order must put the `InputFocused` check BEFORE the bare `q` quit. The design ordering (1) "quit: ctrl+c, or 'q' when stack empty AND not InputFocused" already encodes this when read carefully. | F8 test task explicitly covers all three scenarios (q quit, q literal-in-input, ctrl+c always). G2 implementation task wires the ladder in the design's documented order. No ambiguity remains. |
| 4 | Tab keys `1`-`6` typed inside an input | Spec Req 1 says digits jump tabs. Spec Req 16 says inputs swallow Quick Action letters. Digits are NOT in the Quick Action set. | Design Section 5 step (4) tab intercept runs AFTER the InputFocused guard. So if a user has an input focused and types `2`, the digit goes to the input, not the tab switcher. | F4 / G2 tasks make this explicit: the InputFocused guard fires for digits too, not just `s/m/i//`. This is the safer reading because typing project names like `r2d2` into the tag filter would otherwise tab-switch mid-type. |
| 5 | Verify test seeding helpers | Design Section 2b(L) names `seedMemories(t, s, n int)` as a new helper. Spec scenarios use seeded counts (7 memories, 50 memories for pagination, 5/3 split by type). | Codebase currently has `newTestStorage(t)` only (per Section 2b(L) text and Section 8 testing strategy). No `seedMemories` helper exists. | A6 task creates `seedMemories(t, s, n int)` + `seedPending(t, s, n int)` in a new `internal/dashboard/testhelpers_test.go`. All later integration tests depend on this. |
| 6 | `recent_screen.go` rename vs delete | Design Section 6 says "Rename to `MemoriesScreen`" but file row still says `recent_screen.go` Modify. Existing file `recent_screen.go` becomes `memories_screen.go`. | Same file conceptually, new name. | J1+ tasks treat this as a rename + major rewrite. The file `recent_screen.go` is renamed to `memories_screen.go` in the same commit it gets the new fields. No leftover `recent_screen.go`. |
| 7 | `--theme=value` vs `--theme value` migration scan | Decision #8 says exact-token match "with or without `=value`". | `--theme brand` (space separator) and `--theme=brand` (equals separator) are both common shell usages. | O1 test task covers both forms PLUS a guard for the false-positive case (`--no-splashy` MUST NOT match `--no-splash`). |
| 8 | Spec Req 22 minimum viewport — exact threshold edges | Spec says "< 80 columns wide or < 24 rows tall". | Implementation must treat 80×24 as OK (not warning) and 79 or 23 as warning. | F7 test task spells out the inclusive/exclusive boundary: assert OK at exactly 80×24, warning at 79×24 AND at 80×23. |
| 9 | `OnFocus` semantics for Home refresh | Decision #4: "implicit refresh when Home regains focus via `OnFocus`". | `OnFocus` is already part of `Screen` interface (per Section 7). | G3 task implements `setActiveTab` so it calls `OnFocus()` on the newly-active tab. I-task tests confirm Home re-queries stats on focus. |

---

## Task entry format

Each task uses this block:

```
### X1 — Test|Code|Docs|Infra: <one-line description>
- **Type**: test | code | docs | infra
- **Effort**: S | M | L
- **Depends on**: <task ids or "none">
- **Spec ref**: <Requirement ID(s)>
- **Design ref**: <design section(s)>
- **Files**: <paths affected>
- **Acceptance criteria**:
  - bullet list, at least one verbatim spec scenario or design decision
```

---

## Group A — Storage and test fixtures (prerequisites)

### A1 — Test: confirm `CountPending` returns only pending-status rows
- **Type**: test
- **Effort**: S
- **Depends on**: none
- **Spec ref**: passive-capture Req 9 (precondition)
- **Design ref**: Section 6 (storage row), Reconciliation #2
- **Files**: `internal/storage/pending_test.go` (existing — add or assert presence of test case)
- **Acceptance criteria**:
  - The existing `TestCountPending_ReturnsOnlyPendingStatus` is present and green; if absent, add it
  - Promoted and rejected rows must NOT be counted

### A2 — Test: confirm `ListPending` returns row payloads needed by Inbox tab
- **Type**: test
- **Effort**: S
- **Depends on**: none
- **Spec ref**: Req 9 (Inbox layout)
- **Design ref**: Section 5 (`loadPendingCmd`), Reconciliation #2
- **Files**: `internal/storage/pending_test.go`
- **Acceptance criteria**:
  - Test seeds 3 pending rows with distinct types/titles; `ListPending` returns rows containing source, proposed type, content, and id used by `[A]`/`[E]`/`[R]`
  - Verbatim spec ref: Req 9 Scenario "Inbox renders pending rows with affordances" — the rows returned must carry the fields the UI binds to

### A3 — Test: confirm `MostRecentProjects` ordering
- **Type**: test
- **Effort**: S
- **Depends on**: none
- **Spec ref**: Req 2 (Home header project name)
- **Design ref**: Section 4.1, Section 6 (`stats.go` row)
- **Files**: `internal/storage/stats_test.go`
- **Acceptance criteria**:
  - Existing `TestStats_MostRecentProjects` stays green; assert top-4 by `MAX(updated_at)` DESC
  - Used by Home header to display "current project"

### A4 [x] — Test: `storage.MarkRejected(ctx, id)` updates status from `pending` to `rejected`
- **Type**: test
- **Effort**: S
- **Depends on**: none
- **Spec ref**: Req 12 (Inbox reject), passive-capture Req 9 (payload-untouched contract — extended here to reject path)
- **Design ref**: Reconciliation #1, Section 6 (`pending_screen.go` row)
- **Files**: `internal/storage/pending_test.go` (new test function `TestMarkRejected_*`)
- **Acceptance criteria**:
  - Test seeds a pending row, calls `MarkRejected(ctx, id)`, asserts row's `status = 'rejected'`
  - Test asserts `CountPending` decreases by 1 after MarkRejected
  - Test asserts the row's `payload` column is byte-for-byte unchanged (audit fidelity, mirrors passive-capture Req 9)
  - Test asserts MarkRejected on an already-promoted row returns an error or no-op (locked behavior: returns error, like MarkPromoted does for already-promoted)

### A5 [x] — Code: implement `storage.MarkRejected`
- **Type**: code
- **Effort**: S
- **Depends on**: A4
- **Spec ref**: Req 12
- **Design ref**: Reconciliation #1
- **Files**: `internal/storage/pending.go` (add ~15 lines)
- **Acceptance criteria**:
  - Tests from A4 turn GREEN
  - SQL: `UPDATE pending_events SET status='rejected', updated_at=? WHERE id=? AND status='pending'`
  - Returns `ErrNotPending` (or equivalent existing sentinel) when row is not in `pending` state

### A6 [x] — Code: dashboard test helpers `seedMemories` and `seedPending`
- **Type**: code
- **Effort**: S
- **Depends on**: A1, A2 (familiarity with surfaces)
- **Spec ref**: Req 5, 6, 9, 10, 11, 12 (every integration test that needs seeded rows)
- **Design ref**: Section 2b(L), Section 8 (testing strategy), Reconciliation #5
- **Files**: `internal/dashboard/testhelpers_test.go` (NEW)
- **Acceptance criteria**:
  - `func newTestStorage(t *testing.T) *storage.Storage` — already exists per design; verify and re-export if needed
  - `func seedMemories(t *testing.T, s *storage.Storage, n int)` — inserts n memories with deterministic type/topic_key/content/updated_at
  - `func seedPending(t *testing.T, s *storage.Storage, n int)` — inserts n pending rows with distinct proposed types
  - Both helpers use `t.Helper()` and `t.TempDir()` semantics
  - Table-driven self-tests included (helper of helpers, optional but recommended)

---

## Group B — Theme + palette

### B1 [x] — Test: palette has exactly six semantic tokens and no Rose-Pine-Moon hex
- **Type**: test
- **Effort**: S
- **Depends on**: none
- **Spec ref**: Req 20 (Semantic Palette), tui-removed Req 12 (Rose-Pine-Moon hex absence)
- **Design ref**: Section 3 (palette table)
- **Files**: `internal/dashboard/theme_test.go` (new file or extend existing)
- **Acceptance criteria**:
  - Asserts the palette declares the six semantic roles from Req 20: `nav`, `brand`, `success`, `warning`, `destructive` (alias `Error`), `muted`
  - Asserts none of the 11 forbidden Rose-Pine-Moon hex values from tui-removed Req 12 appear in `theme.go` source
  - Verbatim scenario: tui-removed Req 12 "No Rose-Pine-Moon hex codes remain in dashboard sources"

### B2 [x] — Code: implement semantic palette per Section 3 table
- **Type**: code
- **Effort**: M
- **Depends on**: B1
- **Spec ref**: Req 20, Req 21 (brand)
- **Design ref**: Section 3
- **Files**: `internal/dashboard/theme.go`
- **Acceptance criteria**:
  - Tests from B1 turn GREEN
  - `palette` type and `defaultPalette` value contain every token listed in Section 3 with its exact hex
  - Old Rose-Pine-Moon hex values removed from file

### B3 [x] — Test: status style helpers map level→token correctly
- **Type**: test
- **Effort**: S
- **Depends on**: B2
- **Spec ref**: Req 20 (success/warning/destructive semantic use)
- **Design ref**: Section 3, Section 6 (theme.go row "Keep statusLevel/statusStyle helpers")
- **Files**: `internal/dashboard/theme_test.go`
- **Acceptance criteria**:
  - `statusLevel(StatusOK)` resolves to `Success` color
  - `statusLevel(StatusWarn)` resolves to `Warning`
  - `statusLevel(StatusErr)` resolves to `Error`
  - Verbatim scenario: Req 20 "Success color is not used for navigation" — the test asserts the success token is NOT the value of `Nav` or `NavActive`

### B4 [x] — Code: status style helpers refactored against new palette
- **Type**: code
- **Effort**: S
- **Depends on**: B3
- **Spec ref**: Req 20
- **Design ref**: Section 3
- **Files**: `internal/dashboard/theme.go`
- **Acceptance criteria**:
  - Tests from B3 turn GREEN
  - `statusLevel` and `statusStyle` helpers use only the new tokens

---

## Group C — Helpers extraction from WorkstationScreen

### C1 [x] — Test: behavior tests for the eight helpers
- **Type**: test
- **Effort**: M
- **Depends on**: none
- **Spec ref**: (Infrastructure — supports Req 2, 5, 17 rendering)
- **Design ref**: Section 2b(N), Section 6 (`helpers.go`/`helpers_test.go` rows)
- **Files**: `internal/dashboard/helpers_test.go` (NEW)
- **Acceptance criteria**:
  - Table-driven `t.Run` for each: `relTime`, `truncateLeft`, `wrap`, `paneBox`, `clampCursor`, `padRight`, `centerString`, `formatLastSave`
  - Cases include: empty string, exact-width, overflow, negative cursor, max cursor, zero time, future time
  - At least one case per helper matches the current `workstation_screen.go` behavior verbatim (so the extraction is provably behavior-preserving)

### C2 [x] — Code: extract helpers from `workstation_screen.go` to `helpers.go`
- **Type**: code
- **Effort**: M
- **Depends on**: C1
- **Spec ref**: (Infrastructure)
- **Design ref**: Section 2b(N) "Pure copy with identical signatures"
- **Files**: `internal/dashboard/helpers.go` (NEW); `internal/dashboard/workstation_screen.go` (remove the now-extracted functions OR leave aliases — see acceptance)
- **Acceptance criteria**:
  - Tests from C1 turn GREEN
  - All eight functions live in `helpers.go` with identical signatures
  - `workstation_screen.go` still compiles by either (a) importing the new locations or (b) having its own copies removed and re-referenced — pick (b) to avoid double definition
  - `go test ./...` green

### C3 [x] — Test: WorkstationScreen still compiles and renders against extracted helpers (smoke)
- **Type**: test
- **Effort**: S
- **Depends on**: C2
- **Spec ref**: (Infrastructure — guards against drift before H2 deletes WorkstationScreen)
- **Design ref**: Section 9 commit 2 acceptance
- **Files**: `internal/dashboard/workstation_screen_test.go` (existing if present; otherwise a tiny smoke test alongside)
- **Acceptance criteria**:
  - `go test ./...` green
  - Any pre-existing workstation tests still pass — proves the extraction did not change observable behavior

---

## Group D — Style decoupling

### D1 [x] — Test: `styles.go` builds against single palette with no `ApplyTheme` reference
- **Type**: test
- **Effort**: S
- **Depends on**: B2
- **Spec ref**: tui-removed Req 10 (themes.go and multi-theme symbols absent)
- **Design ref**: Section 1, Section 9 commit 3
- **Files**: `internal/dashboard/styles_test.go` (new or extend)
- **Acceptance criteria**:
  - Test calls every exported style constructor (`headerStyle`, `tabActiveStyle`, `tabInactiveStyle`, `cardStyle`, `cardFocusedStyle`, `badgeStyle`, `statusOK/Warn/Err`) and asserts each returns a non-empty `lipgloss.Style`
  - A source-level assertion (or `grep`-equivalent helper) asserts `styles.go` contains no references to `ApplyTheme`, `ThemeBrand`, `ThemeZBrush`, `ThemeMono`, or `nextTheme`

### D2 [x] — Code: rebuild `styles.go` against the new single palette
- **Type**: code
- **Effort**: M
- **Depends on**: D1, B4
- **Spec ref**: Req 20
- **Design ref**: Section 6 (`styles.go` row enumerates the functions)
- **Files**: `internal/dashboard/styles.go`
- **Acceptance criteria**:
  - Tests from D1 turn GREEN
  - `styles.go` reads only from the single `palette` defined in `theme.go`
  - No symbol from `themes.go` is referenced

### D3 [x] — Test: legacy `Model` still compiles intact for now (transitional)
- **Type**: test
- **Effort**: S
- **Depends on**: D2
- **Spec ref**: (Infrastructure — temporary safety check before H2)
- **Design ref**: Section 9 commit 3
- **Files**: `internal/dashboard/model_test.go` (existing; assert smoke compile)
- **Acceptance criteria**:
  - `go test ./...` green
  - Legacy Model's tests still pass — proves D2's decoupling did not break the legacy paths that still exist between commits 3 and 5

---

## Group E — Brand text + logo replacement

### E1 [x] — Test: `renderBrand()` returns text logo (not ASCII art)
- **Type**: test
- **Effort**: S
- **Depends on**: B2
- **Spec ref**: Req 21 (Brand Logo), tui-removed Req 6 (block-letter `logo.go` absent)
- **Design ref**: Section 4.1, Section 6 (`logo.go` Delete and replace)
- **Files**: `internal/dashboard/logo_test.go` (rewrite — see Reconciliation: logo.go is being deleted, so `renderBrand` must live in `helpers.go` or new `brand.go`; per tui-removed Req 6 the file `logo.go` must not exist)
- **Acceptance criteria**:
  - Test calls `renderBrand()` and asserts the rendered string contains the literal `🧠 Thoughtline` and the tagline `Local memory for game projects`
  - Verbatim scenario: Req 21 "Brand text renders without ASCII art" — assert no multi-line block-letter rendering of `THOUGHTLINE` is present
  - Test asserts the file `internal/dashboard/logo.go` does NOT exist (or that the symbol lives in `helpers.go` or `brand.go`)

### E2 [x] — Code: implement `renderBrand()` and delete `logo.go`
- **Type**: code
- **Effort**: S
- **Depends on**: E1
- **Spec ref**: Req 21, tui-removed Req 6
- **Design ref**: Section 6 (`logo.go` Delete and replace)
- **Files**: `internal/dashboard/helpers.go` (or new `internal/dashboard/brand.go`); DELETE `internal/dashboard/logo.go`
- **Acceptance criteria**:
  - Tests from E1 turn GREEN
  - `renderBrand()` returns `🧠 Thoughtline` styled `Brand` + `Local memory for game projects` styled `Muted`
  - `logo.go` file removed from disk
  - All callers updated to use `renderBrand()` (or the new location)

---

## Group F — Tab layer in flatModel (RED phase)

> All F-tasks are RED tests — they will turn GREEN when Group G lands.

### F1 [x] — Test: tab digit-jump 1..6
- **Type**: test
- **Effort**: S
- **Depends on**: A6
- **Spec ref**: Req 1 (Tab Navigation Contract — digit jump)
- **Design ref**: Section 2b(B), Section 7 (`flatModel` shape)
- **Files**: `internal/dashboard/flat_model_test.go` (NEW)
- **Acceptance criteria**:
  - Verbatim scenario: Req 1 "Direct jump by digit" — GIVEN active=Home, WHEN press `4`, THEN active=Inbox
  - Six sub-tests (one per digit) using `m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})` directly
  - Assertion: `m.activeTab == TabInbox` after; previously-active tab no longer the activeTab

### F2 [x] — Test: tab cycle forward/backward with wrap
- **Type**: test
- **Effort**: S
- **Depends on**: A6
- **Spec ref**: Req 1 (`tab` / `shift+tab` cycle with wrap)
- **Design ref**: Section 2b(B)
- **Files**: `internal/dashboard/flat_model_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 1 "Tab cycle forward and backward"
  - Forward wrap: from `TabHelp` press `tab` → `TabHome`
  - Backward wrap: from `TabHome` press `shift+tab` → `TabHelp`

### F3 [x] — Test: tab state survives Screen stack push/pop
- **Type**: test
- **Effort**: S
- **Depends on**: A6
- **Spec ref**: Req 1 (tab persists across push/pop), Req 17 (esc returns to originating tab)
- **Design ref**: Section 2b(C) OriginatingTab interface assertion
- **Files**: `internal/dashboard/flat_model_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 1 "Tab state survives Screen push/pop"
  - Push detail while active=Memories, press `esc`, assert active=Memories
  - Verify a screen implementing `OriginatingTab() tabKey` returning `TabHome` overrides the previous tab on pop (cross-tab return case)

### F4 [x] — Test: `InputFocused()` guard suppresses global Quick Actions + digits + tab
- **Type**: test
- **Effort**: M
- **Depends on**: A6
- **Spec ref**: Req 16 (Input-focus suppression), Req 23 (q literal in input)
- **Design ref**: Section 2b(B) step (3), Section 7 (`inputFocuser` interface), Reconciliation #4
- **Files**: `internal/dashboard/flat_model_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 16 "Typing m in search input is literal" — focus Search input, send `m`, assert no tab switch AND `m` appended to textinput
  - Verbatim scenario: Req 16 "Typing s in inbox edit content is literal" — focus textarea, send `s`, assert no save flow triggered
  - Per Reconciliation #4: focus textarea, send `2` (digit), assert no tab switch and `2` lands in input
  - Per Req 23: focus textarea, send `q`, assert no quit and `q` lands in input
  - Per Req 23: focus textarea, send `ctrl+c`, assert quit STILL happens (ctrl+c bypasses focus guard)

### F5 [x] — Test: Quick Action `[/]` jumps to Search AND focuses input
- **Type**: test
- **Effort**: S
- **Depends on**: A6
- **Spec ref**: Req 15 (Slash jumps and focuses search)
- **Design ref**: Section 2b(B) step (5), Section 5
- **Files**: `internal/dashboard/flat_model_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 15 "Slash jumps and focuses search"
  - After `/`: `m.activeTab == TabSearch` AND the SearchScreen's `InputFocused()` returns true

### F6 [x] — Test: `OriginatingTab()` return after esc pop
- **Type**: test
- **Effort**: S
- **Depends on**: A6
- **Spec ref**: Req 17 (Memory Detail full-screen view, esc returns to originating tab)
- **Design ref**: Section 2b(C)
- **Files**: `internal/dashboard/flat_model_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 17 "Esc returns to originating tab"
  - Detail pushed with `origin = TabMemories`, esc pop, assert `activeTab == TabMemories`
  - Detail pushed with `origin = TabHome`, esc pop, assert `activeTab == TabHome`

### F7 [x] — Test: Min viewport 80x24 warning + key freeze
- **Type**: test
- **Effort**: M
- **Depends on**: A6
- **Spec ref**: Req 22 (Minimum Viewport)
- **Design ref**: Section 2b(B) step (1), Reconciliation #8
- **Files**: `internal/dashboard/flat_model_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 22 "Under-min viewport renders warning" — resize to 70×20, assert View contains the warning string
  - Verbatim scenario: Req 22 "q still quits under min viewport" — at 70×20 send `q`, assert quit
  - Verbatim scenario: Req 22 "Other keys are ignored under min viewport" — at 70×20 send `1`, `/`, `m`, assert activeTab unchanged
  - Per Reconciliation #8: exactly 80×24 does NOT warn; 79×24 warns; 80×23 warns

### F8 [x] — Test: Quit semantics (q quits / q literal in input / ctrl+c always)
- **Type**: test
- **Effort**: S
- **Depends on**: A6
- **Spec ref**: Req 23 (Quit Semantics)
- **Design ref**: Section 2b(B) step (1), Reconciliation #3
- **Files**: `internal/dashboard/flat_model_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 23 "q on Memories tab quits" — active=Memories, no input focused, press `q`, assert tea.Quit cmd returned
  - Verbatim scenario: Req 23 "q in search input is literal" (already covered by F4 — cross-reference)
  - Verbatim scenario: Req 23 "ctrl+c quits even with input focused" (already covered by F4 — cross-reference)
  - Cross-reference is acceptable but the F8 file MUST contain a discrete `t.Run("q_quits_on_memories", ...)` for traceability

---

## Group G — Tab layer GREEN

### G1 — Code: `tabKey` enum, `tabDef`, `defaultTabs`, `flatModel.tabs`/`activeTab` fields
- **Type**: code
- **Effort**: M
- **Depends on**: F1, F2, F3 (RED tests exist)
- **Spec ref**: Req 1
- **Design ref**: Section 7 (interfaces), Section 6 (flat_model.go row)
- **Files**: `internal/dashboard/flat_model.go`
- **Acceptance criteria**:
  - `tabKey` constants `TabHome..TabHelp` defined
  - `defaultTabs` slice present with exact order: Home, Memories, Search, Inbox, Sessions, Help (digits 1..6)
  - `flatModel` gains `tabs [6]Screen`, `activeTab tabKey`, `status statusMessage`
  - No tests turn GREEN yet (interpreter check only) — go test compiles

### G2 — Code: implement Update ladder (window → quit → stack → focus-guard → tab → quick-actions → tab.Update)
- **Type**: code
- **Effort**: L
- **Depends on**: G1, F1, F2, F3, F4, F5, F7, F8
- **Spec ref**: Req 1, Req 15, Req 16, Req 22, Req 23
- **Design ref**: Section 5 data flow, Section 2b(B) ordering
- **Files**: `internal/dashboard/flat_model.go`
- **Acceptance criteria**:
  - Tests F1, F2, F4, F5, F7, F8 turn GREEN
  - The order in code matches the design exactly:
    1. `tea.WindowSizeMsg` propagates to all tabs and stack screens
    2. `ctrl+c` always quits; `q` quits ONLY when stack empty AND not InputFocused
    3. If stack non-empty, dispatch to stack top (esc handled inside dispatch)
    4. `InputFocused()` guard — if true, route to active tab Screen without intercepting
    5. Tab intercept: `1`-`6`, `tab`, `shift+tab`
    6. Quick Actions intercept: `s`, `/`, `m`, `i`
    7. Fall through to `tabs[activeTab].Update(msg)`

### G3 — Code: `setActiveTab(k)` calls `OnFocus()` on the newly-active tab
- **Type**: code
- **Effort**: S
- **Depends on**: G2
- **Spec ref**: Req 1 (visual marking of active tab), implicit Home refresh on focus (Reconciliation #9)
- **Design ref**: Section 5, Section 2b(F)
- **Files**: `internal/dashboard/flat_model.go`
- **Acceptance criteria**:
  - `setActiveTab` returns the `tea.Cmd` from `OnFocus()` so callers can chain it
  - Tests F1 visual-active-mark assertion turns GREEN (paired with G4 status/footer rendering if needed)

### G4 — Code: `statusMessage` with expiry
- **Type**: code
- **Effort**: S
- **Depends on**: G2
- **Spec ref**: Req 15 (S surfaces a response)
- **Design ref**: Section 2b(D), Section 7 (`statusMessage` struct)
- **Files**: `internal/dashboard/flat_model.go`, `internal/dashboard/commands.go`
- **Acceptance criteria**:
  - `statusMessage` struct with Text/Level/Expires
  - `setStatusMessage(text, ttl)` returns a `tea.Cmd` that emits a `statusClearMsg` after `ttl`
  - `[S]` Quick Action sets the message "Save: use 'tl save ...' (CLI) or tl_save (MCP). In-TUI save coming in a follow-up change." for 5 seconds
  - F5 / Req 15 "S surfaces a response without crashing" scenario verifies status text appears in View()

### G5 — Code: `OriginatingTab()` interface assertion handling on pop
- **Type**: code
- **Effort**: S
- **Depends on**: G2, F3, F6
- **Spec ref**: Req 1, Req 17
- **Design ref**: Section 2b(C), Section 7 (`originator` interface)
- **Files**: `internal/dashboard/flat_model.go`, `internal/dashboard/screen.go` (doc comment for the optional interface)
- **Acceptance criteria**:
  - Tests F3 and F6 turn GREEN
  - On `esc` pop, `flatModel` queries the popped screen via interface assertion; if it implements `OriginatingTab() tabKey`, the returned tabKey becomes `activeTab`
  - Otherwise `activeTab` is unchanged

### G6 — Code: tab dispatch fallback
- **Type**: code
- **Effort**: S
- **Depends on**: G2
- **Spec ref**: Req 7 (Memories keybinds), Req 8 (Search), Req 13 (Sessions), Req 14 (Help)
- **Design ref**: Section 5 step (6)
- **Files**: `internal/dashboard/flat_model.go`
- **Acceptance criteria**:
  - When no intercept fires, `flatModel.Update` calls `tabs[activeTab].Update(msg)` and merges the returned cmd
  - All F-group tests are GREEN at this point

---

## Group H — Cleanup batch (delete legacy mass)

> **Highest-risk commit. Groups F+G tests MUST be GREEN before H lands.**

### H1 [x] — Test: assert legacy files are absent and legacy symbols are not referenced
- **Type**: test
- **Effort**: M
- **Depends on**: G6
- **Spec ref**: tui-removed Req 4, 5, 6, 7, 8, 9, 10, 11
- **Design ref**: Section 9 commit 5
- **Files**: `internal/dashboard/cleanup_assertions_test.go` (NEW)
- **Acceptance criteria**:
  - One sub-test per file: assert `os.Stat` returns ErrNotExist for `cube.go`, `splash.go`, `logo.go`, `workstation_screen.go`, `projects_screen.go`, `tabs_test.go`, `themes.go`, `model.go`, `view.go`, `update.go`
  - One sub-test asserting source-level absence of `ThemeBrand`, `ThemeZBrush`, `ThemeMono`, `ApplyTheme`, `nextTheme` (greps `internal/dashboard/*.go`)
  - One sub-test asserting absence of `tabKey` (legacy enum on legacy Model — note: the NEW `tabKey` in `flat_model.go` is the SAME identifier; the test must distinguish by checking absence of `func (m Model) View()` and `func (m Model) Update(` as proxies for the legacy Model existence)
  - Verbatim: tui-removed Req 11 Scenario "No tabKey enum or legacy Model methods remain" — adjusted per the note above to grep for `func (m Model)` patterns specifically

### H2 [x] — Code: delete legacy files in a single commit
- **Type**: code
- **Effort**: L
- **Depends on**: H1, C2, C3, D2, E2, G6
- **Spec ref**: tui-removed Req 4-11
- **Design ref**: Section 9 commit 5
- **Files**: DELETE: `internal/dashboard/model.go`, `view.go`, `update.go`, `cube.go`, `splash.go`, `themes.go`, `tabs_test.go`, `cube_test.go` (if exists), `splash_test.go` (if exists). `logo.go` already deleted in E2.
- **Acceptance criteria**:
  - Tests from H1 turn GREEN
  - `go test ./...` green — no broken imports
  - `commands.go` simplified: drop splash/cube ticks

### H3 [x] — Code: slim `model_test.go` to roadmap/status tests only
- **Type**: code
- **Effort**: S
- **Depends on**: H2
- **Spec ref**: (Infrastructure)
- **Design ref**: Section 6 (`model_test.go` Modify row), Section 9 commit 5
- **Files**: `internal/dashboard/model_test.go` (rename to `roadmap_status_test.go` if cleaner per design)
- **Acceptance criteria**:
  - Only roadmap/status assertions remain (no test references to deleted legacy Model)
  - File compiles and tests pass

### H4 [x] — Code: `logo_test.go` is already rewritten for `renderBrand` in E1; verify here
- **Type**: code
- **Effort**: S
- **Depends on**: H2
- **Spec ref**: Req 21
- **Design ref**: Section 9 commit 5
- **Files**: `internal/dashboard/logo_test.go` (already migrated in E)
- **Acceptance criteria**:
  - No assertions reference deleted `gradientLogo` / `logoLines` symbols
  - All assertions are about `renderBrand`

---

## Group I — Home tab (HomeScreen)

### I1 [x] — Test: Home renders header, Quick Actions row, and three cards in populated state
- **Type**: test
- **Effort**: M
- **Depends on**: G6, A6
- **Spec ref**: Req 2 (Home tab layout), Req 21 (brand)
- **Design ref**: Section 4.1 mockup, Section 2b(G/H/I)
- **Files**: `internal/dashboard/home_screen_test.go` (NEW)
- **Acceptance criteria**:
  - Verbatim scenario: Req 2 "Populated Home render"
  - Seed `seedMemories(t, s, 5)`; render Home; assert View contains: brand text, tagline, project name, DB path styled muted, Quick Actions hint row, "Project Health" label, "Latest Memories" label, "Recent Activity" label

### I2 [x] — Test: Project Health card has 5 rows in fixed order
- **Type**: test
- **Effort**: S
- **Depends on**: I1
- **Spec ref**: Req 2 (Project Health card present)
- **Design ref**: Section 2b(G) — exact row labels and order
- **Files**: `internal/dashboard/home_screen_test.go`
- **Acceptance criteria**:
  - Render order: `Memories:`, `Sessions:`, `Pending:`, `Disk free:`, `Storage:` — assert ordering by substring index in View output
  - Pending row uses Warning color when count > 0; Disk free uses Warning when <1GB, Error when <100MB

### I3 [x] — Test: Latest Memories card interactive cursor with j/k and no wrap
- **Type**: test
- **Effort**: M
- **Depends on**: I1
- **Spec ref**: Req 3 (Latest Memories Card Interactivity)
- **Design ref**: Section 2b(H) — 5 rows fixed at 80×24
- **Files**: `internal/dashboard/home_screen_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 3 "Cursor moves within card" — `j` twice, cursor moves down 2 rows
  - No-wrap at top (cursor stays at 0 on `k`) and bottom (stays at last row on `j`)

### I4 [x] — Test: enter on Latest Memories pushes Detail; esc restores cursor position
- **Type**: test
- **Effort**: M
- **Depends on**: I3
- **Spec ref**: Req 3 ("Enter opens detail and esc restores cursor")
- **Design ref**: Section 2b(C) + Section 2b(3)
- **Files**: `internal/dashboard/home_screen_test.go` + uses `flat_model_test.go` push/pop integration
- **Acceptance criteria**:
  - Verbatim scenario: Req 3 "Enter opens detail and esc restores cursor" — cursor at row 2, press enter, assert DetailScreen on stack with row-2 memory; press esc, assert cursor still at row 2

### I5 [x] — Test: Recent Activity card renders mixed stream
- **Type**: test
- **Effort**: S
- **Depends on**: I1
- **Spec ref**: Req 2 (Recent Activity card)
- **Design ref**: Section 2b(I) — mixed stream of last 5 events
- **Files**: `internal/dashboard/home_screen_test.go`
- **Acceptance criteria**:
  - Seed memories, sessions, and pending; assert at least two distinct event-type labels appear (e.g., `saved` and `pending`)

### I6 [x] — Test: empty state renders 3 gamedev examples + tl save / tl_save instructions
- **Type**: test
- **Effort**: S
- **Depends on**: I1
- **Spec ref**: Req 4 (Empty State)
- **Design ref**: Section 4.8 mockup, Section 2b(F)
- **Files**: `internal/dashboard/home_screen_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 4 "Empty state example types rendered" — View contains literal `scene-pattern`, `perf-gotcha`, `pipeline-step`
  - Verbatim scenario: Req 4 "Empty state save instructions rendered" — View contains `tl save` AND `tl_save`
  - Verbatim scenario: Req 2 "Empty Home render" — header + Quick Actions render but Latest Memories/Recent Activity cards do NOT

### I7 [x] — Test: `r` and `OnFocus` refresh Home stats
- **Type**: test
- **Effort**: S
- **Depends on**: I1
- **Spec ref**: Req 14 (`r` refresh on Memories — analogous semantic), Req 2 (Home populated render driven by stats)
- **Design ref**: Section 2b(4) — manual refresh on `r` + OnFocus
- **Files**: `internal/dashboard/home_screen_test.go`
- **Acceptance criteria**:
  - Render Home, mutate DB (add a memory), press `r`, assert new total visible
  - Switch to another tab and back; assert `OnFocus` triggered the stats reload (memory count updated)

### I8 [x] — Code: implement `HomeScreen` (rewrite of `dashboard_screen.go`)
- **Type**: code
- **Effort**: L
- **Depends on**: I1, I2, I3, I4, I5, I6, I7, B2, B4, C2, E2, G6
- **Spec ref**: Req 2, 3, 4, 21
- **Design ref**: Section 4.1, Section 4.8, Section 6 (`dashboard_screen.go` Modify major)
- **Files**: `internal/dashboard/dashboard_screen.go` (rewrite — file may be renamed `home_screen.go` per design)
- **Acceptance criteria**:
  - All I-group tests turn GREEN
  - `InputFocused() bool { return false }`
  - Pushes `DetailScreen` with `origin = TabHome` on enter
  - Empty-state branch checked via `storage.Stats().TotalMemories == 0`

---

## Group J — Memories tab (MemoriesScreen)

### J1 [x] — Test: Memories tab shows count + rows with badge/topic_key/preview/scope/relative time
- **Type**: test
- **Effort**: M
- **Depends on**: G6, A6
- **Spec ref**: Req 5 (Memories Tab Layout)
- **Design ref**: Section 4.2 mockup, Section 7 (`MemoriesScreen` shape)
- **Files**: `internal/dashboard/memories_screen_test.go` (NEW)
- **Acceptance criteria**:
  - Verbatim scenario: Req 5 "Memories tab shows count and rows" — seed 7 memories, View contains "7" and at least one row with all 5 fields

### J2 [x] — Test: default sort is `updated_at` DESC
- **Type**: test
- **Effort**: S
- **Depends on**: J1
- **Spec ref**: Req 5 (default sort)
- **Design ref**: Section 7 `MemoriesFilter.Sort` default
- **Files**: `internal/dashboard/memories_screen_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 5 "Default sort order is updated_at DESC"

### J3 [x] — Test: pagination 20/page with `n`/`p` keys and footer "Page X of Y · NNN total"
- **Type**: test
- **Effort**: M
- **Depends on**: J1
- **Spec ref**: Req 5 (total count visible), Req 7 (not explicitly — pagination is from Decision #1)
- **Design ref**: Section 2b(1) (20/page, n/p), Section 4.2 footer, Section 8 testing strategy
- **Files**: `internal/dashboard/memories_screen_test.go`
- **Acceptance criteria**:
  - Seed 50; assert page 1 shows 20 rows and footer matches `Page 1 of 3 · 50 total`
  - `n` advances to page 2; `p` retreats to page 1; `n` at last page is no-op; `p` at page 1 is no-op

### J4 [x] — Test: inline filter bar with `f` cycling focus through 4 filter widgets
- **Type**: test
- **Effort**: M
- **Depends on**: J1
- **Spec ref**: Req 6 (Memories Tab Filters)
- **Design ref**: Section 2b(2) inline always-visible bar, `f` cycles focus
- **Files**: `internal/dashboard/memories_screen_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 6 "All four filter dimensions are reachable" — type/tag/scope/sort each settable
  - `f` cycles focus: row → type → tag → scope → sort → row
  - `c` clears all filters back to default

### J5 [x] — Test: filter actually filters the list
- **Type**: test
- **Effort**: S
- **Depends on**: J4
- **Spec ref**: Req 6 "Filter actually filters the list"
- **Design ref**: Section 5 `loadMemoriesCmd`
- **Files**: `internal/dashboard/memories_screen_test.go`
- **Acceptance criteria**:
  - Seed 5 `decision` + 3 `bug`; apply `type = decision`; assert only 5 rows render

### J6 [x] — Test: keybindings j/k cursor (no wrap), enter pushes Detail, `/` jumps to Search
- **Type**: test
- **Effort**: M
- **Depends on**: J1
- **Spec ref**: Req 7 (Memories Tab Keybindings — every key)
- **Design ref**: Section 7 + Section 4.2 footer
- **Files**: `internal/dashboard/memories_screen_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 7 "Enter pushes detail"
  - Verbatim scenario: Req 7 "Slash jumps to Search"
  - Verbatim scenario: Req 7 "No wrap at bottom"

### J7 [x] — Test: filter persistence across tab switch
- **Type**: test
- **Effort**: S
- **Depends on**: J4
- **Spec ref**: (Decision #5 — not a discrete spec Req but tied to Req 6 + tab persistence)
- **Design ref**: Section 2b(5)
- **Files**: `internal/dashboard/memories_screen_test.go`
- **Acceptance criteria**:
  - Apply `type=decision`, switch to Help, switch back; filter still `type=decision`

### J8 [x] — Test: cursor preservation across detail push/pop
- **Type**: test
- **Effort**: S
- **Depends on**: J1
- **Spec ref**: Req 7 (enter pushes detail), Req 17 (esc returns to originating tab) + Decision #3 cursor preservation
- **Design ref**: Section 2b(3)
- **Files**: `internal/dashboard/memories_screen_test.go`
- **Acceptance criteria**:
  - Move cursor to row 5, push Detail, press esc; cursor still on row 5

### J9 [x] — Code: implement `MemoriesScreen` (rename + rewrite of `recent_screen.go`)
- **Type**: code
- **Effort**: L
- **Depends on**: J1-J8, A6, B2, C2, G6
- **Spec ref**: Req 5, 6, 7
- **Design ref**: Section 6 (`recent_screen.go` Modify major; rename target `memories_screen.go`), Section 7 (MemoriesScreen shape), Reconciliation #6
- **Files**: rename `internal/dashboard/recent_screen.go` → `internal/dashboard/memories_screen.go`
- **Acceptance criteria**:
  - All J-group tests turn GREEN
  - Struct matches Section 7 shape: `storage`, `page`, `perPage=20`, `total`, `rows`, `cursor`, `filter`, `filterFocus`, `tagInput`
  - `InputFocused()` returns true only when the tag filter textinput is focused
  - Pushes `DetailScreen` with `origin = TabMemories` on enter

---

## Group K — Detail view + clipboard

### K1 [x] — Test: DetailScreen renders all 8 metadata fields and supports `[C]` copy
- **Type**: test
- **Effort**: M
- **Depends on**: G6, A6
- **Spec ref**: Req 17 (Detail renders required metadata), Req 18 (read-only — no [E]/[D]), Req 19 (clipboard)
- **Design ref**: Section 4.3 mockup, Section 7 (DetailScreen additions)
- **Files**: `internal/dashboard/detail_screen_test.go` (NEW or rewrite)
- **Acceptance criteria**:
  - Verbatim scenario: Req 17 "Detail renders required metadata" — type, scope, project, topic_key, sync_id, tags, created_at, updated_at all visible
  - Verbatim scenario: Req 18 "No edit hotkey on Memory Detail" — press `e`, assert no edit flow
  - Verbatim scenario: Req 19 "Successful copy on supported platform" — using shimmed clipboard, press `c`, assert clipboard cmd dispatched and success status text rendered
  - Verbatim scenario: Req 19 "Failure surfaces, does not crash" — shim returns error, press `c`, assert status text "Clipboard unavailable..." in yellow

### K2 [x] — Test: `[C]` clipboard backend invocation matrix (success and missing-backend per OS)
- **Type**: test
- **Effort**: M
- **Depends on**: K1
- **Spec ref**: Req 19
- **Design ref**: Section 2b(K), Section 6 (clipboard files), Section 8 testing strategy
- **Files**: `internal/dashboard/clipboard_test.go` (NEW)
- **Acceptance criteria**:
  - Per-OS test using `//go:build windows|darwin|linux` plus a package-level `execCommand` var that shims `exec.Command`
  - Windows: invokes `clip.exe`
  - macOS: invokes `pbcopy`
  - Linux: tries `wl-copy` first, falls back to `xclip -selection clipboard`
  - Missing-backend test: `execLookPath` shim returns ErrNotFound, assert `CopyToClipboard` returns the sentinel error and does not panic

### K3 [x] — Code: `clipboard.go` common interface + per-OS implementations
- **Type**: code
- **Effort**: M
- **Depends on**: K2
- **Spec ref**: Req 19
- **Design ref**: Section 6, Section 7 (interface signature)
- **Files**: `internal/dashboard/clipboard.go` (interface), `clipboard_windows.go`, `clipboard_darwin.go`, `clipboard_linux.go`
- **Acceptance criteria**:
  - K2 tests turn GREEN on the running OS (CI matrix exercises others)
  - `func CopyToClipboard(s string) error` defined exactly as in Section 7
  - Each OS file declares the matching build tag

### K4 [x] — Code: modify `detail_screen.go` for new behavior
- **Type**: code
- **Effort**: M
- **Depends on**: K1, K3, B2, C2
- **Spec ref**: Req 17, 18, 19
- **Design ref**: Section 6 (`detail_screen.go` Modify), Section 7 (DetailScreen additions), Section 4.3 footer
- **Files**: `internal/dashboard/detail_screen.go`
- **Acceptance criteria**:
  - K1 tests turn GREEN
  - Adds `origin tabKey` field and `OriginatingTab() tabKey` accessor (Section 7)
  - Removes any pre-existing `[E]`/`[D]` key bindings — verified by source assertion in H1
  - Adds `[C]` handler → `copyClipboardCmd(content)` → on `clipboardMsg` sets status line ("Copied to clipboard (N chars)" green OR "Clipboard unavailable: ..." yellow)
  - Footer renders `topic_key` and `sync_id` muted; clipboard status appears in DetailScreen footer (not global), per Section 2b(K)

### K5 [x] — Code: `copyClipboardCmd` in `commands.go`
- **Type**: code
- **Effort**: S
- **Depends on**: K3, K4
- **Spec ref**: Req 19
- **Design ref**: Section 5 (commands list)
- **Files**: `internal/dashboard/commands.go`
- **Acceptance criteria**:
  - Returns `tea.Cmd` that calls `CopyToClipboard(s)` and emits `clipboardMsg{text, level}` based on error/success
  - K1 / K4 clipboard scenarios pass

---

## Group L — Inbox tab

### L1 [x] — Test: InboxScreen renders rows with `[A]`/`[E]`/`[R]` affordances
- **Type**: test
- **Effort**: M
- **Depends on**: G6, A6, A5
- **Spec ref**: Req 9 (Inbox layout)
- **Design ref**: Section 4.4 mockup, Section 6 (pending_screen Modify)
- **Files**: `internal/dashboard/inbox_screen_test.go` (NEW)
- **Acceptance criteria**:
  - Verbatim scenario: Req 9 "Inbox renders pending rows with affordances" — seed 2 pending, assert 2 rows + `[A]` `[E]` `[R]` text affordances visible in footer/help line
  - Verbatim scenario: Req 9 "Inbox empty state" — zero pending, no crash, empty surface renders

### L2 [x] — Test: `[A]` accept calls `MarkPromoted` unchanged, row disappears, success status
- **Type**: test
- **Effort**: M
- **Depends on**: L1
- **Spec ref**: Req 10 (Inbox Accept Action), passive-capture Req 9 (passthrough scenario)
- **Design ref**: Section 5 `promoteCmd`, Section 6
- **Files**: `internal/dashboard/inbox_screen_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 10 "Accept promotes unchanged"
  - Verbatim scenario: Req 10 "Accept surfaces success status"
  - Verbatim scenario: passive-capture Req 9 "Passthrough accept preserves proposed values" — assert pending payload byte-for-byte unchanged after promote

### L3 [x] — Test: `[R]` reject calls `MarkRejected`, row disappears, no memory created, other rows untouched
- **Type**: test
- **Effort**: M
- **Depends on**: L1, A5
- **Spec ref**: Req 12 (Inbox Reject)
- **Design ref**: Reconciliation #1, Section 6
- **Files**: `internal/dashboard/inbox_screen_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 12 "Reject removes row without promoting"
  - Verbatim scenario: Req 12 "Reject does not affect other rows"

### L4 [x] — Test: InboxEditScreen — 3 fields (type select / title textinput / content textarea)
- **Type**: test
- **Effort**: M
- **Depends on**: L1
- **Spec ref**: Req 11 (Inbox Edit Action — three editable fields)
- **Design ref**: Section 4.5 mockup, Section 7 (InboxEditScreen shape)
- **Files**: `internal/dashboard/inbox_edit_screen_test.go` (NEW)
- **Acceptance criteria**:
  - Press `E` on a pending row; assert push of InboxEditScreen; assert three fields present
  - `tab` cycles focus through fields (0→1→2→0)
  - `InputFocused()` returns true at all times while edit screen is active

### L5 [x] — Test: edit submit (`ctrl+s`) promotes with new values, payload unchanged
- **Type**: test
- **Effort**: M
- **Depends on**: L4
- **Spec ref**: Req 11 (Edit submit promotes with new values), passive-capture Req 9 (Edited accept)
- **Design ref**: Section 2b(9) — ctrl+s submits, no confirmation step
- **Files**: `internal/dashboard/inbox_edit_screen_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 11 "Edit submit promotes with new values"
  - Verbatim scenario: passive-capture Req 9 "Edited accept overrides type, title, and content" — assert payload byte-for-byte unchanged after promote with edits

### L6 [x] — Test: edit cancel (`esc`) leaves Inbox unchanged
- **Type**: test
- **Effort**: S
- **Depends on**: L4
- **Spec ref**: Req 11 (Edit cancel leaves Inbox unchanged)
- **Design ref**: Section 2b(9), Section 5
- **Files**: `internal/dashboard/inbox_edit_screen_test.go`
- **Acceptance criteria**:
  - Verbatim scenario: Req 11 "Edit cancel leaves Inbox unchanged"

### L7 [x] — Code: implement `InboxScreen` (rewrite of `pending_screen.go`)
- **Type**: code
- **Effort**: L
- **Depends on**: L1, L2, L3, A5, G6
- **Spec ref**: Req 9, 10, 12
- **Design ref**: Section 6 (pending_screen Modify major)
- **Files**: rewrite `internal/dashboard/pending_screen.go` (and rename to `inbox_screen.go` for clarity, optional)
- **Acceptance criteria**:
  - L1, L2, L3 tests turn GREEN
  - `[A]` → `MarkPromoted(id, memID)` via promoteCmd with proposed payload values
  - `[E]` → pushes `InboxEditScreen` with the pending row
  - `[R]` → `MarkRejected(id)` via storage call; row disappears

### L8 [x] — Code: implement `InboxEditScreen`
- **Type**: code
- **Effort**: L
- **Depends on**: L4, L5, L6, L7
- **Spec ref**: Req 11
- **Design ref**: Section 4.5 mockup, Section 7 (InboxEditScreen struct)
- **Files**: `internal/dashboard/inbox_edit_screen.go` (NEW)
- **Acceptance criteria**:
  - L4, L5, L6 tests turn GREEN
  - Fields per Section 7: typeField (textinput constrained to 11-type taxonomy via validator), titleField (textinput), bodyField (textarea), focus (0/1/2)
  - `InputFocused() bool { return true }`
  - `ctrl+s` triggers promoteCmd with edited values; on success pops back to Inbox with green status "Promoted: <title>"
  - `esc` pops without persisting

### L9 [x] — Code: `promoteCmd` in `commands.go`
- **Type**: code
- **Effort**: S
- **Depends on**: L7, L8
- **Spec ref**: Req 10, 11
- **Design ref**: Section 5 (`promoteCmd(id,t,title,body)`)
- **Files**: `internal/dashboard/commands.go`
- **Acceptance criteria**:
  - Saves a new memory with given type/title/body and calls `MarkPromoted(pendingID, newMemoryID)`
  - Returns `promotedMsg` on success; error msg on failure

---

## Group M — Sessions tab

### M1 [x] — Test: Sessions renders from RecentSessions, read-only
- **Type**: test
- **Effort**: S
- **Depends on**: G6, A6
- **Spec ref**: Req 13 (Sessions Tab)
- **Design ref**: Section 4.6 mockup
- **Files**: `internal/dashboard/sessions_screen_test.go` (NEW)
- **Acceptance criteria**:
  - Verbatim scenario: Req 13 "Sessions render" — at least one row with identifying metadata
  - Verbatim scenario: Req 13 "Sessions tab is read-only" — pressing `d`, `e` etc. does not mutate any row

### M2 [x] — Code: implement `SessionsScreen`
- **Type**: code
- **Effort**: M
- **Depends on**: M1, B2, C2
- **Spec ref**: Req 13
- **Design ref**: Section 6 (sessions_screen.go Create), Section 4.6
- **Files**: `internal/dashboard/sessions_screen.go` (NEW)
- **Acceptance criteria**:
  - M1 tests turn GREEN
  - Uses `storage.RecentSessions`; no mutation methods called from this screen

---

## Group N — Help tab + keybindings registry

### N1 — Test: `Keybindings` slice structure exists and is non-empty
- **Type**: test
- **Effort**: S
- **Depends on**: G6
- **Spec ref**: Req 14 (Help Tab keybindings reference)
- **Design ref**: Section 2b(J) — single source of truth, Section 7 (`KeybindGroup`/`Keybind` types)
- **Files**: `internal/dashboard/keybindings_test.go` (NEW)
- **Acceptance criteria**:
  - `Keybindings` is `[]KeybindGroup`
  - Contains groups: Navigation, Quick Actions, Memories tab, Detail view, Inbox (at minimum)
  - Every group has at least one binding

### N2 — Code: implement `keybindings.go` with the full set
- **Type**: code
- **Effort**: S
- **Depends on**: N1
- **Spec ref**: Req 14
- **Design ref**: Section 4.7 mockup (full set), Section 7
- **Files**: `internal/dashboard/keybindings.go` (NEW)
- **Acceptance criteria**:
  - Includes 1-6, tab/shift+tab, esc, q/ctrl+c (Navigation), s, /, m, i (Quick Actions), ↑↓/n/p/f/c/enter (Memories), ↑↓/C (Detail), A/E/R (Inbox)

### N3 — Test: HelpScreen renders Keybindings + roadmap
- **Type**: test
- **Effort**: S
- **Depends on**: N2
- **Spec ref**: Req 14 (Help Tab — keybindings + roadmap)
- **Design ref**: Section 4.7 mockup
- **Files**: `internal/dashboard/help_screen_test.go` (NEW)
- **Acceptance criteria**:
  - Verbatim scenario: Req 14 "Help shows keybindings" — content references digit-key tab switch, Quick Actions, `q`/`ctrl+c`
  - Verbatim scenario: Req 14 "Help renders roadmap from YAML" — at least one roadmap entry's title appears

### N4 — Code: implement `HelpScreen`
- **Type**: code
- **Effort**: M
- **Depends on**: N3, B2, C2
- **Spec ref**: Req 14
- **Design ref**: Section 4.7, Section 6 (help_screen.go Create)
- **Files**: `internal/dashboard/help_screen.go` (NEW)
- **Acceptance criteria**:
  - N3 tests turn GREEN
  - Two-column layout: keybindings left, roadmap right

### N5 — Test: keybinding drift — every handler wired in `flatModel.Update` has an entry in `Keybindings`
- **Type**: test
- **Effort**: M
- **Depends on**: N2, G6, J9, K4, L7, L8
- **Spec ref**: Req 14 (implicit — Help is single source of truth per Section 2b(J))
- **Design ref**: Section 2b(J), Section 8 testing strategy "keybindings drift test"
- **Files**: `internal/dashboard/keybindings_test.go`
- **Acceptance criteria**:
  - Test builds a set of "actual handled keys" by scanning source for tea.KeyMsg case literals OR by registering them via a registry pattern
  - Diff against `Keybindings` slice; assert no key is in source but missing from the slice
  - Allowed exceptions documented in test (e.g., per-Bubble component default keys like textinput cursor moves)

### N6 — Code: ensure the drift test passes (may require refactoring to a registry pattern)
- **Type**: code
- **Effort**: M
- **Depends on**: N5
- **Spec ref**: Req 14
- **Design ref**: Section 2b(J)
- **Files**: `internal/dashboard/keybindings.go`, possibly `flat_model.go` (lookup map)
- **Acceptance criteria**:
  - N5 turns GREEN
  - Future additions to handlers MUST be added to `Keybindings` slice — drift test will catch

---

## Group O — CLI flag migration

### O1 — Test: migration error on `--theme`/`--no-splash`/`--splash-ms` with exit 2 and exact stderr
- **Type**: test
- **Effort**: M
- **Depends on**: none
- **Spec ref**: tui-removed Req 1, 2, 3
- **Design ref**: Section 2b(8), Section 6 (`main.go` Modify), Reconciliation #7
- **Files**: `cmd/thoughtline/main_test.go` (NEW or extend)
- **Acceptance criteria**:
  - Calls the testable pre-scan function with arg slices: `["--theme=brand"]`, `["--theme", "brand"]`, `["--no-splash"]`, `["--splash-ms=500"]`, `["--splash-ms", "500"]`
  - Each returns the documented error string (containing the flag name and pointing to CHANGELOG) and an exit code 2 indicator
  - False-positive guard: `["--no-splashy"]` (per Reconciliation #7) does NOT match `--no-splash`
  - Verbatim scenarios: tui-removed Req 1/2/3 "Source-level absence" — separate assertion that `flag.String("theme", ...)` / `"no-splash"` / `"splash-ms"` literals are NOT present in `main.go`

### O2 — Code: pre-`flag.Parse` migration scan + remove Config fields
- **Type**: code
- **Effort**: M
- **Depends on**: O1
- **Spec ref**: tui-removed Req 1, 2, 3
- **Design ref**: Section 2b(8), Section 6
- **Files**: `cmd/thoughtline/main.go`, `internal/dashboard/run.go` (drop `ThemeName`/`Splash`/`SplashDuration` from Config)
- **Acceptance criteria**:
  - O1 tests turn GREEN
  - Pre-scan function: iterates `os.Args[1:]`, exact-token match with optional `=value` suffix
  - Error text format: `thoughtline ui: --theme was removed in v0.2 (single semantic palette). See CHANGELOG.md.`
  - Flag registrations for the three removed flags are deleted
  - `Config` struct in `internal/dashboard/run.go` no longer has the three fields

---

## Group P — Golden files

### P1 — Test/Infra: `home_default.golden` at 100×30
- **Type**: test
- **Effort**: S
- **Depends on**: I8, B2, C2, E2
- **Spec ref**: Req 2 (Home layout)
- **Design ref**: Section 4.1, Section 2b(M), Section 8
- **Files**: `internal/dashboard/testdata/home_default.golden` (NEW)
- **Acceptance criteria**:
  - Render Home at 100×30 with seeded data, compare to golden; `-update` regenerates

### P2 — Test/Infra: `home_empty.golden` at 100×30
- **Type**: test
- **Effort**: S
- **Depends on**: I8
- **Spec ref**: Req 4
- **Design ref**: Section 4.8
- **Files**: `internal/dashboard/testdata/home_empty.golden`
- **Acceptance criteria**: empty state golden locked

### P3 — Test/Infra: `memories_page1.golden` at 100×30
- **Type**: test
- **Effort**: S
- **Depends on**: J9
- **Spec ref**: Req 5, 7
- **Design ref**: Section 4.2
- **Files**: `internal/dashboard/testdata/memories_page1.golden`

### P4 — Test/Infra: `memories_filtered.golden` at 100×30
- **Type**: test
- **Effort**: S
- **Depends on**: J9
- **Spec ref**: Req 6
- **Design ref**: Section 4.2 (filter bar visible variant)
- **Files**: `internal/dashboard/testdata/memories_filtered.golden`

### P5 — Test/Infra: `inbox_three.golden` at 100×30
- **Type**: test
- **Effort**: S
- **Depends on**: L7
- **Spec ref**: Req 9
- **Design ref**: Section 4.4
- **Files**: `internal/dashboard/testdata/inbox_three.golden`

### P6 — Test/Infra: `inbox_edit_form.golden` at 100×30
- **Type**: test
- **Effort**: S
- **Depends on**: L8
- **Spec ref**: Req 11
- **Design ref**: Section 4.5
- **Files**: `internal/dashboard/testdata/inbox_edit_form.golden`

### P7 — Test/Infra: `detail.golden` at 100×30
- **Type**: test
- **Effort**: S
- **Depends on**: K4
- **Spec ref**: Req 17, 19
- **Design ref**: Section 4.3
- **Files**: `internal/dashboard/testdata/detail.golden`

### P8 — Test/Infra: `help.golden` at 100×30
- **Type**: test
- **Effort**: S
- **Depends on**: N4
- **Spec ref**: Req 14
- **Design ref**: Section 4.7
- **Files**: `internal/dashboard/testdata/help.golden`

### P9 — Test/Infra: `sessions.golden` at 100×30
- **Type**: test
- **Effort**: S
- **Depends on**: M2
- **Spec ref**: Req 13
- **Design ref**: Section 4.6
- **Files**: `internal/dashboard/testdata/sessions.golden`

### P10 — Test/Infra: `brand.golden` (text-only logo block)
- **Type**: test
- **Effort**: S
- **Depends on**: E2
- **Spec ref**: Req 21
- **Design ref**: Section 4.1 brand region, Section 6 (logo_test row)
- **Files**: `internal/dashboard/testdata/brand.golden`

---

## Group Q — Cleanup tail

### Q1 — Code: delete `projects_screen.go` and `items.go`
- **Type**: code
- **Effort**: S
- **Depends on**: J9 (Memories tab absorbs project filter), H2 (legacy delete already done)
- **Spec ref**: tui-removed Req 8 (`projects_screen.go` absent)
- **Design ref**: Section 9 commit 15
- **Files**: DELETE `internal/dashboard/projects_screen.go`, `internal/dashboard/items.go`
- **Acceptance criteria**:
  - Files removed from disk
  - `go test ./...` green (nothing references them)
  - tui-removed Req 8 scenario "File absent on disk" satisfied

### Q2 [x] — Code: delete `cube_test.go` and `splash_test.go` (if not already in H2)
- **Type**: code
- **Effort**: S
- **Depends on**: H2
- **Spec ref**: tui-removed Req 4, 5
- **Design ref**: Section 9 commit 5
- **Files**: DELETE `internal/dashboard/cube_test.go` (if exists), `internal/dashboard/splash_test.go` (if exists)
- **Acceptance criteria**:
  - Files removed if present; no-op if already deleted in H2

---

## Group R — Documentation

### R1 — Docs: CHANGELOG entry with BREAKING flag removal + new TUI summary
- **Type**: docs
- **Effort**: S
- **Depends on**: O2
- **Spec ref**: tui-removed Req 1, 2, 3 (flag removals reference CHANGELOG)
- **Design ref**: Section 9 commit 13
- **Files**: `CHANGELOG.md`
- **Acceptance criteria**:
  - Entry under upcoming release: BREAKING section names `--theme`, `--no-splash`, `--splash-ms`
  - Summary of new tab-based TUI workspace

### R2 — Docs: README updates (TUI section, screenshots placeholder, flags table)
- **Type**: docs
- **Effort**: S
- **Depends on**: P1-P10
- **Spec ref**: (Infrastructure)
- **Design ref**: Section 9 commit 13 spillover
- **Files**: `README.md`
- **Acceptance criteria**:
  - TUI section updated to describe 6 tabs and global hotkeys
  - Screenshots placeholder referencing golden files locations
  - Flags table updated (no `--theme`, no `--no-splash`, no `--splash-ms`)

### R3 — Docs: ADR `0006-tui-memory-workspace.md`
- **Type**: docs
- **Effort**: M
- **Depends on**: all design decisions
- **Spec ref**: (Architecture record)
- **Design ref**: All of design.md as source
- **Files**: `docs/adr/0006-tui-memory-workspace.md` (NEW; or whatever ADR location the repo uses)
- **Acceptance criteria**:
  - Rationale, design choices, palette decision (Tokyo Night Storm provenance), predecessor reference (`tui-redesign-superseded`)

---

## Group S — Verification gate

### S1 — Infra: `go test ./...` green
- **Type**: infra
- **Effort**: S
- **Depends on**: all
- **Spec ref**: (Infrastructure — green-bar gate)
- **Design ref**: Section 9 ("Each commit ends with go test ./... green")
- **Files**: n/a
- **Acceptance criteria**:
  - Full test suite passes locally

### S2 — Infra: `go vet ./...` clean
- **Type**: infra
- **Effort**: S
- **Depends on**: all
- **Spec ref**: (Infrastructure)
- **Design ref**: implied by Section 9
- **Files**: n/a
- **Acceptance criteria**:
  - `go vet ./...` reports no issues

### S3 — Infra: cross-platform clipboard smoke (manual on Win + at least one Unix host)
- **Type**: infra
- **Effort**: M
- **Depends on**: K3
- **Spec ref**: Req 19
- **Design ref**: Section 8 ("Cross-platform — clipboard")
- **Files**: n/a (manual procedure documented in CHANGELOG or test plan)
- **Acceptance criteria**:
  - On Windows: `clip.exe` copy works
  - On macOS or Linux (whichever available): `pbcopy` / `wl-copy` / `xclip` works
  - Missing-backend path surfaces "Clipboard unavailable" status

### S4 — Infra: spec scenario-to-test coverage audit
- **Type**: infra
- **Effort**: M
- **Depends on**: all test tasks
- **Spec ref**: All 23 + 1 + 12 = 36 Requirements
- **Design ref**: Section 8 (testing strategy completeness)
- **Files**: `openspec/changes/tui-memory-workspace/verify-coverage.md` (created during sdd-verify; this task seeds the expectation)
- **Acceptance criteria**:
  - Every Requirement maps to ≥1 test task in this tasks.md
  - Reconcile any uncovered scenarios before declaring the change ready for archive

---

## Dependency and Parallelism Summary

### Tasks that can start in parallel (no dependencies)
- A1, A2, A3, A4 (storage tests — independent of each other)
- C1 (helpers tests — no deps)
- O1 (CLI migration tests — no deps)
- B1 (palette tokens test — no deps)

### Critical path (longest sequential chain)
```
A6 → F1..F8 (parallel within group) → G1 → G2 → G6 → H1 → H2 → I1 → I8 → J9 → L7 → L8 → P1..P10 → S1..S4
```
Roughly: prerequisites (A) → tab-layer RED (F) → tab-layer GREEN (G) → cleanup (H) → Home (I) → Memories (J) → Inbox (L) → goldens (P) → gate (S).

### Primary build-up chain
- **Storage**: A1-A6 (parallel between A1-A4; A5 after A4; A6 after A1-A2)
- **Palette**: B1 → B2 → B3 → B4
- **Helpers**: C1 → C2 → C3
- **Decoupling**: D1 → D2 → D3 (depends on B2)
- **Brand**: E1 → E2 (depends on B2)
- **Tab layer**: (F1-F8 in parallel) → G1 → G2 → (G3, G4, G5, G6 parallel) → cleanup
- **Cleanup**: H1 → H2 → H3 → H4 (depends on G6 + C2/C3 + D2/D3 + E2)
- **Tabs**: I (Home), J (Memories), K (Detail+clipboard), L (Inbox), M (Sessions), N (Help) — mostly parallel after H4
- **CLI**: O1 → O2 (independent of UI)
- **Goldens**: P1-P10 (parallel after their respective screen groups land)
- **Docs**: R1-R3 (parallel near the end)
- **Gate**: S1-S4 (final)

### Parallelization opportunities
- All F-group tests in parallel (F1-F8)
- All J-group tests in parallel before J9
- All P-group golden files in parallel
- Groups I, J, K, L, M, N can be done in parallel once H4 lands

### Bottlenecks / Risks
- **H2 (legacy delete)** is the single highest-risk commit; it must come after all F+G tests are GREEN and before any new tab production code (I, J, L) lands.
- **N5 keybindings drift test** requires every screen's handler set to be finalized first — natural late-stage task.
- **K3 clipboard build-tag files** — only the host OS's tests run locally; cross-OS coverage relies on CI.

---

## Task Count by Group

| Group | Tests | Code/Docs/Infra | Total |
|-------|-------|-----------------|-------|
| A — Storage + fixtures | 4 | 2 | 6 |
| B — Theme + palette | 2 | 2 | 4 |
| C — Helpers extraction | 2 | 1 | 3 |
| D — Style decoupling | 2 | 1 | 3 |
| E — Brand + logo | 1 | 1 | 2 |
| F — Tab layer RED | 8 | 0 | 8 |
| G — Tab layer GREEN | 0 | 6 | 6 |
| H — Cleanup batch | 1 | 3 | 4 |
| I — Home tab | 7 | 1 | 8 |
| J — Memories tab | 8 | 1 | 9 |
| K — Detail + clipboard | 2 | 3 | 5 |
| L — Inbox tab | 6 | 3 | 9 |
| M — Sessions tab | 1 | 1 | 2 |
| N — Help + keybindings | 2 | 3 | 5 |
| O — CLI flag migration | 1 | 1 | 2 |
| P — Golden files | 10 | 0 | 10 |
| Q — Cleanup tail | 0 | 2 | 2 |
| R — Documentation | 0 | 3 | 3 |
| S — Verification gate | 0 | 4 | 4 |
| **Total** | **57** | **38** | **95** |

Note: Strict-TDD ratio = 57 test tasks : 38 production tasks. Every code task has at least one paired test task in the same group.

---

## Effort Summary

| Effort | Count | Notes |
|--------|-------|-------|
| S (<2h) | 51 | Mostly RED tests, single-file edits, infra/docs |
| M (2-4h) | 36 | Most new code tasks, integration tests, multi-file edits |
| L (4-8h) | 8 | Largest screens (HomeScreen, MemoriesScreen, InboxScreen, InboxEditScreen), G2 Update ladder, H2 legacy delete |
| Total estimated effort | ~180-220h | One developer, sequential; less in practice via parallelism |

---

## Commit Series (mapping to design's Section 9)

| # | Commit | Tasks included |
|---|--------|----------------|
| 1 | `test(dashboard): write RED tests for new flatModel tab intercept` | F1, F2, F3, F4, F5, F6, F7, F8, A6 |
| 2 | `refactor(dashboard): extract helpers from workstation_screen` | C1, C2, C3 |
| 3 | `chore(dashboard): decouple styles.go from themes.go` | D1, D2, D3 |
| 4 | `feat(dashboard): semantic palette in theme.go` | B1, B2, B3, B4 |
| 5 | `chore(dashboard): delete legacy Model + view/update + cube/splash/logo + themes + tabs_test` | E1, E2, H1, H2, H3, H4, Q2 |
| 6 | `feat(dashboard): tab layer in flatModel` | G1, G2, G3, G4, G5, G6 — turns F-tests GREEN |
| 7 | `feat(dashboard): Home tab (HomeScreen) with empty state` | I1, I2, I3, I4, I5, I6, I7, I8 |
| 8 | `feat(dashboard): Memories tab with pagination + filters` | J1-J9 |
| 9 | `feat(dashboard): Detail read-only with [C] copy + clipboard backends` | K1, K2, K3, K4, K5 |
| 10 | `feat(dashboard): Inbox tab with [A]/[E]/[R] + edit screen` | A1, A2, A3, A4, A5 (storage prerequisites land here if not earlier), L1-L9 |
| 11 | `feat(dashboard): Sessions tab restyle` | M1, M2 |
| 12 | `feat(dashboard): Help tab + keybindings registry` | N1, N2, N3, N4, N5, N6 |
| 13 | `chore(cli): remove --theme/--no-splash/--splash-ms with migration error` | O1, O2, R1 |
| 14 | `test(dashboard): golden files for all tabs at 100x30` | P1-P10 |
| 15 | `chore(dashboard): delete projects_screen and items.go` | Q1, R2, R3, S1, S2, S3, S4 |

Notes on commit boundaries:
- Commits 1-4 are pre-tab safety work; they end green without touching the legacy Model.
- Commit 5 is the highest-risk commit (legacy mass deletion) and lands ONLY after commits 1-4 are green and the F-group RED tests exist.
- Commit 6 turns the F-group from RED to GREEN.
- Commits 7-12 add tab screens. They CAN be reordered or parallelized (different files), as long as each individual commit is green.
- Commit 10 carries Group A (storage prerequisites) only if not landed earlier. Recommended: land A1-A6 alongside commit 1 (prerequisite block) so later UI commits depend on stable storage.
- Strict-TDD per commit: RED tests + GREEN production land together OR pure deletion + test removal land together. Never RED-only or GREEN-only.

---

## Addendum — Bug-fix tasks (observed in v0.1.0 binary, captured 2026-05-14)

Two real bugs surfaced when the user rebuilt the intermediate state of the dashboard and inspected the live UI. The bugs are now codified as Spec Req 24 (Project Health scoping) and Spec Req 25 (Memory Detail wrap+scroll). The following tasks land them inside the existing commit series — no new commits required.

### I-bug-1 [x] — Test: Project Health card counts match Top Projects scope
- **Type**: test
- **Effort**: S
- **Depends on**: A1, A2, A3, A6
- **Spec ref**: Req 24 (Project Health Card — Consistent Scoping)
- **Design ref**: Section 2b(G), Section 4.1
- **Files**: `internal/dashboard/home_screen_test.go` (extend)
- **Acceptance criteria**:
  - Test seeds 27 memories across 4 projects (`thoughtline`, `SuckItUp`, `hole-game`, `<other>`) and ZERO memories for project literal `"Símbolo del sistema"`
  - Test constructs `HomeScreen` with `cfg.Project = "Símbolo del sistema"` (or any non-matching cwd basename) — simulating the v0.1.0 bug condition
  - `HomeScreen.View()` output contains `Memories: 27` (NOT `Memories: 0`)
  - Top Projects rendered on the same frame contains at least 3 project bullet items
  - Verbatim spec scenario: "Stat card matches Top Projects when DB has 27 memories across 4 projects"

### I-bug-2 [x] — Code: HomeScreen uses unscoped stats for Project Health
- **Type**: code
- **Effort**: S
- **Depends on**: I-bug-1
- **Spec ref**: Req 24
- **Design ref**: Section 2b(G) updated — Project Health queries `storage.Stats(ctx, StatsOptions{})` with NO project filter (or with `project = "*"` semantics) regardless of `cfg.Project`
- **Files**: `internal/dashboard/dashboard_screen.go` → `home_screen.go` (in commit 7's rewrite)
- **Acceptance criteria**:
  - `HomeScreen.OnFocus()` issues a `loadHomeCmd` that calls `storage.Stats` with an UNSCOPED options struct (or equivalent semantics yielding global counts)
  - The cell labels in the Project Health card stay `Memories`, `Sessions`, `Pending` (no need to re-label as "All Memories" — the workspace-level scope is the implicit default)
  - I-bug-1 test goes GREEN
  - Reconciliation note: this OVERRIDES the legacy `dashboard.Config.Project` default behavior. Document in commit message.

### K-bug-1 [x] — Test: Memory Detail wraps long lines and scrolls
- **Type**: test
- **Effort**: S
- **Depends on**: C2 (helpers `wrap` extracted)
- **Spec ref**: Req 25 (Memory Detail — Wrap and Scroll)
- **Design ref**: Section 4.3, Section 8 (testing strategy)
- **Files**: `internal/dashboard/detail_screen_test.go` (extend)
- **Acceptance criteria**:
  - Test 1: construct a `DetailScreen` with content containing a 200-character single line; render at width 80; assert no rendered line exceeds 80 chars AND no content character is dropped
  - Test 2: construct a `DetailScreen` with content of ~100 wrapped lines; render at viewport height 20; assert footer contains a scroll indicator (e.g., regex match `\d+\s*%` OR `Line \d+ of \d+`)
  - Test 3: send `tea.KeyMsg{Type: tea.KeyDown}` to `DetailScreen.Update`; assert the viewport's `YOffset` (or equivalent scroll position) increments by 1
  - Test 4: short content (< viewport height) — assert footer shows `100%` or omits the scroll indicator
  - Verbatim spec scenarios: "Long line wraps at viewport width", "Long content scrolls vertically", "Short content does not show scroll indicator"

### K-bug-2 [x] — Code: DetailScreen wraps content + scroll indicator
- **Type**: code
- **Effort**: M
- **Depends on**: K-bug-1, C2 (uses `wrap` helper)
- **Spec ref**: Req 25
- **Design ref**: Section 4.3 (footer status line allotment), Section 2b(K) extended
- **Files**: `internal/dashboard/detail_screen.go` (modify in commit 9)
- **Acceptance criteria**:
  - Body content is pre-wrapped at viewport width using `wrap(content, viewportWidth)` BEFORE being set on the `bubbles/viewport.Model`
  - `↑` / `↓` and `j` / `k` map to `viewport.LineUp(1)` / `viewport.LineDown(1)`
  - `PgUp` / `PgDn` map to `viewport.HalfViewUp()` / `viewport.HalfViewDown()`
  - Footer renders a scroll indicator: `fmt.Sprintf("%d%%", int(viewport.ScrollPercent()*100))` — locked format for this change (one of design Open Question #12's two options)
  - Re-wrap is triggered on `tea.WindowSizeMsg` (re-call `wrap` with new width; reset content)
  - K-bug-1 tests go GREEN

---

### Commit slot mapping for bug-fix tasks

| Bug task | Lands in commit |
|----------|-----------------|
| I-bug-1, I-bug-2 | **Commit 7** (`feat(dashboard): Home tab (HomeScreen) with empty state`) — together with I1..I8 |
| K-bug-1, K-bug-2 | **Commit 9** (`feat(dashboard): Detail read-only with [C] copy + clipboard backends`) — together with K1..K5 |

Total task count after addendum: **99 tasks** (95 original + 4 bug-fix). Test : code ratio: **59 : 40**. Bug-fix tasks do NOT change the 15-commit structure or critical path.
