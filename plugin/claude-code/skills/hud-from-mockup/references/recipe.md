# HUD from mockup: detailed recipe

Worked examples below are this repo's own. They show the **pattern** to follow when a reference
screen exists; they are not layouts to duplicate (see "Reference, never copy" in `../SKILL.md`).

`S` below is `Assets/GorodiskiGames/Survivalio`. Every other path is repo-relative.

## Worked examples

| | Garage menu (design D22) | Consent |
|---|---|---|
| Host | `Gameplay.unity`, view under `GameView.Huds` | prefab `S/Resources/Prefabs/ConsentHud.prefab` |
| Builder | `S/Scripts/Editor/GarageHudSetup.cs` | `S/Scripts/Editor/ConsentHudSetup.cs` |
| MenuItem | `Festival Derby/Build Garage HUD` (`:78`) and `BuildInOpenScene` (`:110`) | `Tools/NinjaSurvivor/Setup (una vez)/ToolsInGame/Consent/Setup (Prefab)` (`:100`) |
| Second root | `GarageStage` rig (`RenderTexture` + camera + light) outside the canvas | none |
| Idempotency | `Node` find-before-create (`:1605`) | same pattern |
| Kill switch | `ToggleGarageMenu` flips `GameConfig.GarageMenuEnabled` (`:1572`) | n/a |
| Helpers | `Place` (`:1753`), `MakeText` (`:1791`), `MakeButton`, `MakeLabel` (`:1856`), `MakeInputField`, `BuildShopContent` (`:1034`), `BuildStage` (`:1044`) | inline styling |

Where a builder runs matters. `Build()` opens `Gameplay.unity`, builds, marks dirty, saves the
scene and saves assets (`:78-108`). `BuildInOpenScene()` (`:110`) does the same build on whatever
scene is already open and leaves saving to the caller — that is the entry point for a CLI `code
execute`, where the caller owns scene state.

## Pipeline detail

### 0. Explore, decide, track

- Read the mockup PNG. In parallel, map the existing HUD: scene node, `GameView.Huds` entry, view,
  mediator, builder, data source, loc keys, kill-switch. Classify each element: pure UI, rules/data,
  art.
- Ask ONE product question at a time and wait. Record the decision with its date.
- Write `odd/tasks/<feature>.md`: objective, decisions, existing-system map, task list.

### 1. Art gap

- Reuse first: `S/ResourcesStatic/Textures/` (`MenuPanels/`, `ButtonIcons/`, `Slots/`, `BaseShapes/`,
  `Headers/`, `Icons/`, `SlotIcons/`, `Shop/`) and `S/Art/UI/Garage/`.
- A newly created texture folder is outside every atlas until it is packed — check before shipping
  a directory full of loose sprites.
- Replacing an existing sprite: keep the same `.meta` and guid. Adding a new one: new guid.
- View every generated PNG before installing; "all viewed" from an agent has hidden defects before.

### 2. Logic

- Pure cores (System-only, no `UnityEngine`) with a failing test first when a runnable harness
  exists; run Unity's suites through `tools/unity.ps1` (`tools/UNITY_AGENT.md`). Never open Unity
  for a docs-only change.
- View references are optional and null-tolerant so a scene that lacks the new node still boots.

### 3. Scene builder

- Start from `GarageHudSetup.cs`: it is the template, not a suggestion.
- **Reference space.** Author positions in the mockup's own pixels and scale once. The garage
  builder uses `_refWidth = 1672`, `_boardWidth = 2868`, `_k = _boardWidth / _refWidth` (`:70-72`),
  so every number can be checked against the image one by one, and the drawn height lands on
  `2868 * 9 / 16 = 1613 = 941 * _k` — the visible height of a 16:9 Game View.
- **Group the scene.** Node → block → slot. Use `Place` for absolute rects and `Node` for parented
  blocks; list slot rectangles so a later `HudJuice` can stagger them.
- **Wiring.** Resolve serialized references by path with a `Find<T>` helper that logs when the node
  is missing, so a partial build is loud rather than silently null.
- **Second roots for rigs.** A camera or light cannot live under a Canvas, so the showroom rig
  (`GarageCarStage`) is its own root object (`:180`). Keep render rigs out of the HUD subtree.
