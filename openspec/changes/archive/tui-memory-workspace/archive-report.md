# Archive Report — tui-memory-workspace

> Phase: archive
> Date: 2026-05-14
> Final HEAD (pre-archive): `5ab5202 fix(dashboard): Inbox promotion fidelity`
> Status: CLOSED
> Verify verdict: ACCEPT (pass 2 after C1/C2/C3 remediation)

## Summary

`tui-memory-workspace` redesigned the Thoughtline TUI from the legacy multi-theme,
splash-gated, ASCII-art dashboard into a six-tab Memory Workspace (Home, Memories,
Search, Inbox, Sessions, Help) built on a single semantic palette, a Screen-stack
navigation model, and a Strict-TDD-shaped implementation. The change spans
exploration through second-pass verify in a single day (2026-05-14), with 15+
commits, a ~+6k LOC net delta (heavy in tests + new screens + planning), schema
migration v5 for the `rejected` pending-event status, and three breaking CLI flag
removals (`--theme`, `--no-splash`, `--splash-ms`) covered by a friendly migration
error.

The cycle's standout finding is that fixture coincidence — production code
hard-coding values that happened to match the test seed — masked real Inbox
promotion bugs through batch 5 of apply. Verify pass 1 caught all three CRITICAL
findings (C1: Accept hardcoded title + raw payload; C2: Edit dropped session +
captured_at; C3: ListPending hardcoded project filter), the fix-up commit
remediated each with line-level evidence, and verify pass 2 confirmed ACCEPT with
new L6 tests that defeat the fixture coincidence by seeding distinct project
names per test.

The change supersedes (in spirit) the archived `tui-redesign` cycle that
attempted to evolve the legacy multi-theme dashboard incrementally; the team
decided a clean Screen-stack rebuild was lower risk than further patching the
legacy `Model`/`View`/`Update` triad.

## Timeline

| Date | Phase | Outcome |
|------|-------|---------|
| 2026-05-14 | sdd-explore | Salvage matrix + 14 open questions surfaced |
| 2026-05-14 | sdd-propose | Locked decisions + 10 deferred questions for design |
| 2026-05-14 | sdd-spec | 25 + 1 delta + 12 negative requirements (3 specs total) |
| 2026-05-14 | sdd-design | All 19 open questions resolved + 18-token palette locked |
| 2026-05-14 | sdd-tasks | 99 strict-TDD tasks (later 103 after bug-fix addendum) |
| 2026-05-14 | sdd-apply batch 1 | Foundation (commits 1–4) |
| 2026-05-14 | sdd-apply batch 2 | Cleanup (–1700 LOC legacy, commit 5) |
| 2026-05-14 | sdd-apply batch 3 | Tab layer GREEN (commit 6) |
| 2026-05-14 | sdd-apply batch 4 | Tab screens (commits 7–10) |
| 2026-05-14 | sdd-apply batch 5 | Polish + verify gate (commits 11–15) |
| 2026-05-14 | sdd-verify pass 1 | 3 CRITICAL + 5 WARNING + 4 SUGGESTION |
| 2026-05-14 | sdd-apply fix-up | C1/C2/C3 remediated + L6 tests (commit `5ab5202`) |
| 2026-05-14 | sdd-verify pass 2 | ACCEPT (all CRITICALs clear) |
| 2026-05-14 | sdd-archive | This report; spec merge into `openspec/specs/`; folder moved to archive |

## Commit Series (recent, chronological)

| SHA (short) | Subject | Role |
|-------------|---------|------|
| `5ab5202` | fix(dashboard): Inbox promotion fidelity | Fix-up for C1/C2/C3 + L6 tests |
| `334422e` | feat(dashboard): Sessions tab restyle | Batch 5 polish |
| `e5f4711` | feat(dashboard): Inbox tab with [A]/[E]/[R] + edit screen | Batch 4 Inbox |
| `3592355` | feat(dashboard): Detail read-only with [C] copy + clipboard backends | Batch 4 Detail |
| `515185e` | feat(dashboard): Memories tab with pagination + filters | Batch 4 Memories |
| `f37aeec` | feat(dashboard): Home tab (HomeScreen) with empty state | Batch 4 Home |

