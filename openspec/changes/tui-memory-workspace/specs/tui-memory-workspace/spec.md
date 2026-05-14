# tui-memory-workspace Specification

> Change: `tui-memory-workspace`
> Status: draft
> Operation: ADDED (new capability — supersedes archived `tui-dashboard` spec from `tui-redesign-superseded`)

## Purpose

Defines the navigation contract, tab layout, keybinding semantics, semantic color palette, and read-only memory viewing rules for the redesigned Thoughtline TUI. The dashboard is a six-tab workspace (Home, Memories, Search, Inbox, Sessions, Help) with global Quick Actions hotkeys, a single semantic palette, full-screen Memory Detail views, and clipboard copy. Saved memories are read-only; pending captures in the Inbox are editable before promotion.

---

## Requirements

### Requirement 1: Tab Navigation Contract

The TUI MUST expose exactly six tabs in this order: **Home (1), Memories (2), Search (3), Inbox (4), Sessions (5), Help (6)**. Tabs MUST be directly reachable by the digit keys `1` through `6`. `tab` MUST cycle to the next tab and `shift+tab` MUST cycle to the previous tab (with wrap from 6→1 and 1→6). The currently active tab MUST be visually distinguished from inactive tabs. Tab state MUST persist across Screen stack push/pop operations: pushing a screen (e.g., Memory Detail) over the active tab and then popping it MUST return the user to the same tab that was active before the push.

#### Scenario: Direct jump by digit

- GIVEN the active tab is Home (tab 1)
- WHEN the user presses `4`
- THEN the active tab becomes Inbox (tab 4)
- AND the Inbox tab is visually marked as active and the previously active tab is no longer marked active

#### Scenario: Tab cycle forward and backward

- GIVEN the active tab is Inbox (tab 4)
- WHEN the user presses `tab`
- THEN the active tab becomes Sessions (tab 5)
- WHEN the user then presses `shift+tab` twice
- THEN the active tab becomes Inbox (tab 4) then Search (tab 3)

#### Scenario: Tab state survives Screen push/pop

- GIVEN the active tab is Memories (tab 2)
- WHEN the user pushes a Memory Detail screen onto the stack
- AND then presses `esc` to pop the screen
- THEN the active tab is still Memories (tab 2) and the Memories list is rendered

---

### Requirement 2: Home Tab Layout

The Home tab MUST render these sections top-to-bottom: (1) a header containing the text brand `🧠 Thoughtline`, the tagline `Local memory for game projects`, the current project name, and the database file path rendered in the muted color token; (2) a Quick Actions row; (3) a Project Health card; (4) a Latest Memories card with an interactive cursor; (5) a Recent Activity card. When the total memory count is zero, the Home tab MUST render an empty-state surface (see Requirement 4) in place of the Latest Memories and Recent Activity cards.

#### Scenario: Populated Home render

- GIVEN the DB contains at least one memory
- WHEN the Home tab is active
- THEN the header (brand + tagline + project + DB path in muted color), Quick Actions row, Project Health card, Latest Memories card, and Recent Activity card are all rendered

#### Scenario: Empty Home render

- GIVEN the DB contains zero memories
- WHEN the Home tab is active
- THEN the header and Quick Actions row still render
- AND the empty state replaces the Latest Memories and Recent Activity cards

---

### Requirement 3: Latest Memories Card Interactivity

The Latest Memories card on the Home tab MUST support a cursor that moves with `j` / `↓` (down) and `k` / `↑` (up) within the card's rows, without wrapping. Pressing `enter` while the cursor is on a row MUST push a Memory Detail screen for that memory onto the Screen stack. Pressing `esc` from the Memory Detail screen MUST pop the screen and return to the Home tab with the cursor at the same row position that was active when the detail was pushed.

#### Scenario: Cursor moves within card

- GIVEN the Home tab is active and the Latest Memories card has at least 3 rows
- WHEN the user presses `j` twice
- THEN the cursor moves down two rows within the card

#### Scenario: Enter opens detail and esc restores cursor

