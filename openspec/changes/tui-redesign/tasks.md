# Tasks: TUI Redesign — Flat Screens, Single Theme, Action Menu

> Change: `tui-redesign`
> Status: ready
> Strict TDD: RED → GREEN → REFACTOR enforced — every code task is preceded by a test task.

## Reconciliation Notes (predecessor contradictions resolved here)

| Issue | Resolution |
|-------|------------|
| Top Projects ranking metric unspecified in spec | Implemented as `MAX(updated_at) GROUP BY project` per design §3 |
| `--no-splash` flag existence unconfirmed in spec | Confirmed present in `cmd/thoughtline/main.go:157`. Task I11 is pre-confirmed; removal is in scope. |
| `--splash-ms` flag: spec silent, design §file-table includes removal | Removing `--splash-ms` together with `--theme`/`--no-splash` per design file-table |
| `--no-update-check`: not mentioned for removal in design | Retained — update-check is orthogonal to TUI chrome |
| `themes.go` naming: spec non-prescriptive, design says `theme.go` | Tasks follow design: rename to `theme.go` |
| Pending count refresh: spec says "on screen entry"; design adds `r` + 30s ticker | Tasks implement design's stricter superset |
| `RecentMemories` storage method: design says "confirm or implement" | `Storage.RecentAll(ctx, project, limit)` already exists. A5 writes a test confirming it satisfies the dashboard's contract; no new method needed. |
| `MemoriesByProject` storage method | `Storage.Recent(ctx, project, limit)` already satisfies the contract. A7 writes a test confirming; no new method needed. |
| `MostRecentProjects []string` field on Stats | Does NOT exist yet (current `Stats` has `ByProject map[string]int`). A3/A4 add it. |
| Empty-state copy in tui-removed vs design decision E | Spec requires verbatim message. Design decision E defines it as: `No pending events. Passive capture is OFF — enable via tl_capture_passive.` Tasks use design's exact copy. |

---

## Group A — Storage prerequisites

### A1 — Test: `CountPending` contract
- **Type**: test
- **Effort**: S
- **Depends on**: none (can start immediately)
- **Spec ref**: tui-dashboard Req 6 (Pending events count N), tui-dashboard Req 12
- **Files**: `internal/storage/pending_test.go` (modify)
- **Acceptance criteria**:
  - Test inserts 3 `status=pending` rows for project `alpha` and 2 `status=promoted` rows
  - `CountPending(ctx, "alpha")` returns `3`
  - `CountPending(ctx, "other")` returns `0`
  - Test passes RED (function does not exist yet)
  - `go test ./internal/storage/...` compiles with a `// TODO` stub

### A2 — Code: implement `CountPending`
- **Type**: code
- **Effort**: S
- **Depends on**: A1
- **Design ref**: design §Interfaces / Contracts
- **Files**: `internal/storage/pending.go` (modify)
- **Acceptance criteria**:
  - `func (s *Storage) CountPending(ctx context.Context, project string) (int, error)` added
  - Query: `SELECT count(*) FROM pending_events WHERE project = ? AND status = 'pending'`
  - A1 tests go GREEN
  - `go vet ./internal/storage/...` clean

### A3 — Test: `Stats.MostRecentProjects` field
- **Type**: test
- **Effort**: S
- **Depends on**: none (can start parallel to A1)
- **Spec ref**: tui-dashboard Req 5 (Top Projects)
- **Design ref**: design §3 Open Questions — ranking = `MAX(updated_at) GROUP BY project LIMIT 4`
- **Files**: `internal/storage/stats_test.go` (modify)
- **Acceptance criteria**:
  - Test seeds memories for 5 projects with varying `updated_at`
  - `Stats(ctx, StatsOptions{})` result has `MostRecentProjects []string` with ≤4 entries
  - Projects ordered by `MAX(updated_at) DESC`
  - Test passes RED (`MostRecentProjects` field absent)

### A4 — Code: extend `Stats` struct + query
- **Type**: code
- **Effort**: S
- **Depends on**: A3
- **Files**: `internal/storage/stats.go` (modify)
- **Acceptance criteria**:
  - `MostRecentProjects []string` added to `Stats` struct
  - Populated by: `SELECT project FROM memories WHERE deleted_at IS NULL GROUP BY project ORDER BY MAX(updated_at) DESC LIMIT 4`
  - A3 tests go GREEN
  - Existing Stats tests still pass

### A5 — Test: `RecentAll` satisfies Recent Activity screen contract
- **Type**: test
- **Effort**: S
- **Depends on**: none (can start parallel)
- **Spec ref**: tui-screens Req 2 (Recent Activity)
- **Files**: `internal/storage/recent_test.go` (modify or create)
- **Acceptance criteria**:
  - Confirms `Storage.RecentAll(ctx, "", 20)` returns memories ordered by `updated_at DESC`, capped at limit
  - No new method needed — test documents the existing contract
  - Tests pass GREEN immediately (function already exists)

### A6 — (No new code) — A5 confirms `RecentAll` is sufficient
- NOTE: A6 from the prompt's skeleton is collapsed into A5. No implementation task needed.

### A7 — Test: `Storage.Recent` satisfies BrowseProjects drill-in contract
- **Type**: test
- **Effort**: S
- **Depends on**: none (parallel)
- **Spec ref**: tui-screens Req 3 (Browse Projects)
- **Files**: `internal/storage/recent_test.go` (modify)
- **Acceptance criteria**:
  - Confirms `Storage.Recent(ctx, project, limit)` returns only memories for that project, ordered `updated_at DESC`, capped at limit
  - Tests pass GREEN (function already exists)

---

## Group B — Theme foundation

