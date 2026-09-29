# Claude Code Integration Specification

> Change: `adopt-thoughtline-replace-engram`
> Status: proposed
> Operation: ADDED (first formal spec for the Claude Code ↔ Thoughtline integration contract)

## Capability Summary

Defines which configuration and skill files must exist, what they must contain, and in what order they must be edited so that Claude Code uses Thoughtline as its sole memory backend with zero Engram references remaining. Includes a mandatory pre-flight backup requirement and a post-flight smoke-test gate. All edits listed here are to user-space config files — none are in the Thoughtline repository.

## Requirements

### Requirement 1: Pre-flight Backup (Gate)

Before any destructive config edit, a zip backup of `~\.engram\` MUST exist at `~\.engram-backup-<YYYY-MM-DD>.zip`. No config file MUST be modified until this backup is confirmed on disk. This is the primary rollback anchor.

#### Scenario: Backup required before config edits

- GIVEN the migration has completed and the 5 config files are about to be edited
- WHEN the pre-flight check runs
- THEN it verifies `~\.engram-backup-2026-05-04.zip` (or today's date) exists; if missing, the process MUST stop and prompt the user to create the backup

---

### Requirement 2: MCP Server Config — Remove Engram Entry

`~\AppData\Roaming\Code\User\mcp.json` MUST NOT contain a server entry keyed `"engram"` after this change. The `"thoughtline"` entry MUST remain intact and unchanged. No other server entries MUST be modified.

#### Scenario: VSCode MCP panel shows only Thoughtline

- GIVEN `mcp.json` has had the `"engram"` entry removed and VSCode is restarted
- WHEN the user opens the MCP servers panel in VSCode
- THEN only `thoughtline` (and any other non-engram servers) appear; no `engram` entry is listed

---

### Requirement 3: Claude Code MCP Config — Engram File Removed

`~\.claude\mcp\engram.json` MUST be deleted (or renamed to `engram.json.bak` as a safety copy). After this change, no file named `engram.json` in `~\.claude\mcp\` MUST reference an active Engram server. `~\.claude\mcp\thoughtline.json` MUST exist and contain a valid Thoughtline MCP server definition pointing to the Thoughtline binary.

#### Scenario: Claude Code boots without Engram MCP server

- GIVEN `engram.json` is deleted and `thoughtline.json` exists with a valid config
- WHEN Claude Code is launched
- THEN no MCP connection error for Engram appears; Thoughtline MCP server connects successfully

---

### Requirement 4: Allow-List Updated in settings.json

`~\.claude\settings.json` MUST NOT contain any `mcp__plugin_engram_engram__*` entries under the tool allow-list. It MUST contain pre-populated allow-list entries for all **12** Thoughtline MCP tools so the user is never prompted for permission on first use:

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
| `mcp__thoughtline__tl_pending_list` |
| `mcp__thoughtline__tl_pending_get` |

Previously this contained exactly 9 tools. The passive-capture-hooks change adds 3 new tools (`tl_promote`, `tl_pending_list`, `tl_pending_get`), bringing the total to 12.

#### Scenario: tl_save callable without permission prompt

- GIVEN `settings.json` contains the 9 `mcp__thoughtline__tl_*` allow-list entries and the `mcp__plugin_engram_engram__*` entries have been removed
- WHEN Claude Code calls `tl_save` for the first time in a new session
- THEN no permission dialog appears; the tool executes immediately

#### Scenario: No stale Engram entries remain

- GIVEN `settings.json` has been updated
- WHEN it is parsed
- THEN zero keys matching `mcp__plugin_engram_engram__*` are found in the allow-list

---

### Requirement 5: Hook Event Registration — 6 Events

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

### Requirement 6: `tl_promote` Allow-Listed in settings.json

`~\.claude\settings.json` MUST include `mcp__thoughtline__tl_promote`, `mcp__thoughtline__tl_pending_list`, and `mcp__thoughtline__tl_pending_get` in the tool allow-list alongside the existing 9 Thoughtline tools. After this change the allow-list MUST contain 12 entries (see Requirement 4 above).

#### Scenario: tl_promote callable without permission prompt

- GIVEN `settings.json` contains `mcp__thoughtline__tl_promote` in the allow-list
- WHEN Claude Code calls `tl_promote` for the first time in a session
- THEN no permission dialog appears; the tool executes immediately

---

### Requirement 7: CLAUDE.md Protocol Block Replaced

`~\.claude\CLAUDE.md` MUST NOT contain the `<!-- gentle-ai:engram-protocol -->` block after this change. It MUST contain an equivalent Thoughtline memory protocol block that documents `tl_save`, `tl_search`, `mem_session_summary` equivalents, and the proactive save triggers. The replacement block MUST preserve the same structural sections (when to save, when to search, session close protocol) so that Claude Code's behavior is equivalent — just backed by Thoughtline tools.

#### Scenario: No Engram references in CLAUDE.md

- GIVEN `CLAUDE.md` has been updated
- WHEN the file is searched for the string `"engram"` (case-insensitive)
- THEN zero matches are found (excluding any historical changelog comment if present)

#### Scenario: Thoughtline protocol block present

- GIVEN `CLAUDE.md` has been updated
- WHEN the file is read
- THEN it contains a section titled "Thoughtline Persistent Memory — Protocol" (or equivalent) with `tl_save` and `tl_search` documented

---

### Requirement 6: Convention Skill File Replaced

`~\.claude\skills\_shared\engram-convention.md` MUST be replaced by `~\.claude\skills\_shared\thoughtline-convention.md`. The new file MUST document the Thoughtline-specific conventions (tool names, topic_key format, scope rules, session lifecycle) that the old Engram convention file documented for Engram. The old `engram-convention.md` MUST NOT remain as an active skill file (it MAY be renamed to `.bak`).

#### Scenario: Skill file swap complete

- GIVEN `thoughtline-convention.md` exists and `engram-convention.md` has been removed or renamed
- WHEN Claude Code auto-loads skills matching the current project context
- THEN only `thoughtline-convention.md` is loaded; no `engram-convention.md` reference is active

---

### Requirement 8: Edit Order is Mandatory

The 5 config edits MUST be applied in the following order to ensure there is always a working memory backend during the transition:

1. `mcp.json` — remove `engram` server entry (Thoughtline already active; this cuts Engram from the MCP bus)
2. `settings.json` — swap allow-list entries (updated to include 12 Thoughtline tools per Requirements 4 and 6)
3. `~\.claude\mcp\engram.json` — delete (or rename to `.bak`)
4. `~\.claude\CLAUDE.md` — replace protocol block
5. `~\.claude\skills\_shared\engram-convention.md` → `thoughtline-convention.md`

If the apply phase processes these out of order, the verify phase MUST flag it as a CRITICAL failure.

#### Scenario: Post-flight smoke test passes

- GIVEN all 5 files have been updated in the correct order and Claude Code has been restarted
- WHEN the user calls `tl_search` with a `sync_id` known to have been migrated from Engram
- THEN Thoughtline returns the memory with the original `sync_id` and content intact