- GIVEN the cursor is on row index 2 of the Latest Memories card
- WHEN the user presses `enter`
- THEN a Memory Detail screen for the row-2 memory is pushed onto the stack
- WHEN the user then presses `esc`
- THEN the Home tab is active and the Latest Memories cursor is on row index 2

---

### Requirement 4: Empty State

When total memory count is zero, the Home tab MUST render an empty-state surface with friendly copy explaining that no memories exist yet. The surface MUST list exactly three example memory types from the gamedev-oriented taxonomy: `scene-pattern`, `perf-gotcha`, and `pipeline-step`. The surface MUST include a one-liner showing both the CLI invocation (`tl save`) and the MCP invocation (`tl_save`).

#### Scenario: Empty state example types rendered

- GIVEN the DB contains zero memories
- WHEN the Home tab is active
- THEN the empty state surface contains the literal strings `scene-pattern`, `perf-gotcha`, and `pipeline-step`

#### Scenario: Empty state save instructions rendered

- GIVEN the DB contains zero memories
- WHEN the Home tab is active
- THEN the empty state contains both the literal string `tl save` (CLI) and the literal string `tl_save` (MCP)

---

### Requirement 5: Memories Tab Layout

The Memories tab MUST render a paginated list of all saved memories. The total memory count MUST be visible somewhere on the tab (header, footer, or status area). Each row MUST render: a type badge, the memory's `topic_key` (or `title` when no `topic_key` is set), a 1–2 line content preview, the scope value (`project` or `personal`), and the `updated_at` timestamp formatted as a relative time. The default sort order MUST be `updated_at` DESC (most recently updated first).

#### Scenario: Memories tab shows count and rows

- GIVEN the DB contains 7 memories
- WHEN the Memories tab is active
- THEN the tab displays the total count `7` somewhere on screen
- AND at least one row is rendered showing the type badge, topic_key or title, preview, scope, and relative updated time

#### Scenario: Default sort order is updated_at DESC

- GIVEN memory A has `updated_at = 1000` and memory B has `updated_at = 2000`
- WHEN the Memories tab is active and no custom sort has been selected
- THEN memory B appears above memory A in the rendered list

---

### Requirement 6: Memories Tab Filters

The Memories tab MUST expose filter controls for all four of these dimensions: `type`, `tag`, `scope`, and `sort` (sort order toggle between `updated` and `recent`/created). The visual UX for filter activation (inline bar, popup, push-screen) is deferred to design — the spec only requires that all four filter dimensions are reachable from the Memories tab.

#### Scenario: All four filter dimensions are reachable

- GIVEN the Memories tab is active
- WHEN the user activates the filter UX
- THEN the user can set or change a value for `type`, `tag`, `scope`, and `sort` (any UI affordance — inline bar, popup, or pushed screen — satisfies this requirement)

#### Scenario: Filter actually filters the list

- GIVEN the DB contains 5 memories of type `decision` and 3 of type `bug`
- WHEN the user applies the filter `type = decision` on the Memories tab
- THEN only the 5 `decision` memories are rendered

---

### Requirement 7: Memories Tab Keybindings

The Memories tab MUST recognize these keybindings when no text input/textarea is focused:

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down (no wrap) |
| `k` / `↑` | Move cursor up (no wrap) |
| `enter` | Push Memory Detail for the cursor row |
| `/` | Jump to the Search tab (tab 3) |
| `f` | Open the filter UX |
| `r` | Refresh the list (re-query DB) |
| `q` | Quit the program |

Cursor navigation MUST NOT wrap.

#### Scenario: Enter pushes detail

- GIVEN the cursor is on row index 0 of the Memories tab
- WHEN the user presses `enter`
- THEN a Memory Detail screen for that memory is pushed onto the stack and becomes active

#### Scenario: Slash jumps to Search

- GIVEN the Memories tab is active
- WHEN the user presses `/`
- THEN the active tab becomes Search (tab 3)

#### Scenario: No wrap at bottom

- GIVEN the cursor is on the last row of the Memories tab
- WHEN the user presses `j`
- THEN the cursor remains on the last row

---

### Requirement 8: Search Tab

