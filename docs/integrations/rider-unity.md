# JetBrains Rider (Unity)

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

## Unity-specific saves to make on day 1

Capture these early — your future self will thank you:

```jsonc
// Project structure
{
  "type": "convention",
  "topic_key": "convention/unity/folder-layout",
  "title": "Unity folder layout for <project>",
  "tags": ["engine:unity"],
  "content": "**What**: Assets/_Game/{Scenes, Scripts, Art, Audio, Prefabs, ScriptableObjects}; third-party in Assets/_ThirdParty; tooling in Assets/_AI. **Why**: keeps Unity's auto-imports out of source-controlled team folders. **Learned**: prefix _ keeps our folders at the top of the Project window."
}

// Scripting style
{
  "type": "convention",
  "topic_key": "convention/unity/csharp-style",
  "title": "MonoBehaviour conventions",
  "tags": ["engine:unity", "tool:rider"],
  "content": "**What**: SerializeField private fields, no public. **Why**: keeps inspector tweakable without breaking encapsulation. **Where**: enforced via Rider .editorconfig + InspectorTooLargeException analyzer."
}

// Build pipeline gotchas
{
  "type": "perf-gotcha",
  "topic_key": "perf/unity/android-asset-bundles",
  "title": "Asset bundles balloon APK on Android",
  "tags": ["engine:unity", "platform:android", "perf:memory"],
  "content": "**What**: ASTC 6x6 + LZ4 compression on bundles, never default. **Why**: default ETC2 + LZMA gave 220MB APKs. **Learned**: Player Settings → Other → Texture compression must match the per-bundle override or Unity silently re-recompresses."
}
```

## Tagging tip for Rider users

Tag memories with `tool:rider`. Pass `agent_label: "rider"` to `tl_session_start`:

```jsonc
{ "agent_label": "rider" }
```

That groups Rider activity in the dashboard's session list.
