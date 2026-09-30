# JetBrains Rider (Unity)

## Using Thoughtline with Rider + Unity today

Install Thoughtline, then add it under `Settings → Tools → AI Assistant → MCP servers`. What this page teaches was never about the server: **the Unity vocabulary** — which `type` a Unity lesson belongs under, which tags it carries, and what a body worth re-reading looks like. It lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The habit the old step 3 recommended still holds: **keep the memory rules in the project, and point the AI Assistant's "additional context" setting at them.** Rider does not auto-load skill files either, so the agent still has to be told when to save and in what shape.

### Rules to keep in `Assets/_AI/RULES.md`

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

### Unity saves to make on day 1

The three below are the ones this page always recommended, now expressed in the contract Thoughtline accepts. Capture them early — your future self will thank you.

```jsonc
// Project structure
{
  "type": "convention",
  "topic_key": "convention/unity/folder-layout",
  "title": "Unity folder layout for <project>",
  "content": "**Tags**: engine:unity, tool:rider\n\n**What**: Assets/_Game/{Scenes, Scripts, Art, Audio, Prefabs, ScriptableObjects}; third-party in Assets/_ThirdParty; tooling in Assets/_AI. **Why**: keeps Unity's auto-imports out of source-controlled team folders. **Learned**: the leading underscore keeps our folders at the top of the Project window."
}
```

```jsonc
// Scripting style
{
  "type": "convention",
  "topic_key": "convention/unity/csharp-style",
  "title": "MonoBehaviour conventions",
  "content": "**Tags**: engine:unity, tool:rider\n\n**What**: SerializeField private fields, no public. **Why**: keeps inspector tweakable without breaking encapsulation. **Where**: enforced via Rider .editorconfig + InspectorTooLargeException analyzer."
}
```

```jsonc
// Build pipeline gotchas
{
  "type": "perf-gotcha",
  "topic_key": "perf/android/asset-bundles",
  "title": "Asset bundles balloon the APK on Android",
  "content": "**Tags**: engine:unity, platform:android, perf:memory, tool:rider\n\n**What**: ASTC 6x6 + LZ4 compression on bundles, never the default. **Why**: default ETC2 + LZMA gave 220MB APKs. **Learned**: Player Settings -> Other -> Texture compression must match the per-bundle override, or Unity silently recompresses."
}
```

Tag Rider-driven saves with `tool:rider` on that `**Tags**:` line, so they are recognisable later.

---

[Rider](https://www.jetbrains.com/rider/) added MCP support via the AI Assistant plugin. This page is the canonical setup for Unity developers using Rider.

## 1. Install the binary

```powershell
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
thoughtline version
```

Or a release binary from [the latest release](https://github.com/AgusLoza2021/Thoughtline/releases/latest).

## 2. Register Thoughtline as an MCP server

Open `Settings → Tools → AI Assistant → MCP servers → Add` and configure:

| Field   | Value          |
|---------|----------------|
| Name    | `thoughtline`  |
| Command | `thoughtline`  |
| Args    | (leave empty)  |
| Enabled | ✅              |

Apply, then restart the AI Assistant from the toolbar (it picks up the new server on next agent invocation).

## 3. Pin the protocol

Drop the active-protocol markdown into your project's `Assets/_AI/RULES.md` (or whatever convention your team uses). The simplest way is to capture it once with the binary itself:

```powershell
thoughtline protocol --event session-start --project unity-game-name -o Assets/_AI/RULES.md
```

Then point the AI Assistant's "additional context" setting at that file. The agent now knows the `tl_*` tools exist and when to call them.
