# Memory Type Taxonomy Specification

> Change: `adopt-thoughtline-replace-engram`
> Status: proposed
> Operation: ADDED (first formal spec for the type taxonomy domain)

## Capability Summary

Defines the closed, exhaustive set of 11 memory type values that `Type.Valid()` MUST accept, the rules for keeping the set closed, and the type/scope coupling constraint that forces `preference` memories into `scope = personal`. This spec is the normative contract; `types.go` and `validate.go` are its implementation.

## Requirements

### Requirement 1: Closed Type Set (11 values)

The system MUST accept exactly the following values for `memory.Type`, case-sensitively, and MUST reject any other string including empty string, whitespace-only, and mixed-case variants:

| Value | Engram Origin |
|---|---|
| `game-design-decision` | Thoughtline native |
| `scene-pattern` | Thoughtline native |
| `asset-reference` | Thoughtline native |
| `perf-gotcha` | Thoughtline native |
| `pipeline-step` | Thoughtline native |
| `script-pattern` | Thoughtline native |
| `bugfix` | Engram 1:1 |
| `convention` | Thoughtline native (absorbs coerced Engram types) |
| `preference` | Engram 1:1 |
| `decision` | Engram 1:1 (new — added by this change) |
| `architecture` | Engram 1:1 (new — added by this change) |

`Type.Valid()` SHALL iterate `AllTypes()` and return `true` only on an exact byte-level match.

#### Scenario: Known valid types accepted

- GIVEN a `memory.Type` set to any of the 11 values listed above
- WHEN `Type.Valid()` is called
- THEN it returns `true`

#### Scenario: Types outside the set rejected

- GIVEN a `memory.Type` set to `"pattern"`, `"config"`, `"discovery"`, `"manual"`, or any arbitrary string not in the 11-value set
- WHEN `Type.Valid()` is called
- THEN it returns `false`

#### Scenario: Case-sensitivity enforced

- GIVEN a `memory.Type` set to `"Decision"`, `"ARCHITECTURE"`, or `"Bugfix"`
- WHEN `Type.Valid()` is called
- THEN it returns `false`

#### Scenario: Empty string rejected

- GIVEN a `memory.Type` set to `""`
- WHEN `Type.Valid()` is called
- THEN it returns `false`

---

### Requirement 2: `AllTypes()` Enumerates the Full Set

`AllTypes()` MUST return a slice of exactly 11 `Type` values in a stable, deterministic order. The slice MUST include both `decision` and `architecture` after this change is applied. Any code that switches on type values (e.g. UI labels, documentation generators) MUST derive its set from `AllTypes()` — hardcoded subsets are prohibited.

#### Scenario: AllTypes returns 11 values

- GIVEN the updated `types.go` is compiled
- WHEN `AllTypes()` is called
- THEN the returned slice has length 11 and contains both `"decision"` and `"architecture"`

#### Scenario: AllTypes used as source of truth for validation

- GIVEN `Type.Valid()` is implemented as a loop over `AllTypes()`
- WHEN a new type is added to `AllTypes()`
- THEN `Type.Valid()` accepts that type automatically — no separate switch update required

---

### Requirement 3: Preference Type Forces Personal Scope

A `Memory` with `Type = "preference"` MUST have `Scope = "personal"`. `Validate()` MUST return `ErrPreferenceMustBePersonal` if `Type == "preference"` and `Scope != "personal"`. This rule applies to all code paths that call `Validate()`, including the migration binary.

#### Scenario: Preference with personal scope passes validation

- GIVEN a `Memory` with `Type = "preference"` and `Scope = "personal"`
- WHEN `Validate()` is called
- THEN it returns `nil`

#### Scenario: Preference with project scope fails validation

- GIVEN a `Memory` with `Type = "preference"` and `Scope = "project"`
- WHEN `Validate()` is called
- THEN it returns `ErrPreferenceMustBePersonal`

---

### Requirement 4: Non-Preference Types Require Project Scope

A `Memory` with any type other than `"preference"` MUST have `Scope = "project"`. `Validate()` MUST return `ErrNonPreferenceMustBeProject` otherwise. This rule extends to the two newly added types `decision` and `architecture`.

#### Scenario: Decision type with project scope passes

- GIVEN a `Memory` with `Type = "decision"` and `Scope = "project"`
- WHEN `Validate()` is called (all other fields valid)
- THEN it returns `nil`

#### Scenario: Architecture type with personal scope fails

- GIVEN a `Memory` with `Type = "architecture"` and `Scope = "personal"`
- WHEN `Validate()` is called
- THEN it returns `ErrNonPreferenceMustBeProject`
