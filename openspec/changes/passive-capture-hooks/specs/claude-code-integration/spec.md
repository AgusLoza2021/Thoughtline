# Delta for Claude Code Integration

> Change: `passive-capture-hooks`
> Status: proposed
> Operation: MODIFIED (extends existing claude-code-integration spec)

## ADDED Requirements

### Requirement 8: Hook Event Registration — 6 Events

`plugin/claude-code/hooks/hooks.json` MUST be extended with entries for the following 6 Claude Code hook events, each invoking `thoughtline hook <name>` with JSON piped from stdin:

| Event name (Claude Code) | `thoughtline hook` argument |
|--------------------------|-----------------------------|
| `SessionStart` | `session-start` |
| `UserPromptSubmit` | `user-prompt-submit` |
| `PreToolUse` | `pre-tool-use` |
| `PostToolUse` | `post-tool-use` |
| `Stop` | `stop` |
| `SessionEnd` | `session-end` |

Each hook entry MUST specify the command in a cross-platform form that works on both Windows (PowerShell/cmd) and Unix (bash/sh). The hook MUST pipe the event JSON to the binary via stdin.

The hook configuration MUST NOT suppress stdout from the binary globally — errors that reach stderr are visible to the user for debugging. The hook MUST be configured so that a non-zero exit from the command does NOT abort the Claude Code session (use `"on_failure": "continue"` or equivalent field if supported by the hooks JSON schema).

#### Scenario: Hooks file contains all 6 entries

- GIVEN `plugin/claude-code/hooks/hooks.json` has been updated
- WHEN the file is parsed
- THEN exactly 6 entries exist for the events listed above, each with a valid command referencing the `thoughtline` binary

#### Scenario: Hook failure does not crash Claude Code

- GIVEN one of the 6 hooks is registered and `thoughtline hook` exits non-zero (unexpected error)
- WHEN Claude Code fires that hook
- THEN the Claude Code session continues; no crash or blocking dialog appears

#### Scenario: Hook invocation is cross-platform

- GIVEN a Windows machine with `thoughtline.exe` in PATH
- WHEN Claude Code fires `PreToolUse`
- THEN the hook command resolves the binary correctly and exits 0

---

### Requirement 9: `tl_promote` Allow-Listed in settings.json

`~\.claude\settings.json` MUST include `mcp__thoughtline__tl_promote` in the tool allow-list alongside the existing 9 Thoughtline tools. After this change the allow-list MUST contain 10 entries.

(Previously: allow-list contained exactly 9 `mcp__thoughtline__tl_*` entries per Requirement 4 of the base spec.)

#### Scenario: tl_promote callable without permission prompt

- GIVEN `settings.json` contains `mcp__thoughtline__tl_promote` in the allow-list
- WHEN Claude Code calls `tl_promote` for the first time in a session
- THEN no permission dialog appears; the tool executes immediately

---

## MODIFIED Requirements

### Requirement 4: Allow-List Updated in settings.json

`~\.claude\settings.json` MUST NOT contain any `mcp__plugin_engram_engram__*` entries under the tool allow-list. It MUST contain pre-populated allow-list entries for all **10** Thoughtline MCP tools so the user is never prompted for permission on first use:

| Tool name pattern |
|---|
| `mcp__thoughtline__tl_save` |
| `mcp__thoughtline__tl_search` |
| `mcp__thoughtline__tl_get_observation` |
| `mcp__thoughtline__tl_context` |
| `mcp__thoughtline__tl_update` |
| `mcp__thoughtline__tl_delete` |
| `mcp__thoughtline__tl_session_start` |
| `mcp__thoughtline__tl_session_summary` |
| `mcp__thoughtline__tl_stats` |
| `mcp__thoughtline__tl_promote` |

(Previously: the set contained exactly 9 tools — `tl_promote` did not exist.)

#### Scenario: tl_save callable without permission prompt

- GIVEN `settings.json` contains the 10 `mcp__thoughtline__tl_*` allow-list entries and the `mcp__plugin_engram_engram__*` entries have been removed
- WHEN Claude Code calls `tl_save` for the first time in a new session
- THEN no permission dialog appears; the tool executes immediately

#### Scenario: No stale Engram entries remain

- GIVEN `settings.json` has been updated
- WHEN it is parsed
- THEN zero keys matching `mcp__plugin_engram_engram__*` are found in the allow-list
