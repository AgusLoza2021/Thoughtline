# OpenCode

[OpenCode](https://opencode.ai) — sst's open-source terminal AI agent — supports MCP via its config file. Wire Thoughtline as a local stdio MCP server.

## 1. Make sure the binary is on your PATH

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
thoughtline version
```

Or use a release binary from [the latest release](https://github.com/AgusLoza2021/Thoughtline/releases/latest). On Windows, the [auto-installer](../../scripts/install.ps1) handles this for you.

## 2. Register Thoughtline in OpenCode's config

OpenCode reads:

- **Global** config: `~/.config/opencode/opencode.json`
- **Project** config: `opencode.json` or `.opencode/opencode.json` in the project root

Add Thoughtline under the top-level `mcp` key. **Note**: OpenCode's `command` is an **array of strings**, not a single string.

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "thoughtline": {
      "type": "local",
      "command": ["thoughtline", "serve"],
      "enabled": true
    }
  }
}
```

You can also pass environment variables under `environment` if you want to pin the project or DB path:

```json
{
  "mcp": {
    "thoughtline": {
      "type": "local",
      "command": ["thoughtline", "serve"],
      "enabled": true,
      "environment": {
        "THOUGHTLINE_PROJECT": "my-game-project"
      }
    }
  }
}
```

To temporarily turn it off without removing the entry: `"enabled": false`.

Restart OpenCode (or start a new session) and the `tl_*` tools become available.

## 3. Tell OpenCode when to use the tools

OpenCode reads agent rules from `AGENTS.md` at the project root (and falls back to user-level rules). Drop the protocol there:

```markdown
## Thoughtline persistent memory

You have access to thoughtline memory tools (namespace `tl`).

Save proactively after: decisions, conventions, bug fixes, non-obvious feature work, gotchas, user preferences. Use this content shape:

**What** / **Why** / **Where** / **Learned**

Use `topic_key` of the form `category/subject` (lowercase, slash-separated). Re-saves on the same key upsert.

Search proactively when the user references prior work: `tl_search` first, then `tl_get_observation` for full content.

Close working blocks with `tl_session_summary` containing Goal / Discoveries / Accomplished / Next Steps / Relevant Files.
```

## Tagging tip for OpenCode users

Use `tool:opencode` on save and pass `agent_label: "opencode"` to `tl_session_start`:

```jsonc
{ "agent_label": "opencode", "tags": ["tool:opencode"] }
```

See [`design/tag-conventions.md`](../design/tag-conventions.md) for the canonical vocabulary.
