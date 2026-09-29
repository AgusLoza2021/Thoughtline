# Zed

## Using this vocabulary with Zed today

The server this page used to teach is retired. The server you want now is [Engram](https://github.com/Gentleman-Programming/engram) — install it and point Zed at it by following Engram's own instructions. What survives from this repository is the layer *above* the server: which `type` to save under, where tags belong, and what a body worth re-reading looks like. That layer was never engine-bound, and it lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The other thing that outlived the server is the observation the old step 3 made: **Zed's assistant does not auto-load skill files.** Its agent has to be told when to save and in what shape — true of Thoughtline, still true of Engram.

### Rules to drop into `.zed/rules.md`

The protocol text never belonged to the server, so it is unchanged apart from the tool names:

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
  "content": "**Tags**: engine:unity, tool:zed\n\n**What**: Assets/_Game/{Scenes, Scripts, Art, Audio, Prefabs, ScriptableObjects}. **Why**: keeps Unity's auto-imports out of source-controlled team folders. **Learned**: the leading underscore keeps our folders at the top of the Project window."
}
```

Tag Zed-driven saves with `tool:zed` on that `**Tags**:` line, so they are recognisable later.

---

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
