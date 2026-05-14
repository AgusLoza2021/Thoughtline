# Proposal: tui-memory-workspace — Tabbed Memory Workspace, Read-Only, Semantic Palette

> Change: `tui-memory-workspace`
> Phase: propose
> Supersedes: `tui-redesign` (archived as `tui-redesign-superseded`)

## Intent

The current Thoughtline TUI confuses first-time users — especially junior game-devs who just ran `thoughtline ui` for the first time. The block-letter ASCII logo dominates the viewport, an animated cube competes for attention, a splash screen blocks input for hundreds of milliseconds, and a 3-theme switcher (Brand/ZBrush/Mono) exposes implementation detail the user did not ask for. Navigation is a 5-item menu that hides the most useful surfaces (memories, search, inbox) two screens deep. There is no clear "you are here," no global way back to the most-used actions, and the Rose-Pine-Moon palette uses color decoratively rather than to encode meaning.

We are rebuilding the dashboard around a flat, tab-based workspace: six numbered tabs (Home, Memories, Search, Inbox, Sessions, Help), reachable from anywhere with a single keystroke, plus a globally-available Quick Actions row (`[S]ave · [/]earch · [M]emories · [I]nbox`) that works on every tab when no text input is focused. Memory content is **read-only** in the TUI — the dashboard is a viewer, not an editor. Color is reassigned semantically: cyan/blue for navigation, purple for brand and memory-type accents, green/yellow/red exclusively for success/warning/destructive, gray for technical metadata. The brand surface becomes a simple `🧠 Thoughtline` text logo with a one-line tagline.

This proposal supersedes the in-flight `tui-redesign` change. That predecessor produced a useful exploration of the codebase and several reusable Screen components (`SearchScreen`, `DetailScreen`, `PendingScreen`), but its 3-pane LazyGit-style `WorkstationScreen` and stack-of-screens navigation model are not the direction the user wants. We salvage the screen interface, the storage queries, and several formatting helpers; we delete the legacy `Model`/`view.go`/`update.go` trio, the multi-theme infrastructure, the ASCII cube, the splash screen, and the block-letter logo. The result is a TUI a junior user can master in 60 seconds.

## Scope

### In Scope

- **Six-tab workspace** wired into `flatModel`:
  - Tab 1 **Home** — brand header, Quick Actions row, Project Health card, Latest Memories card (interactive cursor), Recent Activity card, friendly empty state when no memories exist.
  - Tab 2 **Memories** — paginated list of ALL saved memories with total count; rows render type badge, topic_key or title, 1–2 line preview, scope, updated time; filters for type/tag/scope and sort by updated/recent.
  - Tab 3 **Search** — restyled FTS search (textinput + results list).
  - Tab 4 **Inbox** — pending captures with `[A]` accept (promote as-is), `[E]` edit type+title+content body then accept, `[R]` reject.
  - Tab 5 **Sessions** — restyled session list (structural changes such as open/closed grouping deferred to design).
  - Tab 6 **Help** — keybindings reference + Roadmap (moved off Home).
- **Tab navigation** at `flatModel` level: keys `1`-`6` jump directly, `tab` / `shift+tab` cycle. Tab state coexists with the existing Screen stack: pushing a screen (Memory Detail) overlays the active tab; `esc` pops the screen and returns to the same tab.
- **Global Quick Actions hotkeys**: `[S]` save (open inbox-save flow or no-op stub for now), `[/]` search (jump to tab 3 + focus input), `[M]` memories (jump to tab 2), `[I]` inbox (jump to tab 4). Active on every tab. **Suppressed when any `textinput` or `textarea` is focused** (search bar, inbox edit form) — typing those letters in an input must produce literal characters.
- **Memory Detail = full-screen stack push** (not a true overlay/modal). Reachable from Home's Latest Memories card and from the Memories tab via `enter`. `esc` returns to the originating tab. Detail view is read-only: shows full content, metadata (type, scope, project, tags, topic_key, sync_id, created/updated), and offers `[C]` copy.
- **Read-only contract for saved memories**: no `[E]` edit, no `[D]` delete hotkeys anywhere in the TUI. Saved memories cannot be mutated from the dashboard. Pending captures in the Inbox are drafts (not yet promoted) and remain editable — this does not violate the contract.
- **Inbox `[E]` edit-before-accept**: editable fields are **type, title, and content body**. Body editor uses the Bubbles `textarea` component. On submit, the edited values are promoted and the pending row is marked promoted. Confirmation/preview step deferred to design.
- **`[C]` copy to clipboard** from Memory Detail. Cross-platform via build tags:
  - Windows: `clip.exe` subprocess (no third-party lib).
  - macOS: `pbcopy` subprocess.
  - Linux: try `wl-copy`, fall back to `xclip -selection clipboard`, fall back to in-app status message "clipboard unavailable."