The Search tab MUST render a `textinput` component at the top of the tab and a results list below it. Pressing `enter` while the textinput is focused MUST execute the FTS query and populate the results list. Pressing `enter` while the cursor is on a result row MUST push a Memory Detail screen for that result onto the stack.

#### Scenario: Query executes on enter

- GIVEN the Search tab is active with the textinput focused and the user has typed `wal mode`
- WHEN the user presses `enter`
- THEN the FTS query runs and matching results render in the results list

#### Scenario: Enter on result pushes detail

- GIVEN the Search tab has rendered results and the cursor is on result row 0
- WHEN the user presses `enter`
- THEN a Memory Detail screen for that result is pushed onto the stack

---

### Requirement 9: Inbox Tab Layout

The Inbox tab MUST render a list of pending captures (rows from `pending_events` where `status = 'pending'`). Each row MUST render: the source of the capture (e.g., hook event_type or tool_name), the proposed memory type, a content preview, and an optional confidence indicator when available. Each row MUST display three per-row hotkey affordances visible to the user: `[A]` accept, `[E]` edit, `[R]` reject.

#### Scenario: Inbox renders pending rows with affordances

- GIVEN there are 2 pending captures in the DB
- WHEN the Inbox tab is active
- THEN exactly 2 rows are rendered
- AND each row shows the source, proposed type, preview, and the `[A]`/`[E]`/`[R]` hotkey affordances

#### Scenario: Inbox empty state

- GIVEN there are zero pending captures
- WHEN the Inbox tab is active
- THEN the tab renders an empty state surface (no rows) and does not crash

---

### Requirement 10: Inbox Accept Action

Pressing `[A]` while the cursor is on a pending row MUST promote that row by calling `MarkPromoted` with the proposed payload values unchanged. After the call succeeds, the row MUST disappear from the Inbox list. A brief success status MUST be surfaced to the user (status line, toast, or equivalent).

#### Scenario: Accept promotes unchanged

- GIVEN the cursor is on a pending row with proposed type `decision`, title `T`, and content `C`
- WHEN the user presses `A`
- THEN `MarkPromoted` is called and a new memory is created with type `decision`, title `T`, content `C`
- AND the pending row disappears from the Inbox

#### Scenario: Accept surfaces success status

- GIVEN a successful accept has just occurred
- WHEN the Inbox renders the next frame
- THEN a user-visible success status (e.g., `promoted` or equivalent) is briefly displayed

---

### Requirement 11: Inbox Edit Action

Pressing `[E]` on a pending row MUST open an edit form with exactly three editable fields:

| Field | Editor |
|-------|--------|
| `type` | Select-from-valid-types (constrained to the 11-type taxonomy) |
| `title` | Single-line `textinput` |
| `content` | Multi-line Bubbles `textarea` |

While any of these editors are focused, the edit form is considered the active focused input (see Requirement 16). Submitting the form MUST promote the pending row with the edited values via the passive-capture pre-promotion edit path (see `passive-capture` spec delta). Cancelling the form (e.g., via `esc` outside the textarea, or a dedicated cancel key) MUST return the user to the Inbox tab with no changes persisted.

#### Scenario: Edit submit promotes with new values

- GIVEN the user pressed `E` on a pending row, changed `type` from `bug` to `decision`, set `title` to `Switch to WAL`, and edited `content`
- WHEN the user submits the edit form
- THEN a new memory is created with type `decision`, title `Switch to WAL`, and the edited content
- AND the pending row disappears from the Inbox

#### Scenario: Edit cancel leaves Inbox unchanged

- GIVEN the user pressed `E` on a pending row and modified the title
- WHEN the user cancels the edit form
- THEN no memory is created
- AND the pending row is still present in the Inbox with the original proposed values

---

### Requirement 12: Inbox Reject Action

Pressing `[R]` on a pending row MUST mark that row as rejected (no promotion). The row MUST disappear from the Inbox list after the action completes. No memory MUST be created.

#### Scenario: Reject removes row without promoting

- GIVEN the cursor is on a pending row
- WHEN the user presses `R`
- THEN no new memory is created
- AND the row disappears from the Inbox list

