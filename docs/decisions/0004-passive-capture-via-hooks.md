# ADR 0004 — Passive Capture via Claude Code Hooks

**Status**: Accepted
**Date**: 2026-05-07

---

## Context

Thoughtline's explicit-save model (`tl_save`) is intentional: the model curates
what becomes a memory. But users lose value from sessions where no save happened —
debugging sessions, exploratory tool chains, prompt exchanges that contained a
useful decision buried in the middle of a long session.

`claude-mem` (AGPL-3.0) solves this with fully automatic capture + LLM compression.
We want the value (capture everything) without the trade-offs (auto-promote with
possible hallucinated summaries, AGPL license, Node.js runtime dependency).

Two constraints drove the design:

1. **Opt-in only.** Hook payloads contain raw user prompts and tool I/O — possibly
   secrets. Default-OFF is the only acceptable privacy posture.
2. **Model curates, not pipeline.** The hook captures blindly; the model decides
   which events become memories. This preserves Thoughtline's "explicit, structured
   memory" thesis.

---

## Decision

Add a two-stage local pipeline:

1. **Capture stage** (`thoughtline hook <event-name>`): a one-shot CLI subcommand
   invoked by Claude Code hook events. Reads JSON from stdin, writes a raw row to
   `pending_events`. Always exits 0 — hook failures must never break the session.
   Enabled by `THOUGHTLINE_PASSIVE_CAPTURE=1`.

2. **Promote stage** (`tl_promote` MCP tool): the model reviews `pending_events`
   via `tl_pending_list` / `tl_pending_get` and calls `tl_promote` to convert
   selected events into typed memories. Per-event transactions; partial failure
   is fine.

3. **Retention janitor** (`thoughtline worker`): archives events past the
   retention window (default 7 days) and hard-deletes archived events past the
   hard-delete window (default 30 days). Cron/Task-Scheduler friendly.

Schema: new `pending_events` table in the existing SQLite file. Unique constraint
on `(project, event_hash)` for idempotency. No FTS5 trigger — pending events are
never surfaced by `tl_search`.

---

## Consequences

**Positive**:
- Users who opt in get a session history they can triage at any time.
- The model decides what's worth saving — no hallucinated auto-summaries.
- Idempotent hook invocations mean duplicate events from retries are safe.
- The pipeline is fully testable: unit tests for hash and domain types, storage
  tests for each mutation, CLI integration tests with a temp DB, MCP integration
  tests for all 3 new tools.

**Negative / Risks**:
- Raw payloads stored unencrypted. Privacy is user responsibility when opting in.
- The pending queue grows unbounded if the worker is never run. Documentation
  and a setup guide mitigate this; enforcement is future work.
- Hook payload schema from Claude Code may drift. Mitigation: we store the raw
  blob; extractors degrade gracefully to NULL for unknown fields.

**Deferred**:
- LLM compression of pending events (will be its own ADR when needed).
- Config-file activation (env-var only in v1; future: `os.UserConfigDir()/thoughtline/config.toml`).
- Encryption-at-rest (same scope as the existing `memories` table).