- **New semantic palette** replaces Rose-Pine-Moon:
  - cyan/blue → primary navigation, active tab, focused element.
  - purple → Thoughtline brand, memory-type accents, headings.
  - green → success only (saved, promoted, OK status).
  - yellow → warning only (pending, FTS degraded, disk low).
  - red → destructive/error only (rejected, failed, error states).
  - gray → muted technical metadata (sync_id, paths, timestamps).
  - Exact hex codes deferred to design.
- **Text brand logo**: `🧠 Thoughtline` + tagline `Local memory for game projects`. No ASCII block letters. No gradient. Replaces `logo.go`.
- **Empty state** on Home when memory count = 0: friendly copy + three gamedev examples (`scene-pattern`, `perf-gotcha`, `pipeline-step`) + one-liner showing how to save the first memory via CLI (`tl save ...`) or MCP (`tl_save` from Claude).
- **Visual style**: Bubbletea terminal-first. Rounded borders preferred (with fallback to ASCII borders deferred to design). Clearer hierarchy, warmer wording, consistent card titles and spacing.
- **Legacy code cleanup** (sequenced as the first batch of apply work):
  - Decouple `styles.go` from `themes.go` (rebuild lipgloss vars against single `palette`).
  - Delete `themes.go` (multi-theme infrastructure).
  - Delete legacy `model.go` / `view.go` / `update.go` (legacy tab-based `Model`).
  - Delete `cube.go`, `splash.go`, `logo.go` (block-letter art).
  - Delete `workstation_screen.go` AFTER extracting `relTime`, `truncateLeft`, `wrap`, `paneBox`, `clampCursor`, `padRight`, `centerString`, `formatLastSave` to new `helpers.go`.
  - Delete `tabs_test.go` (tests dead legacy `tabKey`).
  - Split `model_test.go`: keep roadmap/status tests, delete legacy `Model` tests.
  - Rewrite `logo_test.go` for the new text brand.
- **CLI flag cleanup** in `cmd/thoughtline/main.go`:
  - Remove `--theme`, `--no-splash`, `--splash-ms`.
  - Remove `dashboard.Config` fields `ThemeName`, `Splash`, `SplashDuration`.
  - Keep `--no-update-check`.
  - Migration UX (error message vs silent removal) deferred to design.

### Out of Scope

- Editing or deleting saved memories from the TUI (read-only contract).
- True overlay/modal rendering (semi-transparent on top of underlying tab) — Memory Detail uses full-screen push.
- Web viewer or HTTP/REST dashboard.
- New memory types or schema changes.
- Search ranking/algorithm changes (Search tab is restyle only).
- Auto-update or self-update mechanism changes.
- Migrating the `Save` Quick Action to a full save form — `[S]` will land as a focused stub that points the user to CLI/MCP save until a follow-up change builds the in-TUI save flow.
- Telemetry, analytics, or remote logging.
- Theme customization or user-configurable color palettes (single semantic palette only).
- Cross-database memory operations (single-DB scope, unchanged).

## Capabilities

### New Capabilities

