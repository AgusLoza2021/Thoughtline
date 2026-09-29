# Agent setup

## Tell the agent the protocol today

Most editors don't auto-load skill protocols the way Claude Code does. You have
to tell the agent **when** to save and **what** a memory worth keeping looks
like. The server ships the tools; the instruction is still yours to write.

The block below is reusable across editors. Drop it into your project's rules
file — `.cursorrules`, `.zed/rules.md`, `~/.config/opencode/AGENTS.md`,
`GEMINI.md` — or, for Claude Code, take the longer standalone version in
[integrations/claude-code-protocol.md](integrations/claude-code-protocol.md).

It is maintained here and _only_ here. The per-editor guides below refer to it
rather than repeating it: a copy of this block in seven files is how the version
you are reading went stale in all seven at once.

### The protocol block

```markdown
## Persistent memory

You have Thoughtline's memory tools available.

Save proactively with `tl_save` after: a decision, a convention, a bug fix,
non-obvious feature work, a gotcha, or a stated preference.

Set `type` from the catalogue: `discovery`, `architecture`, `bugfix`,
`decision`, `config`, `pattern`, `preference`, plus the gamedev extensions
`convention`, `perf-gotcha`, `pipeline-step`, `script-pattern`,
`scene-pattern`, `asset-reference` and `game-design-decision`. Always pass it -
the server checks the field against that list and refuses anything else, so an
invented type is a failed save rather than a memory nobody will find again.

`preference` uses `scope: "personal"`; everything else uses `scope: "project"`
(the default). There is no third value - a save asking for `global` is refused.

Tags go in the `tags` field, a list of lowercase `key:value` strings:

tl_save(tags=["engine:unity", "platform:android", "pipeline:fbx"])

`tl_search` filters on that field and `tl_get_observation` prints it back. A
`**Tags**:` first line in `content` still works and is what a human reading raw
markdown sees, but it is indexed as body text: it can be searched, not
filtered, and an edit to the body can lose it.

`topic_key` is `category/subject`, lowercase and slash-separated, e.g.
`convention/unity/folder-layout`. Re-saving the same key REPLACES the title,
content and tags rather than appending, so reuse a key only for a topic that
evolves. A query containing `/` is matched against `topic_key` before full-text
search runs, so the key you choose is also the cheapest way to find it again.

Search proactively when the user refers to earlier work: `tl_context` first,
then `tl_search` with keywords, then `tl_get_observation` for the full record.

Keep the memory good without being asked: `tl_update` to correct or extend one
instead of saving the same topic twice; `tl_delete` for a claim you know is
false; `tl_link` to relate two memories; `tl_stats` to see what the store
holds. There is no pin and no review queue - if a memory must not be missed,
say so in its content, and if one has aged out, retire it yourself.

Close a working block with `tl_session_summary`: Goal / Discoveries /
Accomplished / Next Steps / Relevant Files. End a completed task with a
`## Key Learnings:` section and the small learnings get captured for free.
```

### A complete save, to see the shape

```jsonc
{
  "type": "bugfix",
  "topic_key": "bug/unity/editor-import-hang",
  "tags": ["engine:unity", "pipeline:blender-to-unity", "tool:cursor"],
  "title": "Unity hangs on import after a Blender re-export",
  "content": "**What**: Cleared Library/ArtifactDB after the .blend re-export. **Why**: Unity reused a stale artifact and the importer deadlocked on the same GUID. **Learned**: a re-export that keeps asset names but changes mesh topology is the trigger; a full reimport is cheaper than debugging it."
}
```

Tag the saves you make from your editor with `tool:<editor>` in `tags`. The
`tool:` namespace is listed in the [tag conventions](design/tag-conventions.md),
and `tl_search(tags=["tool:cursor"])` finds every memory that carries it.

---

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

## Verify any setup

```bash
thoughtline ui
```

Open the dashboard and watch the Sessions and Browse tabs while you ask the agent something it should remember. If `tl_save` was called, you'll see the new memory show up. If not, the protocol prompt didn't reach the agent — re-check the rules file.

---

## Multiple editors, one database

It's fine to wire Thoughtline into **all** of them at the same time. They all read the same SQLite file. The `agent_label` on each session lets the dashboard show which editor was driving when.

What's not OK is running **two MCP servers in parallel** against the same database file (e.g. one Claude Code session and one Cursor session at the same time). SQLite handles concurrent reads fine but Thoughtline assumes a single writer. If you want to multiplex, use separate `THOUGHTLINE_DB` paths per agent and merge later — or just don't run two assistants at the same time, which is the saner answer.
