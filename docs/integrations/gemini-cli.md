# Gemini CLI

The [Google Gemini CLI](https://github.com/google-gemini/gemini-cli) supports MCP via its `settings.json`. Wire Thoughtline as an stdio MCP server.

## 1. Make sure the binary is on your PATH

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
thoughtline version
```

Or use a release binary from [the latest release](https://github.com/AgusLoza2021/Thoughtline/releases/latest). On Windows, the [auto-installer](../../scripts/install.ps1) handles this for you.

## 2. Register Thoughtline in Gemini's settings.json

Gemini CLI reads:

- **User-level**: `~/.gemini/settings.json`
- **Project-level**: `.gemini/settings.json` in the project root (overrides user)

Add Thoughtline under the top-level `mcpServers` key:

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

If you already have entries under `mcpServers`, **merge** — don't overwrite.

Restart the CLI session and the `tl_*` tools become available. Gemini CLI exposes a `/mcp` slash command to inspect connected servers; use it to verify Thoughtline is listed.

### Optional knobs

Gemini CLI's MCP server schema supports more keys than we use by default. Worth knowing:

| Key | Use |
|---|---|
| `cwd` | Run `thoughtline serve` from a specific directory (useful if you rely on the cwd-basename project default). |
| `env` | Set env vars per server. Supports `$VAR_NAME` and `${VAR_NAME}` expansion. |
| `timeout` | Request timeout in milliseconds. Bump it if you have a very large database (the default is fine for ~100k memories). |
| `trust` | Bypass tool-call confirmation dialogs. Don't enable unless you know what each tool does — you do, but be deliberate. |
| `includeTools` / `excludeTools` | Allowlist or blocklist specific `tl_*` tools. Useful if you want a read-only Gemini session: `"includeTools": ["tl_search", "tl_get_observation", "tl_context", "tl_stats"]`. |

Example with everything:

```json
{
  "mcpServers": {
    "thoughtline": {
      "command": "thoughtline",
      "args": ["serve"],
      "env": {
        "THOUGHTLINE_PROJECT": "my-game-project"
      },
      "timeout": 15000
    }
  }
}
```

## 3. Tell Gemini when to use the tools

Gemini CLI reads agent rules from `GEMINI.md` (project root) and `~/.gemini/GEMINI.md` (user-level). Drop the protocol there:

```markdown
## Thoughtline persistent memory

You have access to thoughtline memory tools (namespace `tl`).

Save proactively after: decisions, conventions, bug fixes, non-obvious feature work, gotchas, user preferences. Use this content shape:

**What** / **Why** / **Where** / **Learned**

Use `topic_key` of the form `category/subject` (lowercase, slash-separated). Re-saves on the same key upsert.

Search proactively when the user references prior work: `tl_search` first, then `tl_get_observation` for full content.

Close working blocks with `tl_session_summary` containing Goal / Discoveries / Accomplished / Next Steps / Relevant Files.
```

## Tagging tip for Gemini users

Use `tool:gemini-cli` on save and pass `agent_label: "gemini-cli"` to `tl_session_start`:

```jsonc
{ "agent_label": "gemini-cli", "tags": ["tool:gemini-cli"] }
```

See [`design/tag-conventions.md`](../design/tag-conventions.md) for the canonical vocabulary.