- **`tui-memory-workspace`** — tab-based TUI workspace with Home, Memories, Search, Inbox, Sessions, Help; global Quick Actions hotkeys; semantic palette; read-only memory viewing with clipboard copy; friendly empty state.

### Modified Capabilities

- **`passive-capture`** — Inbox tab introduces user-observable behavior change: pending captures can now be edited (type + title + content body) before promotion via `[E]`. The underlying storage `MarkPromoted` write path is unchanged; the edit happens in-TUI before the promote call. The CLI/MCP capture path itself is unmodified.
- **`dashboard-ui`** (the existing dashboard capability, if previously declared) — full surface replacement: navigation model (stack-of-screens → tabs+stack), palette (Rose-Pine-Moon → semantic), brand (ASCII art → text), startup (splash → no-splash), theme flag (`--theme` removed).

## Approach

**Tab + stack coexistence in `flatModel`.** The architectural centerpiece is keeping `flatModel`'s existing Screen stack while adding tab navigation. `flatModel` gains `tabs []tabDef` and `activeTab tabKey` fields plus a key intercept at the top of `Update` that handles `1`-`6`, `tab`, `shift+tab`, and the global Quick Actions hotkeys (only when no input is focused). Each tab is itself a Screen — they implement the same interface as detail/search/pending screens do today. The Screen stack still handles overlays: pushing a Memory Detail screen renders on top of whatever tab is active, and popping it returns to that tab. This preserves the value of the existing Screen interface while solving the navigation flatness problem.

**Cleanup-first sequencing.** The `themes.go` / `theme.go` / `styles.go` / legacy `model.go` cluster is mutually entangled — `styles.go` calls `ApplyTheme(ThemeBrand)` from `themes.go`, and legacy `model.go` holds a `Theme` field. We cannot land the new palette while the old multi-theme machinery still compiles, because they fight over the same lipgloss vars. The first batch of apply tasks is therefore pure deletion and decoupling: sever `styles.go` from `themes.go`, delete the legacy `Model` and its triad, delete `cube.go`/`splash.go`/`logo.go`, extract `workstation_screen.go` helpers to `helpers.go`, then delete `WorkstationScreen`. Only after the slate is clean do we add the new tab infrastructure and Home tab. Strict TDD means each deletion is preceded by a test rewrite/removal in the same commit; the new RED tests for `flatModel` tab behavior are written before the legacy tests die.

**Salvage strategy from the predecessor.** Per the exploration's salvage matrix: `flat_model.go`, `screen.go`, `run.go`, `status.go`, `roadmap.go`/`roadmap.yaml`, `diskfree_*.go`, `updatecheck.go`, `teatest_smoke_test.go`, and the storage queries (`pending.go`, `stats.go`, `recent.go`, `sessions.go`) are kept as-is. `SearchScreen`, `RecentScreen`, `PendingScreen`, `DetailScreen` are kept with modifications (restyle, adapt to tab context, add accept/reject for Inbox, add `[C]` copy and remove edit/delete for Detail). `theme.go`'s struct shape is reused with new hex values; only the palette content changes. `DashboardScreen` is rewritten to become the Home tab (header, Quick Actions row, Project Health card, Latest Memories card, Recent Activity card). The predecessor's exploration of pending-capture promotion is reused for the Inbox `[A]`/`[E]`/`[R]` actions.

## Affected Areas

