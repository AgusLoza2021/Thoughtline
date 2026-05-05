# Cursor

[Cursor](https://cursor.com) speaks MCP since 0.50. Wire Thoughtline as a stdio MCP server in two steps.

## 1. Make sure the binary is on your PATH

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
thoughtline version
```

Or use a release binary from [the latest release](https://github.com/AgusLoza2021/Thoughtline/releases/latest).

## 2. Add the server

Open `Cursor Settings → MCP → Add Server` and paste:

```json
{
  "mcpServers": {
    "thoughtline": {
      "command": "thoughtline",
      "args": []
    }
  }
}
```

Cursor stores its config at `~/.cursor/mcp.json` (macOS / Linux) or `%USERPROFILE%\.cursor\mcp.json` (Windows). You can also edit it by hand.

Restart Cursor. The `tl_*` tools will be available in the agent dropdown.

## 3. Tell Cursor when to use the tools

Cursor doesn't auto-load skill protocols the way Claude Code does. Add this to your project's `.cursorrules` (or the global rules) so the agent knows the protocol:

```markdown
## Thoughtline persistent memory

You have access to thoughtline memory tools (namespace `tl`).

Save proactively after: decisions, conventions, bug fixes, non-obvious feature work, gotchas, user preferences. Use this content shape:

**What** / **Why** / **Where** / **Learned**

Use `topic_key` of the form `category/subject` (lowercase, slash-separated). Re-saves on the same key upsert.

Search proactively when the user references prior work: `tl_search` first, then `tl_get_observation` for full content.

Close working blocks with `tl_session_summary` containing Goal / Discoveries / Accomplished / Next Steps / Relevant Files.
```

## Tagging tip for Cursor users

Use `tool:cursor` on save so you can later filter for memories captured from Cursor specifically:

```jsonc
{ "tags": ["engine:unity", "tool:cursor"] }
```

See [`design/tag-conventions.md`](../design/tag-conventions.md) for the canonical vocabulary.
