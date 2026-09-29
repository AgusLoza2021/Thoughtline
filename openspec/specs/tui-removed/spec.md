# tui-removed Specification

> Status: shipped

## Purpose

Captures the surfaces, files, symbols, and palette values that MUST be absent after `tui-memory-workspace` is applied. `sdd-verify` uses this spec to assert that the legacy multi-theme dashboard, ASCII art chrome, splash screen, predecessor bridge surfaces, and Rose-Pine-Moon palette have actually been removed from the codebase — not merely overridden.

Every requirement here is a negative assertion phrased so it maps to a single source-level grep, file-existence check, or built-binary flag inspection.

---

## Requirements

### Requirement 1: `--theme` CLI Flag Is Absent

The compiled `thoughtline` binary MUST NOT register a `--theme` CLI flag. The flag MUST NOT appear in the `thoughtline ui --help` output. The string `--theme` MUST NOT appear as a flag registration in `cmd/thoughtline/main.go`.

#### Scenario: Source-level absence of --theme registration

- GIVEN the source tree is scanned
- WHEN one greps `cmd/thoughtline/main.go` for the literal `"theme"` as a flag name argument to `flag.String` / `flag.Var` / equivalent
- THEN no match is found

---

### Requirement 2: `--no-splash` CLI Flag Is Absent

The compiled `thoughtline` binary MUST NOT register a `--no-splash` CLI flag. The string `no-splash` MUST NOT appear as a flag registration in `cmd/thoughtline/main.go`.

#### Scenario: Source-level absence of --no-splash registration

- GIVEN the source tree is scanned
- WHEN one greps `cmd/thoughtline/main.go` for the literal `"no-splash"`
- THEN no match is found

---

### Requirement 3: `--splash-ms` CLI Flag Is Absent

The compiled `thoughtline` binary MUST NOT register a `--splash-ms` CLI flag. The string `splash-ms` MUST NOT appear as a flag registration in `cmd/thoughtline/main.go`.

#### Scenario: Source-level absence of --splash-ms registration

- GIVEN the source tree is scanned
- WHEN one greps `cmd/thoughtline/main.go` for the literal `"splash-ms"`
- THEN no match is found

---

### Requirement 4: `internal/dashboard/cube.go` Does Not Exist

The file `internal/dashboard/cube.go` MUST NOT exist in the repository after this change is applied.

#### Scenario: File absent on disk

- GIVEN the repository is checked out at the post-change commit
- WHEN one stats `internal/dashboard/cube.go`
- THEN the file does not exist

---

### Requirement 5: `internal/dashboard/splash.go` Does Not Exist

The file `internal/dashboard/splash.go` MUST NOT exist in the repository after this change is applied.

#### Scenario: File absent on disk

- GIVEN the repository is checked out at the post-change commit
- WHEN one stats `internal/dashboard/splash.go`
- THEN the file does not exist

---

### Requirement 6: `internal/dashboard/logo.go` (Block-Letter Version) Does Not Exist

The file `internal/dashboard/logo.go` MUST NOT exist as the block-letter ASCII art module after this change. Any new brand rendering (text-only `🧠 Thoughtline`) MUST live in a different file (e.g., `helpers.go`, `home_screen.go`, or a new `brand.go`) — not in `logo.go`.

#### Scenario: Block-letter logo.go absent on disk

- GIVEN the repository is checked out at the post-change commit
- WHEN one stats `internal/dashboard/logo.go`
- THEN the file does not exist

---

### Requirement 7: `internal/dashboard/workstation_screen.go` Does Not Exist

The file `internal/dashboard/workstation_screen.go` MUST NOT exist in the repository after this change is applied. Shared helpers previously defined there MUST have been extracted to `internal/dashboard/helpers.go` before deletion.

#### Scenario: File absent on disk

- GIVEN the repository is checked out at the post-change commit
- WHEN one stats `internal/dashboard/workstation_screen.go`
- THEN the file does not exist

---

### Requirement 8: `internal/dashboard/projects_screen.go` Does Not Exist

The file `internal/dashboard/projects_screen.go` MUST NOT exist in the repository after this change is applied. Project filtering MUST be reachable through the Memories tab filter UX instead of a separate screen.

#### Scenario: File absent on disk

- GIVEN the repository is checked out at the post-change commit
- WHEN one stats `internal/dashboard/projects_screen.go`
- THEN the file does not exist

---

### Requirement 9: `internal/dashboard/tabs_test.go` Does Not Exist

