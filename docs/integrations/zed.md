# Zed

<!-- retired-v0.1.0 -->
> **Retired — the v0.1.0 MCP server this page was written for is unmaintained.**
> Its install and wiring steps are kept at the bottom as a record of how the project
> worked, not as a path to follow. For what this project is now — a gamedev memory
> vocabulary that runs on
> [Engram](https://github.com/Gentleman-Programming/engram) — read the
> [README](../../README.md), the [memory domain](../design/memory-domain.md) and the
> [tag conventions](../design/tag-conventions.md).

## Using this vocabulary with Zed today

The server this page used to teach is retired. The server you want now is [Engram](https://github.com/Gentleman-Programming/engram) — install it and point Zed at it by following Engram's own instructions. What survives from this repository is the layer *above* the server: which `type` to save under, where tags belong, and what a body worth re-reading looks like. That layer was never engine-bound, and it lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The other thing that outlived the server is the observation the old step 3 made: **Zed's assistant does not auto-load skill files.** Its agent has to be told when to save and in what shape — true of Thoughtline, still true of Engram.

### Rules to drop into `.zed/rules.md`

The protocol text never belonged to the server, so it is unchanged apart from the tool names:

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
  "content": "**Tags**: engine:unity, tool:zed\n\n**What**: Assets/_Game/{Scenes, Scripts, Art, Audio, Prefabs, ScriptableObjects}. **Why**: keeps Unity's auto-imports out of source-controlled team folders. **Learned**: the leading underscore keeps our folders at the top of the Project window."
}
```

Tag Zed-driven saves with `tool:zed` on that `**Tags**:` line, so they are recognisable later.

---

> **Legacy — the v0.1.0 setup.** Everything below documents `thoughtline`, the
> retired server, and its `tl_*` tools. It is a record of how the project worked,
> not instructions to follow.

[Zed](https://zed.dev) added MCP support to its assistant in late 2025. Wire Thoughtline as a context server.

## 1. Install the binary

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
thoughtline version
```

Or a release binary from [the latest release](https://github.com/AgusLoza2021/Thoughtline/releases/latest).

## 2. Register Thoughtline

Open `Zed → Settings → Open Settings (JSON)` (or `cmd+,` then "Open settings file") and add the entry under `assistant.context_servers`:

```jsonc
{
  "assistant": {
    "context_servers": {
      "thoughtline": {
        "source": "custom",
        "command": "thoughtline",
        "args": []
      }
    }
  }
}
```

Reload the window (`cmd/ctrl+shift+P → reload window`). The `tl_*` tools will appear in the assistant's tool list.
