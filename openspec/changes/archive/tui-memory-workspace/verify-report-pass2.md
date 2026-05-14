# Verify Report (Pass 2 — Post Fix-up) — tui-memory-workspace

> Phase: verify (second pass)
> Date: 2026-05-14
> Verifier: sdd-verify (opus 4.7)
> Subject: HEAD 5ab5202 (after fix-up commit for C1/C2/C3)
> Predecessor: verify-report.md (3 CRITICAL findings)

## Remediation Status

| Finding | Status | Detail |
|---------|--------|--------|
| C1 — Accept [A] hardcoded title + raw payload | CLEAR | `inbox_screen.go:230` calls `parseProposedMemory(ev.Payload, ev.EventType)`; `acceptCmd` uses `ev.Project` (line 232,237), `ev.SessionID` (243), `ev.CapturedAt` (244). No `"test-workspace"` literal anywhere in `inbox_screen.go`. Defensive retry on `ErrSessionNotFound` (252-258) preserves capture. |
| C2 — Edit submit hardcoded Project + dropped session/captured_at | CLEAR | `NewInboxEditScreen(ev pending.Event)` (inbox_edit_screen.go:55) takes the full event. `promoteCmd` uses `ev.Project` (236,242), `ev.SessionID` (248), `ev.CapturedAt` (249). Initial field values come from `parseProposedMemory` (56). No `"test-workspace"` literal. Same `ErrSessionNotFound` fallback (256-262). |
| C3 — loadCmd hardcoded Project filter | CLEAR | `loadCmd` (inbox_screen.go:203-217) calls `ListPending` with only `Status: "pending"` and `Limit: 50` — no Project filter. `storage.ListPending` (pending.go:141-144) now treats empty `Project` as unscoped (skips the `project = ?` predicate). The unscoped query is now the primary path, not a fallback. |
| L6 test — new integration test | PRESENT | `TestInboxScreen_L6_AcceptPreservesProjectAndContent` (inbox_screen_test.go:260) seeds project `"my-game-x"`, drives `acceptCmd`, loads via `RecentAll("my-game-x")` and asserts Project/Type/Title/Content; also asserts zero leakage into `"test-workspace"`. `TestInboxEditScreen_L6_SubmitPreservesProjectAndSession` (line 338) seeds project `"another-game"`, drives `promoteCmd` after editing typeField/titleField/bodyField, asserts edited values landed correctly and no leak. Both projects are distinct from the `"test-workspace"` fixture coincidence. |

## Build & Lint Health

- `go test ./...` — PASS. 13/13 packages green (cmd/migrate, cmd/thoughtline, internal/brain, internal/config, internal/dashboard, internal/events, internal/links, internal/memory, internal/pending, internal/server, internal/storage, plugin/claude-code/hooks; `cmd/migrate/cmd` has `[no test files]`).
- `go vet ./...` — PASS. Clean output, no diagnostics.

## Spot-checks (nothing else broke)

- **`storage.ListPending` semantic change is safe.** The only behavioral diff is "empty Project means unscoped" (pending.go:141-144 wraps the predicate in `if p.Project != ""`). Confirmed callers:
  - `internal/server/tl_pending_list.go:86-93` — passes `Project: project` from the MCP request, so when the caller supplies a project it still filters (unchanged behavior). When the MCP caller omits project, it now goes unscoped — this is the intended improvement and matches the inbox semantics.
  - `internal/dashboard/pending_screen.go:128` — passes the screen's project explicitly (unchanged behavior).
  - `internal/storage/pending_test.go:121-167` — all four `ListPending` sub-tests pass non-empty `Project: "proj-a"`, so the old code path is still exercised. Tests still green.
  - `internal/dashboard/inbox_screen.go:211` — now intentionally unscoped (the fix).
- **Fix-up commit scope is contained.** `git show --stat 5ab5202`: 8 files, 541+/-45 lines, all in `internal/dashboard/{golden_screens_test,helpers,inbox_edit_screen,inbox_screen,inbox_screen_test}.go`, `internal/storage/pending.go`, and the two openspec markdowns (`apply-progress.md` and `verify-report.md`). No drift into unrelated systems.
- **`parseProposedMemory` correctness.** Helper at `helpers.go:29-59` uses `json.Unmarshal` with `_` to ignore parse errors → silent fallback. Fallback rules: type → `defaultTypeForEvent(eventType)` (convention for tool/session hooks, decision otherwise); title → `"<EventType> capture"` or `"Inbox capture"` when EventType empty; content → raw payload (preserves capture). This honors the spec intent (Req 24 fidelity) and prevents the original C1 bug of `Type: memory.Type("PostToolUse")` (a hook name leaking into the type column).
- **`"test-workspace"` is gone from production.** Grep over `internal/dashboard/inbox_screen.go` and `inbox_edit_screen.go` returns zero matches; remaining occurrences are only in test files and `golden_screens_test.go` / `testhelpers_test.go` / `flat_model_test.go` (legitimate fixture identity for other tests).

## Carryover Items (from first pass, unchanged)

| Item | Status | Note |
|------|--------|------|
| W1 — tabKey grep | unchanged | accepted deviation (constant present but not yet shared with keybindings layer) |
| W2 — `#C4A7E7` 4 occurrences | unchanged | accepted (Brand carry-over from pre-redesign palette) |
| W3 — `workstation_screen.go` on disk | unchanged | accepted (pre-disclosed in apply-progress; deletion deferred) |
| W4 — clipboard cross-platform smoke | unchanged | deferred (manual macOS/Linux smoke after archive) |
| W5 — schema v5 migration | unchanged | accepted (pre-disclosed; no destructive change) |
| S1-S4 (suggestions) | unchanged | non-blocking |

## New Findings

None.

The fix-up commit introduced no new CRITICAL, WARNING, or SUGGESTION items. The defensive `ErrSessionNotFound` retry is a behavior change (the original spec did not mandate it) but it is the safer default: the alternative is dropping a promoted memory because its session link is stale, which is worse than losing the link. Not flagged as a SUGGESTION because the commit message explicitly justifies it.

## Recommendation

**ACCEPT for archive.** All three CRITICAL findings from the first pass are remediated with line-level evidence. The new L6 tests defeat the fixture coincidence that hid the bugs originally (distinct project names, full-field assertions, explicit leak-check into the old `"test-workspace"` fixture). Build is green, vet is clean, and no spillover into unrelated systems.

Next phase: `sdd-archive`.
