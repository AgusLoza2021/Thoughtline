# tui-dashboard Specification

> Change: `tui-redesign`
> Status: draft
> Operation: ADDED (new capability — no prior spec exists)

## Purpose

Defines the layout, navigation, keybindings, and data contract for the Thoughtline dashboard screen — the first screen a user sees after `thoughtline ui`. The dashboard replaces the former tab-based, multi-theme, animated TUI.

---

## Requirements

### Requirement 1: Dashboard Screen Layout

The dashboard screen MUST render exactly these 7 sections, top-to-bottom, in this order:

| Order | Section | Notes |
|-------|---------|-------|
| 1 | Header card | Double-border |
| 2 | ASCII logo | "THOUGHTLINE", 5+ rows tall, monospaced grid, gradient per row |
| 3 | Tagline | `> thoughtline v<version> — game-dev memory that survives` |
| 4 | Stat card | Double-border, 2×2 grid, 4 numeric cells |
| 5 | Top Projects | Bulleted list, max 3 items + "...and N more" when overflow |
| 6 | Action menu | 5 items, vertical, `▸` cursor on active item |
| 7 | Footer | `j/k navigate • enter select • s search • q quit` |

No section MAY be omitted even when the underlying data is unavailable (sections MUST render in an empty/zero state instead).

#### Scenario: Full dashboard render

- GIVEN the DB is readable and contains at least one memory and one project
- WHEN the dashboard screen is the top of the navigation stack
- THEN all 7 sections render in the correct top-to-bottom order
- AND the header card and stat card each use a double-border style

#### Scenario: Empty DB dashboard render

- GIVEN the DB is readable and contains zero memories and zero projects
- WHEN the dashboard screen is the top of the stack
- THEN the stat card MUST show all four numeric cells as `0`
- AND the Top Projects section MUST render an empty bullet or a "No projects yet" placeholder (not absent)
- AND all other sections still render normally

---

### Requirement 2: ASCII Logo

The ASCII logo MUST satisfy:

- At least 5 rows tall
- Rendered as a monospaced character grid (no proportional spacing)
- A color gradient applied per-row (5 distinct colors top-to-bottom: mauve → lavender → blue → teal → green)
- The exact glyph style (block, proportional, FIGlet font) is deferred to design

#### Scenario: Logo row count

- GIVEN the dashboard renders
- WHEN the logo component is measured
- THEN it MUST span at least 5 rendered terminal rows

#### Scenario: Logo gradient

- GIVEN the dashboard renders
- WHEN each row of the logo is inspected
- THEN consecutive rows MUST have distinct foreground color values
- AND the gradient MUST progress through at least 5 distinct color steps

---

### Requirement 3: Tagline

The tagline MUST read `> thoughtline v<version> — game-dev memory that survives` where `<version>` is sourced from the build-time version variable (not hardcoded). If the build variable is absent, the version MUST render as `dev`.

#### Scenario: Version in tagline

- GIVEN the binary is built with a version linker flag set to `1.2.3`
- WHEN the dashboard renders
- THEN the tagline reads `> thoughtline v1.2.3 — game-dev memory that survives`

#### Scenario: Dev build tagline

- GIVEN the binary is built without a version linker flag
- WHEN the dashboard renders
- THEN the tagline reads `> thoughtline vdev — game-dev memory that survives`

---

### Requirement 4: Stat Card

The stat card MUST display exactly 4 numeric cells arranged in a 2×2 grid inside a double-border card. Each cell renders a label and a non-negative integer count. The specific 4 metrics are deferred to design. Counts MUST be sourced from live DB queries, not hardcoded.

#### Scenario: Stat card zero state

- GIVEN the DB contains no memories
- WHEN the stat card renders
- THEN all 4 cells show `0`

#### Scenario: Stat card non-zero state

- GIVEN the DB contains at least 1 memory
- WHEN the stat card renders
- THEN at least one cell shows a value greater than `0`

---

### Requirement 5: Top Projects Section

