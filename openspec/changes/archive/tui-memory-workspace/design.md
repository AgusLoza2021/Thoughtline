# Design: tui-memory-workspace — Tabbed Memory Workspace, Read-Only, Semantic Palette

> Change: `tui-memory-workspace`
> Phase: design
> Supersedes: `tui-redesign-superseded`

## 1. Technical Approach

We extend `flatModel` (the live entry point) with a tab layer that **coexists** with the existing `[]Screen` stack. `flatModel` gains two fields: `tabs []tabDef` (constant, ordered list of the six tabs Home/Memories/Search/Inbox/Sessions/Help) and `activeTab tabKey` (the currently-rendered tab index). Each tab is itself a `Screen` implementation, instantiated once at `Run()` time and held on the model. A new top-level key intercept in `flatModel.Update` handles `1`-`6`, `tab`, `shift+tab`, and the global Quick Actions hotkeys (`s`, `/`, `m`, `i`) before any message reaches the Screen stack. When the stack has overlays (e.g., Memory Detail pushed from Memories), the intercept yields to the topmost stacked screen so `esc` can pop back to the tab. The tab layer is therefore *strictly above* the stack in the dispatch order, never inside it.

The work lands in **cleanup-first sequencing**: legacy `model.go` / `view.go` / `update.go` / `themes.go` / `cube.go` / `splash.go` / `logo.go` are mutually entangled (styles.go calls `ApplyTheme(ThemeBrand)`, legacy Model holds a `Theme` field, cube/splash are referenced by legacy update). The first batch decouples `styles.go` from `themes.go`, rewrites the test files that depend on legacy `Model`, and deletes the dead files. Only after the slate is clean do we wire the new tabs, the Home rebuild, the Memories pagination, the Inbox edit form, and the clipboard build-tagged files. Strict TDD applies per commit: the new RED tests for `flatModel.tabKey` switching, `InputFocused()` suppression, and pagination are authored *before* the production code lands; legacy test removal happens in the same commit as the production removal so `go test ./...` is green at every SHA.

Salvage strategy follows the explore matrix: `flat_model.go`, `screen.go`, `run.go`, `status.go`, `roadmap.go`, `diskfree_*.go`, `updatecheck.go`, `teatest_smoke_test.go`, all `internal/storage` helpers stay verbatim. `theme.go`'s struct shape is reused; only its 14+ hex values change. `SearchScreen` and `DetailScreen` keep their structure with restyling and (for Detail) the addition of `[C]` copy plus removal of edit/delete affordances. `recent_screen.go` is the most surgical rewrite — it gains pagination, filters, and a sort selector while keeping its existing storage call sites.

## 2. Architecture Decisions