### B1 — Test: palette constants defined and parseable
- **Type**: test
- **Effort**: S
- **Depends on**: none (parallel)
- **Spec ref**: tui-dashboard Req 8 (Single Fixed Theme)
- **Design ref**: design §Final Hex Palette (14 named tokens)
- **Files**: `internal/dashboard/theme_test.go` (create)
- **Acceptance criteria**:
  - Test enumerates all 14 palette tokens: `Foreground`, `Muted`, `Border`, `LogoGradient[0..4]`, `StatNumber`, `Cursor`, `MenuSelectedBg`, `StatusOK`, `StatusWarn`, `StatusErr`, `Tag`
  - Each is a valid 6-character hex string (regex `^#[0-9A-Fa-f]{6}$`)
  - `LogoGradient` has exactly 5 elements
  - Test passes RED (`theme.go` not yet created)

### B2 — Code: `internal/dashboard/theme.go` with exact palette
- **Type**: code
- **Effort**: S
- **Depends on**: B1
- **Design ref**: design §Final Hex Palette
- **Files**: `internal/dashboard/theme.go` (create), `internal/dashboard/themes.go` (delete — done here or in I9; do it here)
- **Acceptance criteria**:
  - Single `palette` struct (unexported) with all 14 fields from the design table
  - `var defaultPalette = palette{...}` with exact hex codes committed
  - `LogoGradient [5]lipgloss.Color`
  - `themes.go` deleted (rename/replace)
  - B1 tests go GREEN
  - No `Theme1`, `Theme2`, `Theme3` symbols in package

### B3 — Test: lipgloss style helpers produce 3 distinct status styles
- **Type**: test
- **Effort**: S
- **Depends on**: B2
- **Spec ref**: tui-dashboard Req 11 (Status Indicator)
- **Files**: `internal/dashboard/theme_test.go` (extend)
- **Acceptance criteria**:
  - `statusStyle(OK)`, `statusStyle(WARN)`, `statusStyle(ERR)` each return a non-nil `lipgloss.Style`
  - All three render distinct foreground colors (compare rendered strings differ)

### B4 — Code: lipgloss style helpers in `theme.go`
- **Type**: code
- **Effort**: S
- **Depends on**: B3
- **Files**: `internal/dashboard/theme.go` (extend)
- **Acceptance criteria**:
  - `type statusLevel int` with `statusOK`, `statusWARN`, `statusERR` constants
  - `func statusStyle(l statusLevel, p palette) lipgloss.Style` returns appropriately coloured style
  - B3 tests go GREEN

---

## Group C — Screen abstraction

### C1 — Test: `Screen` interface contract
- **Type**: test
- **Effort**: S
- **Depends on**: none (parallel)
- **Spec ref**: tui-dashboard Req 9 (Navigation Contract)
- **Design ref**: design §Interfaces / Contracts
- **Files**: `internal/dashboard/screen_test.go` (create)
- **Acceptance criteria**:
  - `mockScreen` stub that implements the interface compiles
  - Interface has exactly: `Init() tea.Cmd`, `Update(tea.Msg) (Screen, tea.Cmd)`, `View(width, height int, p palette) string`, `Title() string`, `OnFocus() tea.Cmd`
  - Test passes RED (`screen.go` not yet created)

### C2 — Code: `Screen` interface in `internal/dashboard/screen.go`
- **Type**: code
- **Effort**: S
- **Depends on**: C1
- **Design ref**: design §Interfaces / Contracts
- **Files**: `internal/dashboard/screen.go` (create)
- **Acceptance criteria**:
  - Exact interface signature from design committed
  - C1 tests go GREEN

### C3 — Test: screen stack push / pop / peek
- **Type**: test
- **Effort**: S
- **Depends on**: C2
- **Spec ref**: tui-dashboard Req 9
- **Files**: `internal/dashboard/model_test.go` (modify)
- **Acceptance criteria**:
  - `Push(s Screen)` adds screen to top; `Peek()` returns it
  - `Pop()` removes top; `Peek()` returns previous
  - `Pop()` on stack with 1 item is a no-op (dashboard is always present)
  - Tests pass RED (Model not yet refactored)

### C4 — Code: `Model` struct with stack methods
- **Type**: code
- **Effort**: M
- **Depends on**: C3, B2
- **Design ref**: design §Interfaces / Contracts (`Model` struct definition)
- **Files**: `internal/dashboard/model.go` (modify)
- **Acceptance criteria**:
  - `Model` struct gains: `stack []Screen`, `palette palette`, `pendingCount int`, `diskFreePct int`, `lastErr error`, `width int`, `height int`
  - Old `tabKey`/cube/splash/theme-selection fields removed
  - `Push`, `Pop`, `Peek` methods on `*Model`
  - C3 tests go GREEN

---

## Group D — Logo + asset bytes

### D1 — Test: `thoughtlineLogo` row count and column budget
- **Type**: test
- **Effort**: S
- **Depends on**: none (parallel)
- **Spec ref**: tui-dashboard Req 2 (ASCII Logo)
- **Design ref**: design §ASCII Logo (5 rows, ≤70 cols)
- **Files**: `internal/dashboard/logo_test.go` (create)
- **Acceptance criteria**:
  - `thoughtlineLogo` is a `[]string` with `len >= 5`
  - Each element has `len([]rune(row)) <= 70`
  - Test passes RED (`logo.go` not yet created)

### D2 — Code: `thoughtlineLogo` in `internal/dashboard/logo.go`
- **Type**: code
- **Effort**: M
- **Depends on**: D1
- **Design ref**: design §ASCII Logo — block-letter style, 5 rows, ≤70 cols, hardcoded `var thoughtlineLogo []string`
- **Files**: `internal/dashboard/logo.go` (create)
- **Acceptance criteria**:
  - `var thoughtlineLogo = []string{...}` with 5 hand-crafted block-letter rows for "THOUGHTLINE"
  - Each row ≤70 runes wide
  - D1 tests go GREEN

