# Proposal: TUI Redesign — Flat Screens, Single Theme, Action Menu

## Intent

The current Thoughtline TUI (`internal/dashboard/`) ships with an animated 3D ASCII cube, a splash screen, three switchable themes, and tab-based navigation. For the target audience (junior gamedevs, artists, pipeline TDs running `thoughtline ui` for the first time) this hides the actual value — searching and browsing local memories — behind chrome and animation. We are redesigning the dashboard to a flat stack-of-screens layout inspired by the Engram TUI: hardcoded ASCII logo, a 4-cell stat card, a 5-item action menu, single curated theme. Goal: a first-time user can find and read a memory in under five keystrokes from launch.

## Scope

### In Scope
- Replace tab-based navigation with stack-of-screens (`esc` goes back).
- New dashboard layout (top-to-bottom): double-border header card with `THOUGHTLINE ONLINE` tag, hardcoded ASCII logo with 5-color vertical gradient, italic version tagline, double-border 2x2 stat card, top-projects section, 5-item action menu, footer help.
- 5 action items: `Search memories`, `Recent activity`, `Browse projects`, `Pending events (N)`, `Quit`. The pending-events item always renders; `(N)` shows 0 when passive capture is off.
- Single theme (Rose-Pine-inspired, gamedev-tuned warm/dark/punchy). Final hex codes deferred to design.
- Delete `cube.go`, `splash.go`, multi-theme infrastructure in `themes.go`, and tab plumbing in `model.go` / `update.go` / `view.go`.
- Preserve: `thoughtline ui` entrypoint and its flags, simplified status bar (`MEM OK / N errors`), read-only behavior, dashboard test scaffolding.
- Strict TDD: write screen tests before deleting old code.

### Out of Scope
- LLM features in the TUI.
- Editing memories from the TUI (read-only stays).
- `--pro` / `--legacy` flag retaining the old TUI.
- Web / localhost viewer.
- Renaming the `internal/dashboard/` package.

## Capabilities

### New Capabilities
- `tui-dashboard`: flat stack-of-screens TUI with logo, stat card, action menu, and child screens for search, recent activity, project browser, and pending events.

### Modified Capabilities
- None.

## Approach

Rebuild `internal/dashboard/` around a screen-stack `Model` (`stack []screen`, `Push/Pop`), drop the `tabKey` enum, and replace the multi-theme `Theme` struct with a single `palette` value. Each screen (`dashboardScreen`, `searchScreen`, `recentScreen`, `projectsScreen`, `pendingScreen`) implements `Init/Update/View` and is composed via lipgloss. Logo and header live as `var` literals (hardcoded ASCII + gradient). The 5-item action menu reuses `bubbles/list` with a custom delegate to render the `▸` cursor. Stat-card numbers come from existing storage queries (counts surfaced for `tl_context`). License posture: Engram TUI is MIT (same as Thoughtline) — architectural patterns are free to copy; verbatim code lifts get a per-file attribution header per MIT. This is materially different from claude-mem (AGPL) which we do NOT copy from.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/dashboard/model.go` | Modified | Drop `tabKey`, add screen stack |
| `internal/dashboard/update.go` | Modified | Route input per top-of-stack screen, `esc` pops |
| `internal/dashboard/view.go` | Modified | Render top-of-stack only |
| `internal/dashboard/themes.go` | Modified | Collapse to single palette |
| `internal/dashboard/styles.go` | Modified | Header card, stat card, menu styles |
| `internal/dashboard/cube.go` | Removed | Animated cube deleted |
| `internal/dashboard/splash.go` | Removed | Splash screen deleted |
| `internal/dashboard/items.go` | Modified | Action menu items |
| `internal/dashboard/model_test.go` | Modified | Adapt to screen-stack model |
| `internal/dashboard/tabs_test.go` | Removed | Tabs no longer exist |
| `cmd/thoughtline` | Modified | Drop `--no-splash` / `--theme` flags if present |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| User misses the cube as a brand asset | Low | Document rationale in commit body; logo gradient becomes the new brand mark |
| TUI tests break during redesign | High | Strict TDD: write new screen tests before deleting old; keep `model_test.go` green at every commit |
| Lipgloss render diffs across Windows Terminal / VS Code / cmd.exe | Medium | Cross-platform CI already runs; manual smoke test before archive |
| Stat-card SQL queries slow first paint | Low | Cache counts at startup, refresh on `r`; revisit in design |

## Rollback Plan

Single-feature branch, single squashed merge. To revert: `git revert <merge-sha>`. The old dashboard is fully preserved in git history (cube, splash, themes recoverable verbatim). No DB migration, no config schema change, no on-disk artifact change — rollback is purely code.

## Dependencies

- Existing: `bubbletea` v1.3.10, `bubbles`, `lipgloss` (already in `go.mod`).
- No new third-party deps.

## Success Criteria

- [ ] First-time user can open the TUI, find a memory, and read it in under 5 keystrokes (`s` → type → `enter`).
- [ ] `cube.go` and `splash.go` deleted; `themes.go` reduced to one palette.
- [ ] `go test ./...` green; `go vet ./...` clean; `gofmt` clean.
- [ ] Binary size delta within +/-3% (no runaway dependency growth).
- [ ] All five menu actions reachable with `j/k` + `enter`; `esc` pops back from every child screen.
- [ ] Single-theme palette committed (hex codes locked in design phase).

## Open Questions (deferred to design)

1. Final hex palette for the single theme.
2. ASCII glyphs for "THOUGHTLINE": 8-wide block letters vs proportional.
3. "Top Projects" ranking: top-N by memory count, or by recent activity?
4. Stat-card 4th cell: `tools` or `recent-activity-this-week`?
5. Pending-events `(N)`: live SQL on every render, or cached at startup with manual refresh?
6. Deletion sequencing: single commit removing cube/splash/themes, or per-file commits for cleaner bisect?

## License Hygiene

Engram TUI is MIT — same license as Thoughtline. Architectural ideas (logo + stat card + menu pattern) are free to adopt. Any verbatim code lift requires the standard MIT attribution header in the destination file, which we will add cleanly. This is explicitly different from claude-mem (AGPL) — we do NOT copy code from AGPL sources into this MIT project.