| Path | Impact | Description |
|---|---|---|
| `internal/dashboard/flat_model.go` | Modified | Add `tabs []tabDef`, `activeTab tabKey`, global key intercept (1-6, tab/shift+tab, Quick Actions), input-focus guard for hotkey suppression. |
| `internal/dashboard/screen.go` | Kept | Interface unchanged. |
| `internal/dashboard/run.go` | Kept | Entry point unchanged; receives a slimmer `Config`. |
| `internal/dashboard/model.go` | Removed | Legacy tab-based `Model`. |
| `internal/dashboard/view.go` | Removed | Legacy `Model.View()`. |
| `internal/dashboard/update.go` | Removed | Legacy `Model.Update()`. |
| `internal/dashboard/themes.go` | Removed | Multi-theme infrastructure. |
| `internal/dashboard/styles.go` | Modified | Decouple from `Theme` struct; rebuild against single `palette`. |
| `internal/dashboard/theme.go` | Modified | Replace 14 hex values with new semantic palette. |
| `internal/dashboard/logo.go` | Removed | Block-letter ASCII art. |
| `internal/dashboard/logo_test.go` | Modified | Rewrite for new text brand `renderBrand()`. |
| `internal/dashboard/cube.go` | Removed | Animated ASCII cube. |
| `internal/dashboard/splash.go` | Removed | Splash screen. |
| `internal/dashboard/commands.go` | Modified | Simplify after legacy removal (drop splash tick/cube tick). |
| `internal/dashboard/dashboard_screen.go` | Modified | Rewrite as Home tab: Quick Actions row, Project Health card, interactive Latest Memories card, Recent Activity card, empty state. |
| `internal/dashboard/workstation_screen.go` | Removed | Bridge design not in new spec. Delete AFTER helper extraction. |
| `internal/dashboard/helpers.go` | Created | Extract `relTime`, `truncateLeft`, `wrap`, `paneBox`, `clampCursor`, `padRight`, `centerString`, `formatLastSave` from `WorkstationScreen` before its deletion. |
| `internal/dashboard/recent_screen.go` | Modified | Adapt to Memories tab: pagination, filters (type/tag/scope), sort. |
| `internal/dashboard/projects_screen.go` | Removed | Project filtering becomes a Memories tab filter, not a separate screen. |
| `internal/dashboard/search_screen.go` | Modified | Restyling only; add focus signal so Quick Actions hotkeys suppress when input focused. |
| `internal/dashboard/pending_screen.go` | Modified | Become Inbox tab: `[A]` accept, `[E]` edit (type + title + content via Bubbles `textarea`) then accept, `[R]` reject. |
| `internal/dashboard/detail_screen.go` | Modified | Remove edit/delete affordances; add `[C]` copy; improved metadata rendering. |
| `internal/dashboard/sessions_screen.go` (new or extracted) | Created | Tab 5: restyled session list. |
| `internal/dashboard/help_screen.go` (new) | Created | Tab 6: keybindings reference + roadmap rendering. |
| `internal/dashboard/screen_stubs.go` | Modified | Remove `WorkstationScreen` stub; add stubs/constructors for any forward references. |
| `internal/dashboard/items.go` | Removed or stripped | Old `bubbles/list` item types; remove if unused. |
| `internal/dashboard/tabs_test.go` | Removed | Tests dead legacy `tabKey`. |
| `internal/dashboard/model_test.go` | Modified | Keep roadmap/status tests; delete legacy `Model` tests. |
| `internal/dashboard/teatest_smoke_test.go` | Kept | Infrastructure. |
| `internal/dashboard/roadmap.go`, `roadmap.yaml` | Kept | Move rendering target to Help tab. |
| `internal/dashboard/status.go` | Kept | Reused in Project Health card. |
| `internal/dashboard/diskfree_*.go` | Kept | Cross-platform disk status. |
| `internal/dashboard/clipboard_windows.go` (new) | Created | `clip.exe` subprocess. |
| `internal/dashboard/clipboard_darwin.go` (new) | Created | `pbcopy` subprocess. |
| `internal/dashboard/clipboard_linux.go` (new) | Created | `wl-copy` → `xclip` fallback chain. |
| `internal/dashboard/clipboard_test.go` (new) | Created | Per-OS unit tests with mocked exec. |
| `cmd/thoughtline/main.go` | Modified | Remove `--theme`, `--no-splash`, `--splash-ms` flags; remove corresponding `Config` fields; migration UX deferred to design. |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Legacy test mass (~40 of 80+ tests in `internal/dashboard/`) breaks when legacy `Model` is deleted | High | Strict TDD: rewrite/remove tests in the same commit that removes the production code; new `flatModel` tab tests are written RED first. Apply phase sequences this as batch #1 before any new UI code lands. |
| Palette decoupling (`styles.go` ↔ `themes.go` ↔ legacy `model.go`) is more entangled than expected | Medium | Treat decoupling as a discrete first task with its own tests. No new palette work begins until `styles.go` rebuilds cleanly against a single `palette` value. |
| `flatModel` tab refactor breaks the existing Screen stack semantics (push/pop, focus) | Medium | Tab key intercept lives ABOVE the Screen stack dispatch; if no tab key matches, the message falls through to the active Screen unchanged. Existing `teatest_smoke_test.go` plus new tab-switching tests guard the boundary. |
| Global Quick Actions hotkeys fire while user is typing in search/inbox edit, corrupting input | High if unmitigated | Mandatory input-focus guard: every Screen exposes an `InputFocused() bool` method (or equivalent signal); `flatModel` checks it before consuming Quick Action keys. New test: focus search input, press `m`, assert no tab switch. |
| Clipboard cross-platform fragility (xclip missing on Wayland-only Linux, `clip.exe` PATH issues in WSL/Cygwin shells) | Medium | Build tags isolate per-OS code; each backend returns an error that surfaces as an in-app status message ("clipboard unavailable") rather than crashing. No hard dependency on any specific clipboard binary. |
| Bubbles `textarea` integration in Inbox edit introduces a new component the codebase hasn't used before | Low | `textarea` is part of `github.com/charmbracelet/bubbles` which is already a direct dependency. Inbox edit gets its own Screen with isolated tests; `InputFocused()` returns true while editing to suppress global hotkeys. |
| User confusion during migration when `--theme` flag is silently removed | Low | Migration UX is an explicit deferred design question; default position is a friendly error message naming the flag and pointing at the changelog. |
| Helper extraction from `WorkstationScreen` to `helpers.go` introduces subtle behavior drift (e.g., `relTime` rounding) | Low | Extract verbatim with no signature changes; add a `helpers_test.go` covering the extracted functions before deleting `workstation_screen.go`. |
| Rounded borders unsupported on Windows legacy terminals (cmd.exe without VT) | Low | Lipgloss already handles fallback at the rendering layer; explicit fallback policy deferred to design. |