### D3 — Test: gradient renderer applies 5 distinct colors row-by-row
- **Type**: test
- **Effort**: S
- **Depends on**: D2, B2
- **Spec ref**: tui-dashboard Req 2 (Logo gradient)
- **Files**: `internal/dashboard/logo_test.go` (extend)
- **Acceptance criteria**:
  - `renderLogo(p palette)` returns a string
  - Splitting on newline yields ≥5 non-empty lines
  - Consecutive lines differ in their ANSI color escape prefix (proxy: each rendered row starts with a different lipgloss color code)

### D4 — Code: gradient render helper in `logo.go`
- **Type**: code
- **Effort**: S
- **Depends on**: D3
- **Files**: `internal/dashboard/logo.go` (extend)
- **Acceptance criteria**:
  - `func renderLogo(p palette) string` applies `p.LogoGradient[i]` per row via lipgloss
  - Rows joined with `lipgloss.JoinVertical(lipgloss.Left, rows...)`
  - If terminal width < 70: returns `lipgloss.NewStyle().Foreground(p.LogoGradient[0]).Render("THOUGHTLINE")` (single-color fallback per design decision J)
  - D3 tests go GREEN

---

## Group E — Disk-free + status indicator

### E1 — Test: `freeDiskPct` on Unix
- **Type**: test
- **Effort**: S
- **Depends on**: none (parallel)
- **Spec ref**: tui-dashboard Req 11 (Status Indicator)
- **Design ref**: design §D — `syscall.Statfs`, parent dir of DB path
- **Files**: `internal/dashboard/diskfree_unix_test.go` (create, build tag `//go:build !windows`)
- **Acceptance criteria**:
  - `freeDiskPct("/tmp")` (or OS temp dir) returns value in range [0, 100] with nil error
  - Test passes RED on Unix (function not yet implemented)

### E2 — Code: `freeDiskPct` on Unix
- **Type**: code
- **Effort**: S
- **Depends on**: E1
- **Files**: `internal/dashboard/diskfree_unix.go` (create, build tag `//go:build !windows`)
- **Acceptance criteria**:
  - Uses `syscall.Statfs`; computes `(Bavail * 100) / Blocks`
  - Returns `(0, err)` on failure (maps to status `ERR`)
  - E1 tests go GREEN on Unix

### E3 — Test: `freeDiskPct` on Windows
- **Type**: test
- **Effort**: S
- **Depends on**: none (parallel to E1)
- **Files**: `internal/dashboard/diskfree_windows_test.go` (create, build tag `//go:build windows`)
- **Acceptance criteria**:
  - `freeDiskPct(os.TempDir())` returns value in [0, 100] with nil error on Windows

### E4 — Code: `freeDiskPct` on Windows
- **Type**: code
- **Effort**: S
- **Depends on**: E3
- **Files**: `internal/dashboard/diskfree_windows.go` (create, build tag `//go:build windows`)
- **Acceptance criteria**:
  - Uses `golang.org/x/sys/windows.GetDiskFreeSpaceEx`
  - Returns `(0, err)` on failure
  - E3 tests go GREEN on Windows

### E5 — Test: status indicator maps percent to OK / WARN / ERR
- **Type**: test
- **Effort**: S
- **Depends on**: B4
- **Spec ref**: tui-dashboard Req 11 (threshold table)
- **Files**: `internal/dashboard/theme_test.go` (extend) or `internal/dashboard/status_test.go` (create)
- **Acceptance criteria**:
  - `diskStatusLevel(25, nil)` → `statusOK`
  - `diskStatusLevel(8, nil)` → `statusWARN`
  - `diskStatusLevel(0, errSomeError)` → `statusERR`
  - Threshold: `< 10%` = WARN, DB error = ERR, else OK

### E6 — Code: status indicator function
- **Type**: code
- **Effort**: S
- **Depends on**: E5
- **Files**: `internal/dashboard/status.go` (create) or inline in `dashboard_screen.go`
- **Acceptance criteria**:
  - `func diskStatusLevel(pct int, dbErr error) statusLevel`
  - `func formatStatusLine(pct int, dbErr error, p palette) string` returns `MEM: OK 25%` / `MEM: WARN` / `MEM: ERR` per spec
  - E5 tests go GREEN

---

## Group F — Dashboard screen

### F1 — Test: dashboard renders all 7 sections in empty state
- **Type**: test
- **Effort**: M
- **Depends on**: C2, B4, D4, E6 (all foundations must compile)
- **Spec ref**: tui-dashboard Req 1 (layout), Req 4 (stat card zero state), Req 5 (Top Projects empty)
- **Files**: `internal/dashboard/dashboard_screen_test.go` (create)
- **Acceptance criteria**:
  - `DashboardScreen.View(100, 30, defaultPalette)` with zero-value stats produces a string
  - String contains: header border, logo rows ≥5, tagline prefix `> thoughtline v`, double-border stat card, `No projects yet`, `▸`, footer `j/k navigate • enter select • s search • q quit`
  - All 4 stat cells show `0`
  - Test passes RED (`DashboardScreen` not yet created)

### F2 — Code: `DashboardScreen` in `internal/dashboard/dashboard_screen.go`
- **Type**: code
- **Effort**: L
- **Depends on**: F1, A2, A4, B4, C2, D4, E6
- **Design ref**: design §File Changes, §Data Flow
- **Files**: `internal/dashboard/dashboard_screen.go` (create)
- **Acceptance criteria**:
  - Implements `Screen` interface
  - Renders all 7 sections: header (double-border, status indicator), logo (gradient), tagline (version var), stat card (double-border, 4 cells: memories/sessions/projects/this-week), top projects (≤3 bullets + overflow line), action menu (5 items + `▸` cursor), footer
  - Stat cell failure renders `—` per design decision K
  - F1 tests go GREEN