- **Paint through a `MaterialPropertyBlock` on body slots only** — never on shared materials. The
  stage does this in `ApplyPaintToModel` (`GarageCarStage.cs`), setting both `_BaseColor` (URP Lit)
  and `_Color` (built-in) so the tint survives whichever shader the imported model carries.

### 4. Captures

- Capture in the Game View at 16:9 and 20:9. Compare each PNG to the mockup; at least two
  refinement passes, then a defects pass (double elements, stray dots, wrong badge/selection state,
  layout gaps).
- Screen-space-overlay UI does not appear in camera captures; switch the canvas to Screen Space -
  Camera at runtime for those, or screenshot the Game View.
- With the Editor in the background the player loop freezes and captures come back stale
  (AGENTS.md §9); set `Application.runInBackground = true` at runtime for anything live.
- **TMP built while inactive can bake a solid bar.** A text mesh authored while its root was
  deactivated survives every open, so the mediator calls `ForceMeshUpdate(true)` on the texts of the
  overlay it is opening — one rebuild per open, not per frame (`GarageHudView.cs`, `RefreshTextMeshes`).

### 5. Loc

- `Loc.GetOr(table, key, fallback)` for everything the view or mediator writes.
- Fixed labels: `LocalizeAllHudsSetup` (menu `Tools/NinjaSurvivor/ToolsInGame/Localization/5 -
  Localize Static Hud Labels`). It matches the label's current text against a key map, so only
  labels NO mediator writes in runtime belong in the map — two writers on one string is the bug it
  avoids. Re-running it after another setup rewrote the placeholder text re-hooks the labels.
- DonJose is caps-only: check any new punctuation/glyph against the font before shipping.

### 6. Commits

- `git branch --show-current`; then `git add -- <my paths> && git commit --only -m "<msg>" -- <my paths>`.
- One work unit per commit: builder + view/mediator + docs together. A generated scene save is part
  of the work unit that owns the builder.
- Never `git add -A`; leave foreign hunks alone (§13).

## Agent prompt templates that worked (condensed)

- **Explorer (read-only)**: "Map the existing HUD: scene node, `GameView.Huds` entry, view,
  mediator, `*Setup.cs` builder, data source, loc keys, kill-switch, scene overrides. Return what
  exists vs what the mockup needs (pure UI / rules / art) and the exact API a builder must bind."
- **Logic writer (TDD)**: "Implement <rules> as a System-only pure core. RED first (show it), then
  GREEN. View refs optional and null-tolerant. Return the API and the commands run."
- **Builder writer**: "Extend `<X>Setup.cs` (idempotent, find-before-create, MenuItem under
  `Festival Derby/...`, doc comment says what it does and does NOT do). Author positions in
  reference-mockup pixels scaled by `_k`. Do not enable the feature. Iterate with your own captures
  at 16:9 and 20:9. Return builder path, MenuItem, capture paths and remaining diffs."
- **Polish/defects writer**: "From these captures list defects; find the root cause before fixing
  (stray dots are usually a capture-camera problem, not the font). Rerun the builder and captures;
  report evidence."

## Gotchas and their home

| Gotcha | Where it is documented |
|---|---|
| HUDs are scene objects resolved by type through `GameView.Huds` | `GarageHudSetup.cs:21-23`; AGENTS.md §4 |
| The builder overwrites manual scene edits | `GarageHudSetup.cs:31-33` |
| Reference-space scaling (`_k`) keeps the mockup checkable | `GarageHudSetup.cs:70-72` |
| A second root owns rigs (camera/light) outside the canvas | `GarageHudSetup.cs:24-25`, `:180` |
| Kill switch is separate from the builder and ships OFF | `GarageHudSetup.cs:34`; AGENTS.md §7 |
| TMP built while inactive bakes a solid bar | `GarageHudView.cs` `RefreshTextMeshes` |
| Static labels vs runtime labels: two writers is the bug | `LocalizeAllHudsSetup.cs` header comment |
| Paint through MPB, body slots only | `GarageCarStage.ApplyPaintToModel` |
| Canvas scaler match value depends on aspect | `Documentation/Knowledge/canvas-scaler-aspect-adaptive.md` |
| Background Editor freezes the player loop and stale captures | AGENTS.md §9 |
| New art folder is outside every atlas until packed | Unity SpriteAtlas packs listed folders only; add the folder to the atlas |

## General precedents

- Additive FX on a hand-tuned card: kill-switch on the component and a pure math core.
- Scene overrides beat prefab values; clear them by property path only.
