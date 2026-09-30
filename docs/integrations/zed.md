# Zed

## Using Thoughtline with Zed today

Install Thoughtline, then point Zed at it. What this page teaches is the layer that outlives any engine: which `type` to save under, where tags belong, and what a body worth re-reading looks like. It lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The observation the old step 3 made still holds: **Zed's assistant does not auto-load skill files.** Its agent has to be told when to save and in what shape — true of Thoughtline.

### Rules to drop into `.zed/rules.md`

The protocol text never belonged to the server, so it is unchanged apart from the tool names:

```markdown
## Persistent memory — the words

You have Thoughtline's memory tools available (`tl_*`).

Set `type` from this vocabulary: `decision`, `convention`, `bugfix`,
`perf-gotcha`, `pipeline-step`, `script-pattern`, `scene-pattern`,
`asset-reference`, `game-design-decision`, `architecture`, or `preference`.
Always pass it - Thoughtline validates the field, and a save whose type is
not in this vocabulary is rejected rather than filed under a default.

`preference` uses `scope: "personal"`; everything else uses `scope: "project"`
(the default). Thoughtline accepts those two and no others.

The first line of `content` is a `**Tags**:` line, comma-separated, in
`key:value` form:

**Tags**: engine:unity, platform:android, pipeline:fbx

`tl_save` also takes a `tags` array; this vocabulary keeps its tags on that
first line so a memory stays self-describing when it is copied, quoted or
moved between stores. Thoughtline's full-text search indexes the body, so the
line stays findable either way.

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
