# Agent setup

Thoughtline speaks the **Model Context Protocol** over stdio, so anything that's an MCP client can use it. This page is the index of per-tool setup guides — one click and you have the right config block for your IDE.

> Already installed `thoughtline`? Skip to your editor below. If not, see [INSTALLATION.md](INSTALLATION.md) first.

---

## Per-editor guides

| Editor | Status | Guide |
|---|---|---|
| [Claude Code](https://docs.claude.com/en/docs/claude-code) (CLI + VS Code extension) | First-class | [INSTALLATION.md § Register the MCP server](INSTALLATION.md#register-the-mcp-server) and the [Claude Code plugin](../plugin/claude-code/README.md) |
| [Cursor](https://cursor.com) | Tested | [integrations/cursor.md](integrations/cursor.md) |
| [Zed](https://zed.dev) | Tested | [integrations/zed.md](integrations/zed.md) |
| [Windsurf](https://windsurf.com) (Codeium Cascade) | Tested | [integrations/windsurf.md](integrations/windsurf.md) |
| [OpenCode](https://opencode.ai) | Tested | [integrations/opencode.md](integrations/opencode.md) |
| [Gemini CLI](https://github.com/google-gemini/gemini-cli) | Tested | [integrations/gemini-cli.md](integrations/gemini-cli.md) |
| [JetBrains Rider](https://www.jetbrains.com/rider/) (Unity workflow) | Tested | [integrations/rider-unity.md](integrations/rider-unity.md) |

---

## The minimal MCP server entry

Every guide is a variation on the same idea: register an MCP server that runs `thoughtline serve` over stdio. The block looks roughly the same everywhere — only the **file location** and the **top-level key** change.

```jsonc
{
  "thoughtline": {
    "command": "thoughtline",
    "args": ["serve"]
  }
}
```

Where to drop it:

| Editor | Config file | Wrap key |
|---|---|---|
| Claude Code | `~/.claude.json` (Win: `%USERPROFILE%\.claude.json`) | `mcpServers` |
| Cursor | `~/.cursor/mcp.json` (Win: `%USERPROFILE%\.cursor\mcp.json`) | `mcpServers` |
| Zed | `~/.config/zed/settings.json` | `assistant.context_servers` |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` | `mcpServers` |
| OpenCode | `~/.config/opencode/opencode.json` (or project `opencode.json`) | `mcp` |
| Gemini CLI | `~/.gemini/settings.json` (or project `.gemini/settings.json`) | `mcpServers` |

Each per-editor guide gives you the exact wrapper plus tool-specific tips (tagging conventions, where to drop the protocol prompt, agent_label values for `tl_session_start`).

---

## Tagging your agent

Every per-editor guide ends with a "tagging tip" that recommends `tool:<editor>` so the dashboard's Tags tab can group memories by where they were captured. The canonical list:

| Editor | Recommended tag | `agent_label` for `tl_session_start` |
|---|---|---|
| Claude Code | `tool:claude-code` | `"claude-code"` |
| Cursor | `tool:cursor` | `"cursor"` |
| Zed | `tool:zed` | `"zed"` |
| Windsurf | `tool:windsurf` | `"windsurf"` |
| OpenCode | `tool:opencode` | `"opencode"` |
| Gemini CLI | `tool:gemini-cli` | `"gemini-cli"` |
| Rider (Unity) | `tool:rider` | `"rider"` |

See [docs/design/tag-conventions.md](design/tag-conventions.md) for the full taxonomy (engine tags, platform tags, pipeline tags).

---

## Tell the agent the protocol

Most editors don't auto-load skill protocols the way Claude Code does. You have to tell the agent **when** to call `tl_save` and `tl_search`, otherwise it won't.

The boilerplate is reusable across editors. Drop it into your project's rules file (`.cursorrules`, `.zed/rules.md`, `~/.config/opencode/AGENTS.md`, `GEMINI.md`, etc.):

```markdown
## Thoughtline persistent memory

You have access to thoughtline memory tools (namespace `tl`).

**Save proactively** after: decisions, conventions, bug fixes, non-obvious feature work, gotchas, user preferences. Use this content shape:

**What** / **Why** / **Where** / **Learned**

Use `topic_key` of the form `category/subject` (lowercase, slash-separated). Re-saves on the same key upsert.

**Search proactively** when the user references prior work: call `tl_search` first, then `tl_get_observation` for the full content of a hit.

**Bookend working sessions** with `tl_session_start` at the beginning and `tl_session_summary` at the end. The summary should contain Goal / Discoveries / Accomplished / Next Steps / Relevant Files.
```

The Claude Code plugin injects this automatically via `SessionStart` and post-compaction hooks (see [plugin/claude-code/](../plugin/claude-code/)).

---

## Verify any setup

```bash
thoughtline ui
```

Open the dashboard and watch the Sessions and Browse tabs while you ask the agent something it should remember. If `tl_save` was called, you'll see the new memory show up. If not, the protocol prompt didn't reach the agent — re-check the rules file.

---

## Multiple editors, one database

It's fine to wire Thoughtline into **all** of them at the same time. They all read the same SQLite file. The `agent_label` on each session lets the dashboard show which editor was driving when.

What's not OK is running **two MCP servers in parallel** against the same database file (e.g. one Claude Code session and one Cursor session at the same time). SQLite handles concurrent reads fine but Thoughtline assumes a single writer. If you want to multiplex, use separate `THOUGHTLINE_DB` paths per agent and merge later — or just don't run two assistants at the same time, which is the saner answer.
