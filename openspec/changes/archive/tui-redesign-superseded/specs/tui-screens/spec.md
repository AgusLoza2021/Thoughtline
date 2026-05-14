# tui-screens Specification

> Change: `tui-redesign`
> Status: draft
> Operation: ADDED (new capability — no prior spec exists)

## Purpose

Defines the behavioral contracts for the four non-dashboard drill-in screens reachable from the dashboard action menu: Search, Recent Activity, Browse Projects, and Pending Events.

---

## Requirements

### Requirement 1: Search Screen

The search screen MUST provide a text input field and a result list. Submitting a non-empty query via `enter` MUST execute a full-text search against the local DB using the same query engine as `tl_search`. An empty query MUST be a no-op (no search is executed, no "no results" state is shown). `esc` MUST return to the dashboard. The screen MUST NOT allow editing or deleting memories.

#### Scenario: Submit non-empty query

- GIVEN the search screen is active with an empty input
- WHEN the user types `spawning bug` and presses `enter`
- THEN a DB search is executed with the query `spawning bug`
- AND matching memories are displayed in the result list below the input

#### Scenario: Empty query is no-op

- GIVEN the search screen is active with an empty input field
- WHEN the user presses `enter`
- THEN no search is executed and the result list does not show a "no results" message

#### Scenario: esc returns to dashboard

- GIVEN the search screen is active
- WHEN `esc` is pressed
- THEN the search screen is popped and the dashboard is displayed
- AND the dashboard cursor is on `Search memories`

#### Scenario: q acts as esc on search screen

- GIVEN the search screen is active
- WHEN `q` is pressed
- THEN the search screen is popped and the dashboard is displayed (program does NOT quit)

---

### Requirement 2: Recent Activity Screen

The recent activity screen MUST display the top-N most recently updated memories across all projects, ordered by `updated_at DESC`. The default N MUST be 20. `j`/`k` MUST scroll through the list. `enter` on a memory item MUST open a detail view for that memory. `esc` MUST return to the dashboard.

#### Scenario: List ordered by recency

- GIVEN the DB contains memories with different `updated_at` values
- WHEN the recent activity screen renders
- THEN the memory with the most recent `updated_at` MUST appear at the top of the list

#### Scenario: Default count cap

- GIVEN the DB contains more than 20 memories
- WHEN the recent activity screen renders
- THEN at most 20 items are shown in the list

#### Scenario: j/k scroll

- GIVEN the recent activity screen has at least 2 items
- WHEN `j` is pressed
- THEN the highlight moves to the next item in the list

#### Scenario: enter opens detail

- GIVEN the recent activity screen is active with item 1 highlighted
- WHEN `enter` is pressed
- THEN a detail view for that memory is pushed onto the stack

#### Scenario: esc returns to dashboard

- GIVEN the recent activity screen is active
- WHEN `esc` is pressed
- THEN the screen is popped and the dashboard is displayed

---

### Requirement 3: Browse Projects Screen

The browse projects screen MUST display the list of all distinct project names present in the DB, ordered alphabetically. `enter` on a project name MUST drill into a "memories in project X" view, which is the recent activity screen filtered to that project (same ordering: `updated_at DESC`, same N cap). `esc` MUST return to the dashboard.

#### Scenario: Alphabetical order

- GIVEN the DB contains projects `zombie-game`, `alpha-studio`, `pipeline-tools`
- WHEN the browse projects screen renders
- THEN the items appear in order: `alpha-studio`, `pipeline-tools`, `zombie-game`

#### Scenario: enter drills into project

- GIVEN `alpha-studio` is highlighted in the browse projects screen
- WHEN `enter` is pressed
- THEN a filtered recent activity screen for project `alpha-studio` is pushed
- AND only memories belonging to `alpha-studio` are listed

#### Scenario: esc returns to dashboard

- GIVEN the browse projects screen is active
- WHEN `esc` is pressed
- THEN the screen is popped and the dashboard is displayed

---

### Requirement 4: Pending Events Screen

The pending events screen MUST list pending events for the current default project, ordered by `captured_at DESC`. Each row MUST display at minimum: `event_type`, `captured_at` (human-readable timestamp), hash prefix (first 8 characters of `event_hash`), and payload preview (first 60 characters of `payload`). `enter` on an event MUST push a detail screen showing the full payload. `esc` MUST return to the dashboard.

When there are no pending events, the screen MUST display the message:
`No pending events. Set THOUGHTLINE_PASSIVE_CAPTURE=1 to enable capture, or run `thoughtline worker` to clean up.`

#### Scenario: Pending events list renders

- GIVEN the current project has 3 pending events
- WHEN the pending events screen renders
- THEN exactly 3 rows are shown, each containing event_type, timestamp, hash prefix, and payload preview

#### Scenario: Ordered by captured_at DESC

- GIVEN pending events with different `captured_at` timestamps
- WHEN the screen renders
- THEN the most recently captured event appears at the top

#### Scenario: Empty state message

- GIVEN the current project has zero pending events
- WHEN the pending events screen renders
- THEN the empty state message is displayed

#### Scenario: enter opens full payload

- GIVEN a pending event row is highlighted
- WHEN `enter` is pressed
- THEN a detail screen with the full payload is pushed onto the stack

#### Scenario: esc returns to dashboard

- GIVEN the pending events screen is active
- WHEN `esc` is pressed
- THEN the screen is popped and the dashboard is displayed
