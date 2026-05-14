# ADR 0006 — TUI Memory Workspace

**Status**: Accepted
**Date**: 2026-05-14

---

## Context

The v0.1.0 `thoughtline ui` TUI (delivered by the `tui-redesign` change) shipped
a workstation-style layout: a three-pane sidebar / center / right-detail view
with vim-style hjkl navigation and a screen-stack for drill-ins. Once it landed
in front of junior users we observed three classes of friction:

- **Discoverability**. Memories were buried two clicks deep behind a sidebar
  "Recent" section. New users with `tl_capture` events sitting in the pending
  queue had no idea the inbox existed.
- **Color drift**. Three themes (`brand`, `zbrush`, `mono`) used distinct hex
  values for the same semantic role (success/warn/error), so screenshots in
  docs and bug reports could not agree on what "the green color" meant. Color
  was decoration rather than information.
- **Concrete v0.1.0 bugs**. The Project Health card scoped its counts to the
  cwd basename so a user running the binary from `/tmp` saw 0 memories. The
  Memory Detail view rendered untruncated content past the viewport with no
  scroll. There was no way to copy a memory body to the clipboard. The Inbox
  could not edit a draft before promotion.

`tui-redesign` was archived as superseded after this change started. The
predecessor remains in `openspec/changes/archive/tui-redesign-superseded/` for
forensic reference.

---

## Decision

### D1 — Six-tab flat workspace

Replace the screen-stack root with a six-tab layer (`Home`, `Memories`,
`Search`, `Inbox`, `Sessions`, `Help`) reachable by digit keys `1`–`6` and
cycled by `tab` / `shift+tab`. Tabs and screen-stack coexist: tabs are the
default surface, the stack is used for drill-ins like `Memory Detail` and the
`InboxEditScreen`. Pressing `esc` pops the stack and returns the user to the
originating tab (via the `originator` interface).

### D2 — Single semantic palette

Collapse the three themes into one. Color encodes role:

| Role | Hex | Usage |
|------|-----|-------|
| Navigation | `#7DCFFF` cyan / `#7AA2F7` blue | tabs, cursors |
| Brand | `#C4A7E7` purple | app mark, memory-type badges |
| Success | `#9ECE6A` green | open sessions, "saved" pills |
| Warning | `#E0AF68` yellow | pending counts, in-progress glyphs |
| Error | `#F7768E` red | rejected, error messages |
| Meta | `#7A7A85` gray | timestamps, descriptions |

The Brand `#C4A7E7` is carried over from the Rose-Pine-Moon palette of the
predecessor at the user's explicit request. The `tui-removed` delta originally
listed this hex as forbidden; the carry-over is the documented exception (see
the spec's note about the Brand color).

### D3 — Inbox-mediated edit and clipboard

Memory rows are read-only once saved. The only writeable surface is the
Inbox: `[A]` promotes as-is, `[E]` pushes an `InboxEditScreen` pre-filled
with the pending payload (the user edits, then promotes), `[R]` calls a new
`storage.MarkRejected(ctx, id)` method. `[C]` from the Detail view copies the
body to the system clipboard via platform-specific build-tag files
(`clip.exe` on Windows, `pbcopy` on Darwin, `wl-copy` / `xclip` on Linux).

### D4 — Strict TDD execution

The plan is decomposed into 99 tasks across 15 commits. Every code task
(`X2`, `X4`, …) is preceded by its test task (`X1`, `X3`, …) in the same
commit. The drift test in `keybindings_test.go` guards `Keybindings`
documentation against the wiring in `flat_model.go`'s Update ladder so the
Help tab cannot lie about hotkeys.

### D5 — CLI flag removal with friendly migration error

`--theme`, `--no-splash`, and `--splash-ms` are removed. Invoking any of them
prints

```
thoughtline ui: --<flag> was removed in v0.2 (single semantic palette). See CHANGELOG.md.
```

to stderr and exits with code 2. False-positive guards in
`cmd/thoughtline/main_test.go` confirm `--no-splashy` does NOT match
`--no-splash` and `--themed` does NOT match `--theme`.

---

## Consequences

### Positive

- Junior users grasp the surface in under sixty seconds (six visible labels,
  no nested navigation).
- Semantic colors are documentable and testable — a screenshot's meaning is
  fully recoverable from the palette table.
- Clipboard available cross-platform with no third-party dependency.
- Schema v5 adds `'rejected'` to the `pending_events` status CHECK constraint
  so the Inbox `[R]` action is a first-class state transition.
- Ten golden-file tests at 100×30 catch unintentional layout drift on every
  CI run.

### Negative

- BREAKING removal of `--theme`, `--no-splash`, `--splash-ms`. Users with
  shell aliases or scripts pinned to those flags get a one-line migration
  error and exit code 2 the first time they upgrade.
- Schema v5 migration runs on next start. The migration is additive (only
  loosens a CHECK constraint) so it is irreversible-on-rollback but
  data-preserving.

### Carry-over and exceptions

- The Brand color `#C4A7E7` is preserved from the predecessor Rose-Pine-Moon
  palette. The `tui-removed` delta lists this hex as forbidden in general but
  the Brand role is the explicit exception, documented here and in the spec.

---

## References

- `openspec/changes/tui-memory-workspace/proposal.md`
- `openspec/changes/tui-memory-workspace/design.md`
- `openspec/changes/tui-memory-workspace/specs/tui-memory-workspace/spec.md`
- `openspec/changes/archive/tui-redesign-superseded/` (predecessor)
- ADR 0003 (state dir) — referenced for the migration error UX precedent