The Top Projects section MUST display up to 3 project names as a bulleted list. When the total number of distinct projects exceeds 3, a footer line `...and N more` MUST appear immediately below the list, where N is `(total − 3)`. When there are 3 or fewer projects, the footer line MUST NOT appear. The ranking metric (by memory count vs. recent activity) is deferred to design.

#### Scenario: Three or fewer projects

- GIVEN the DB contains exactly 2 distinct projects
- WHEN the dashboard renders
- THEN the Top Projects section shows exactly 2 bullet items and no "...and N more" line

#### Scenario: More than three projects

- GIVEN the DB contains exactly 5 distinct projects
- WHEN the dashboard renders
- THEN the Top Projects section shows exactly 3 bullet items
- AND the footer reads `...and 2 more`

---

### Requirement 6: Action Menu

The action menu MUST contain exactly these 5 items in this order:

| Index | Label |
|-------|-------|
| 1 | `Search memories` |
| 2 | `Recent activity` |
| 3 | `Browse projects` |
| 4 | `Pending events (N)` |
| 5 | `Quit` |

The `▸` cursor MUST appear to the left of the currently active item. N in `Pending events (N)` MUST render as a non-negative integer. When passive capture is OFF or the count is 0, N MUST render as `0`. The active item MUST be visually distinguished from inactive items.

#### Scenario: Default cursor position

- GIVEN the dashboard is freshly entered
- WHEN the action menu renders
- THEN the `▸` cursor is on index 1 (`Search memories`)

#### Scenario: Pending events count zero

- GIVEN passive capture is disabled
- WHEN the action menu renders
- THEN the fourth item reads `Pending events (0)`

#### Scenario: Pending events count non-zero

- GIVEN passive capture is enabled and there are 7 pending events
- WHEN the action menu renders
- THEN the fourth item reads `Pending events (7)`

---

### Requirement 7: Footer Help Line

The dashboard MUST render a footer line at the bottom reading exactly:
`j/k navigate • enter select • s search • q quit`

#### Scenario: Footer presence

- GIVEN the dashboard renders
- WHEN the footer section is inspected
- THEN it contains `j/k navigate • enter select • s search • q quit`

---

### Requirement 8: Single Fixed Theme

The TUI MUST use exactly one theme. There is NO `--theme` flag and NO runtime theme-switching. The exact hex palette is deferred to design but MUST be specified once and applied consistently across all screens.

#### Scenario: No theme flag accepted

- GIVEN a user runs `thoughtline ui --theme classic`
- WHEN the program parses flags
- THEN the program prints an error message and exits with a non-zero exit code
- AND the error message MUST indicate the flag is not recognized

---

### Requirement 9: Navigation Contract (Stack-Based)

The TUI MUST maintain a navigation stack. From the dashboard, `enter` on a menu item pushes the corresponding screen onto the stack. `esc` on any non-dashboard screen pops the top screen and returns to the previous screen. `q` on the dashboard quits. `q` on any non-dashboard screen behaves identically to `esc` (pops the stack, does NOT quit). `ctrl+c` on any screen MUST quit the program immediately.

#### Scenario: Drill into search from dashboard

- GIVEN the dashboard is on top of the stack
- WHEN the cursor is on `Search memories` and `enter` is pressed
- THEN the search screen is pushed onto the stack and becomes the active screen

#### Scenario: Back from child screen with esc

- GIVEN the search screen is on top of the stack (dashboard beneath it)
- WHEN `esc` is pressed
- THEN the search screen is popped and the dashboard is the active screen
- AND the action menu cursor remains on `Search memories`

#### Scenario: q on child screen acts as esc

- GIVEN the search screen is on top of the stack
- WHEN `q` is pressed
- THEN the search screen is popped and the dashboard becomes active (program does NOT quit)

#### Scenario: q on dashboard quits

- GIVEN the dashboard is the only screen on the stack
- WHEN `q` is pressed
- THEN the program exits with exit code 0

#### Scenario: ctrl+c always quits

- GIVEN any screen is active (dashboard or child)
- WHEN `ctrl+c` is pressed
- THEN the program exits immediately

---

### Requirement 10: Dashboard Keybindings