## Rollback Plan

This change is **pure code, no DB migration, no schema change, no data mutation**. Rollback options:

- **Squash-merge revert**: if the change lands as a single squashed merge, `git revert <merge-sha>` restores the legacy TUI in one commit. Users on the bad release downgrade to the prior version with no data loss.
- **Per-commit revert**: if commits are preserved through merge, the cleanup batch (legacy deletion) and the build batch (new tabs) can be reverted independently. Reverting only the build batch leaves a working "minimal TUI" state.
- **Flag-gated escape**: NOT pursued — adding a `--legacy-ui` flag would keep dead code alive and contradicts the cleanup goal.
- **DB**: untouched. No rollback action required on the storage layer.

## Dependencies

- **`github.com/charmbracelet/bubbletea`** — already a direct dependency. No version change required.
- **`github.com/charmbracelet/bubbles`** — already a direct dependency. The `textarea` component (Inbox edit) lives here; no new module required.
- **`github.com/charmbracelet/lipgloss`** — already a direct dependency. No version change required.
- **`github.com/charmbracelet/x/exp/teatest`** — used by `teatest_smoke_test.go`. Confirm it is a direct require in `go.mod` (it is, per the kept smoke test). New tab + Inbox integration tests will use it.
- **Clipboard**: no third-party library. Windows uses `os/exec` to invoke `clip.exe` (always available on Windows 7+). macOS uses `os/exec` to invoke `pbcopy`. Linux uses `os/exec` to invoke `wl-copy` then `xclip`. `golang.org/x/sys` is **NOT** required for this approach — subprocess-only keeps the dep footprint zero.
- **No new direct dependencies introduced by this change.**

## Success Criteria