#### Scenario: Reject does not affect other rows

- GIVEN the Inbox shows 3 pending rows and the cursor is on row 1
- WHEN the user presses `R`
- THEN row 1 disappears
- AND rows 0 and 2 remain visible and untouched

---

### Requirement 13: Sessions Tab

The Sessions tab MUST render a list of sessions sourced from the existing `RecentSessions` storage query. Each row MUST show enough metadata to identify the session. The Sessions tab MUST NOT expose any mutation actions (no delete, no edit, no merge). Open/closed grouping and session search are deferred to design.

#### Scenario: Sessions render

- GIVEN at least one session exists in the DB
- WHEN the Sessions tab is active
- THEN at least one session row is rendered with identifying metadata

#### Scenario: Sessions tab is read-only

- GIVEN the Sessions tab is active and at least one row is rendered
- WHEN any key other than navigation/tab-switch/quit is pressed
- THEN no session row is modified, deleted, or merged

---

### Requirement 14: Help Tab

The Help tab MUST render a keybindings reference covering at minimum: the digit tab-switch keys (`1`–`6`), `tab`/`shift+tab`, the Quick Actions hotkeys (`S`, `/`, `M`, `I`), the navigation keys (`j`, `k`, `↑`, `↓`, `enter`, `esc`), `r` refresh, `q` quit, and `ctrl+c` quit. The Help tab MUST also render the roadmap content sourced from `roadmap.yaml`.

#### Scenario: Help shows keybindings

- GIVEN the Help tab is active
- WHEN the user reads the tab
- THEN the rendered content includes references to digit-key tab switching, Quick Actions hotkeys, and `q`/`ctrl+c` quit semantics

#### Scenario: Help renders roadmap from YAML

- GIVEN `roadmap.yaml` contains at least one roadmap entry
- WHEN the Help tab is active
- THEN that roadmap entry's title or label appears in the rendered Help tab

---

### Requirement 15: Global Quick Actions Hotkeys

The hotkeys `[S]`, `[/]`, `[M]`, `[I]` MUST be active on every tab when no `textinput` or `textarea` is focused. Their actions:

| Key | Action |
|-----|--------|
| `S` | Trigger the save flow (stub permitted for v1: MUST NOT crash and MUST surface some user-visible response such as a status message pointing to `tl save` / `tl_save`) |
| `/` | Jump to the Search tab (tab 3) AND focus the search textinput |
| `M` | Jump to the Memories tab (tab 2) |
| `I` | Jump to the Inbox tab (tab 4) |

#### Scenario: M jumps to Memories from any tab

- GIVEN the active tab is Sessions (tab 5) and no input is focused
- WHEN the user presses `m`
- THEN the active tab becomes Memories (tab 2)

#### Scenario: Slash jumps and focuses search

- GIVEN the active tab is Home and no input is focused
- WHEN the user presses `/`
- THEN the active tab becomes Search (tab 3)
- AND the search textinput is focused

#### Scenario: S surfaces a response without crashing

- GIVEN the active tab is Home and no input is focused
- WHEN the user presses `s`
- THEN the program does not crash
- AND a user-visible response (status message, prompt, or open screen) appears

---

### Requirement 16: Input-Focus Suppression

When any `textinput` or `textarea` is focused (e.g., the Search tab's input, the Inbox edit form's title or content fields), the global Quick Actions hotkeys (`S`, `/`, `M`, `I`) MUST be suppressed. Pressing those keys while an input is focused MUST insert the literal character into the input, NOT trigger a tab switch or save flow.

#### Scenario: Typing m in search input is literal

- GIVEN the Search tab is active and the search textinput is focused
- WHEN the user presses `m`
- THEN the literal character `m` is appended to the textinput's value
- AND the active tab remains Search (no tab switch occurs)

#### Scenario: Typing s in inbox edit content is literal

- GIVEN the Inbox edit form is open and the content `textarea` is focused
- WHEN the user presses `s`
- THEN the literal character `s` is appended to the textarea's value
- AND no save flow is triggered