### F3 — Test: cursor `j` / `k` with no-wrap
- **Type**: test
- **Effort**: S
- **Depends on**: F2
- **Spec ref**: tui-dashboard Req 10
- **Files**: `internal/dashboard/dashboard_screen_test.go` (extend)
- **Acceptance criteria**:
  - Cursor starts at index 0 (`Search memories`)
  - `j` from index 0 → index 1; `j` from index 4 → stays at 4
  - `k` from index 0 → stays at 0; `k` from index 4 → index 3

### F4 — Code: keymap handlers for `j` / `k`
- **Type**: code
- **Effort**: S
- **Depends on**: F3
- **Files**: `internal/dashboard/dashboard_screen.go` (extend Update)
- **Acceptance criteria**:
  - `Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})` increments cursor (clamped at 4)
  - `Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})` decrements cursor (clamped at 0)
  - F3 tests go GREEN

### F5 — Test: `s` shortcut navigates to Search regardless of cursor
- **Type**: test
- **Effort**: S
- **Depends on**: F2, C4
- **Spec ref**: tui-dashboard Req 10 (s shortcut)
- **Files**: `internal/dashboard/dashboard_screen_test.go` (extend)
- **Acceptance criteria**:
  - Cursor on index 2 (Browse projects); `s` pressed → returned `tea.Cmd` is a `pushScreenCmd` for SearchScreen

### F6 — Code: `s` shortcut handler
- **Type**: code
- **Effort**: S
- **Depends on**: F5
- **Files**: `internal/dashboard/dashboard_screen.go` (extend Update)
- **Acceptance criteria**:
  - F5 tests go GREEN

### F7 — Test: `r` triggers stat + Top Projects + pending refresh
- **Type**: test
- **Effort**: S
- **Depends on**: F2
- **Spec ref**: tui-dashboard Req 10 (`r` → refresh)
- **Design ref**: design §5 pending caching — refresh on `r`
- **Files**: `internal/dashboard/dashboard_screen_test.go` (extend)
- **Acceptance criteria**:
  - `Update(keyMsg("r"))` returns a non-nil `tea.Cmd` (the load-stats command)
  - The returned cmd is not `tea.Quit`

### F8 — Code: refresh command on `r`
- **Type**: code
- **Effort**: S
- **Depends on**: F7
- **Files**: `internal/dashboard/dashboard_screen.go` (extend Update)
- **Acceptance criteria**:
  - F7 tests go GREEN

### F9 — Test: `enter` on each menu item pushes the right screen
- **Type**: test
- **Effort**: M
- **Depends on**: F2, C4
- **Spec ref**: tui-dashboard Req 9 (drill-in)
- **Files**: `internal/dashboard/dashboard_screen_test.go` (extend)
- **Acceptance criteria**:
  - `enter` on index 0 → cmd pushes `SearchScreen`
  - `enter` on index 1 → cmd pushes `RecentScreen`
  - `enter` on index 2 → cmd pushes `BrowseProjectsScreen`
  - `enter` on index 3 → cmd pushes `PendingEventsScreen` (even when count=0)
  - `enter` on index 4 → returns `tea.Quit`

### F10 — Code: drill-in dispatch on `enter`
- **Type**: code
- **Effort**: S
- **Depends on**: F9
- **Files**: `internal/dashboard/dashboard_screen.go` (extend Update)
- **Acceptance criteria**:
  - F9 tests go GREEN

### F11 — Test: 30s ticker refreshes pending count without blocking input
- **Type**: test
- **Effort**: S
- **Depends on**: F2, A2
- **Spec ref**: tui-dashboard Req 12 (pending count refresh)
- **Design ref**: design §5 — 30s ticker
- **Files**: `internal/dashboard/dashboard_screen_test.go` (extend)
- **Acceptance criteria**:
  - `DashboardScreen.Init()` returns a non-nil `tea.Cmd` that sets up a ticker
  - Sending a `tickMsg` to `Update` returns a fresh load-pending cmd

### F12 — Code: 30s ticker command
- **Type**: code
- **Effort**: S
- **Depends on**: F11
- **Files**: `internal/dashboard/dashboard_screen.go` (extend Init/Update)
- **Acceptance criteria**:
  - `Init()` starts a 30s ticker via `tea.Every`
  - F11 tests go GREEN

---

## Group G — Drill-in screens

### G1 — Test: SearchScreen — empty query is no-op
- **Type**: test
- **Effort**: S
- **Depends on**: C2
- **Spec ref**: tui-screens Req 1
- **Files**: `internal/dashboard/search_screen_test.go` (create)
- **Acceptance criteria**:
  - `Update(enterKey)` with empty input field returns nil cmd and no result list

### G2 — Test: SearchScreen — non-empty query executes search
- **Type**: test
- **Effort**: S
- **Depends on**: G1
- **Spec ref**: tui-screens Req 1 scenario: Submit non-empty query
- **Files**: `internal/dashboard/search_screen_test.go` (extend)
- **Acceptance criteria**:
  - `Update(enterKey)` with input `"spawning bug"` returns a `tea.Cmd` that executes storage search
  - After `searchResultsMsg` received, `View()` contains result titles

### G3 — Test: SearchScreen — `esc` / `q` returns to dashboard
- **Type**: test
- **Effort**: S
- **Depends on**: G1
- **Spec ref**: tui-screens Req 1, tui-dashboard Req 9
- **Files**: `internal/dashboard/search_screen_test.go` (extend)
- **Acceptance criteria**:
  - `Update(escKey)` returns a `popScreenCmd`
  - `Update(qKey)` also returns a `popScreenCmd` (not quit)

