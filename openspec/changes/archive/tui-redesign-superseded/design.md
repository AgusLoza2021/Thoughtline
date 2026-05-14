# Design: TUI Redesign — Flat Screens, Single Theme, Action Menu

## Technical Approach

Rebuild `internal/dashboard/` around a `[]Screen` stack with a single `Model` owning storage, palette, viewport size, and per-screen state. Each `Screen` is an interface (`Init / Update / View / Title`) so the root Model dispatches input to `stack[len-1]`. Old `tabKey`, `cube`, `splash`, and multi-`Theme` infrastructure are removed in a per-file commit series guarded by Strict TDD. Stat data sources from the existing `storage.Stats(...)` plus one new helper `storage.CountPending(...)`. Refresh model: `tea.Cmd` on screen entry → typed `*loadedMsg` → fold into model.

Targets dark terminals only (Windows Terminal + VS Code primary; cmd.exe ConPTY best-effort). Minimum viewport 80×24; smaller renders a single-line warning.

## Architecture Decisions

### Open Questions (proposal)

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| 1 | Final hex palette | Rose-Pine-Moon adapted (table below) | Warm dark, matches gamedev "punchy" target; gradient legible on Windows Terminal |
| 2 | ASCII glyphs | A. Hardcoded 5-row block (`var thoughtlineLogo`) | Deterministic, commit-safe, no figlet runtime dep, ≤70 cols fits 80-col terminals |
| 3 | Top Projects ranking | `MAX(updated_at) GROUP BY project` (recent activity) | Devs care about current sprint; lifetime totals reward dead repos |
| 4 | Stat-card 4 cells | memories, sessions, projects, this-week activity | Tools count is constant (12) and wastes a cell; weekly cadence matches gamedev sprints |
| 5 | Pending events caching | Cache on Model; refresh on dashboard `Init` + on `esc` pop-back + on `r` + 30s ticker | COUNT(*) per render is wasteful given key-driven re-renders; observable contract still satisfied |
| 6 | Deletion sequencing | Per-file series (8 commits, see Migration) | Bisect-friendly; tests stay green per commit |

### Subsection Decisions (A–K)

| Sub | Question | Decision | Rationale |
|-----|----------|----------|-----------|
| A | Screen-stack data structure | `stack []Screen` with `Push(s)` / `Pop()` methods on Model | Engram's `prevScreen` is too rigid for project picker → memories → detail drill-in |
| B | Min terminal size | 80×24; below → centered single-line warning, freezes input except `q`/`ctrl+c`. Content centered, max width 100 cols | Standard tty floor; matches Engram TUI; avoids broken layouts |
| C | Refresh model | Screen entry → `tea.Cmd` → typed `*loadedMsg` → Update folds into Model state. See Data Flow | Bubbletea idiomatic, already used by current code |
| D | Disk-free measurement | Parent dir of resolved DB path (`filepath.Dir(cfg.DBPath)`). Windows: `GetDiskFreeSpaceExW` via `golang.org/x/sys/windows`. Unix: `syscall.Statfs`. Failure → status `ERR` | DB-relative is what the user actually cares about; per-OS calls are short |
| E | Pending events empty-state copy | `No pending events. Passive capture is OFF — enable via tl_capture_passive.` (single literal — spec scenarios assert verbatim) | Friendly + actionable, mentions the actual tool name |
| F | Test strategy | Unit tests on Model handlers (golang stdlib). teatest harness for full screen-flow integration. Snapshot tests via `testify/golden` (no new dep beyond teatest) | teatest is `github.com/charmbracelet/x/exp/teatest` — already a transitive dep; adds direct require |
| G | Cross-platform | Primary: Windows Terminal, VS Code terminal, modern macOS/Linux ttys. Best-effort: cmd.exe via ConPTY. Document in README | Matches user base; cmd.exe edge cases noted not blocked |
| H | Removed-flag error | `thoughtline ui: flag provided but not defined: -theme. The --theme flag was removed in v0.2 (single-theme TUI). See CHANGELOG.md.` (custom error wrapping `flag.Parse`) | Clear migration path; keeps `flag` package's standard prefix |
| I | Migration docs | CHANGELOG: BREAKING entry listing removed flags + new defaults. README: update screenshots + flags table. No user action required (DB unchanged) | One-time documentation pass, no schema or config migration |
| J | Logo at 80 cols | Logo designed for ≤70 cols width with 5 padding cols each side. If terminal width < 70, logo replaced by literal text `THOUGHTLINE` in single-color brand-pink (no gradient) | 70-col budget verified against block-letter art for THOUGHTLINE (11 chars × 6 width = 66 cols + 4 inter-letter spacing) |
| K | Stat-query failure | Each cell shows `—`; status indicator goes `ERR`; footer error line shows underlying error | Matches existing `m.err` rendering pattern; user knows it's a known failure not a hang |

