# Cursor

## Using this vocabulary with Cursor today

The server this page used to teach is retired. The server you want now is [Engram](https://github.com/Gentleman-Programming/engram) — install it and point Cursor at it by following Engram's own instructions. What survives from this repository is the layer *above* the server: which `type` to save under, where tags belong, and what a body worth re-reading looks like. That layer was never engine-bound, and it lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The other thing that outlived the server is the observation the old step 3 made: **Cursor does not auto-load skill files.** Its agent has to be told when to save and in what shape — that was true of Thoughtline and it is still true of Engram. Paste this into `.cursorrules`, or into your global rules:

```markdown
## Persistent memory — the words

You have Engram's memory tools available.

Set `type` from this vocabulary: `decision`, `convention`, `bugfix`,
`perf-gotcha`, `pipeline-step`, `script-pattern`, `scene-pattern`,
`asset-reference`, `game-design-decision`, `architecture`, or `preference`.
Always pass it - Engram does not validate the field, its default is `manual`,
and a memory typed `manual` is not in this vocabulary.

`preference` uses `scope: "personal"`; everything else uses `scope: "project"`
(the default). Engram also accepts `global`; this vocabulary does not use it.

The first line of `content` is a `**Tags**:` line, comma-separated, in
`key:value` form:

**Tags**: engine:unity, platform:android, pipeline:fbx

Engram has no tags field, so that line is where tags live - and its full-text
search indexes the body, so the line stays findable. You cannot filter by tag;
it is an aid to recall, not an index.

`topic_key` is `category/subject`, lowercase and slash-separated, e.g.
`convention/unity/folder-layout`. Re-saving the same key REPLACES the title and
content rather than appending, so reuse a key only for a topic that evolves.
```

The behaviour block that goes with it — when to save, how to search, how to
correct a memory rather than save it twice — is kept in one place, at
[`docs/AGENT-SETUP.md`](../AGENT-SETUP.md) in the Thoughtline repository. It is
not repeated here: a copy of that block in seven files went stale in all seven
at once.

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
