# Cursor

<!-- retired-v0.1.0 -->
> **Retired — the v0.1.0 MCP server this page was written for is unmaintained.**
> Its install and wiring steps are kept at the bottom as a record of how the project
> worked, not as a path to follow. For what this project is now — a gamedev memory
> vocabulary that runs on
> [Engram](https://github.com/Gentleman-Programming/engram) — read the
> [README](../../README.md), the [memory domain](../design/memory-domain.md) and the
> [tag conventions](../design/tag-conventions.md).

## Using this vocabulary with Cursor today

The server this page used to teach is retired. The server you want now is [Engram](https://github.com/Gentleman-Programming/engram) — install it and point Cursor at it by following Engram's own instructions. What survives from this repository is the layer *above* the server: which `type` to save under, where tags belong, and what a body worth re-reading looks like. That layer was never engine-bound, and it lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The other thing that outlived the server is the observation the old step 3 made: **Cursor does not auto-load skill files.** Its agent has to be told when to save and in what shape — that was true of Thoughtline and it is still true of Engram. Paste this into `.cursorrules`, or into your global rules:

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
  "type": "convention",
  "topic_key": "convention/unity/folder-layout",
  "title": "Unity folder layout for <project>",
  "content": "**Tags**: engine:unity, tool:cursor\n\n**What**: Assets/_Game/{Scenes, Scripts, Art, Audio, Prefabs, ScriptableObjects}. **Why**: keeps Unity's auto-imports out of source-controlled team folders. **Learned**: the leading underscore keeps our folders at the top of the Project window."
}
```

Tag Cursor-driven saves with `tool:cursor` on that `**Tags**:` line, so they are recognisable later.

---

> **Legacy — the v0.1.0 setup.** Everything below documents `thoughtline`, the
> retired server, and its `tl_*` tools. It is a record of how the project worked,
> not instructions to follow.

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
