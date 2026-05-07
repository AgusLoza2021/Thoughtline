# Delta for Claude Code Integration

> Change: `passive-capture-hooks`
> Status: shipped
> Operation: MODIFIED (extends existing claude-code-integration spec)

## Summary of Changes

The passive-capture-hooks change extends the existing claude-code-integration spec with:

1. **Requirement 5 (new)**: Hook Event Registration — 6 Events
   - Registers `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `Stop`, `SessionEnd` to invoke `thoughtline hook <name>` via stdin JSON piping
   - Cross-platform command form with `on_failure: continue` semantics
   
2. **Requirement 4 (updated)**: Allow-List Updated in settings.json
   - Updated from 9 tools to 12 tools (added `tl_promote`, `tl_pending_list`, `tl_pending_get`)
   - Updated requirement numbering: all subsequent requirements in the base spec are renumbered

3. **Requirement 8 (new)**: Edit Order is Mandatory
   - Originally Requirement 7 in the base spec; renumbered due to new requirements 5 and 6

## Full Updated Requirements

See `openspec/specs/claude-code-integration/spec.md` (merged spec) for the authoritative version after this change is applied. This delta file is preserved for archive reference only.

All details are captured in the merged spec at `openspec/specs/claude-code-integration/spec.md` which now includes:
- Original Requirements 1–7 (renumbered to 1–4, then 7–8)
- New Requirement 5: Hook Event Registration
- New Requirement 6: tl_promote Allow-Listed

The merged spec is the source of truth going forward.