Earlier batches (1–3) introduced the brand surface, semantic palette, Screen
stack, FlatModel router, Help/Search tabs, and the legacy-deletion sweep
(–1700 LOC across `cube.go`, `splash.go`, `logo.go` block-letter art,
`themes.go`, `model.go`/`view.go`/`update.go`, `tabs_test.go`,
`projects_screen.go`, `resize_test.go`, `teatest_smoke_test.go`, `items.go`,
`browse_screen_test.go`, `model_test.go`, plus the original `logo_test.go`).

## Final Scope

- **Files removed (15)**: `cube.go`, `splash.go`, `logo.go`, `themes.go`,
  `model.go`, `view.go`, `update.go`, `tabs_test.go`, `resize_test.go`,
  `teatest_smoke_test.go`, `projects_screen.go`, `items.go`,
  `browse_screen_test.go`, `model_test.go`, original `logo_test.go`.
- **Files created (~35)**: 6 new Screens (`HomeScreen`, `MemoriesScreen`,
  `InboxScreen`, `InboxEditScreen`, `SessionsScreen`, `HelpScreen`) + `brand.go`
  + 4 clipboard backends + 10 golden test fixtures + new test files +
  ADR-0006 + `verify-coverage.md`.
- **Storage surface change**: 1 new method (`MarkRejected`) + schema **v5**
  migration adding the `rejected` status to `pending_events.status` CHECK.
- **Breaking CLI flag removals (3)**: `--theme`, `--no-splash`, `--splash-ms`
  removed; a friendly migration error fires when the flag is passed.
- **Net LOC**: ~+6,000 (mostly tests + new screens + planning docs).
- **Test ratio**: 59 % test code : 40 % production code (Strict-TDD-enforced).

## Findings Carryover (from verify pass 1 → pass 2)

| ID | Class | Description | Status |
|----|-------|-------------|--------|
| C1 | CRITICAL | Accept `[A]` used hardcoded title + raw payload; ignored proposed values | CLEAR in pass 2 |
| C2 | CRITICAL | Edit submit hardcoded project; dropped `session_id` + `captured_at` | CLEAR in pass 2 |
| C3 | CRITICAL | `loadCmd` hardcoded `Project` filter masked unscoped use case | CLEAR in pass 2 |
| W1 | WARNING | `tabKey` constant present but not shared with keybindings layer | accepted deviation |
| W2 | WARNING | `#C4A7E7` reused 4× as Brand (also as Tag/StatNumber legacy aliases) | accepted per design Section 3 |
| W3 | WARNING | `internal/dashboard/workstation_screen.go` still on disk (vs. Req 7) | accepted — pre-disclosed batch-2 scope creep |
| W4 | WARNING | Cross-platform clipboard smoke (macOS pbcopy, Linux wl-copy/xclip) unverified | deferred to CI or platform-equipped contributor |
| W5 | WARNING | Schema v5 migration added during batch 4 (not in original design) | accepted — pre-disclosed, idempotent, data-preserving |
| S1–S4 | SUGGESTION | Quality improvements for follow-up (parse-strictness, golden coverage, etc.) | open for future SDD changes |

## Successor-Friendly Notes

- The Inbox `[E]` edit flow now correctly inherits `project`, `session_id`,
  and `captured_at` from the pending event (the original implementation had
  fixture-masked bugs that verify caught and fix-up resolved). When changing
  the flow, use the L6 integration tests
  (`TestInboxScreen_L6_AcceptPreservesProjectAndContent` and
  `TestInboxEditScreen_L6_SubmitPreservesProjectAndSession`) as the contract —
  they deliberately use distinct project names per test to defeat fixture
  coincidence.