---

### Requirement 17: Memory Detail Full-Screen View

The Memory Detail view MUST be implemented as a Screen pushed onto the existing Screen stack (full-screen push, not a true semi-transparent overlay). It MUST render: the full memory content, and the metadata fields `type`, `scope`, `project`, `topic_key`, `sync_id`, `tags`, `created_at`, `updated_at`. Pressing `esc` while the Memory Detail is the top of the stack MUST pop the screen and return the user to the tab that was active when the push occurred.

#### Scenario: Detail renders required metadata

- GIVEN a Memory Detail is pushed for a memory with non-empty `type`, `scope`, `project`, `topic_key`, `sync_id`, `tags`, `created_at`, and `updated_at`
- WHEN the Detail renders
- THEN all eight of those metadata fields are visible in the rendered output

#### Scenario: Esc returns to originating tab

- GIVEN the active tab was Memories (tab 2) when the user pushed a Memory Detail
- WHEN the user presses `esc` on the Memory Detail
- THEN the Memory Detail is popped
- AND the active tab is Memories (tab 2)

---

### Requirement 18: Read-Only Contract for Saved Memories

No `[E]` (edit) or `[D]` (delete) hotkeys MUST be registered on any non-Inbox screen, including Memory Detail, Memories tab, Home tab's Latest Memories card, Search tab results, and Sessions tab. The Memory Detail view MUST support only `[C]` (copy) and `esc` (back) as memory-action hotkeys. Pending captures in the Inbox tab are drafts (not yet promoted memories) and remain editable per Requirements 11 — this does NOT violate the read-only contract.

#### Scenario: No edit hotkey on Memory Detail

- GIVEN a Memory Detail is the active screen
- WHEN the user presses `e`
- THEN no edit flow is opened
- AND the memory is not modified

#### Scenario: No delete hotkey on Memories tab

- GIVEN the Memories tab is active with the cursor on a row
- WHEN the user presses `d`
- THEN no delete flow is opened
- AND the memory row is not removed from the DB

#### Scenario: Source-level absence of edit/delete bindings

- GIVEN the dashboard source code is scanned
- WHEN one searches for key bindings registered on non-Inbox screens
- THEN no key binding for `e` or `d` is found that mutates or attempts to mutate a saved memory

---

### Requirement 19: Clipboard Copy

The `[C]` hotkey on the Memory Detail screen MUST copy the memory's content to the OS clipboard. The implementation MUST be cross-platform via subprocess: Windows uses `clip.exe`, macOS uses `pbcopy`, Linux tries `wl-copy` first and falls back to `xclip -selection clipboard`. If all available clipboard backends fail or are unavailable, the failure MUST be surfaced as an in-app status message (e.g., `clipboard unavailable`) and MUST NOT crash the program.

#### Scenario: Successful copy on supported platform

- GIVEN the Memory Detail is active and a working clipboard backend is available
- WHEN the user presses `c`
- THEN the memory content is sent to the clipboard backend
- AND a success status is briefly surfaced

#### Scenario: Failure surfaces, does not crash

- GIVEN the Memory Detail is active and all clipboard backends fail
- WHEN the user presses `c`
- THEN the program does not crash
- AND a user-visible status message indicating clipboard unavailability is rendered

---

### Requirement 20: Semantic Palette

The TUI MUST define exactly six color tokens in the palette: `nav` (cyan/blue family), `brand` (purple family), `success` (green family), `warning` (yellow family), `destructive` (red family), and `muted` (gray family). Each token MUST be used only for its semantic purpose. The exact hex codes are deferred to design.

| Token | Semantic role |
|-------|---------------|
| `nav` | Primary navigation, active tab marker, focused element |
| `brand` | Thoughtline brand logo, memory-type accents, headings |
| `success` | Successful operations (promoted, saved, OK status) |
| `warning` | Pending state, degraded subsystem, low disk |
| `destructive` | Rejected, failed, error state |
| `muted` | Technical metadata (sync_id, paths, timestamps, DB path) |

#### Scenario: Palette has exactly six tokens