### G4 — Code: `SearchScreen` in `search_screen.go`
- **Type**: code
- **Effort**: M
- **Depends on**: G1, G2, G3
- **Design ref**: design §File Changes `search_screen.go`
- **Files**: `internal/dashboard/search_screen.go` (create)
- **Acceptance criteria**:
  - Uses `bubbles/textinput` for query field
  - Result list below input
  - G1/G2/G3 tests go GREEN

### G5 — Test: RecentScreen — memories ordered by updated_at DESC
- **Type**: test
- **Effort**: S
- **Depends on**: C2, A5
- **Spec ref**: tui-screens Req 2
- **Files**: `internal/dashboard/recent_screen_test.go` (create)
- **Acceptance criteria**:
  - Injecting 3 memories with descending `updated_at`, first item in `View()` output matches newest

### G6 — Test: RecentScreen — `enter` pushes detail screen
- **Type**: test
- **Effort**: S
- **Depends on**: G5
- **Spec ref**: tui-screens Req 2
- **Files**: `internal/dashboard/recent_screen_test.go` (extend)
- **Acceptance criteria**:
  - `Update(enterKey)` with item selected returns a `pushScreenCmd` for `DetailScreen`

### G7 — Code: `RecentScreen` in `recent_screen.go`
- **Type**: code
- **Effort**: M
- **Depends on**: G5, G6
- **Files**: `internal/dashboard/recent_screen.go` (create)
- **Acceptance criteria**:
  - Default N=20; uses `Storage.RecentAll`
  - `j`/`k` scroll; `enter` pushes detail; `esc`/`q` pops
  - G5/G6 tests go GREEN

### G8 — Test: BrowseProjectsScreen — alphabetical list
- **Type**: test
- **Effort**: S
- **Depends on**: C2
- **Spec ref**: tui-screens Req 3
- **Files**: `internal/dashboard/browse_screen_test.go` (create)
- **Acceptance criteria**:
  - Projects `zombie-game`, `alpha-studio`, `pipeline-tools` → rendered order: `alpha-studio`, `pipeline-tools`, `zombie-game`

### G9 — Test: BrowseProjectsScreen — `enter` pushes project-filtered RecentScreen
- **Type**: test
- **Effort**: S
- **Depends on**: G8
- **Spec ref**: tui-screens Req 3
- **Files**: `internal/dashboard/browse_screen_test.go` (extend)
- **Acceptance criteria**:
  - `Update(enterKey)` on `alpha-studio` returns a `pushScreenCmd` for `RecentScreen` scoped to `alpha-studio`

### G10 — Code: `BrowseProjectsScreen` in `projects_screen.go`
- **Type**: code
- **Effort**: M
- **Depends on**: G8, G9
- **Files**: `internal/dashboard/projects_screen.go` (create)
- **Acceptance criteria**:
  - Project list from `Stats.ByProject` sorted alphabetically
  - `enter` pushes scoped `RecentScreen`; `esc`/`q` pops
  - G8/G9 tests go GREEN

### G11 — Test: PendingEventsScreen — list with required columns
- **Type**: test
- **Effort**: S
- **Depends on**: C2, A2
- **Spec ref**: tui-screens Req 4
- **Files**: `internal/dashboard/pending_screen_test.go` (create)
- **Acceptance criteria**:
  - 3 pending events in DB → `View()` contains `event_type`, `captured_at` human-readable, 8-char hash prefix, payload[0:60]

### G12 — Test: PendingEventsScreen — empty state shows exact copy
- **Type**: test
- **Effort**: S
- **Depends on**: G11
- **Spec ref**: tui-screens Req 4 (empty state), tui-removed Req 3 (single palette, no multi-theme)
- **Design ref**: design decision E — exact copy: `No pending events. Passive capture is OFF — enable via tl_capture_passive.`
- **Files**: `internal/dashboard/pending_screen_test.go` (extend)
- **Acceptance criteria**:
  - 0 pending events → `View()` contains that exact string verbatim

### G13 — Test: PendingEventsScreen — `enter` pushes detail with full payload
- **Type**: test
- **Effort**: S
- **Depends on**: G11
- **Spec ref**: tui-screens Req 4
- **Files**: `internal/dashboard/pending_screen_test.go` (extend)
- **Acceptance criteria**:
  - `Update(enterKey)` with item highlighted returns a `pushScreenCmd` for detail screen containing full payload

### G14 — Code: `PendingEventsScreen` in `pending_screen.go`
- **Type**: code
- **Effort**: M
- **Depends on**: G11, G12, G13, A2
- **Files**: `internal/dashboard/pending_screen.go` (create)
- **Acceptance criteria**:
  - Uses `Storage.ListPending` scoped to current project, ordered `captured_at DESC`
  - Empty state message is exact design decision E copy
  - `enter` pushes detail; `esc`/`q` pops
  - G11/G12/G13 tests go GREEN

### G15 — Test: detail screen renders full memory content
- **Type**: test
- **Effort**: S
- **Depends on**: C2
- **Spec ref**: tui-screens Req 2 (enter opens detail), Req 4 (enter opens full payload)
- **Files**: `internal/dashboard/detail_screen_test.go` (create)
- **Acceptance criteria**:
  - `DetailScreen` with a memory/payload injected → `View()` contains the full content without truncation
  - `esc`/`q` returns a `popScreenCmd`