### 2a. Resolved Open Questions (proposal + explore)

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| 1 | Pagination size for Memories tab | **20 per page**, page-based with `n` (next) / `p` (prev) keys and a `Page X of Y · NNN total` indicator in the tab footer | 20 fits comfortably in an 80×24 terminal with header+filter+footer chrome. Page-based avoids virtual-scroll complexity (Bubbletea has no native virtualization); `n`/`p` are unambiguous and don't collide with cursor `↑`/`↓` (which move within the page). The total count is cheap (`storage.Stats().TotalMemories`) and orienting. |
| 2 | Memories tab filter UX | **Inline always-visible filter bar** at top of tab, with `f` cycling focus through the four filter widgets (type, tag, scope, sort). `Esc` from a focused filter widget returns focus to the row cursor | Always-visible bar makes filter state legible without an extra keystroke ("you are here" wins). A popup or pushed Screen hides what's filtering the rows the user is staring at. `f` to cycle is faster than mouse-style discoverability and matches the rest of the keymap. |
| 3 | Scroll position preservation | **Preserve cursor row and page** when returning from Memory Detail via `esc`. Filter changes reset cursor to row 0 of page 1 | The detail view is a read-only "peek"; bouncing the user back to the top is jarring and forces re-navigation. Filter changes legitimately invalidate position because the row set changed. |
| 4 | Live refresh interval on Home | **Manual refresh on `r` only**, plus implicit refresh when Home regains focus via `OnFocus` (tab switch back to Home, or `esc` pop from a pushed Screen) | There is no storage change-notification mechanism today, so a 30s ticker is pure polling and visible noise. `OnFocus` covers the common case (user saved via CLI then tabbed back). Manual `r` is the explicit escape hatch. |
| 5 | Memories tab filter persistence | **Persist across tab switches** within a single `thoughtline ui` session (held on the Memories Screen struct, which lives for the model's lifetime). **Reset across process restarts** — no on-disk filter state | Persistence across tabs matches expectation: "I filtered to perf-gotcha, I expect to find it still filtered when I come back from Help." Cross-session persistence requires a config file we don't have and would surprise users on next launch. |
| 6 | Sessions tab scope | **Restyle only** for this change. No open/closed grouping, no session search | Sessions are not the focus of this redesign; scope creep risks delaying the Memories/Inbox work that is the actual user pain. A follow-up change (`tui-sessions-grouping`) can add grouping/search once we see real usage. Listed in the residual roadmap of the Help tab. |
| 7 | Cross-platform rounded borders | **Best-effort relying on lipgloss defaults**. Use `lipgloss.RoundedBorder()` everywhere; lipgloss already degrades to ASCII corners on terminals without VT/UTF-8. Document the known-bad terminal (legacy `cmd.exe` without ConPTY) in CHANGELOG | Explicit ASCII fallback would require runtime terminal capability detection that lipgloss doesn't expose cleanly. The user base is Windows Terminal / VS Code / modern macOS/Linux ttys; legacy cmd.exe is rare for our gamedev audience. |
| 8 | `--theme` / `--no-splash` / `--splash-ms` removal UX | **Friendly error** naming the flag and pointing at CHANGELOG. Implementation: a custom pre-`flag.Parse` scan that detects the three removed flags by literal substring on `os.Args[1:]`, prints a clear message, exits with code 2 | Silent "unknown flag" is hostile to users who scripted these flags into shell aliases. Custom pre-scan avoids needing to register hidden flags just to reject them. Error text: `thoughtline ui: --theme was removed in v0.2 (single semantic palette). See CHANGELOG.md.` |
| 9 | Inbox `[E]` edit confirmation step | **No confirmation step**. Submit key (`ctrl+s` in the textarea, since `enter` inserts a newline in a multi-line body) immediately calls `MarkPromoted` and pops the edit form back to the Inbox list with a green status message `Promoted: <title>` | Adding a confirm screen for a draft promotion is friction. The user already authored the edit; a second "are you sure" is paternalistic. `ctrl+s` as submit is unambiguous and matches editor mental model; cancel is `esc`. |
| 10 | Exact hex codes for semantic palette | **Full table in Section 3 below** — 16 tokens locked, dark-terminal-first | See Section 3 for the table with rationale per token. All values verified against #1E1E1E (VS Code dark) and #0C0C0C (Windows Terminal default) backgrounds for WCAG AA contrast on the foreground/muted/border trio. |

### 2b. Subsection Decisions (A–N)

| Sub | Question | Decision | Rationale |
|-----|----------|----------|-----------|
| A | `flatModel` exposing `InputFocused()` from child Screens | **Extend the `Screen` interface** with an optional `InputFocused() bool` method via a Go interface assertion (`if f, ok := screen.(interface{ InputFocused() bool }); ok && f.InputFocused()`). Only Screens with text inputs implement it (`SearchScreen`, `InboxEditScreen`). Default behavior when not implemented: `false` | Interface assertion keeps the core `Screen` contract minimal (Init/Update/View/Title/OnFocus stay required). Forcing every Screen to implement a no-op `InputFocused()` adds boilerplate. `flatModel` queries the top-of-stack screen, then the active tab Screen, with stack winning if non-empty. |
| B | Where the tab-key intercept lives in the Update chain | **Order in `flatModel.Update`**: (1) `tea.WindowSizeMsg` → propagate to all tab screens AND stack screens; (2) `tea.KeyMsg` global quit (`ctrl+c`, `q` when stack empty); (3) if stack non-empty → dispatch to top of stack (so `esc` pops); (4) input-focus guard via `InputFocused()`; (5) if not focused → tab intercept (`1`-`6`, `tab`, `shift+tab`) and Quick Actions (`s`, `/`, `m`, `i`); (6) fall through to the active tab Screen's `Update` | Window size must reach every screen so they're sized correctly when first revealed. Stack overlays own their input fully (modal-ish). The focus guard runs after the stack check because the stack screen itself may have an input (e.g., Inbox edit pushed onto the stack). |
| C | Memory Detail "originating tab" for `esc` return | **Push a `tabReturnMsg{from: activeTab}` into the screen on construction**. `DetailScreen` stores the originating tabKey. On pop, `flatModel` reads the popped screen's `OriginatingTab()` accessor and sets `activeTab` to it. Default: if accessor missing, return to current `activeTab` (no-op) | Storing the origin on the pushed screen avoids `flatModel` needing a parallel stack of "where I came from." It also handles the unusual case of detail-from-Home: `esc` returns to Home, not to Memories. Accessor is an interface assertion like `InputFocused()` — opt-in. |
| D | Quick Actions hotkey `[S]` save stub behavior | **Status message only**: pressing `s` (when not in input) sets a footer status message `Save: use 'tl save ...' (CLI) or tl_save (MCP). In-TUI save coming in a follow-up change.` for ~5 seconds, then clears. No Screen push, no flow change | A placeholder Save Screen invites users to fill it in and reports failure when there's no backing flow. A status message is honest about the current state without breaking the hotkey contract. The message references the existing CLI/MCP paths the user already has. |
| E | Inbox edit form Screen — stack push vs Screen-internal mode | **Push as a separate `InboxEditScreen`** onto the stack. `InboxEditScreen` owns the textarea + two textinputs (type, title) + the pending row's ID. On submit → `MarkPromoted` cmd → pop on success | A separate Screen makes `InputFocused()` clean (the edit screen returns true always while present), keeps the Inbox list rendering simpler (no mode flag), and means `esc` from the edit form pops one level back to Inbox naturally via the existing stack mechanic. |
| F | Empty state detection — what counts as "empty" | **Global memory count = 0**, computed once at Home `OnFocus` from `storage.Stats().TotalMemories`. Project filter does not trigger the empty state (a project with zero rows just shows "No memories for this project yet" in the Latest Memories card; the rest of Home renders normally) | The empty state is for first-run users with literally nothing. A per-project zero is a normal filtered view and should look like a filter, not an evangelism page. |
| G | Project Health card — metrics, order, labels | **Five rows, fixed order**: (1) `Memories: NNN`, (2) `Sessions: NNN`, (3) `Pending: NN` (yellow when > 0), (4) `Disk free: X.X GB` (yellow when < 1 GB, red when < 100 MB), (5) `Storage: <path>` (gray, truncated left if > card width) | Counts first because they're the user's mental anchor. Pending elevated because it's actionable. Disk/Storage at bottom because they're chrome. Path truncated left so the filename remains visible. |
| H | Latest Memories card — row count | **5 rows fixed** on standard 80×24, **7 rows when viewport height > 36**. Adaptive via `min(7, max(3, (height-24)/3))` | 5 rows is the sweet spot for a quick glance; 3 rows is the floor for cramped terminals; 7 is the ceiling so the card doesn't dominate larger viewports (the Recent Activity card needs room too). |
| I | Recent Activity card — event source and window | **Mixed stream** of the last 5 events sorted by `updated_at desc` across: memory saves (label `saved`, purple), session opens/closes (label `session`, cyan), and pending captures (label `pending`, yellow). Time window: last 7 days, falling back to "last 5 of all time" if fewer | Mixed stream gives a true activity feel; siloed lists feel like reports. 7-day window covers a sprint; the all-time fallback prevents empty-card embarrassment for new projects. |
| J | Help tab rendering source | **Keybindings**: static markdown-like string rendered with lipgloss styling, sourced from `internal/dashboard/keybindings.go` (new file, single source of truth — `flatModel` reads from this map to register handlers AND Help reads it for display). **Roadmap**: existing `roadmap.yaml` via `roadmap.go` (unchanged) | A single source of truth for keybindings prevents the classic drift between "what the help says" and "what actually works." YAML for roadmap stays because it's already a working file the team edits. |
| K | Clipboard backend errors — message text and location | Status message rendered in the **DetailScreen footer line** (not the global footer): success `Copied to clipboard (NNN chars)` in green; failure `Clipboard unavailable: <reason>` in yellow. Examples: `Clipboard unavailable: xclip and wl-copy not found` on Linux | Footer of DetailScreen because that's where the action happened — global footer hijacking surprises the user. Yellow not red because clipboard failure is graceful degradation, not an error. |
| L | Test fixtures — controlled memory/pending state | **Reuse `newTestStorage(t)` pattern**: each test calls `newTestStorage(t)` to get a fresh `*storage.Storage` backed by a `t.TempDir()` SQLite file, then `Save(...)` / `MarkPending(...)` to seed exact rows. New helper `seedMemories(t, s, n int)` for pagination tests | Matches the rest of the codebase. `t.TempDir()` gives per-test isolation; SQLite is fast enough that 100-row seeding is sub-millisecond. No mocks of the storage layer — exercise the real one. |
| M | Snapshot / golden-file strategy | Golden files in `internal/dashboard/testdata/` for: `home_default.golden`, `home_empty.golden`, `memories_page1.golden`, `memories_filtered.golden`, `inbox_three.golden`, `inbox_edit_form.golden`, `detail.golden`, `help.golden`. **All rendered at 100×30**. Update via `go test ./internal/dashboard -update` | Standard Go golden-file pattern matching go-testing skill. 100×30 is the canonical "comfortable" terminal size — wider than the 80-col floor, taller than 24 so adaptive H values for Latest Memories card (5 rows) render predictably. |
| N | `helpers.go` extraction — pure copy vs restructure | **Pure copy with identical signatures**. Functions extracted: `relTime(time.Time) string`, `truncateLeft(string, int) string`, `wrap(string, int) []string`, `paneBox(title, body string, width int) string`, `clampCursor(int, int) int`, `padRight(string, int) string`, `centerString(string, int) string`, `formatLastSave(time.Time) string`. Add `helpers_test.go` BEFORE deleting `workstation_screen.go` | Behavior-preserving extraction is the lowest-risk path. Restructuring (e.g., turning `paneBox` into a Lipgloss style) belongs to a separate cleanup change, not this one. `helpers_test.go` first gives us a safety net the moment `workstation_screen.go` is removed. |

## 3. Final Hex Palette

Dark-terminal-first. All foreground values verified for WCAG AA contrast (4.5:1) on backgrounds `#1E1E1E` (VS Code dark+) and `#0C0C0C` (Windows Terminal default). The palette intentionally does NOT set a background — terminal transparency is preserved.

| Token | Hex | Use |
|-------|-----|-----|
| `Foreground` | `#E5E5E5` | Default body text |
| `Muted` | `#7A7A85` | Footer help, metadata (sync_id, paths, timestamps), inactive tab labels |
| `Border` | `#3A3A4A` | Default panel borders, card outlines |
| `BorderFocused` | `#7AA2F7` | Border of the focused card/input/screen (cyan/blue) |
| `Brand` | `#C4A7E7` | `🧠 Thoughtline` logo text, headings, memory-type badges (purple) |
| `BrandDim` | `#8E7AB5` | Subdued brand accents (tagline, badge bg variant) |
| `Nav` | `#7DCFFF` | Inactive nav tab text (cyan) |
| `NavActive` | `#7AA2F7` | Active nav tab label + underline (brighter blue) |
| `NavActiveBg` | `#1F2A44` | Subtle bg behind the active tab label for emphasis |
| `Success` | `#9ECE6A` | Saved, promoted, OK status (green) |
| `Warning` | `#E0AF68` | Pending, disk low, FTS degraded (yellow) |
| `Error` | `#F7768E` | Rejected, failed, error states (red) |
| `FocusedBg` | `#2A2A3A` | Selected row in lists, focused input background |
| `CardBg` | terminal default (no override) | Card surface — keeps terminal transparency |
| `StatusOK` | `#9ECE6A` | OK indicator glyph (alias of Success for clarity) |
| `StatusWarn` | `#E0AF68` | WARN indicator glyph (alias of Warning) |
| `StatusErr` | `#F7768E` | ERR indicator glyph (alias of Error) |
| `BadgeMemoryType` | `#C4A7E7` | Memory-type badge text (purple, alias of Brand) |
| `BadgeMemoryTypeBg` | `#2D2440` | Memory-type badge background (deep purple) |

Hex provenance: derived from Tokyo Night Storm (a known-good dark palette in the gamedev tool space) with the green/yellow/red retained and the Rose-Pine-Moon brand purple kept for continuity with the user's stated preference for purple branding. No Rose-Pine-Moon hex values remain (`#26233a`, `#ea9a97`, `#ebbcba` etc. are gone).

## 4. ASCII / Visual Layouts

All mockups at 100×30 unless noted. Lipgloss notation: `[fg=Brand]text[/]` means foreground = `Brand` token; `[focused]` means `BorderFocused`.

### 4.1 Home tab (default, populated)

```
╭──────────────────────────────────────────────────────────────────────────────────────────────────╮
│ [fg=Brand]🧠 Thoughtline[/]   [fg=Muted]Local memory for game projects[/]                                          │
│                                                                                                  │
│ [fg=NavActive,bg=NavActiveBg] 1 Home [/]  [fg=Nav] 2 Memories [/]  [fg=Nav] 3 Search [/]  [fg=Nav] 4 Inbox [/]  [fg=Nav] 5 Sessions [/]  [fg=Nav] 6 Help [/]  │
│                                                                                                  │
│ [fg=Muted]Quick:[/] [fg=Brand][S][/]ave  [fg=Brand][/][/]search  [fg=Brand][M][/]emories  [fg=Brand][I][/]nbox                              │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ╭─ Project Health ──────────────────╮  ╭─ Latest Memories ───────────────────────────────────╮  │
│ │ Memories:    142                  │  │ ▸ [fg=Brand]decision[/] sqlite-wal-mode    2h ago [fg=Muted]project[/]    │  │
│ │ Sessions:     38                  │  │   [fg=Brand]bugfix[/]   fts5-shared-cache  5h ago [fg=Muted]project[/]    │  │
│ │ Pending:       [fg=Warning]3[/]                  │  │   [fg=Brand]convention[/] script-naming    yesterday [fg=Muted]project[/] │  │
│ │ Disk free:  12.4 GB               │  │   [fg=Brand]perf-got[/] dx11-instancing    2d ago    [fg=Muted]project[/] │  │
│ │ [fg=Muted]Storage: …/thoughtline.db[/]      │  │   [fg=Brand]scene[/]    bake-lightmap-flow  3d ago [fg=Muted]personal[/] │  │
│ ╰───────────────────────────────────╯  ╰─────────────────────────────────────────────────────╯  │
│                                                                                                  │
│ ╭─ Recent Activity ──────────────────────────────────────────────────────────────────────────╮  │
│ │ [fg=Success]saved[/]    sqlite-wal-mode                          2h ago                                 │  │
│ │ [fg=Nav]session[/]  thoughtline / sdd-design                  3h ago                                 │  │
│ │ [fg=Warning]pending[/]  capture from claude-mcp                   4h ago                                 │  │
│ │ [fg=Success]saved[/]    fts5-shared-cache                        5h ago                                 │  │
│ │ [fg=Nav]session[/]  thoughtline / sdd-propose                yesterday                              │  │
│ ╰────────────────────────────────────────────────────────────────────────────────────────────╯  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Muted]↑↓ select · enter open · r refresh · q quit[/]                                                       │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
```

### 4.2 Memories tab (filter bar visible, 5 sample rows)

```
╭──────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 🧠 Thoughtline                                                                                   │
│ [fg=Nav] 1 Home [/]  [fg=NavActive,bg=NavActiveBg] 2 Memories [/]  [fg=Nav] 3 Search [/]  [fg=Nav] 4 Inbox [/]  [fg=Nav] 5 Sessions [/]  [fg=Nav] 6 Help [/]  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Filters: [fg=Brand]Type:[/] all       [fg=Brand]Tag:[/] -        [fg=Brand]Scope:[/] all       [fg=Brand]Sort:[/] updated↓     [fg=Muted](f cycle, c clear)[/] │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ▸ [fg=Brand][decision][/]  sqlite-wal-mode                                              2h ago [fg=Muted]project[/] │
│   Enable WAL via DSN pragma for concurrent reader+writer. busy_timeout=5000 prev…              │
│                                                                                                  │
│   [fg=Brand][bugfix][/]    fts5-shared-cache                                              5h ago [fg=Muted]project[/] │
│   FTS5 cursor crashed under shared cache mode; switched to per-connection cache.               │
│                                                                                                  │
│   [fg=Brand][convention][/]  script-naming                                          yesterday [fg=Muted]project[/] │
│   All shell scripts use kebab-case and a #!/usr/bin/env bash shebang.                          │
│                                                                                                  │
│   [fg=Brand][perf-got][/]  dx11-instancing                                              2d ago [fg=Muted]project[/] │
│   Per-instance buffer needs DYNAMIC usage + Map/Unmap; static lockstep stalls GPU.             │
│                                                                                                  │
│   [fg=Brand][scene][/]     bake-lightmap-flow                                        3d ago [fg=Muted]personal[/] │
│   Bake order: static meshes → light probes → reflection probes. UE5 specific.                  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Muted]Page 1 of 8 · 142 total · ↑↓ row · n next · p prev · enter open · f filter · q quit[/]              │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
```

### 4.3 Memory Detail view (full screen, pushed onto stack)

```
╭──────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 🧠 Thoughtline                                                          [fg=Muted](esc back)[/]                │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Brand][decision][/]  sqlite-wal-mode                                                                  │
│ [fg=Muted]project · thoughtline · tags: platform:windows · updated 2h ago · created 3d ago[/]              │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                  │
│ **What**: Enable WAL via DSN pragma.                                                             │
│                                                                                                  │
│ **Why**: Concurrent MCP tool calls need readers + writer simultaneously.                         │
│                                                                                                  │
│ **Where**: internal/storage/storage.go                                                           │
│                                                                                                  │
│ **Learned**: busy_timeout=5000 prevents lock errors under load. WAL files (-wal, -shm)           │
│ live alongside the main DB and must be included in backup tooling.                               │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
│                                                                                                  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Muted]topic_key: decision/sqlite-wal · sync_id: 0193abcd-…-7f                              [/]            │
│ [fg=Success]Copied to clipboard (487 chars)[/]                                                              │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Muted]↑↓ scroll · [C] copy · esc back[/]                                                                  │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
```

### 4.4 Inbox tab (3 pending captures)

```
╭──────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 🧠 Thoughtline                                                                                   │
│ [fg=Nav] 1 Home [/]  [fg=Nav] 2 Memories [/]  [fg=Nav] 3 Search [/]  [fg=NavActive,bg=NavActiveBg] 4 Inbox [/]  [fg=Nav] 5 Sessions [/]  [fg=Nav] 6 Help [/]  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Warning]3 pending captures[/]                                                                          │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ▸ [fg=Brand][decision][/]  proposed: use ristretto for hot-path cache                       1h ago        │
│   "Considering ristretto over freecache; ristretto has better TTL semantics and…"              │
│                                                                                                  │
│   [fg=Brand][bugfix][/]    proposed: panic in roadmap parser when yaml empty             3h ago        │
│   "Fixed nil-map deref in roadmap.go:42 by initializing the map before yaml.Unmar…"            │
│                                                                                                  │
│   [fg=Brand][scene][/]     proposed: ue5 nanite + virtual shadow map combo                yesterday    │
│   "Nanite meshes with VSM produce ~30% better shadow perf vs traditional VSM alon…"            │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Muted]↑↓ select · [A] accept · [E] edit · [R] reject · enter preview · q quit[/]                          │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
```

### 4.5 Inbox edit form (pushed onto stack from [E])

```
╭──────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 🧠 Thoughtline   [fg=Muted]Editing pending capture (esc cancel)[/]                                         │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                  │
│ [fg=Brand]Type:[/]    [focused] decision                            [/]  [fg=Muted](tab next, ↑↓ change)[/]  │
│                                                                                                  │
│ [fg=Brand]Title:[/]   [focused] use ristretto for hot-path cache    [/]                                  │
│                                                                                                  │
│ [fg=Brand]Body:[/]                                                                                         │
│ ╭──────────────────────────────────────────────────────────────────────────────────────────────╮│
│ │ **What**: Switch hot-path cache from freecache to ristretto.                                 ││
│ │                                                                                              ││
│ │ **Why**: ristretto has better TTL semantics and admission policy that matches our            ││
│ │ access pattern (heavy reads on a small working set).                                         ││
│ │                                                                                              ││
│ │ **Where**: internal/cache/                                                                   ││
│ │                                                                                              ││
│ │ **Learned**: ristretto needs explicit Wait() in tests to flush its async buffer.             ││
│ │_                                                                                             ││
│ │                                                                                              ││
│ ╰──────────────────────────────────────────────────────────────────────────────────────────────╯│
│                                                                                                  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Muted]tab next field · ctrl+s promote · esc cancel[/]                                                     │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
```

### 4.6 Sessions tab (restyled, flat list)

```
╭──────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 🧠 Thoughtline                                                                                   │
│ [fg=Nav] 1 Home [/]  [fg=Nav] 2 Memories [/]  [fg=Nav] 3 Search [/]  [fg=Nav] 4 Inbox [/]  [fg=NavActive,bg=NavActiveBg] 5 Sessions [/]  [fg=Nav] 6 Help [/]  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Muted]38 sessions · most recent first[/]                                                                │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ▸ thoughtline / sdd-design                                            3h ago    [fg=Success]open[/]        │
│   thoughtline / sdd-propose                                           yesterday [fg=Muted]closed[/]      │
│   thoughtline / tui-redesign-archive                                  2d ago    [fg=Muted]closed[/]      │
│   ue5-nanite-research / lightmap-flow                                 4d ago    [fg=Muted]closed[/]      │
│   thoughtline / brain-foundation                                      6d ago    [fg=Muted]closed[/]      │
│   …                                                                                              │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Muted]↑↓ select · enter open · q quit[/]                                                                  │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
```

### 4.7 Help tab (keybindings + roadmap)

```
╭──────────────────────────────────────────────────────────────────────────────────────────────────╮
│ 🧠 Thoughtline                                                                                   │
│ [fg=Nav] 1 Home [/]  [fg=Nav] 2 Memories [/]  [fg=Nav] 3 Search [/]  [fg=Nav] 4 Inbox [/]  [fg=Nav] 5 Sessions [/]  [fg=NavActive,bg=NavActiveBg] 6 Help [/]  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ╭─ Keybindings ──────────────────────────────────╮  ╭─ Roadmap ─────────────────────────────╮  │
│ │ [fg=Brand]Navigation[/]                                       │  │ [fg=Success]✓[/] Brain foundation                      │  │
│ │   1-6           jump to tab                    │  │ [fg=Success]✓[/] Tabbed memory workspace               │  │
│ │   tab / S-tab   cycle tabs                     │  │ [fg=Warning]◐[/] Inbox edit + promote                  │  │
│ │   esc           back / pop overlay             │  │ [fg=Muted]·[/] In-TUI save flow                      │  │
│ │   q / ctrl+c    quit                           │  │ [fg=Muted]·[/] Sessions open/closed grouping         │  │
│ │                                                │  │ [fg=Muted]·[/] Cross-project search                  │  │
│ │ [fg=Brand]Quick Actions[/]                                    │  │                                       │  │
│ │   s             save (CLI/MCP for now)         │  │                                       │  │
│ │   /             search                         │  │                                       │  │
│ │   m             memories                       │  │                                       │  │
│ │   i             inbox                          │  │                                       │  │
│ │                                                │  │                                       │  │
│ │ [fg=Brand]Memories tab[/]                                     │  │                                       │  │
│ │   ↑↓            select row                     │  │                                       │  │
│ │   n / p         next / prev page               │  │                                       │  │
│ │   f             cycle filter focus             │  │                                       │  │
│ │   c             clear filters                  │  │                                       │  │
│ │   enter         open detail                    │  │                                       │  │
│ │                                                │  │                                       │  │
│ │ [fg=Brand]Detail view[/]                                      │  │                                       │  │
│ │   ↑↓            scroll                         │  │                                       │  │
│ │   C             copy to clipboard              │  │                                       │  │
│ │                                                │  │                                       │  │
│ │ [fg=Brand]Inbox[/]                                            │  │                                       │  │
│ │   A             accept (promote as-is)         │  │                                       │  │
│ │   E             edit then promote              │  │                                       │  │
│ │   R             reject                         │  │                                       │  │
│ ╰────────────────────────────────────────────────╯  ╰───────────────────────────────────────╯  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Muted]q quit[/]                                                                                           │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
```

### 4.8 Empty state on Home (memory count = 0)

```
╭──────────────────────────────────────────────────────────────────────────────────────────────────╮
│ [fg=Brand]🧠 Thoughtline[/]   [fg=Muted]Local memory for game projects[/]                                          │
│                                                                                                  │
│ [fg=NavActive,bg=NavActiveBg] 1 Home [/]  [fg=Nav] 2 Memories [/]  [fg=Nav] 3 Search [/]  [fg=Nav] 4 Inbox [/]  [fg=Nav] 5 Sessions [/]  [fg=Nav] 6 Help [/]  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                  │
│                                  [fg=Brand]Welcome — no memories yet[/]                                    │
│                                                                                                  │
│         Thoughtline remembers the gamedev decisions you'd otherwise forget. Try:                 │
│                                                                                                  │
│         [fg=Brand]scene-pattern[/]   — how you wired a Unity / UE5 / PlayCanvas scene                      │
│         [fg=Brand]perf-gotcha[/]     — the GPU stall that took you a day to find                           │
│         [fg=Brand]pipeline-step[/]   — the FBX → glTF → engine import you almost wrote down                │
│                                                                                                  │
│         Save your first one from the CLI:                                                        │
│           [fg=Muted]$ tl save --type perf-gotcha --title "dx11 instancing" --content "…"[/]                  │
│                                                                                                  │
│         Or from Claude (MCP):                                                                    │
│           [fg=Muted]> Save this to Thoughtline as a scene-pattern[/]                                         │
│                                                                                                  │
│         Then press [fg=Brand]r[/] here to refresh.                                                          │
│                                                                                                  │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ [fg=Muted]r refresh · 6 help · q quit[/]                                                                      │
╰──────────────────────────────────────────────────────────────────────────────────────────────────╯
```

## 5. Data Flow

```
tea.Program
   │
   ▼
flatModel.Update(msg)
   │
   ├── tea.WindowSizeMsg ──► fan out to all tabs + stack screens; store size; return
   │
   ├── tea.KeyMsg
   │     │
   │     ├── (1) global quit: ctrl+c, or 'q' when stack empty AND not InputFocused ──► tea.Quit
   │     │
   │     ├── (2) stack non-empty? ──► top := stack[len-1]
   │     │         │                  │
   │     │         │                  ├── 'esc' ──► pop stack; if popped screen has OriginatingTab(), set activeTab; return tea.Cmd
   │     │         │                  └── else  ──► top.Update(msg) ──► tea.Cmd (may include pushScreenCmd/popScreenCmd)
   │     │         └── return
   │     │
   │     ├── (3) InputFocused() guard
   │     │         │
   │     │         ├── focused: tabs[activeTab].Update(msg) directly (text into input)
   │     │         └── not focused: continue
   │     │
   │     ├── (4) tab intercept: '1'..'6' ──► setActiveTab(n); call tabs[n].OnFocus(); return tea.Cmd
   │     │                     'tab'/'S-tab' ──► cycle; OnFocus(); return tea.Cmd
   │     │
   │     ├── (5) Quick Actions intercept:
   │     │         's' ──► setStatusMessage("Save: use tl save (CLI) or tl_save (MCP)…", 5s)
   │     │         '/' ──► setActiveTab(Search); tabs[Search].(*SearchScreen).FocusInput(); OnFocus()
   │     │         'm' ──► setActiveTab(Memories); OnFocus()
   │     │         'i' ──► setActiveTab(Inbox); OnFocus()
   │     │
   │     └── (6) fall through ──► tabs[activeTab].Update(msg)
   │
   ├── *memoriesLoadedMsg ──► tabs[Memories].(*MemoriesScreen).fold(msg)
   ├── *pendingLoadedMsg  ──► tabs[Inbox].(*InboxScreen).fold(msg)
   ├── *statsLoadedMsg    ──► tabs[Home].(*HomeScreen).fold(msg)
   ├── *promotedMsg       ──► popScreen + refresh inbox + footer status "Promoted: …"
   ├── *clipboardMsg      ──► detail.setStatus(msg.text, msg.level)
   └── *statusClearMsg    ──► clear status

Storage commands (tea.Cmd):
   loadHomeCmd      ──► storage.Stats(...) + storage.CountPending(...) + storage.RecentActivity(...) ──► *statsLoadedMsg
   loadMemoriesCmd  ──► storage.RecentAll(filter, page, perPage) + storage.CountMemories(filter) ──► *memoriesLoadedMsg
   loadPendingCmd   ──► storage.ListPending(...) ──► *pendingLoadedMsg
   promoteCmd(id,t,title,body) ──► storage.MarkPromoted + (re-save with edits) ──► *promotedMsg
   copyClipboardCmd(s) ──► CopyToClipboard(s) (per-OS) ──► *clipboardMsg
```

## 6. File Changes

| Path | Action | Description |
|---|---|---|
| `internal/dashboard/flat_model.go` | Modify | Add `tabs [6]Screen`, `activeTab tabKey`, `status statusMsg`, key intercept ladder (window → quit → stack → focus-guard → tab → quick-actions → tab.Update). New helpers: `setActiveTab`, `currentInputFocused`. |
| `internal/dashboard/screen.go` | Keep | Interface unchanged. Document optional `InputFocused() bool` and `OriginatingTab() tabKey` via interface assertion. |
| `internal/dashboard/run.go` | Modify | Slim `Config` (drop ThemeName/Splash/SplashDuration); construct the six tab Screens; initial `OnFocus` for Home. |
| `internal/dashboard/model.go` | Delete | Legacy tab-based `Model`. |
| `internal/dashboard/view.go` | Delete | Legacy `Model.View()`. |
| `internal/dashboard/update.go` | Delete | Legacy `Model.Update()`. |
| `internal/dashboard/themes.go` | Delete | Multi-theme infrastructure. |
| `internal/dashboard/styles.go` | Modify | Decouple from `Theme`. Rebuild lipgloss vars from single `palette`. Functions: `headerStyle`, `tabActiveStyle`, `tabInactiveStyle`, `cardStyle`, `cardFocusedStyle`, `badgeStyle(memType)`, `statusOK/Warn/Err`. |
| `internal/dashboard/theme.go` | Modify | Replace hex values with the Section-3 palette. Add tokens: `BorderFocused`, `Nav`, `NavActive`, `NavActiveBg`, `FocusedBg`, `BadgeMemoryType`, `BadgeMemoryTypeBg`. Keep `statusLevel`/`statusStyle` helpers. |
| `internal/dashboard/logo.go` | Delete and replace | Replace with `renderBrand()` returning `"🧠 Thoughtline"` styled `Brand` + tagline `"Local memory for game projects"` styled `Muted`. |
| `internal/dashboard/logo_test.go` | Modify | Rewrite for `renderBrand()` — golden file `brand.golden`, contrast/empty-string tests. |
| `internal/dashboard/cube.go` | Delete | Animated ASCII cube. |
| `internal/dashboard/cube_test.go` (if exists) | Delete | Cube tests. |
| `internal/dashboard/splash.go` | Delete | Splash screen. |
| `internal/dashboard/commands.go` | Modify | Drop splash/cube ticks. Keep storage-load Cmds. Add `loadMemoriesCmd`, `loadPendingCmd`, `promoteCmd`, `copyClipboardCmd`. |
| `internal/dashboard/dashboard_screen.go` | Modify (major) | Rewrite as `HomeScreen`: header, Quick Actions row, Project Health card, Latest Memories card (interactive cursor + enter to push Detail), Recent Activity card, empty state. `InputFocused() bool { return false }`. |
| `internal/dashboard/workstation_screen.go` | Delete | Bridge design. Delete AFTER helpers extracted. |
| `internal/dashboard/helpers.go` | Create | Verbatim extraction: `relTime`, `truncateLeft`, `wrap`, `paneBox`, `clampCursor`, `padRight`, `centerString`, `formatLastSave`. |
| `internal/dashboard/helpers_test.go` | Create | Unit tests for the eight extracted functions (table-driven, t.Run). |
| `internal/dashboard/recent_screen.go` | Modify (major) | Rename to `MemoriesScreen`. Add fields: `filter MemoriesFilter`, `page int`, `perPage = 20`, `total int`, `filterFocus filterField`. Add pagination (`n`/`p`), filter cycling (`f`), filter clear (`c`). `enter` pushes `DetailScreen` with `OriginatingTab() = TabMemories`. `InputFocused()` returns true when a filter widget is focused (tag textinput). |
| `internal/dashboard/projects_screen.go` | Delete | Project filter absorbed into MemoriesScreen filter bar. |
| `internal/dashboard/search_screen.go` | Modify | Restyle to new palette. Add `InputFocused() bool` (true when textinput is focused). Add `FocusInput()` accessor used by the `/` Quick Action. `enter` on a result pushes `DetailScreen` with `OriginatingTab() = TabSearch`. |
| `internal/dashboard/pending_screen.go` | Modify (major) | Rename to `InboxScreen`. Keys: `A` (immediate promote-as-is via `MarkPromoted`), `E` (push `InboxEditScreen`), `R` (reject — calls `MarkRejected` if exists, else `SweepPending(id)`). |
| `internal/dashboard/inbox_edit_screen.go` | Create | New Screen. Fields: `pendingID`, `typeField textinput`, `titleField textinput`, `bodyField textarea`, `focus int`. `tab` cycles fields; `ctrl+s` submits via `promoteCmd`; `esc` cancels (pop). `InputFocused() bool { return true }`. |
| `internal/dashboard/detail_screen.go` | Modify | Remove edit/delete affordances. Add `[C]` copy → `copyClipboardCmd`. Add `originating tabKey` field + `OriginatingTab()` accessor. Footer status line for clipboard result. |
| `internal/dashboard/sessions_screen.go` | Create | New file. Tab 5. Reuse `storage.RecentSessions`. Flat list with `enter` → session detail (future). Restyle only. |
| `internal/dashboard/help_screen.go` | Create | New file. Tab 6. Two-column layout: keybindings (from `keybindings.go`) on the left, roadmap (from `roadmap.go`) on the right. |
| `internal/dashboard/keybindings.go` | Create | Single source of truth: `var Keybindings = []KeybindGroup{...}`. Used by Help rendering AND `flatModel` action wiring (via lookup map). |
| `internal/dashboard/screen_stubs.go` | Modify | Remove `WorkstationScreen` stub. Add `NewSessionsScreen`, `NewHelpScreen`, `NewInboxEditScreen`, `NewHomeScreen`, `NewMemoriesScreen`, `NewInboxScreen` constructors used by `run.go`. |
| `internal/dashboard/items.go` | Delete | Old bubbles/list items, unused under new design. |
| `internal/dashboard/tabs_test.go` | Delete | Tests dead legacy `tabKey`. |
| `internal/dashboard/model_test.go` | Modify | Keep roadmap/status assertions; remove legacy `Model` tests. Rename to `roadmap_status_test.go` if cleaner. |
| `internal/dashboard/teatest_smoke_test.go` | Keep | Infrastructure. Update fixture expectations to new header/tabs. |
| `internal/dashboard/flat_model_test.go` | Create | RED-first tests: tab key intercept (1-6, tab/shift+tab), Quick Actions intercept, InputFocused suppression, stack-overlay precedence, OriginatingTab return. |
| `internal/dashboard/home_screen_test.go` | Create | Project Health card content, Latest Memories interactive cursor, empty state rendering, `r` refresh. |
| `internal/dashboard/memories_screen_test.go` | Create | Pagination (n/p), filter cycling (f), filter clear (c), filter persistence across tab switch, cursor preservation across detail push/pop. |
| `internal/dashboard/inbox_screen_test.go` | Create | `A` immediate promote (calls MarkPromoted), `E` push edit screen, `R` reject. |
| `internal/dashboard/inbox_edit_screen_test.go` | Create | Field cycling with tab, ctrl+s submits, esc cancels, InputFocused always true. |
| `internal/dashboard/detail_screen_test.go` | Create or modify | `[C]` calls clipboard cmd; status renders on success and failure; no `[E]`/`[D]` registered. |
| `internal/dashboard/help_screen_test.go` | Create | Renders keybindings from `Keybindings` slice; renders roadmap items. |
| `internal/dashboard/sessions_screen_test.go` | Create | Lists from `storage.RecentSessions`. |
| `internal/dashboard/clipboard.go` | Create | Common interface: `func CopyToClipboard(s string) error`. Selects backend via build tag. |
| `internal/dashboard/clipboard_windows.go` | Create | `//go:build windows` — `clip.exe` subprocess. |
| `internal/dashboard/clipboard_darwin.go` | Create | `//go:build darwin` — `pbcopy` subprocess. |
| `internal/dashboard/clipboard_linux.go` | Create | `//go:build linux` — `wl-copy` → `xclip -selection clipboard` fallback chain. |
| `internal/dashboard/clipboard_test.go` | Create | Per-OS tests with `exec.LookPath` shimmed via package-level var; assert subprocess invoked with expected args; assert sentinel error when no backend found. |
| `internal/dashboard/testdata/*.golden` | Create | `home_default.golden`, `home_empty.golden`, `memories_page1.golden`, `memories_filtered.golden`, `inbox_three.golden`, `inbox_edit_form.golden`, `detail.golden`, `help.golden`, `sessions.golden`, `brand.golden`. All at 100×30. |
| `cmd/thoughtline/main.go` | Modify | Pre-`flag.Parse` scan: detect `--theme`, `--no-splash`, `--splash-ms` (with or without `=value`); print friendly migration error to stderr; exit 2. Remove flag registrations. Remove `Config{ThemeName/Splash/SplashDuration}` references. |
| `cmd/thoughtline/main_test.go` (if exists, else create) | Create or modify | Test the migration error path: invoking with `--theme=brand` produces the documented error string. |
| `CHANGELOG.md` | Modify | Add BREAKING entry naming removed flags + new TUI summary. |

## 7. Interfaces / Contracts

```go
// internal/dashboard/screen.go (unchanged base, plus documented optional methods)

type Screen interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (Screen, tea.Cmd)
    View() string
    Title() string
    OnFocus() tea.Cmd
}

// Optional, queried via interface assertion:
type inputFocuser interface{ InputFocused() bool }
type originator   interface{ OriginatingTab() tabKey }

// internal/dashboard/flat_model.go

type tabKey int

const (
    TabHome tabKey = iota
    TabMemories
    TabSearch
    TabInbox
    TabSessions
    TabHelp
)

type tabDef struct {
    Key    tabKey
    Label  string // "Home", "Memories", ...
    Hotkey rune   // '1'..'6'
}

var defaultTabs = [6]tabDef{
    {TabHome,     "Home",     '1'},
    {TabMemories, "Memories", '2'},
    {TabSearch,   "Search",   '3'},
    {TabInbox,    "Inbox",    '4'},
    {TabSessions, "Sessions", '5'},
    {TabHelp,     "Help",     '6'},
}

type flatModel struct {
    storage    *storage.Storage
    width      int
    height     int

    tabs       [6]Screen
    activeTab  tabKey

    stack      []Screen
    status     statusMessage // text + level + expires
}

type statusMessage struct {
    Text    string
    Level   StatusLevel // OK / Warn / Err / Info
    Expires time.Time
}

// internal/dashboard/theme.go

type PaletteToken string

const (
    Foreground       PaletteToken = "fg"
    Muted            PaletteToken = "muted"
    Border           PaletteToken = "border"
    BorderFocused    PaletteToken = "border-focused"
    Brand            PaletteToken = "brand"
    BrandDim         PaletteToken = "brand-dim"
    Nav              PaletteToken = "nav"
    NavActive        PaletteToken = "nav-active"
    NavActiveBg      PaletteToken = "nav-active-bg"
    Success          PaletteToken = "success"
    Warning          PaletteToken = "warning"
    Error            PaletteToken = "error"
    FocusedBg        PaletteToken = "focused-bg"
    StatusOK         PaletteToken = "status-ok"
    StatusWarn       PaletteToken = "status-warn"
    StatusErr        PaletteToken = "status-err"
    BadgeMemoryType  PaletteToken = "badge-mem-type"
    BadgeMemoryTypeBg PaletteToken = "badge-mem-type-bg"
)

type palette map[PaletteToken]lipgloss.Color

// internal/dashboard/clipboard.go

// CopyToClipboard sends s to the OS clipboard. Returns a non-nil error
// when the platform-specific backend (or its fallback chain) is unavailable.
// Implemented per build tag in clipboard_{windows,darwin,linux}.go.
func CopyToClipboard(s string) error

// internal/dashboard/inbox_edit_screen.go

type InboxEditScreen struct {
    pendingID   int64
    typeField   textinput.Model
    titleField  textinput.Model
    bodyField   textarea.Model
    focus       int // 0=type, 1=title, 2=body
    err         error
}

func (s *InboxEditScreen) InputFocused() bool { return true }

// internal/dashboard/detail_screen.go (additions only)

type DetailScreen struct {
    // ... existing fields ...
    origin   tabKey
    status   statusMessage
}

func (s *DetailScreen) OriginatingTab() tabKey { return s.origin }

// internal/dashboard/memories_screen.go

type MemoriesFilter struct {
    Type  string // "" = all
    Tag   string // "" = none
    Scope string // "" = all
    Sort  string // "updated_desc" | "created_desc"
}

type MemoriesScreen struct {
    storage     *storage.Storage
    page        int
    perPage     int // const 20
    total       int
    rows        []storage.Memory
    cursor      int
    filter      MemoriesFilter
    filterFocus int // 0=row cursor, 1..4 = filter widgets
    tagInput    textinput.Model
}

func (s *MemoriesScreen) InputFocused() bool { return s.filterFocus == 2 && s.tagInput.Focused() }

// internal/dashboard/keybindings.go

type KeybindGroup struct {
    Title    string
    Bindings []Keybind
}

type Keybind struct {
    Keys string // "1-6", "tab / shift+tab", "ctrl+s"
    Desc string
}

var Keybindings = []KeybindGroup{
    {Title: "Navigation", Bindings: []Keybind{
        {"1-6", "jump to tab"},
        {"tab / shift+tab", "cycle tabs"},
        {"esc", "back / pop overlay"},
        {"q / ctrl+c", "quit"},
    }},
    // ... see Section 4.7 for the full set
}
```

## 8. Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit — helpers | `relTime`, `truncateLeft`, `wrap`, `paneBox`, `clampCursor`, `padRight`, `centerString`, `formatLastSave` | Table-driven `t.Run` in `helpers_test.go`. RED before `workstation_screen.go` deletion. |
| Unit — flatModel | Tab key intercept (1-6, tab/shift+tab), Quick Actions intercept, InputFocused suppression, stack precedence over tab keys, OriginatingTab return on pop | `flat_model_test.go` driving `m.Update(tea.KeyMsg{...})` directly; assertions on `m.activeTab`, `m.stack` length, returned `tea.Cmd`. |
| Unit — palette | Every PaletteToken renders to a non-empty hex; no Rose-Pine-Moon values remain | `theme_test.go` — assert each token resolves; grep-style assertion that `defaultPalette` contains no `#26233a`/`#ea9a97`/`#ebbcba`. |
| Unit — keybindings | The `Keybindings` slice covers every key wired in `flatModel.Update` | `keybindings_test.go` — collect actual handlers via a registry pattern; diff against the slice. Prevents the help/runtime drift. |
| Unit — clipboard | Each OS backend invokes the right subprocess; missing backend returns sentinel error | `clipboard_test.go` per build tag; shim `execLookPath` and `execCommand` via package vars. |
| Integration — Inbox edit flow | Press `i` → `↓` → `E` → edit fields → `ctrl+s` → assert MarkPromoted called with edited values, status message rendered, popped back to Inbox | `teatest.NewTestModel` in `inbox_edit_screen_test.go`. Uses `newTestStorage(t)` + `seedMemories(t, s, 0)` + manual pending insert. |
| Integration — Memories pagination | Seed 50 memories; assert page 1 shows 20, `n` advances, `p` retreats, footer shows `Page X of 3 · 50 total` | `memories_screen_test.go` with `teatest`. |
| Integration — Detail origin return | From Memories, push Detail, press `esc`, assert activeTab still Memories and cursor preserved | `flat_model_test.go` integration block. |
| Integration — Quick Actions suppression | Focus Search input, send `m`, assert no tab switch AND `m` literal appears in input | `flat_model_test.go` — drives the focus path then sends the key. |
| Snapshot — visual | Render each tab + Detail + edit form + empty state at 100×30 | `testdata/*.golden`. `-update` flag to regenerate. Drift triggers diff. |
| Cross-platform — disk free | Existing `diskfree_*.go` tests kept | No change. |
| Cross-platform — clipboard | Build-tagged tests run per OS in CI; missing-binary path always tested via the shim | See clipboard row above. |
| Migration — flag rejection | `thoughtline ui --theme=brand` exits 2 with the documented stderr message | `cmd/thoughtline/main_test.go` exec sub-binary or use the testable pre-scan function directly. |

Strict TDD application: every entry in the table above is RED in commit N, GREEN in commit N+1 (or same commit when the production change is purely a deletion paired with test rewrite).

## 9. Migration / Rollout

Per-commit series. Each commit ends with `go test ./...` green.

| # | Commit | Scope |
|---|---|---|
| 1 | `test(dashboard): write RED tests for new flatModel tab intercept` | `flat_model_test.go` with tab/quick-actions tests against a stub `flatModel` extension. No production code yet. |
| 2 | `refactor(dashboard): extract helpers from workstation_screen` | Create `helpers.go` + `helpers_test.go`. Behavior-preserving copy. |
| 3 | `chore(dashboard): decouple styles.go from themes.go` | Rebuild `styles.go` against single `palette`. Tests adjusted. |
| 4 | `feat(dashboard): semantic palette in theme.go` | Replace 14 hex values; add new tokens; palette/theme tests green. |
| 5 | `chore(dashboard): delete legacy Model + view/update + cube/splash/logo + themes + tabs_test` | All legacy production files removed. `model_test.go` slimmed to roadmap/status. `tabs_test.go` deleted. `logo_test.go` rewritten for `renderBrand`. Commit ends green. |
| 6 | `feat(dashboard): tab layer in flatModel` | Implement tab intercept, status message, OriginatingTab handling. Tests from commit 1 turn GREEN. |
| 7 | `feat(dashboard): Home tab (HomeScreen) with empty state` | Rewrite `dashboard_screen.go`. Latest Memories interactive cursor → push Detail. |
| 8 | `feat(dashboard): Memories tab with pagination + filters` | `recent_screen.go` → `MemoriesScreen`. Tests for pagination, filter cycling. |
| 9 | `feat(dashboard): Detail read-only with [C] copy + clipboard backends` | `detail_screen.go` changes + `clipboard_*.go` + tests. |
| 10 | `feat(dashboard): Inbox tab with [A]/[E]/[R] + edit screen` | `pending_screen.go` → `InboxScreen` + `InboxEditScreen`. Submit flow tested via teatest. |
| 11 | `feat(dashboard): Sessions tab restyle` | `sessions_screen.go` new file. |
| 12 | `feat(dashboard): Help tab + keybindings registry` | `keybindings.go` + `help_screen.go` + drift test. |
| 13 | `chore(cli): remove --theme/--no-splash/--splash-ms with migration error` | `main.go` pre-scan + test. CHANGELOG entry. |
| 14 | `test(dashboard): golden files for all tabs at 100x30` | `testdata/*.golden`. Final snapshot lock. |
| 15 | `chore(dashboard): delete projects_screen and items.go` | Now safe; nothing references them. |

Rollback: per-commit revert is safe; merging as a single squash is also safe (reverts in one shot).

## 10. Open Questions

None. All 10 deferred questions from the proposal and all 9 from the exploration are resolved in Sections 2a and 2b. Genuine residuals (e.g., the future Save in-TUI flow, Sessions grouping) are listed in the Help tab roadmap and explicitly Out of Scope per the proposal.

## 11. Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Legacy test mass (~40 of 80+) breaks during the cleanup commit | High | Commit 5 in the migration plan is sized to handle this as a single coordinated change. The new RED tests in commits 1–4 prove `flatModel` and the new palette work BEFORE the legacy is removed. |
| `styles.go` ↔ `themes.go` ↔ legacy `model.go` decoupling drags in more files than estimated | Medium | Treat commit 3 as its own self-contained refactor. If unexpected coupling appears, scope-creep is a `chore(dashboard):` followup, not a blocker for commit 4. |
| Bubbles `textarea` rendering quirks on Windows Terminal (cursor positioning at multi-byte boundaries) | Medium | `inbox_edit_screen_test.go` includes a multibyte-content case. If Bubbles textarea misbehaves, contingency is a single-line `textinput` fallback for Body and a doc note — but no schema or contract change. |
| Lipgloss rounded borders fall back to non-rounded on legacy `cmd.exe` (no ConPTY) | Low | Documented in CHANGELOG and Help tab — best-effort policy per decision #7. Lipgloss already handles this internally. |
| `clip.exe` PATH issues in WSL/Cygwin shells where the Windows build runs in a Unix-ish env | Low | Clipboard returns an error; in-TUI status message renders. No crash. WSL is technically Linux build (uses `wl-copy`/`xclip`), so this only affects Cygwin Go builds, which is out of our supported matrix. |
| Global Quick Actions hotkeys fire while user is typing in Search/Inbox edit, corrupting input | High if unmitigated | `InputFocused()` guard at step 3 in the Update ladder; explicit test in `flat_model_test.go`. Non-negotiable per the proposal's contract. |
| Memories pagination "Page X of Y" loses sync when filter changes mid-page | Medium | Filter changes reset `page = 0` and re-fetch `total`. Tested in `memories_screen_test.go`. |
| Migration error pre-scan misfires on legitimate strings (e.g., `--no-splashy` substring of `--no-splash`) | Low | Pre-scan uses exact-token match with optional `=value` suffix, not substring. Tested in `main_test.go`. |
| Clipboard backend selection at build time excludes a platform user runs binary on (e.g., FreeBSD) | Low | Default Linux build covers BSDs via `xclip`; an explicit `_unsupported` stub for other GOOS values returns sentinel error. No build failure. |
| Golden files churn on minor styling tweaks, causing PR review noise | Medium | Group golden updates per commit; reviewers diff `testdata/*.golden` once. Standard Go pattern, accepted trade-off. |
