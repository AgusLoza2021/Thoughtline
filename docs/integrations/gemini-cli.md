# Gemini CLI

## Using Thoughtline with Gemini CLI today

Install Thoughtline, then register it in `~/.gemini/settings.json`. What this page teaches is the layer that outlives any engine: which `type` to save under, where tags belong, and what a body worth re-reading looks like. It lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The observation the old step 3 made still holds: **Gemini CLI only learns rules from `GEMINI.md`** (project root and `~/.gemini/`); it does not auto-load skill files. Its agent has to be told when to save and in what shape — true of Thoughtline.

Useful while you are there: Gemini CLI's `includeTools` / `excludeTools` keys let you allowlist specific tools. For a read-only Gemini session, name the read side — `["tl_search", "tl_get_observation", "tl_context", "tl_stats"]` — and leave the writers out.

### Rules to drop into `GEMINI.md`

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
  "type": "bugfix",
  "topic_key": "bug/unity/editor-import-hang",
  "title": "Unity hangs on import after a Blender re-export",
  "content": "**Tags**: engine:unity, pipeline:blender-to-unity, tool:gemini-cli\n\n**What**: Cleared Library/ArtifactDB after the .blend re-export. **Why**: Unity reused a stale artifact and the importer deadlocked on the same GUID. **Learned**: a re-export that keeps asset names but changes mesh topology is the trigger; a full reimport is cheaper than debugging it."
}
```

Tag Gemini-driven saves with `tool:gemini-cli` on that `**Tags**:` line, so they are recognisable later.

---

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