The file `internal/dashboard/tabs_test.go` (tests for the legacy `tabKey` enum on the legacy `Model`) MUST NOT exist in the repository after this change is applied.

#### Scenario: File absent on disk

- GIVEN the repository is checked out at the post-change commit
- WHEN one stats `internal/dashboard/tabs_test.go`
- THEN the file does not exist

---

### Requirement 10: `internal/dashboard/themes.go` Does Not Exist

The file `internal/dashboard/themes.go` (multi-theme infrastructure: `ThemeBrand`, `ThemeZBrush`, `ThemeMono`, `ApplyTheme`, `nextTheme`) MUST NOT exist in the repository after this change is applied.

#### Scenario: File absent on disk

- GIVEN the repository is checked out at the post-change commit
- WHEN one stats `internal/dashboard/themes.go`
- THEN the file does not exist

#### Scenario: No references to multi-theme symbols remain

- GIVEN the source tree is scanned
- WHEN one greps `internal/dashboard/*.go` for the literals `ThemeBrand`, `ThemeZBrush`, `ThemeMono`, `ApplyTheme`, or `nextTheme`
- THEN no matches are found

---

### Requirement 11: Legacy `Model` Type Is Absent

The legacy tab-based dashboard `Model` type and its triad (`model.go` / `view.go` / `update.go`) MUST be absent. Specifically: no `tabKey` enum, no `func New(` constructor returning the legacy `Model`, no `func (m Model) View()` method, and no `func (m Model) Update(` method MUST exist in `internal/dashboard/*.go`.

#### Scenario: Legacy files absent on disk

- GIVEN the repository is checked out at the post-change commit
- WHEN one stats `internal/dashboard/model.go`, `internal/dashboard/view.go`, and `internal/dashboard/update.go`
- THEN none of those three files exist

#### Scenario: No tabKey enum or legacy Model methods remain

- GIVEN the source tree is scanned
- WHEN one greps `internal/dashboard/*.go` for the literals `tabKey`, `func (m Model) View()`, and `func (m Model) Update(`
- THEN no matches are found

---

### Requirement 12: Rose-Pine-Moon Hex Values Are Absent (with one intentional carry-over)

The Rose-Pine-Moon palette hex values previously hard-coded in `internal/dashboard/theme.go` MUST NOT appear anywhere under `internal/dashboard/*.go` after this change is applied, with **one explicit exception**: `#C4A7E7` is intentionally re-used as the new semantic `Brand` (and `BadgeMemoryType`) token because the user-locked design preference is purple branding and this hex is the visually correct purple. The reuse is deliberate, the value's *role* has changed (from "LogoGradient[0] / Tag" to "Brand purple"), and it is the only hex carried forward.

The forbidden hex codes are (case-insensitive):

| Hex | Original role | Status |
|-----|---------------|--------|
| `#E0DEF4` | Foreground | **forbidden** |
| `#6E6A86` | Muted | **forbidden** |
| `#393552` | Border | **forbidden** |
| `#C4A7E7` | LogoGradient[0] / Tag | **carry-over — now `Brand` / `BadgeMemoryType`** |
| `#A88DC9` | LogoGradient[1] | **forbidden** |
| `#9CCFD8` | LogoGradient[2] / StatusOK | **forbidden** |
| `#3E8FB0` | LogoGradient[3] | **forbidden** |
| `#56949F` | LogoGradient[4] | **forbidden** |
| `#EB6F92` | StatNumber / StatusErr | **forbidden** |
| `#F6C177` | Cursor / StatusWarn | **forbidden** |
| `#2A273F` | MenuSelectedBg | **forbidden** |

#### Scenario: No forbidden Rose-Pine-Moon hex codes remain in dashboard sources

- GIVEN the source tree is scanned
- WHEN one greps (case-insensitive) `internal/dashboard/*.go` for any of `#E0DEF4`, `#6E6A86`, `#393552`, `#A88DC9`, `#9CCFD8`, `#3E8FB0`, `#56949F`, `#EB6F92`, `#F6C177`, `#2A273F` (10 codes — the original 11 minus the carry-over `#C4A7E7`)
- THEN no matches are found

#### Scenario: `#C4A7E7` is allowed but ONLY as Brand / BadgeMemoryType

- GIVEN the source tree is scanned
- WHEN one greps for `#C4A7E7` in `internal/dashboard/*.go`
- THEN matches MAY appear, but ONLY in `theme.go` and ONLY assigned to the `Brand` and/or `BadgeMemoryType` palette tokens
- AND no match appears as `LogoGradient[*]` or `Tag` (legacy roles)
