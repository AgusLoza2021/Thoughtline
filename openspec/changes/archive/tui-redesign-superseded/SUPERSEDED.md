# SUPERSEDED — 2026-05-14

This change was archived **before implementation completed** because the product
direction shifted.

## Why archived

The user changed their mind about the TUI design. The original `tui-redesign`
proposed a flat stack-of-screens layout inspired by the Engram TUI: hardcoded
ASCII block-letter logo with 5-color gradient, single Rose-Pine-Moon palette,
5-item action menu, drill-in screens with `esc` pop.

The new direction (change `tui-memory-workspace`) replaces that with:

- **Tabbed navigation** (1 Home / 2 Memories / 3 Search / 4 Inbox / 5 Sessions / 6 Help) — NOT stack-of-screens.
- **Memory Workspace** framing — Home shows Latest Memories card, Project Health card, Quick Actions, Recent Activity.
- **Friendly empty state** with 3 gamedev examples.
- **Inbox tab** (replaces Pending) with accept/reject actions.
- **New palette**: cyan/blue navigation, purple brand, green success, yellow warn, red error, gray meta — NOT Rose-Pine-Moon.
- **Simpler header**: text `🧠 Thoughtline` instead of ASCII block-letter logo.
- **Read-only total** confirmed — no edit/delete from TUI.

## Status of partial implementation

Some files were already created in `internal/dashboard/` toward this old
direction (e.g. `dashboard_screen.go`, `search_screen.go`, `recent_screen.go`,
`pending_screen.go`, `screen.go`, single-palette `theme.go`, `logo.go`). They
remain in the repo and will be evaluated by the new change's exploration phase
— some pieces (Screen interface, palette tokens, storage helpers like
`CountPending`) may be salvageable; the block-letter logo and stack-only
navigation are explicitly out.

## How to recover ideas

Everything in this folder is preserved verbatim. The new change SHOULD reference:

- `proposal.md` — for the "junior gamedev first-time experience" framing
- `design.md` — for the screen-interface contract pattern (still useful even without stack-only nav)
- `tasks.md` — for the Strict-TDD task breakdown shape
- `specs/tui-dashboard/spec.md` — for the layout contract (sections, status indicator)

## Successor

See `openspec/changes/tui-memory-workspace/` (created right after this archive).