### G16 — Code: `DetailScreen` in `detail_screen.go`
- **Type**: code
- **Effort**: S
- **Depends on**: G15
- **Files**: `internal/dashboard/detail_screen.go` (create)
- **Acceptance criteria**:
  - Scrollable viewport (use `bubbles/viewport`)
  - Accepts either `storage.SearchResult` or raw payload string
  - G15 tests go GREEN

---

## Group H — Resize behavior

### H1 — Test: terminal < 80×24 renders warning and freezes
- **Type**: test
- **Effort**: S
- **Depends on**: C4
- **Spec ref**: tui-dashboard Req 1 (min viewport implied), design decision B
- **Files**: `internal/dashboard/model_test.go` (extend)
- **Acceptance criteria**:
  - Sending `tea.WindowSizeMsg{Width: 79, Height: 23}` → `Model.View()` contains warning string
  - All keypresses except `q`/`ctrl+c` are no-ops in this state

### H2 — Test: terminal ≥ 80×24 centers content with max width 100
- **Type**: test
- **Effort**: S
- **Depends on**: H1
- **Files**: `internal/dashboard/model_test.go` (extend)
- **Acceptance criteria**:
  - `tea.WindowSizeMsg{Width: 120, Height: 40}` → rendered content width ≤ 100

### H3 — Code: resize handler in `view.go` and `update.go`
- **Type**: code
- **Effort**: S
- **Depends on**: H1, H2
- **Files**: `internal/dashboard/view.go` (modify), `internal/dashboard/update.go` (modify)
- **Acceptance criteria**:
  - `Model.width`, `Model.height` updated on `tea.WindowSizeMsg`
  - `View()` renders warning centered when below threshold
  - H1/H2 tests go GREEN

---

## Group I — Wire-up and deletion series

### I1 — Code: wire new Model + screen stack into `thoughtline ui` entrypoint
- **Type**: code
- **Effort**: M
- **Depends on**: C4, F2 (dashboard screen must exist)
- **Design ref**: migration commit 1 + 2
- **Files**: `internal/dashboard/run.go` (modify), `internal/dashboard/model.go` (modify), `cmd/thoughtline/main.go` (modify — remove old `ThemeName`/`Splash`/`SplashDuration` from `Config` struct usage)
- **Acceptance criteria**:
  - `dashboard.Run(ctx, st, cfg)` now initialises the new `Model` with `DashboardScreen` as root
  - Old splash/cube code paths are dead but files still present
  - `go build ./...` succeeds
  - Commit message: `feat(dashboard): introduce Screen interface and stack`

### I2 — Test: top-level integration — `thoughtline ui` launches and `q` exits
- **Type**: test
- **Effort**: M
- **Depends on**: I1, J1 (teatest must be available)
- **Spec ref**: tui-dashboard Req 9 (q on dashboard quits)
- **Files**: `internal/dashboard/integration_test.go` (create)
- **Acceptance criteria**:
  - Using `teatest`, program launches, receives WindowSizeMsg(100, 30), then `q` key → program exits with code 0
  - Test passes GREEN

### I3 — Code: remove tab-based navigation from `update.go`
- **Type**: code
- **Effort**: M
- **Depends on**: I2
- **Design ref**: migration commit 3 (`refactor(dashboard): route Update through stack; remove tabKey`)
- **Files**: `internal/dashboard/update.go` (modify), `internal/dashboard/model.go` (modify)
- **Acceptance criteria**:
  - All `tabKey`/`activeTab`/tab-switching code removed
  - `Update` routes to `stack[len-1].Update`
  - `ctrl+c` always quits; `q` on dashboard quits; `q`/`esc` on child pops
  - `go test ./internal/dashboard/...` green
  - Commit: `refactor(dashboard): route Update through stack; remove tabKey`

### I4 — Test: `tabKey` symbol is absent from compiled package
- **Type**: test
- **Effort**: S
- **Depends on**: I3
- **Spec ref**: tui-removed Req 4
- **Files**: `internal/dashboard/model_test.go` or dedicated removed_test.go
- **Acceptance criteria**:
  - Compilation succeeds only if `tabKey` and `activeTab` identifiers are absent (build-time check via compile test)

### I5 — Code: delete `cube.go` and all references
- **Type**: code
- **Effort**: S
- **Depends on**: I3
- **Design ref**: migration commit 6
- **Files**: `internal/dashboard/cube.go` (delete), any file referencing cube symbols (audit + remove)
- **Acceptance criteria**:
  - File deleted from repo
  - `rg "cube" internal/dashboard/` finds no symbol references (comments OK)
  - `go build ./...` succeeds
  - Commit: `chore(dashboard): delete cube.go`

### I6 — Test: `cube.go` absent on disk
- **Type**: test
- **Effort**: S
- **Depends on**: I5
- **Spec ref**: tui-removed Req 1
- **Files**: `internal/dashboard/removed_test.go` (create or extend)
- **Acceptance criteria**:
  - `os.Stat("internal/dashboard/cube.go")` returns `os.ErrNotExist` (path relative to module root via `runtime`/`testdata`)

### I7 — Code: delete `splash.go` and all references
- **Type**: code
- **Effort**: S
- **Depends on**: I5
- **Design ref**: migration commit 7
- **Files**: `internal/dashboard/splash.go` (delete), references removed
- **Acceptance criteria**:
  - File deleted; `go build ./...` succeeds
  - Commit: `chore(dashboard): delete splash.go`

### I8 — Test: `splash.go` absent on disk
- **Type**: test
- **Effort**: S
- **Depends on**: I7
- **Spec ref**: tui-removed Req 2
- **Files**: `internal/dashboard/removed_test.go` (extend)
- **Acceptance criteria**:
  - `os.Stat("internal/dashboard/splash.go")` returns `os.ErrNotExist`

