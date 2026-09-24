# Windsurf (Codeium Cascade)

<!-- retired-v0.1.0 -->
> **Retired — the v0.1.0 MCP server this page was written for is unmaintained.**
> Its install and wiring steps are kept at the bottom as a record of how the project
> worked, not as a path to follow. For what this project is now — a gamedev memory
> vocabulary that runs on
> [Engram](https://github.com/Gentleman-Programming/engram) — read the
> [README](../../README.md), the [memory domain](../design/memory-domain.md) and the
> [tag conventions](../design/tag-conventions.md).

## Using this vocabulary with Windsurf today

The server this page used to teach is retired. The server you want now is [Engram](https://github.com/Gentleman-Programming/engram) — install it and register it in Windsurf's `mcp_config.json` by following Engram's own instructions. What survives from this repository is the layer *above* the server: which `type` to save under, where tags belong, and what a body worth re-reading looks like. That layer was never engine-bound, and it lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The other thing that outlived the server is the observation the old step 3 made: **Windsurf does not auto-load skill protocols.** Its agent has to be told when to save and in what shape — true of Thoughtline, still true of Engram.

> **Still true, and worth more attention than it used to get — tool cap.** Windsurf documents a limit of **100 total tools** that Cascade can access at once. The retired server added 9; Engram's own tool set is larger than that, so if you run several MCP servers at once, this is the number to watch. Disable servers you are not using rather than trimming memory tools.

### Rules to drop into `.windsurfrules`

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
  "type": "perf-gotcha",
  "topic_key": "perf/android/asset-bundles",
  "title": "Asset bundles balloon the APK on Android",
  "content": "**Tags**: engine:unity, platform:android, perf:memory, tool:windsurf\n\n**What**: ASTC 6x6 + LZ4 compression on bundles, never the default. **Why**: default ETC2 + LZMA gave 220MB APKs. **Learned**: Player Settings -> Other -> Texture compression must match the per-bundle override, or Unity silently recompresses."
}
```

Tag Windsurf-driven saves with `tool:windsurf` on that `**Tags**:` line, so they are recognisable later.

---

> **Legacy — the v0.1.0 setup.** Everything below documents `thoughtline`, the
> retired server, and its `tl_*` tools. It is a record of how the project worked,
> not instructions to follow.

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

> **Enterprise users**: MCP must be turned on manually in settings. Ask your admin if Cascade doesn't show the tools.