- GIVEN the dashboard's palette type is inspected
- WHEN the field count is measured
- THEN exactly 6 color tokens are declared (`nav`, `brand`, `success`, `warning`, `destructive`, `muted`)

#### Scenario: Success color is not used for navigation

- GIVEN the dashboard source is scanned
- WHEN one inspects where the `success` token is applied
- THEN it is not used to style the active tab marker, focused element, or any non-success indicator

---

### Requirement 21: Brand Logo

The brand surface MUST render the text `🧠 Thoughtline` followed by the tagline `Local memory for game projects`. No ASCII block-letter art MUST be rendered. No gradient MUST be applied to the brand text. The current project name and the database file path MUST render below the brand in the `muted` color token.

#### Scenario: Brand text renders without ASCII art

- GIVEN the Home tab is active
- WHEN the brand surface renders
- THEN the rendered output contains the literal string `🧠 Thoughtline` and the tagline `Local memory for game projects`
- AND no multi-line block-letter rendering of "THOUGHTLINE" is present

#### Scenario: Project and DB path use muted color

- GIVEN the Home tab is active and a project name and DB path are known
- WHEN the header renders
- THEN the project name and DB path are styled with the `muted` color token

---

### Requirement 22: Minimum Viewport

When the terminal viewport is smaller than 80 columns wide or 24 rows tall, the TUI MUST render a single-line warning message indicating that the terminal is too small. While the warning is rendered, all input MUST be frozen except `q` and `ctrl+c`, which still quit the program.

#### Scenario: Under-min viewport renders warning

- GIVEN the terminal is resized to 70×20
- WHEN the TUI renders the next frame
- THEN a single-line warning indicating the terminal is too small is displayed

#### Scenario: q still quits under min viewport

- GIVEN the warning is being rendered due to a 70×20 terminal
- WHEN the user presses `q`
- THEN the program exits

#### Scenario: Other keys are ignored under min viewport

- GIVEN the warning is being rendered due to a 70×20 terminal
- WHEN the user presses `1`, `/`, `m`, or any non-quit key
- THEN no tab switch occurs and no other action is taken

---

### Requirement 23: Quit Semantics

`q` pressed on any tab when no `textinput` or `textarea` is focused MUST quit the program. `q` pressed while an input/textarea IS focused MUST insert the literal character `q` into the input. `ctrl+c` MUST always quit the program regardless of focus state.

#### Scenario: q on Memories tab quits

- GIVEN the Memories tab is active and no input is focused
- WHEN the user presses `q`
- THEN the program exits

#### Scenario: q in search input is literal

- GIVEN the Search tab is active and the search textinput is focused
- WHEN the user presses `q`
- THEN the literal `q` is appended to the input
- AND the program does NOT exit

#### Scenario: ctrl+c quits even with input focused

- GIVEN the Inbox edit content textarea is focused
- WHEN the user presses `ctrl+c`
- THEN the program exits

---

### Requirement 24: Project Health Card — Consistent Scoping

The Project Health card on the Home tab MUST scope its `Memories`, `Sessions`, and `Pending` counts to the SAME data set rendered by the Latest Memories card, the Recent Activity card, and the Top Projects section on the same tab. Specifically: when those sections render data from "all projects" (the default when no explicit project filter is active), the Project Health counts MUST also be "all projects" totals — NOT scoped to the working-directory basename project. This requirement supersedes any earlier implicit scoping inherited from the legacy `dashboard.Config.Project` default.

Background bug observed in v0.1.0: the legacy dashboard showed `0 memories / 0 sessions / 0 projects` in the stat card while Top Projects listed 3+ real projects, because the stat card was scoped to `cwd-basename` and the user's working directory did not match any saved memory's project name.

#### Scenario: Stat card matches Top Projects when DB has 27 memories across 4 projects

- GIVEN the DB contains 27 active memories spread across 4 projects (`thoughtline`, `SuckItUp`, `hole-game`, `<other>`) and 0 of those memories have `project = "Símbolo del sistema"` (or whatever the cwd basename happens to be)
- WHEN the Home tab renders the Project Health card AND the Top Projects / Latest Memories sections on the same frame
- THEN `Memories: 27` is shown in the Project Health card (the total, not 0)
- AND the Top Projects section lists at least 3 project names
- AND the two figures are reconcilable (no row shows "0" while another row shows non-empty data sourced from the same DB)

