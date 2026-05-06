# Windsurf (Codeium Cascade)

[Windsurf](https://windsurf.com) — Codeium's AI IDE — supports MCP via Cascade. Wire Thoughtline as an stdio MCP server in two steps.

## 1. Make sure the binary is on your PATH

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
thoughtline version
```

Or use a release binary from [the latest release](https://github.com/AgusLoza2021/Thoughtline/releases/latest). On Windows, the [auto-installer](../../scripts/install.ps1) handles this for you.

## 2. Register Thoughtline in Windsurf's MCP config

Windsurf reads MCP servers from:

- **macOS / Linux**: `~/.codeium/windsurf/mcp_config.json`
- **Windows**: `%USERPROFILE%\.codeium\windsurf\mcp_config.json`

Create or edit that file and add:

```json
{
  "mcpServers": {
    "thoughtline": {
      "command": "thoughtline",
      "args": ["serve"]
    }
  }
}
```

If you already have other entries under `mcpServers`, **merge** — don't overwrite.

Restart Windsurf (or reload Cascade from its panel). The `tl_*` tools will appear in Cascade's tool list.

> **Heads up — tool cap.** Windsurf documents a cap of **100 total tools** that Cascade can access at once. Thoughtline adds 9 (`tl_save`, `tl_search`, `tl_get_observation`, `tl_context`, `tl_update`, `tl_delete`, `tl_session_start`, `tl_session_summary`, `tl_stats`). If you have many other MCP servers connected, you may need to disable some to stay under the limit.

> **Enterprise users**: MCP must be turned on manually in settings. Ask your admin if Cascade doesn't show the tools.

## 3. Tell Cascade when to use the tools

Windsurf doesn't auto-load skill protocols. Drop the boilerplate into your project's rules file (Windsurf reads `.windsurfrules`) or your team's shared rules doc:

```markdown
## Thoughtline persistent memory

You have access to thoughtline memory tools (namespace `tl`).

Save proactively after: decisions, conventions, bug fixes, non-obvious feature work, gotchas, user preferences. Use this content shape:

**What** / **Why** / **Where** / **Learned**

Use `topic_key` of the form `category/subject` (lowercase, slash-separated). Re-saves on the same key upsert.

Search proactively when the user references prior work: `tl_search` first, then `tl_get_observation` for full content.

Close working blocks with `tl_session_summary` containing Goal / Discoveries / Accomplished / Next Steps / Relevant Files.
```

## Tagging tip for Windsurf users

Use `tool:windsurf` on save and pass `agent_label: "windsurf"` to `tl_session_start`:

```jsonc
{ "agent_label": "windsurf", "tags": ["engine:unity", "tool:windsurf"] }
```

That lets the dashboard's Sessions tab show which editor drove the session, and Tags tab group memories by client.

See [`design/tag-conventions.md`](../design/tag-conventions.md) for the canonical vocabulary.