### I9 — Code: delete `themes.go` (already replaced by `theme.go` in B2)
- **Type**: code
- **Effort**: S
- **Depends on**: B2, I7
- **Design ref**: migration commit 8
- **Files**: `internal/dashboard/themes.go` (confirm deleted — should already be gone from B2), `internal/dashboard/styles.go` (modify — strip multi-theme `ApplyTheme`, rebuild from `palette`)
- **Acceptance criteria**:
  - `themes.go` does not exist
  - `styles.go` builds against `palette` only — no multi-theme conditionals
  - No `Theme1`, `Theme2`, `Theme3` identifiers in package
  - Commit: `refactor(dashboard): collapse themes.go → theme.go (single palette)`

### I10 — Test: `Theme1`/`Theme2`/`Theme3` symbols absent
- **Type**: test
- **Effort**: S
- **Depends on**: I9
- **Spec ref**: tui-removed Req 3
- **Files**: `internal/dashboard/removed_test.go` (extend)
- **Acceptance criteria**:
  - Compile-time: package compiles cleanly with no multi-theme symbols
  - `rg "Theme[123]" internal/dashboard/` returns zero matches

### I11 — Code: read `main.go`, identify all flags to remove, then remove them
- **Type**: code
- **Effort**: S
- **Depends on**: I9
- **Design ref**: migration commit 9 — confirmed flags: `--theme`, `--no-splash`, `--splash-ms`
- **Files**: `cmd/thoughtline/main.go` (modify)
- **Pre-step** (implementer must do first): confirm current `runDashboard` flag registrations before editing
- **Acceptance criteria**:
  - `fs.String("theme", ...)`, `fs.Bool("no-splash", ...)`, `fs.Int("splash-ms", ...)` calls removed
  - `ThemeName`, `Splash`, `SplashDuration` fields removed from `dashboard.Config` (or zeroed if struct shared)
  - Custom `flag.Usage` / error wrapper added: when `flag.Parse` returns error containing `-theme`/`-no-splash`/`-splash-ms`, print the design decision H message: `thoughtline ui: flag provided but not defined: -theme. The --theme flag was removed in v0.2 (single-theme TUI). See CHANGELOG.md.`
  - `--no-update-check` retained
  - `printUsage()` updated to remove stale flag references
  - Commit: `feat(cli): remove --theme/--no-splash/--splash-ms with migration error`

### I12 — Test: removed flags print exact error message and exit non-zero
- **Type**: test
- **Effort**: S
- **Depends on**: I11
- **Spec ref**: tui-removed Req 3 (--theme), Req 5 (--no-splash), design decision H (error format)
- **Files**: `cmd/thoughtline/main_test.go` (create or extend)
- **Acceptance criteria**:
  - `thoughtline ui --theme classic` → exit code non-zero, stderr contains `--theme flag was removed`
  - `thoughtline ui --no-splash` → exit code non-zero
  - `thoughtline ui --splash-ms 500` → exit code non-zero
  - `thoughtline ui --no-update-check` → exit code 0 (retained flag works)

### I13 — Test: `tabs_test.go` absent on disk
- **Type**: test
- **Effort**: S
- **Depends on**: I3
- **Spec ref**: tui-removed Req 4 (tabs_test.go deleted)
- **Files**: `internal/dashboard/removed_test.go` (extend)
- **Acceptance criteria**:
  - `os.Stat("internal/dashboard/tabs_test.go")` returns `os.ErrNotExist`
  - Note: delete the actual file in I3; this test only confirms absence

---

## Group J — Test infrastructure

### J1 — Test+Code: confirm `teatest` available and add direct require
- **Type**: code + infra
- **Effort**: S
- **Depends on**: none (can start first, unblocks I2 and integration tests)
- **Design ref**: design decision F — `github.com/charmbracelet/x/exp/teatest`
- **Files**: `go.mod` (modify), `go.sum` (modify), `internal/dashboard/teatest_smoke_test.go` (create)
- **Acceptance criteria**:
  - `go.mod` has `require github.com/charmbracelet/x/exp/teatest` as a direct dependency at a pinned version
  - Smoke test: create a trivial `tea.Program`, run it with `teatest.NewTestModel`, send `q`, assert exit
  - `go test ./internal/dashboard/... -run TestTeatestSmoke` passes GREEN

### J2 — Test: snapshot test for dashboard render (golden file)
- **Type**: test
- **Effort**: M
- **Depends on**: F2, J1
- **Design ref**: design §Testing Strategy — golden file `testdata/dashboard.golden`
- **Files**: `internal/dashboard/dashboard_screen_test.go` (extend), `internal/dashboard/testdata/dashboard.golden` (create on first run)
- **Acceptance criteria**:
  - `DashboardScreen.View(100, 30, defaultPalette)` with a fixed stats fixture matches golden file
  - Run with `-update` flag re-generates the golden file
  - On CI, golden file must match exactly (no regeneration)

---

## Group K — Documentation

### K1 — Code: ADR `0005-tui-redesign-engram-style.md`
- **Type**: docs
- **Effort**: S
- **Depends on**: I9 (all deletions done before writing the ADR)
- **Files**: `docs/decisions/0005-tui-redesign-engram-style.md` (create)
- **Acceptance criteria**:
  - Covers: rationale for flat-screen redesign, what was removed (cube/splash/multi-theme), palette choice (Rose-Pine-Moon adapted), architectural pattern origin (Engram TUI, MIT), license attribution note

### K2 — Code: update `README.md`
- **Type**: docs
- **Effort**: S
- **Depends on**: I12
- **Files**: `README.md` (modify)
- **Acceptance criteria**:
  - Removes `--theme {brand|zbrush|mono}`, `--no-splash`, `--splash-ms` from flags table
  - TUI section updated (no cube reference, no theme flag)
  - Screenshot or mockup updated / placeholder added

