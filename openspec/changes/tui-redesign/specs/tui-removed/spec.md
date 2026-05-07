# tui-removed Specification

> Change: `tui-redesign`
> Status: draft
> Operation: ADDED (records explicit deletions — verifier checks absence)

## Purpose

Enumerates code, files, flags, and behaviors that MUST be absent after the tui-redesign change lands. The verify phase uses this spec to confirm deletions and breaking-change contracts.

---

## Requirements

### Requirement 1: Animated Cube Deleted

The file `internal/dashboard/cube.go` MUST NOT exist in the repository after this change. No other file in the codebase MAY import or reference any symbol previously defined in `cube.go`. There MUST be no `cube`-prefixed types or functions remaining in the `internal/dashboard` package.

#### Scenario: cube.go absent

- GIVEN the tui-redesign change has been applied
- WHEN `os.Stat("internal/dashboard/cube.go")` is evaluated
- THEN it returns `os.ErrNotExist`

#### Scenario: No references to cube symbols

- GIVEN the tui-redesign change has been applied
- WHEN the entire codebase is scanned for references to `cube` symbols from the dashboard package
- THEN no references are found

---

### Requirement 2: Splash Screen Deleted

The file `internal/dashboard/splash.go` MUST NOT exist in the repository after this change. No other file MAY import or reference any symbol previously defined in `splash.go`.

#### Scenario: splash.go absent

- GIVEN the tui-redesign change has been applied
- WHEN `os.Stat("internal/dashboard/splash.go")` is evaluated
- THEN it returns `os.ErrNotExist`

#### Scenario: No references to splash symbols

- GIVEN the tui-redesign change has been applied
- WHEN the codebase is scanned for references to splash symbols from the dashboard package
- THEN no references are found

---

### Requirement 3: Multi-Theme System Removed

The multi-theme infrastructure MUST be removed. Specifically:

- No `Theme1`, `Theme2`, or `Theme3` named constants or variables MAY exist anywhere in the `internal/dashboard` package.
- No `Theme` struct with multiple named variants MAY exist.
- `themes.go` MAY be renamed to `theme.go` but is not required to be. Regardless of filename, the result MUST be a single palette value (not a collection).
- The `--theme` CLI flag MUST NOT be accepted by `thoughtline ui`. If passed, the program MUST print an error and exit with a non-zero code. The flag MUST NOT be silently ignored.

#### Scenario: --theme flag rejected

- GIVEN the tui-redesign change has been applied
- WHEN the user runs `thoughtline ui --theme classic`
- THEN the program prints an error message referencing the unknown flag
- AND the program exits with a non-zero exit code

#### Scenario: No multi-theme constants

- GIVEN the tui-redesign change has been applied
- WHEN the `internal/dashboard` package is compiled and inspected
- THEN no exported or unexported identifiers named `Theme1`, `Theme2`, `Theme3`, or any variant of `ThemeN` exist

---

### Requirement 4: Tab-Based Navigation Removed

Tab-based navigation MUST be fully removed from the dashboard model. Specifically:

- No `tabKey` type or constant MAY exist in the `internal/dashboard` package.
- No `activeTab` field MAY exist in the dashboard `Model` struct.
- No `tabs_test.go` file MAY exist (the file is deleted, not emptied).
- The screen-stack navigation defined in `tui-dashboard` MUST be the sole navigation mechanism.

#### Scenario: tabKey absent

- GIVEN the tui-redesign change has been applied
- WHEN the `internal/dashboard` package is compiled
- THEN no symbol named `tabKey` or `activeTab` is present

#### Scenario: tabs_test.go absent

- GIVEN the tui-redesign change has been applied
- WHEN `os.Stat("internal/dashboard/tabs_test.go")` is evaluated
- THEN it returns `os.ErrNotExist`

---

### Requirement 5: Removed CLI Flags Produce Errors

Any CLI flag that existed before this change but is removed by this change MUST produce a visible error when passed, not silent acceptance. Confirmed candidates from the proposal:

| Flag | Action |
|------|--------|
| `--theme <name>` | Error + non-zero exit |
| `--no-splash` | Error + non-zero exit (if flag existed before this change) |

The verify phase MUST confirm the presence of `--theme` and `--no-splash` in `cmd/thoughtline/main.go` BEFORE the change to establish a baseline, then confirm their absence AFTER.

#### Scenario: --no-splash flag rejected

- GIVEN the tui-redesign change has been applied AND `--no-splash` existed before the change
- WHEN the user runs `thoughtline ui --no-splash`
- THEN the program prints an error and exits with a non-zero exit code

#### Scenario: Flags fail loudly, not silently

- GIVEN the tui-redesign change has been applied
- WHEN any removed flag is passed to `thoughtline ui`
- THEN the exit code is non-zero (MUST NOT be 0)
- AND stderr or stdout contains a message indicating the flag is not recognized