### Final Hex Palette (single theme: `palette`)

| Token | Hex | Use |
|-------|-----|-----|
| `Background` | terminal default (no override) | We do NOT set bg to keep transparency on themed terminals |
| `Foreground` | `#E0DEF4` | Default text |
| `Muted` | `#6E6A86` | Footer help, secondary metadata |
| `Border` | `#393552` | Double-border lines |
| `LogoGradient[0]` | `#C4A7E7` | Mauve (top row) |
| `LogoGradient[1]` | `#A88DC9` | Lavender |
| `LogoGradient[2]` | `#9CCFD8` | Blue-cyan |
| `LogoGradient[3]` | `#3E8FB0` | Teal |
| `LogoGradient[4]` | `#56949F` | Green-teal (bottom row) |
| `StatNumber` | `#EB6F92` | Big stat numbers (rose pink) |
| `Cursor` | `#F6C177` | `▸` glyph (gold) |
| `MenuSelectedBg` | `#2A273F` | Active menu item bg |
| `StatusOK` | `#9CCFD8` (blue) we use rose-pine "foam" | OK indicator |
| `StatusWarn` | `#F6C177` (gold) | WARN indicator |
| `StatusErr` | `#EB6F92` (rose) | ERR indicator |
| `Tag` | `#C4A7E7` | Optional badge |

Single-platform note: dark terminals only. README documents this.

### ASCII Logo (5 rows, ≤70 cols)

Hardcoded as `var thoughtlineLogo = []string{...}`. Block-letter style (similar to figlet `ANSI Shadow` but trimmed to 5 rows). Provided as a string slice; each row gets `LogoGradient[i]` applied via `lipgloss.NewStyle().Foreground(...).Render()` then joined with `lipgloss.JoinVertical`. Tasks phase commits the actual glyph art.

## Data Flow

```
        ┌─── tea.Program ─────────────────────────────────────┐
        │                                                     │
   key  │  Update(msg) ──► top := stack[len-1]                │
   ───► │                  ├─► local handler (cursor, select) │
        │                  └─► return tea.Cmd                 │
        │                                                     │
   ◄───   View() ◄── stack[len-1].View() ◄── render palette   │
        │                                                     │
        │   tea.Cmd (loadStatsCmd) ──► storage.Stats(...)     │
        │                              storage.CountPending() │
        │                              ─►  *loadedMsg         │
        │                                       │             │
        │   Update(*loadedMsg) ◄────────────────┘             │
        │     fold into Model.{stats,pending,projects}        │
        └─────────────────────────────────────────────────────┘
```