### K3 — Code: update `CHANGELOG.md`
- **Type**: docs
- **Effort**: S
- **Depends on**: I12
- **Design ref**: migration commit 10
- **Files**: `CHANGELOG.md` (modify)
- **Acceptance criteria**:
  - BREAKING section: `--theme`, `--no-splash`, `--splash-ms` removed; describes migration path
  - Lists new screens: Dashboard, Search, Recent, Browse Projects, Pending Events

### K4 — Code: update `docs/COMPARISON.md`
- **Type**: docs
- **Effort**: S
- **Depends on**: K3
- **Files**: `docs/COMPARISON.md` (modify if exists)
- **Acceptance criteria**:
  - "Built-in TUI" row reflects new flat-screen dashboard (no cube, no tabs)

### K5 — Code: update `docs/PROGRESS.md`
- **Type**: docs
- **Effort**: S
- **Depends on**: K3
- **Files**: `docs/PROGRESS.md` (modify if exists)
- **Acceptance criteria**:
  - TUI Redesign milestone marked complete with summary

---

## Group L — Verification gate

### L1 — Verify: `go test ./...` green
- **Type**: infra
- **Effort**: S
- **Depends on**: all code tasks
- **Acceptance criteria**: zero test failures, zero build errors

### L2 — Verify: `go vet ./...` clean
- **Type**: infra
- **Effort**: S
- **Depends on**: L1
- **Acceptance criteria**: zero vet warnings

### L3 — Verify: license hygiene
- **Type**: infra
- **Effort**: S
- **Depends on**: L1
- **Acceptance criteria**: `bash scripts/check-no-claude-mem.sh` passes (if script exists)

### L4 — Verify: cross-platform CI green
- **Type**: infra
- **Effort**: S
- **Depends on**: L2
- **Acceptance criteria**: GitHub Actions CI matrix passes on Linux/macOS/Windows

### L5 — Verify: manual smoke test
- **Type**: infra
- **Effort**: S
- **Depends on**: L4
- **Acceptance criteria**:
  - `thoughtline ui` launches, dashboard renders all 7 sections
  - All 5 menu items navigable with `j/k` + `enter`
  - `esc` returns to dashboard from every child screen
  - `q` on dashboard exits cleanly

---

## Dependency and Parallelism Summary

### Can start in parallel (no dependencies):
- A1, A3, A5, A7 (storage tests)
- B1 (theme test)
- C1 (screen interface test)
- D1 (logo test)
- E1, E3 (disk-free tests)
- J1 (teatest infra — start FIRST, unblocks I2)

### Critical path (longest sequential chain):
```
J1 → I1 → I2 → I3 → I5 → I7 → I9 → I11 → I12 → K2/K3 → L1 → L2 → L3 → L4 → L5
```

### Primary build-up chain for dashboard screen:
```
A1→A2, A3→A4, B1→B2→B3→B4, C1→C2→C3→C4, D1→D2→D3→D4, E1→E2+E3→E4, E5→E6
→ F1→F2→F3→F4→F5→F6→F7→F8→F9→F10→F11→F12
```

### Drill-in screens (parallel with each other after C2 is ready):
```
G1→G2→G3→G4 (Search)
G5→G6→G7 (Recent)
G8→G9→G10 (Browse)
G11→G12→G13→G14 (Pending)
G15→G16 (Detail — shared by G6, G13)
```

---

## Task Count by Group

| Group | Tests | Code/Docs/Infra | Total |
|-------|-------|-----------------|-------|
| A — Storage | 3 | 2 (+ 2 confirming existing) | 7 |
| B — Theme | 2 | 2 | 4 |
| C — Screen abstraction | 2 | 2 | 4 |
| D — Logo | 2 | 2 | 4 |
| E — Disk-free + status | 4 | 3 | 7 |
| F — Dashboard screen | 6 | 6 | 12 |
| G — Drill-in screens | 9 | 7 | 16 |
| H — Resize | 2 | 1 | 3 |
| I — Wire-up + delete | 6 | 7 | 13 |
| J — Test infra | 1 | 1 | 2 |
| K — Docs | 0 | 5 | 5 |
| L — Verification | 0 | 5 | 5 |
| **Total** | **37** | **43** | **80** |

---

## Effort Summary

| Effort | Count | Notes |
|--------|-------|-------|
| S (small, < 2h) | 64 | Most unit tests and focused implementations |
| M (medium, 2–4h) | 14 | Dashboard screen, drill-in screens, Model refactor, integration test |
| L (large, 4–8h) | 2 | F2 (DashboardScreen full impl), G14 (PendingEventsScreen) |
| **Total estimated** | | ~80–100h (spread across parallel tracks) |

---

## Commit Series (per design decision #6)

1. `feat(dashboard): introduce Screen interface and stack` — C1→C4, J1
2. `feat(dashboard): dashboard_screen with logo + stat card + menu` — A1→A4, B1→B4, D1→D4, E1→E6, F1→F12
3. `refactor(dashboard): route Update through stack; remove tabKey` — I1, I2, I3, I4, H1→H3
4. `feat(dashboard): port search into search_screen` — G1→G4
5. `feat(dashboard): recent/projects/pending screens` — G5→G16
6. `chore(dashboard): delete cube.go` — I5, I6
7. `chore(dashboard): delete splash.go` — I7, I8
8. `refactor(dashboard): collapse themes.go → theme.go (single palette)` — I9, I10, I13
9. `feat(cli): remove --theme/--no-splash/--splash-ms with migration error` — I11, I12
10. `docs: CHANGELOG + README screenshots` — K1→K5, J2, L1→L5
