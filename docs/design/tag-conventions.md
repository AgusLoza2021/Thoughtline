# Tag conventions

Tags are free-form, but consistency is what makes search across hundreds of memories actually work. This is the canonical lowercase `key:value` vocabulary for the gamedev memories in this repository. Use it as-is, and extend it where your project demands it.

Tags are **lowercased** and use `key:value` form (or a single token when there's no ambiguity). Multiple values per memory are fine and encouraged: a single texture import gotcha can carry `engine:unity`, `platform:android`, `asset:texture`, `pipeline:fbx-to-unity`.

## Where tags actually live

The envelope has a `tags` field, and `tl_search` filters on it, so a tag you expect to query by belongs there:

```
tl_save(type="bugfix", tags=["engine:unity", "pipeline:fbx", "phase:greybox"])
```

The field takes a list of strings, and each one has to match `^[a-z0-9][a-z0-9:_-]{0,40}$` — lowercase letters and digits, plus `:` and `-`, starting with a letter or a digit, at most 41 characters. An uppercase or spaced tag is rejected rather than normalised for you, and the error names the rule.

Two other places a tag can live:

1. **Inside the `topic_key`**, when the type's key pattern has a slot for it. `perf/android/static-batching` already carries `platform:android`; `scene/playcanvas/interactive-prop` already carries `engine:playcanvas`. The key patterns are listed per type in [`memory-domain.md`](memory-domain.md). This is identity rather than metadata: the key is what an upsert matches on, so a platform or an engine the pattern has a slot for belongs in the key.
2. **On a `**Tags**:` first line inside `content`** — comma-separated, drawn from the namespaces below. It still works, and it is still what a human reading raw markdown sees. Full-text search indexes the body, so those tags stay *findable*. They cannot be filtered, and an edit that rewrites the body can lose them without warning.

> **Use the field.** A tag that only exists on the first line is invisible to `tl_search(tags=[...])`, which is the query you will want next month.

## Engine

| Tag | When to use |
|-----|-------------|
| `engine:unity` | Anything Unity-specific (MonoBehaviour, ScriptableObject, .meta files, AssetBundles) |
| `engine:godot` | Godot 4.x/3.x scenes, GDScript, signals, NodePath patterns |
| `engine:unreal` | UE 5.x/4.x, Blueprints, C++ classes, UASSET workflows |
| `engine:playcanvas` | PlayCanvas Editor, scripts, asset registry |
| `engine:bevy` | Bevy ECS, plugins, schedule ordering |
| `engine:custom` | Your own engine, no library dependency |

## Platform

| Tag | When to use |
|-----|-------------|
| `platform:windows` | Windows builds, DirectX, Win-specific quirks |
| `platform:macos` | Apple Silicon, Metal, notarization |
| `platform:linux` | Vulkan, Steam Deck, Proton |
| `platform:android` | APK / AAB, ARM, Vulkan vs GLES |
| `platform:ios` | iOS builds, App Store, Metal |
| `platform:web` | WebGL / WebGPU, browser quirks, mobile-web |
| `platform:switch` | Switch SDK, NSP / NSO, perf budgets |
| `platform:playstation` | PS4/PS5, devkit, TRC |
| `platform:xbox` | Xbox One/Series, devkit, XR |
| `platform:steam` | Steam features (achievements, workshop, deck) |

## Pipeline

Capture the source → target hop so you can search "everything that affects FBX imports".

| Tag | When to use |
|-----|-------------|
| `pipeline:blender-to-unity` | Blender export → Unity import workflow |
| `pipeline:blender-to-godot` | Blender → Godot via .blend or .glb |
| `pipeline:blender-to-unreal` | Blender → Unreal via FBX or USD |
| `pipeline:fbx` | Anything FBX (export settings, scale mangle, axis flip) |
| `pipeline:gltf` | glTF / GLB pipeline |
| `pipeline:usd` | USD / USDZ pipeline |
| `pipeline:substance` | Substance Painter / Designer outputs |
| `pipeline:photoshop` | PSD authoring + export to engine |
| `pipeline:audio` | DAW → engine sound pipeline (Wwise, FMOD, native) |

## Asset category

| Tag | When to use |
|-----|-------------|
| `asset:mesh` | Static and skeletal meshes |
| `asset:texture` | Albedo, normal, roughness, lightmaps |
| `asset:material` | Material graphs / shader instances |
| `asset:shader` | Custom shaders, HLSL/GLSL/USF |
| `asset:animation` | Anim clips, state machines, blend trees |
| `asset:vfx` | Particle systems, Niagara, VFX Graph |
| `asset:audio` | SFX, music, voice |
| `asset:ui` | UI prefabs, UMG, UI Toolkit, Control nodes |
| `asset:level` | Scenes, levels, world chunks |
| `asset:prefab` | Prefab / Blueprint / PackedScene reusables |

## Phase

Where in the project lifecycle the memory was captured. Useful for retrospectives.

| Tag | When to use |
|-----|-------------|
| `phase:prototype` | Throwaway code, GDD experiments |
| `phase:vertical-slice` | First playable, content-light |
| `phase:alpha` | Feature complete, content incomplete |
| `phase:beta` | Content complete, polish + bugs |
| `phase:gold` | Shipping / shipped |
| `phase:post-launch` | Patches, DLC, live ops |

## Tooling

| Tag | When to use |
|-----|-------------|
| `tool:rider` | JetBrains Rider (Unity scripting) |
| `tool:visual-studio` | Visual Studio (Unreal C++) |
| `tool:vscode` | VS Code |
| `tool:cursor` | Cursor IDE |
| `tool:zed` | Zed editor |
| `tool:claude-code` | Claude Code agent |
| `tool:cli` | Command-line tooling, build scripts |

## Performance buckets

Use these when the memory is a `perf-gotcha` — they keep the search results sharp.

| Tag | When to use |
|-----|-------------|
| `perf:drawcalls` | Draw call count / batching |
| `perf:gc` | Garbage collection / allocations |
| `perf:cpu` | CPU-bound frames |
| `perf:gpu` | GPU-bound frames |
| `perf:memory` | RAM / VRAM budgets, leaks |
| `perf:loadtime` | Cold start, level load, streaming |
| `perf:network` | Multiplayer latency, bandwidth |

## Examples

These are real `tl_save` payloads, abbreviated with `...` where the per-type required sections go. Note where each tag ends up: platform and engine ride inside the `topic_key` when there is a slot for them, and the `tags` field carries the rest.

```jsonc
// Texture import gotcha (Android) — the platform is inherent in the key,
// and the remaining namespaces travel in the field
{
  "type": "perf-gotcha",
  "topic_key": "perf/android/texture-import",
  "tags": ["engine:unity", "asset:texture", "pipeline:fbx", "perf:memory"],
  "title": "Cap texture max size at 1024 on Android",
  "content": "**Symptom**: ...\n**Root cause**: ...\n**Fix**: ..."
}

// Blueprint vs C++ decision (UE5) — `decision/<area>/<choice>` has no engine
// slot, so the engine tag has to travel in the field
{
  "type": "decision",
  "topic_key": "decision/gameplay/blueprint-vs-cpp",
  "tags": ["engine:unreal", "phase:vertical-slice", "tool:visual-studio"],
  "title": "Gameplay in C++, content wiring in Blueprints",
  "content": "**What**: ...\n**Why**: ..."
}

// Godot inventory UI scene pattern — the engine slot is present, and the tag
// is carried anyway, because a key is easier to change than it is to re-find
{
  "type": "scene-pattern",
  "topic_key": "scene/godot/inventory-ui",
  "tags": ["engine:godot", "asset:ui", "asset:prefab"],
  "title": "Inventory UI: Control node over a pooled list",
  "content": "**Pattern**: ..."
}

// Steam Deck perf budget — no engine tag, because no engine is the subject
{
  "type": "perf-gotcha",
  "topic_key": "perf/linux/steam-deck-budget",
  "tags": ["platform:linux", "platform:steam", "perf:gpu", "perf:loadtime"],
  "title": "Steam Deck: hold 33 ms at 800p in the cellar room",
  "content": "**Symptom**: ...\n**Fix**: ..."
}
```

## Adding new tags

If you're reaching for a tag that isn't here:

1. **Check first** — `tl_search` for the concept; you may be inventing a synonym.
2. **Stay lowercase + colon-separated** — `engine:unity` not `Engine_Unity`.
3. **Add it to this doc** in a PR — that's how the canonical list grows.
