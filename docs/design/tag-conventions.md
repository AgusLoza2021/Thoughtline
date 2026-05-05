# Tag conventions

Tags are free-form, but consistency is what makes search across hundreds of memories actually work. This is the canonical lowercase `key:value` vocabulary Thoughtline recommends. Use it as-is, extend where your project demands it.

Tags are **lowercased** and use `key:value` form (or a single token when there's no ambiguity). Multiple values per memory are fine and encouraged: a single texture import gotcha can carry `engine:unity`, `platform:android`, `asset:texture`, `pipeline:fbx-to-unity`.

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

```jsonc
// Texture import gotcha on Android
{ "tags": ["engine:unity", "platform:android", "asset:texture", "pipeline:fbx", "perf:memory"] }

// Blueprint vs C++ decision in UE5
{ "tags": ["engine:unreal", "phase:vertical-slice", "tool:visual-studio"] }

// Godot scene-pattern for inventory UI
{ "tags": ["engine:godot", "asset:ui", "asset:prefab"] }

// Steam Deck perf budget
{ "tags": ["platform:linux", "platform:steam", "perf:gpu", "perf:loadtime"] }
```

## Adding new tags

If you're reaching for a tag that isn't here:

1. **Check first** — `tl_search` for the concept; you may be inventing a synonym.
2. **Stay lowercase + colon-separated** — `engine:unity` not `Engine_Unity`.
3. **Add it to this doc** in a PR — that's how the canonical list grows.
