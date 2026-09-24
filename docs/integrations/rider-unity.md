# JetBrains Rider (Unity)

<!-- retired-v0.1.0 -->
> **Retired — the v0.1.0 MCP server this page was written for is unmaintained.**
> Its install and wiring steps are kept at the bottom as a record of how the project
> worked, not as a path to follow. For what this project is now — a gamedev memory
> vocabulary that runs on
> [Engram](https://github.com/Gentleman-Programming/engram) — read the
> [README](../../README.md), the [memory domain](../design/memory-domain.md) and the
> [tag conventions](../design/tag-conventions.md).

## Using this vocabulary with Rider + Unity today

The server this page used to teach is retired. The server you want now is [Engram](https://github.com/Gentleman-Programming/engram) — install it and add it under `Settings → Tools → AI Assistant → MCP servers` by following Engram's own instructions. What survives from this repository is the part that was never about the server: **the Unity vocabulary** — which `type` a Unity lesson belongs under, which tags it carries, and what a body worth re-reading looks like. It lives in the [memory domain](../design/memory-domain.md) and the [tag conventions](../design/tag-conventions.md).

The other thing that outlived the server is the habit the old step 3 recommended: **keep the memory rules in the project, and point the AI Assistant's "additional context" setting at them.** Rider does not auto-load skill files either, so the agent still has to be told when to save and in what shape.

### Rules to keep in `Assets/_AI/RULES.md`

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

### Unity saves to make on day 1

The three below are the ones this page always recommended, now expressed in the contract Engram actually accepts. Capture them early — your future self will thank you.

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

> **Legacy — the v0.1.0 setup.** Everything below documents `thoughtline`, the
> retired server, and its `tl_*` tools. It is a record of how the project worked,
> not instructions to follow.

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
