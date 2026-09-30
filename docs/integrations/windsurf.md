# Windsurf (Codeium Cascade)

## Using Thoughtline with Windsurf today

Install Thoughtline, then register it in Windsurf's `mcp_config.json`. What this page teaches is the layer that outlives any engine: which `type` to save under, where tags belong, and what a body worth re-reading looks like. It lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The observation the old step 3 made still holds: **Windsurf does not auto-load skill protocols.** Its agent has to be told when to save and in what shape — true of Thoughtline.

> **Worth more attention than it used to get — tool cap.** Windsurf documents a limit of **100 total tools** that Cascade can access at once. Thoughtline contributes a `tl_*` set of its own, so if you run several MCP
servers at once, this is the number to watch. Disable servers you are not using rather than trimming memory tools.

### Rules to drop into `.windsurfrules`

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
  "topic_key": "perf/android/asset-bundles",
  "title": "Asset bundles balloon the APK on Android",
  "content": "**Tags**: engine:unity, platform:android, perf:memory, tool:windsurf\n\n**What**: ASTC 6x6 + LZ4 compression on bundles, never the default. **Why**: default ETC2 + LZMA gave 220MB APKs. **Learned**: Player Settings -> Other -> Texture compression must match the per-bundle override, or Unity silently recompresses."
}
```

Tag Windsurf-driven saves with `tool:windsurf` on that `**Tags**:` line, so they are recognisable later.

---

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