- The Update ladder in `internal/dashboard/flat_model.go` is the
  architectural centerpiece of the redesigned TUI — keybinding routing, screen
  stack dispatch, and tab switching all flow through it. Modify with care and
  with the N5 (no-drift) keybinding test in mind.
- `internal/dashboard/keybindings.go` is the single source of truth for
  hotkeys. The N5 test guards drift between display affordances (e.g.,
  `[A]`/`[E]`/`[R]` on Inbox rows) and the actual key handlers.
- Golden files at `internal/dashboard/testdata/*.golden` lock the visual
  contract at 100×30. Regenerate via `go test ./internal/dashboard -update`
  when an intentional visual change is made.
- Schema v5 migration is idempotent — it will not re-run on already-migrated
  DBs. The migration adds `'rejected'` to the `pending_events.status` CHECK
  constraint and preserves all existing rows.
- Predecessor cycle `tui-redesign` is **not** archived under a date-prefixed
  folder; its lineage is documented inline in this change's `proposal.md` and
  `explore.md`. There is currently no `tui-redesign-superseded/` directory
  under `openspec/changes/archive/` (the user-supplied plan referenced one in
  error; no such folder exists at the time of this archive).

## Open Follow-up Suggestions

- **`tui-payload-parsing-strictness`** — could tighten `parseProposedMemory`
  in `internal/dashboard/helpers.go` to validate `proposed_type` against the
  canonical `memory.Type` enum and surface a typed warning when the payload
  proposes an unknown type (currently silent fallback to
  `defaultTypeForEvent`).
- **`tui-sessions-grouping-and-search`** — Sessions tab restyle landed in
  this change, but open/closed grouping + a session search affordance are
  deferred. Good candidate for a small follow-up.
- **`tui-in-app-save`** — `[S]` Quick Action currently surfaces a status
  message pointing the user to `tl save` / `tl_save`; a full in-TUI save flow
  (form + write through the existing storage path) is out of scope for this
  change and would be a natural successor.
- **`tui-clipboard-platform-coverage`** — wire macOS `pbcopy` and
  Linux `wl-copy` / `xclip` smoke tests into CI, or document a manual
  cross-platform smoke matrix for releases.

## Lessons Learned

- **Test-fixture coincidence is a real masking risk.** Production code
  hard-coding values that happen to match the test seed (`"test-workspace"`)
  hid three CRITICAL Inbox bugs through five apply batches. The fix-up's L6
  tests defeat this by using distinct project names per test
  (`"my-game-x"`, `"another-game"`) and asserting explicit non-leakage into
  the legacy fixture name. Adopt this pattern wherever production behavior
  varies by project / session / captured_at fields.
- **"Risk acknowledged in apply-progress" is NOT "risk resolved."** Three of
  the five WARNINGs (W3 `workstation_screen.go`, W5 schema v5 migration, and
  the eventual C1–C3 fixture issue) were pre-disclosed by `sdd-apply` and
  accepted by the orchestrator without challenge. The pattern works for
  scope-creep WARNINGs but failed for the CRITICALs because the orchestrator
  treated disclosure as resolution. Future cycles should require that
  apply-progress disclosures map to either (a) an explicit acceptance with
  reason or (b) a dedicated remediation task — not silent rollover.
- **Strict-TDD ratio of 59 : 40 (tests : code) paid off when verify caught
  real bugs.** The implementation tests at unit-level all passed; the bugs
  were only visible at broader integration scope (`acceptCmd` end-to-end with
  a real `pending_events` row and a real `memories` write). Broader-scope
  integration tests are worth their weight, even in a strict-TDD discipline.

## Cycle Status

**CLOSED.**

The SDD cycle for `tui-memory-workspace` is complete. Spec merge into
`openspec/specs/` is done. The change folder under `openspec/changes/` is
ready to be moved to `openspec/changes/archive/tui-memory-workspace/` by the
orchestrator's `git mv` and commit step.