- A first-time user, after `thoughtline ui`, can identify all six tabs and reach any of them in **one keystroke** (`1`-`6`).
- Quick Actions hotkeys (`S`, `/`, `M`, `I`) work from any tab when no input is focused, and **produce literal characters** in search and Inbox edit inputs when those inputs are focused (verified by automated `teatest` flow).
- A user can find and open a specific memory in **5 keystrokes or fewer** from cold start: `2` (Memories) → `↓` × 2 → `enter`.
- Memory Detail view shows full content, copies to clipboard with `[C]`, and returns to the originating tab with `esc`. Clipboard failure surfaces as a status message, never a crash.
- All saved memories are read-only in the TUI: no `[E]`/`[D]` hotkeys are registered on any non-Inbox screen. Verified by source-level test (`grep` of registered key bindings) and by automated keystroke test.
- Inbox `[E]` opens a Bubbles `textarea` for the content body, allows editing type and title, and on submit promotes the pending capture via `MarkPromoted`. Verified by `teatest` flow.
- Empty state renders when memory count = 0 and includes the three gamedev example types (`scene-pattern`, `perf-gotcha`, `pipeline-step`) and instructions for both `tl save` (CLI) and `tl_save` (MCP).
- No occurrence of `--theme`, `--no-splash`, or `--splash-ms` in the binary's flag set. Migration UX behaves per design decision.
- No occurrence of Rose-Pine-Moon hex values (`#26233a`, `#ea9a97`, `#ebbcba`, etc.) in `theme.go`. New palette is used everywhere lipgloss styles are constructed.
- `go test ./...` is green at the end of every commit (strict TDD).
- Deleted files: `model.go`, `view.go`, `update.go`, `themes.go`, `logo.go`, `cube.go`, `splash.go`, `workstation_screen.go`, `projects_screen.go`, `tabs_test.go`.
- New files: `helpers.go`, `sessions_screen.go`, `help_screen.go`, `clipboard_windows.go`, `clipboard_darwin.go`, `clipboard_linux.go`, `clipboard_test.go`, plus new test files for the Home tab, Memories tab, Inbox tab edit flow, and global hotkey suppression.

## Open Questions (deferred to design)

1. **Pagination size for the Memories tab**: 20 per page? 50? Page indicator vs infinite/virtual scroll?
2. **Memories tab filter UX**: inline always-visible filter bar, popup triggered by `f`, or separate filter screen pushed onto the stack?
3. **Scroll position preservation**: when going Memories → detail → `esc` back, does the cursor return to the same row, or reset to the top?
4. **Live refresh interval on Home**: auto-refresh every 30s (current legacy behavior), only on `r`, or on a Bubbletea tick driven by storage change notifications?
5. **Memories tab filter persistence**: do filter selections persist across tab switches and across `thoughtline ui` sessions, or reset each time?
6. **Sessions tab scope**: just restyle, or also add open/closed grouping and a session search field?
7. **Cross-platform rounded borders**: required with explicit ASCII fallback for non-VT terminals, or best-effort relying on lipgloss defaults?
8. **`--theme` / `--no-splash` / `--splash-ms` removal migration UX**: friendly error message naming the removed flag and pointing at the changelog, or silent removal with the flag rejected as "unknown flag"?
9. **Inbox `[E]` edit confirmation step**: confirmation/preview screen before promoting, or immediate submit when the user presses the submit key in the textarea form?
10. **Exact hex codes for the semantic palette**: cyan/blue, purple, green, yellow, red, gray — concrete values to be locked in design phase, validated against both light and dark terminal backgrounds.

## License Hygiene

This change does **NOT** lift code from `claude-mem` (AGPL) or any AGPL-licensed source. The dashboard surface is original Bubbletea/Lipgloss work. Engram-inspired patterns carried over from the predecessor `tui-redesign` remain license-clean: Engram is MIT-licensed, and the borrowed patterns (Screen interface shape, palette struct, push/pop stack) are structural rather than verbatim. No new third-party UI assets, fonts, or icon sets are introduced. The single emoji (`🧠`) in the brand logo is Unicode, not a licensed asset.
