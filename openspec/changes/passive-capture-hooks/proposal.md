# Proposal: Passive Capture Hooks (opt-in)

## Intent

Today Thoughtline only captures memory when the AI explicitly calls `tl_save`. Valuable signal from a Claude Code session (tool uses, prompts, errors, session boundaries) is lost unless the model decides to summarize it. We want **opt-in passive capture**: Claude Code hooks stream raw events into a queue, and the AI later **promotes** selected events into typed Thoughtline memories. This preserves "explicit by design" as the default while unlocking richer recall for users who turn it on.

## Scope

### In Scope

- New CLI subcommand `thoughtline hook <event-name>`: reads JSON from stdin, validates, inserts a row into `pending_events`. Idempotent (UNIQUE on `(session_id, event_seq)` or `event_hash`).
- New CLI subcommand `thoughtline worker`: drains the queue. v1 behavior = mark events as `archived` after retention window. No LLM calls.
- New SQLite table `pending_events` (raw payload + status + dedup key + timestamps). Lives in the existing user-data DB.
- Opt-in switch: env var `THOUGHTLINE_PASSIVE_CAPTURE=1` AND/OR config-file flag. Default OFF — `thoughtline hook` exits 0 silently when disabled.
- Plugin integration: extend `plugin/claude-code/hooks/hooks.json` to register the 6 events (`SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `Stop`, `SessionEnd`) → each invokes `thoughtline hook <name>`.
- New MCP tool `tl_promote`: takes one or more `pending_event` IDs + a target memory `type` (from the 11-type taxonomy) + title/topic_key, inserts a real memory, marks events as `promoted`.
- Docs: `docs/integrations/claude-code-passive-capture.md` covering opt-in, privacy posture, promote workflow.

### Out of Scope

- LLM-based compression / summarization of raw events (separate ADR later).
- Multi-provider support (Gemini, OpenRouter, etc.).
- Embeddings or semantic ranking on raw events.
- Web viewer surfacing of `pending_events`.
- Cross-machine sync of pending events.

## Capabilities

### New Capabilities
- `passive-capture`: opt-in queue of raw Claude Code hook events with dedup, retention, and a promote-to-memory flow.

### Modified Capabilities
- `claude-code-integration`: hooks manifest gains 6 event registrations + opt-in semantics.

## Approach

Two-stage pipeline, both stages local and synchronous:

1. **Capture stage** (`thoughtline hook`): Claude Code spawns the binary per event. Binary checks the opt-in flag, validates JSON, computes a stable dedup hash, INSERT OR IGNORE into `pending_events`, exits fast (<50ms target). On any error: log and exit 0 — never break the host session.
2. **Promote stage** (`tl_promote` MCP tool): the AI (or user) reviews raw events via existing search/list tooling and explicitly promotes a curated subset into typed memories. Raw queue is a staging area, not a memory source.

Worker is a janitor only in v1 (status transitions + retention pruning). The architecture leaves room for an LLM compression worker later without changing the hook contract.

We read claude-mem (AGPL-3.0) for **architectural ideas only**. No code, prompts, schemas, or string literals are copied. License hygiene is enforced by code review and a clause in `docs/COMPARISON.md`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/thoughtline/` | New | `hook` and `worker` subcommands |
| `internal/store/` | Modified | New `pending_events` table + migration |
| `internal/mcp/` | Modified | Register `tl_promote` tool |
| `plugin/claude-code/hooks/hooks.json` | Modified | Register 6 hook event mappings |
| `docs/integrations/claude-code-passive-capture.md` | New | Opt-in user guide |
| `docs/COMPARISON.md` | Modified | License hygiene clause |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Raw queue becomes noise; nothing ever gets promoted | High | Default OFF; retention prunes unpromoted events; docs emphasize promote-first workflow |
| Binary crash breaks Claude Code session | Med | `hook` always exits 0; all errors logged not raised; integration test simulates failure |
| AGPL contamination from claude-mem | Low | No code/prompt/schema copying — architectural ideas only; clause in `COMPARISON.md`; PR review checklist |
| Hook latency degrades user experience | Med | Fast-path: opt-in check before JSON parse; <50ms budget; benchmark in tests |
| DB lock contention with concurrent hooks | Med | SQLite WAL already enabled; `INSERT OR IGNORE` is single-statement |
| Privacy: raw prompts stored unencrypted on disk | Med | Document clearly; opt-in only; future encryption-at-rest is separate change |

## Rollback Plan

- Flip default off (already off) → captures stop immediately.
- Remove hook entries from `plugin/claude-code/hooks/hooks.json` → Claude Code stops invoking us.
- `pending_events` table is additive; safe to leave in DB or drop via migration-down.
- `tl_promote` is additive; unregistering is a one-line change.

## Dependencies

- Existing SQLite store + migration framework.
- Existing `mark3labs/mcp-go` tool registration plumbing.
- Claude Code hooks API (already used by current plugin).

## Success Criteria

- [ ] With opt-in OFF (default), `thoughtline hook` exits 0 with no DB writes — verified by test.
- [ ] With opt-in ON, all 6 event types insert one row per unique event; duplicates are ignored.
- [ ] `tl_promote` converts a `pending_event` row into a real memory with correct taxonomy type and marks the source row `promoted`.
- [ ] Hook command median latency <50ms on a warm DB.
- [ ] No AGPL-licensed source/prompt strings appear in the diff (grep + manual review).
- [ ] Docs page explains opt-in, privacy, and promote workflow end-to-end.

## Open Questions

1. Config: env var only, config file only, or both? (Lean: both — env wins.)
2. Worker model: long-running daemon vs. cron-style invocation from a hook? (Lean: invoked, not daemon, in v1.)
3. `tl_promote` signature: single event vs. batch? (Lean: batch from day one — cheaper round-trips.)
4. Retention default for unpromoted events? (Suggested: 7 days.)
5. Should `SessionEnd` auto-trigger a promote prompt to the AI, or stay fully manual? (Lean: manual in v1.)
6. Where does the opt-in setting live if config-file? Reuse existing config or new section?

## License Hygiene Clause

claude-mem is AGPL-3.0. We have read it for architectural inspiration (hook list, queue-then-promote shape). We do **not** copy its code, prompts, JSON schemas, table schemas, or string literals. Any contributor PR touching this change must affirm the same. A short note will be added to `docs/COMPARISON.md` recording this posture.