#### Scenario: Explicit project filter scopes ALL sections consistently

- GIVEN the user has applied a project filter (e.g., via a future filter UX or via the `THOUGHTLINE_PROJECT` env var)
- WHEN the Home tab renders
- THEN every section (Project Health counts, Latest Memories, Recent Activity, Top Projects) reflects the SAME project filter
- AND no section silently ignores the filter while another section honors it

---

### Requirement 25: Memory Detail — Wrap and Scroll

The Memory Detail view MUST render long content lines with soft wrapping at the viewport width (no horizontal overflow, no horizontal scrollbar). The view MUST support vertical scrolling via at minimum `↑` / `↓` (one line) and `PgUp` / `PgDn` (one half-viewport). When the content exceeds the visible viewport height, a scroll indicator (e.g., `12 / 47 lines` or `27%`) MUST render in the Detail screen's footer so the user knows there is more content below the fold.

Background bug observed in v0.1.0: the legacy Detail view rendered long markdown content without wrapping, causing text to overflow the right edge of the terminal AND offered no visible way to scroll down, so users could only see the first viewport-height worth of content with the rest unreachable.

#### Scenario: Long line wraps at viewport width

- GIVEN the Memory Detail is active for a memory whose content contains a line longer than the viewport width (e.g., a 200-character paragraph in an 80-column terminal)
- WHEN the Detail renders
- THEN the line wraps at the viewport's right edge (or at word boundaries within it)
- AND no character of the content is hidden off the right side
- AND no horizontal scrollbar or horizontal-overflow indicator is needed

#### Scenario: Long content scrolls vertically

- GIVEN the Memory Detail is active for a memory whose rendered (wrapped) content exceeds the available Detail viewport height by at least 10 lines
- WHEN the user presses `↓` (or `j`)
- THEN the viewport scrolls down by one rendered line
- WHEN the user presses `PgDn`
- THEN the viewport scrolls down by approximately one half-viewport height
- WHEN the user has scrolled past the first viewport
- THEN the footer scroll indicator reflects the current position (e.g., `25%` or `Line 12 of 47`)

#### Scenario: Short content does not show scroll indicator

- GIVEN the Memory Detail is active for a memory whose content fits entirely in the viewport
- WHEN the Detail renders
- THEN no scroll indicator is shown (or it shows `100%` / `End`)
- AND `↓` and `PgDn` are no-ops

---

## Open Questions (deferred to design)

These items are intentionally NOT decided in this spec. Design MUST resolve them before tasks are written.

1. **Pagination size for the Memories tab** — 20 per page, 50, infinite/virtual scroll, page indicator format.
2. **Memories tab filter UX** — inline always-visible bar vs popup triggered by `f` vs pushed screen.
3. **Scroll position preservation across Memories → Detail → back** — same row vs reset to top.
4. **Home live-refresh interval** — 30s auto-refresh vs `r`-only vs storage-change-driven tick.
5. **Memories tab filter persistence** — persist across tab switches and across `thoughtline ui` sessions vs reset.
6. **Sessions tab open/closed grouping and search field** — restyle-only vs add grouping/search.
7. **Cross-platform rounded borders fallback** — explicit ASCII fallback for non-VT terminals vs best-effort.
8. **`--theme` / `--no-splash` / `--splash-ms` removal migration UX** — friendly error message vs silent "unknown flag" rejection.
9. **Inbox `[E]` edit confirmation step** — preview/confirm screen before promote vs immediate submit.
10. **Exact hex codes for the semantic palette tokens.**
11. **Project Health scoping policy** (Req 24) — locked to "all projects" default per Req 24 wording, but design must spell out how this interacts with a future per-project filter UX.
12. **Detail scroll indicator format** (Req 25) — `27%` vs `Line 12 of 47` vs both. Design picks one.
