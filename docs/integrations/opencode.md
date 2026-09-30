# OpenCode

## Using Thoughtline with OpenCode today

Install Thoughtline, then register it in OpenCode's config. What this page teaches is the layer that outlives any engine: which `type` to save under, where tags belong, and what a body worth re-reading looks like. It lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The observation the old step 3 made still holds: **OpenCode only learns rules from `AGENTS.md`** (falling back to user-level rules); it does not auto-load skill files. Its agent has to be told when to save and in what shape — true of Thoughtline.

### Rules to drop into `AGENTS.md`

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
  "type": "perf-gotcha",
  "topic_key": "perf/unity/android-asset-bundles",
  "title": "Asset bundles balloon the APK on Android",
  "content": "**Tags**: engine:unity, platform:android, perf:memory, tool:opencode\n\n**What**: ASTC 6x6 + LZ4 compression on bundles, never the default. **Why**: default ETC2 + LZMA gave 220MB APKs. **Learned**: Player Settings -> Other -> Texture compression must match the per-bundle override, or Unity silently recompresses."
}
```

Tag OpenCode-driven saves with `tool:opencode` on that `**Tags**:` line, so they are recognisable later.

---

[OpenCode](https://opencode.ai) — sst's open-source terminal AI agent — supports MCP via its config file. Wire Thoughtline as a local stdio MCP server.

## 1. Make sure the binary is on your PATH

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
thoughtline version
```

Or use a release binary from [the latest release](https://github.com/AgusLoza2021/Thoughtline/releases/latest). On Windows, the [auto-installer](../../scripts/install.ps1) handles this for you.

## 2. Register Thoughtline in OpenCode's config

OpenCode reads:

- **Global** config: `~/.config/opencode/opencode.json`
- **Project** config: `opencode.json` or `.opencode/opencode.json` in the project root

Add Thoughtline under the top-level `mcp` key. **Note**: OpenCode's `command` is an **array of strings**, not a single string.

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "thoughtline": {
      "type": "local",
      "command": ["thoughtline", "serve"],
      "enabled": true
    }
  }
}
```

You can also pass environment variables under `environment` if you want to pin the project or DB path:

```json
{
  "mcp": {
    "thoughtline": {
      "type": "local",
      "command": ["thoughtline", "serve"],
      "enabled": true,
      "environment": {
        "THOUGHTLINE_PROJECT": "my-game-project"
      }
    }
  }
}
```

To temporarily turn it off without removing the entry: `"enabled": false`.

Restart OpenCode (or start a new session) and the `tl_*` tools become available.
