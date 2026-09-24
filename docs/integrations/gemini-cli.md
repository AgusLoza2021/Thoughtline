# Gemini CLI

<!-- retired-v0.1.0 -->
> **Retired — the v0.1.0 MCP server this page was written for is unmaintained.**
> Its install and wiring steps are kept at the bottom as a record of how the project
> worked, not as a path to follow. For what this project is now — a gamedev memory
> vocabulary that runs on
> [Engram](https://github.com/Gentleman-Programming/engram) — read the
> [README](../../README.md), the [memory domain](../design/memory-domain.md) and the
> [tag conventions](../design/tag-conventions.md).

## Using this vocabulary with Gemini CLI today

The server this page used to teach is retired. The server you want now is [Engram](https://github.com/Gentleman-Programming/engram) — install it and register it in `~/.gemini/settings.json` by following Engram's own instructions. What survives from this repository is the layer *above* the server: which `type` to save under, where tags belong, and what a body worth re-reading looks like. That layer was never engine-bound, and it lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The other thing that outlived the server is the observation the old step 3 made: **Gemini CLI only learns rules from `GEMINI.md`** (project root and `~/.gemini/`); it does not auto-load skill files. Its agent has to be told when to save and in what shape — true of Thoughtline, still true of Engram.

Useful while you are there: Gemini CLI's `includeTools` / `excludeTools` keys let you allowlist specific tools. For a read-only Gemini session, name Engram's read side — `["mem_search", "mem_get_observation", "mem_context", "mem_stats"]` — and leave the writers out.

### Rules to drop into `GEMINI.md`

```markdown
## Persistent memory

You have Engram's memory tools available.

Save proactively after: a decision, a convention, a bug fix, non-obvious feature
work, a gotcha, or a stated preference.

Set `type` from the project's vocabulary: `decision`, `convention`, `bugfix`,
`perf-gotcha`, `pipeline-step`, `script-pattern`, `scene-pattern`,
`asset-reference`, `game-design-decision`, `architecture`, or `preference`
(which must use `scope: "personal"`; everything else is `project`).

The first line of `content` is a `**Tags**:` line, comma-separated, in
`key:value` form:

**Tags**: engine:unity, platform:android, pipeline:fbx

Engram has no tags field, so that line is where tags live - and Engram's
full-text search indexes the body, so the line stays findable. You cannot
filter by tag; it is an aid to recall, not an index.

`topic_key` is `category/subject`, lowercase and slash-separated, e.g.
`convention/unity/folder-layout`. Re-saving the same key REPLACES the title and
content rather than appending, so reuse a key only for a topic that evolves.

Search proactively with `mem_search` when the user refers to earlier work, then
`mem_get_observation` for the full record.

Close a working block with `mem_session_summary`: Goal / Discoveries /
Accomplished / Next Steps / Relevant Files.
```

### A complete save, to see the shape

```jsonc
{
  "type": "bugfix",
  "topic_key": "bug/unity/editor-import-hang",
  "title": "Unity hangs on import after a Blender re-export",
  "content": "**Tags**: engine:unity, pipeline:blender-to-unity, tool:gemini-cli\n\n**What**: Cleared Library/ArtifactDB after the .blend re-export. **Why**: Unity reused a stale artifact and the importer deadlocked on the same GUID. **Learned**: a re-export that keeps asset names but changes mesh topology is the trigger; a full reimport is cheaper than debugging it."
}
```

Tag Gemini-driven saves with `tool:gemini-cli` on that `**Tags**:` line, so they are recognisable later.

---

> **Legacy — the v0.1.0 setup.** Everything below documents `thoughtline`, the
> retired server, and its `tl_*` tools. It is a record of how the project worked,
> not instructions to follow.

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