The dashboard MUST recognize these keybindings and no others for menu navigation:

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down (no wrap — stops at last item) |
| `k` / `↑` | Move cursor up (no wrap — stops at first item) |
| `enter` | Drill into selected item |
| `s` | Navigate directly to Search screen regardless of cursor position |
| `r` | Refresh stat card and Top Projects (re-query DB) |
| `q` | Quit |

Cursor navigation MUST NOT wrap (pressing `j` on the last item stays on the last item; pressing `k` on the first item stays on the first item).

#### Scenario: Cursor moves down

- GIVEN the cursor is on `Search memories` (index 1)
- WHEN `j` is pressed
- THEN the cursor moves to `Recent activity` (index 2)

#### Scenario: No wrap at top

- GIVEN the cursor is on `Search memories` (index 1)
- WHEN `k` is pressed
- THEN the cursor remains on `Search memories` (index 1)

#### Scenario: No wrap at bottom

- GIVEN the cursor is on `Quit` (index 5)
- WHEN `j` is pressed
- THEN the cursor remains on `Quit` (index 5)

#### Scenario: s shortcut bypasses cursor

- GIVEN the cursor is on `Browse projects` (index 3)
- WHEN `s` is pressed
- THEN the search screen is pushed and becomes active
- AND the dashboard cursor position is irrelevant (shortcut does not depend on cursor)

---

### Requirement 11: Status Indicator

The header card MUST include a status indicator that reflects DB health and disk usage. The indicator MUST show one of three states:

| State | Condition | Display |
|-------|-----------|---------|
| OK | DB readable AND free disk ≥ 10% | `MEM: OK <free%>` |
| WARN | DB readable AND free disk < 10% | `MEM: WARN` |
| ERR | DB read failed during last refresh | `MEM: ERR` |

The status MUST be refreshed on every entry to the dashboard screen (i.e., on `Init` of the dashboard model or on pop-back from a child screen). It MUST NOT re-query on every keystroke.

#### Scenario: Status OK

- GIVEN the DB is readable and free disk is 25%
- WHEN the dashboard renders
- THEN the header shows `MEM: OK 25%`

#### Scenario: Status WARN

- GIVEN the DB is readable and free disk is 8%
- WHEN the dashboard renders
- THEN the header shows `MEM: WARN`

#### Scenario: Status ERR

- GIVEN the last DB read returned an error
- WHEN the dashboard renders
- THEN the header shows `MEM: ERR`

#### Scenario: Status refreshes on re-entry

- GIVEN the user navigated away from the dashboard to the search screen
- WHEN the user presses `esc` to return to the dashboard
- THEN the status indicator is re-queried (not served from a stale cached value from the previous entry)

---

### Requirement 12: Pending Events Count Refresh

The pending events count (N in `Pending events (N)`) MUST be refreshed on every entry to the dashboard screen. The caching strategy (per-render SQL vs. cached with refresh-on-enter) is deferred to design, but the observable contract is: after returning from a child screen, the count MUST reflect the current DB state.

#### Scenario: Pending count refreshes on re-entry

- GIVEN 3 pending events exist when the dashboard is first shown
- AND a new event is captured while the user is on the search screen
- WHEN the user returns to the dashboard via `esc`
- THEN `Pending events (N)` renders with the updated count

#### Scenario: enter on Pending events when N equals zero

- GIVEN passive capture is off and `Pending events (0)` is the active item
- WHEN `enter` is pressed
- THEN the pending events screen is pushed and becomes active (the item is always selectable)

---

## Open Questions (deferred to design)

These questions are intentionally NOT decided in this spec. Design MUST resolve them before tasks are written.

1. **Final hex palette** — exact color values for the single theme.
2. **ASCII glyph style** — 8-wide block vs. proportional vs. FIGlet font for "THOUGHTLINE".
3. **Top Projects ranking** — by memory count or by most-recent-update activity?
4. **Stat card 4 metrics** — which 4 counts to display (e.g., total memories, projects, tool-use, recent-week activity)?
5. **Pending events count caching strategy** — live SQL on every render vs. cached at entry with `r` refresh.
6. **Deletion commit sequencing** — `cube.go`, `splash.go`, `themes.go` in one commit or separate commits for cleaner bisect.
