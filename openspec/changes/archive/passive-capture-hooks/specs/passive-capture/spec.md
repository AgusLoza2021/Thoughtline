# Passive Capture Specification (Delta from Change)

> Change: `passive-capture-hooks`
> Status: shipped (merged into `openspec/specs/passive-capture/spec.md`)
> Operation: ADDED (new capability — no prior spec exists)

## Summary

This is the original delta spec as proposed. The authoritative version has been merged into `openspec/specs/passive-capture/spec.md`.

See `openspec/specs/passive-capture/spec.md` (merged spec) for the current version which includes v1 known limitations and accepted deviations from design.

## Key Points from This Change

- New `pending_events` SQLite table with schema v3 migration
- Two-stage pipeline: `thoughtline hook` (capture) → `tl_promote` (curation)
- Three new MCP tools: `tl_pending_list`, `tl_pending_get`, `tl_promote`
- `thoughtline worker` janitor for retention + archival
- Default OFF opt-in via `THOUGHTLINE_PASSIVE_CAPTURE=1` env var
- 6 Claude Code hook events registered in `plugin/claude-code/hooks/hooks.json`
- Per-item transaction semantics for `tl_promote` (partial success allowed)
- All spec requirements implemented and tested; verify report: PASS-WITH-WARNINGS

## Known v1 Limitations (Carried Forward to Merged Spec)

1. Hook logging: stderr-only (no log file created)
2. Advisory lock simplified: busy_timeout only (no explicit BEGIN EXCLUSIVE)
3. Orphan-memory race: non-atomic seam between memory save and event status update

See merged spec for details and follow-up recommendations.