Screen-entry refresh: `Push(s)` calls `s.Init()`; `Pop()` returns control to `stack[len-1]` and calls its `OnFocus()` (new method) which re-issues the load cmd.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/dashboard/screen.go` | Create | `Screen` interface + helpers `Push`/`Pop` |
| `internal/dashboard/model.go` | Modify | Drop `tabKey`/`cube`/`splash` fields; add `stack []Screen`, `palette`, `pendingCount`, `lastErr`, `width`/`height` |
| `internal/dashboard/update.go` | Modify | Strip tabs; route to `stack[len-1].Update`; global `ctrl+c`, dashboard-level `q` quits, child `q`/`esc` pops |
| `internal/dashboard/view.go` | Modify | Render `stack[len-1].View()`; min-size guard; centered max-100-col layout |
| `internal/dashboard/dashboard_screen.go` | Create | Dashboard `Screen` impl: header, logo, tagline, stat-card, top-projects, action-menu, footer |
| `internal/dashboard/search_screen.go` | Create | Migrates current Search tab into a Screen |
| `internal/dashboard/recent_screen.go` | Create | "Recent activity" screen (memories list) |
| `internal/dashboard/projects_screen.go` | Create | Browse projects screen |
| `internal/dashboard/pending_screen.go` | Create | Pending events screen with empty-state copy from decision E |
| `internal/dashboard/logo.go` | Create | `var thoughtlineLogo []string` + render helper |
| `internal/dashboard/theme.go` | Create | Renames `themes.go`; single `palette` value; remove `Theme` struct |
| `internal/dashboard/themes.go` | Delete | Replaced by `theme.go` |
| `internal/dashboard/styles.go` | Modify | Strip multi-theme `ApplyTheme`; styles built from `palette` constants |
| `internal/dashboard/cube.go` | Delete | Animated cube removed |
| `internal/dashboard/splash.go` | Delete | Splash removed |
| `internal/dashboard/items.go` | Modify | Remove `sessionItem`/`tagItem` if unused; keep `memoryItem` |
| `internal/dashboard/diskfree.go` | Create | `func freeDiskPct(path string) (int, error)` with build tags `_unix.go` / `_windows.go` |
| `internal/storage/pending.go` | Modify | Add `func (s *Storage) CountPending(ctx, project) (int, error)` |
| `internal/storage/stats.go` | Modify | Add `MostRecentProjects []string` field populated from `MAX(updated_at) GROUP BY project LIMIT 4` |
| `internal/dashboard/model_test.go` | Modify | Adapt to screen-stack |
| `internal/dashboard/dashboard_screen_test.go` | Create | Per-section render + cursor tests |
| `internal/dashboard/tabs_test.go` | Delete | Tabs gone |
| `cmd/thoughtline/main.go` | Modify | Remove `--theme`, `--no-splash`, `--splash-ms` flags + custom error for old flags |
| `CHANGELOG.md` | Modify | BREAKING entry |
| `README.md` | Modify | Update screenshots + flags table |
| `go.mod` | Modify | Add `github.com/charmbracelet/x/exp/teatest` direct require |

## Interfaces / Contracts

```go
// internal/dashboard/screen.go
type Screen interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (Screen, tea.Cmd)
    View(width, height int, p palette) string
    Title() string
    OnFocus() tea.Cmd // called when screen returns to top of stack
}

// Model gets:
type Model struct {
    storage      *storage.Storage
    cfg          Config
    palette      palette
    stack        []Screen
    width, height int
    Quitting     bool

    // Cached data refreshed on dashboard focus.
    stats        storage.Stats
    pendingCount int
    diskFreePct  int
    lastErr      error
}

// internal/storage/pending.go
func (s *Storage) CountPending(ctx context.Context, project string) (int, error)
```

## Testing Strategy

| Layer | What to test | Approach |
|-------|--------------|----------|
| Unit | Model key handling: `j`/`k` cursor (no wrap), `s` shortcut, `enter` push, `esc` pop, `q` policy | Direct `Update()` calls with `tea.KeyMsg` |
| Unit | Stat card empty + non-zero, top-projects 0/2/5 cases, pending count 0 vs 7 | Synthetic `storage.Stats` injection via small in-memory store |
| Unit | Logo: row count ≥5, distinct gradient colors per row | String inspection of rendered output |
| Integration | Drill-in: dashboard → search → esc → dashboard, status re-queried | `teatest` driver |
| Integration | Min-size warning fires below 80×24 | `tea.WindowSizeMsg{Width:79,Height:23}` |
| Snapshot | Dashboard render with fixed stats fixture | Golden file `testdata/dashboard.golden` |
| Cross-platform | `freeDiskPct` returns plausible value on each OS | Per-OS test files via build tags |

Strict TDD: each commit in the series writes/updates tests BEFORE the implementation change.

## Migration / Rollout

Per-file commit series (decision #6):

1. `feat(dashboard): introduce Screen interface and stack` (new files only, unused)
2. `feat(dashboard): dashboard_screen with logo + stat card + menu` (+ tests)
3. `refactor(dashboard): route Update through stack; remove tabKey`
4. `feat(dashboard): port search into search_screen`
5. `feat(dashboard): recent/projects/pending screens`
6. `chore(dashboard): delete cube.go`
7. `chore(dashboard): delete splash.go`
8. `refactor(dashboard): collapse themes.go → theme.go (single palette)`
9. `feat(cli): remove --theme/--no-splash/--splash-ms with migration error`
10. `docs: CHANGELOG + README screenshots`

User-facing breakage: `--theme` and `--no-splash` rejected. CHANGELOG documents. No DB migration.

## Open Questions

- [ ] None — all 6 proposal questions and 11 sub-decisions resolved above.

## Risks

- Lipgloss double-border + Foreground combos render slightly differently in cmd.exe — mitigated by best-effort tier
- `teatest` API churn: it's in `x/exp` — pin a specific commit in `go.mod`
- 70-col logo budget assumes block letters; if final glyphs are wider, fall back to single-color text per decision J
