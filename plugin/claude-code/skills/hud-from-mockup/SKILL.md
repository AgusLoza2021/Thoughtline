---
name: hud-from-mockup
description: "Trigger: mockup, rediseñar HUD, rebuild screen like this image, popup redesign, armar este HUD, garage HUD. Pipeline to build or rebuild a FestivalDerby HUD from an owner mockup using the idempotent scene builder (GarageHudSetup as the worked example): builder first, captures, juice, loc."
metadata:
  version: "1.0"
---

# HUD from an owner mockup

Pipeline to build or rebuild a HUD from a reference image or from an existing in-repo screen,
with an idempotent Editor builder as the source of truth.

`S` below is `Assets/GorodiskiGames/Survivalio`. Every other path is repo-relative.

## Reference, never copy (this repo is public)

A reference teaches **structure**: hierarchy, spacing rhythm, hit-target sizes, what is grouped with
what, which state each element shows. It is never a source of pixels.

- Allowed references: the owner's own mockups, and screens already in this repo (the garage is the
  worked example of our own language).
- A screenshot of someone else's game is a mood board only. Never trace it, slice it, re-draw it
  1:1 or reuse its icons, fonts, names or copy. Reduce it to a written layout brief first (zones,
  priorities, states) and build from the brief, not from the image.
- Art comes from `S/ResourcesStatic/Textures/**` / `S/Art/UI/**` or is generated new. Palette, fonts
  and sprites are the project's kit (`../hud-premium-restyle/SKILL.md`), not the reference's.
- To take an existing screen as the example: read its builder (`GarageHudSetup.cs`) for the
  **pattern** (find-before-create, reference space, helpers) and write a new builder for the new
  screen. Do not duplicate its node tree, constants or layout numbers; if two screens need the same
  helper, extract it (see the restyle skill) instead of pasting it.
- Do not commit the reference image unless the owner made it for this project.

## Activation contract

Load when the owner hands over a mockup image (or a reference screenshot) and asks to build,
rebuild or restyle a HUD, popup, overlay or menu page to match it. This skill owns the pipeline
around the HUD. The kit, palette and sprite set live in `../hud-premium-restyle/SKILL.md`. Read
`references/recipe.md` before delegating any of it.

## Hard rules

- **HUDs are scene objects, not runtime prefabs.** Every screen is a `Mediator<TView>` pair whose
  view is a `MonoBehaviour` that already lives in `Gameplay.unity` and is listed in
  `GameView.Huds` (`S/Scripts/UI/GameView.cs:12`, `public BaseHud[] Huds`). `HudManager` resolves
  views **by type** against that array and never instantiates a prefab at runtime. A new screen is
  therefore four things: a scene node, a `GameView.Huds` entry, a view and a mediator.
  `ConsentHud` is the one prefab HUD (`Resources/Prefabs/ConsentHud.prefab`), and it is built by
  the boot flow, not by `HudManager`.
- **The builder is the source of truth.** A HUD authored in the scene is edited by changing its
  Editor builder and re-running it — never by hand-editing nodes in the Hierarchy, because the next
  run overwrites manual work. `S/Scripts/Editor/GarageHudSetup.cs` is the worked example; the
  pattern is documented at the top of that file (`:20-35`).
- **Find before create.** Builders look a node up by name **in the direct parent** before creating
  it (`GarageHudSetup.Node`, `:1605`), so a rerun re-wires instead of duplicating and a half-built
  screen can be repaired. The lookup is deliberately local: sub-blocks reuse names such as `Icon`
  or `Value` in every slot, and a deep search would re-parent one slot's child into another.
- **Never enable a feature from the builder.** The garage builder authors the screen and leaves
  `GameConfig.GarageMenuEnabled` OFF; `ToggleGarageMenu` (`:1572`) is the separate switch. New
  features ship disabled behind a `GameConfig` kill-switch (AGENTS.md §7).
- **Text goes through `Loc.GetOr(table, key, fallback)`**, never a literal string. Fixed labels are
  attached by `LocalizeAllHudsSetup` (menu `Tools/NinjaSurvivor/ToolsInGame/Localization/5 - Localize
  Static Hud Labels`); labels a mediator writes at runtime are localized in code — putting a
  `LocalizedTextView` on a runtime label creates two writers for one string.
- **DonJose Black is caps-only** (`S/Fonts/DonJose_FONT_INFO.md`): it draws lowercase as versals on
  purpose. Author copy uppercase; `%` and `—`/`→` fall back to another font.
- **Every HUD animation restores its state in `OnDisable`, idempotently.** A HUD is one persistent
  GameObject for the whole session; a tween left mid-flight (e.g. scale 0) hides that screen
  forever with no log.
- **Author in one coordinate space.** Positions are written in the reference mockup's own pixels and
  scaled once into the canvas board (see `references/recipe.md`). A constant calibrated on a single
  aspect breaks every other aspect.
- **Art lives in the repo.** Reuse `S/ResourcesStatic/Textures/**` and `S/Art/UI/**`; check the
  atlas/import folder before drawing anything new.
- Never build (the owner builds from the Editor menu, §2), never run Editor actions on a scene
  another session owns (§13), never `git add -A`, Conventional commits, no AI attribution.

## Decision gates

| Question | Answer | Route |
|---|---|---|
| Scope | Reskin only, same nodes | `../hud-premium-restyle/SKILL.md` |
| Scope | New rules, stats or economy | pure core + tests first, then the view |
| Existing HUD | Built by its `*Setup.cs` | change the tool, rerun it, save the scene |
| Existing HUD | Hand-tuned in the scene | ask the owner before running any builder over it |
| Art | Exists | reuse from `Textures/` or `Art/UI/` |
| Art | Missing | generate; install new names with a new guid, replacements with the same guid and `.meta` |

## Execution steps

1. **Read the mockup; map in parallel.** Identify the existing HUD — view, mediator, builder, data
   source, `GameView.Huds` entry, loc keys, any scene overrides. Classify each mockup element as
   pure UI, rules/data, or art. Ask the owner ONE product question at a time and wait. Open
   `odd/tasks/<feature>.md` with the objective, decisions and task list.
2. **Art gap.** Generate the missing sprites and view every PNG before installing. Install into an
   existing `Textures/` folder (a new folder is outside every atlas silently) with the correct guid.
3. **Logic.** Pure cores (System-only) with tests through the project's Unity suites
   (`tools/unity.ps1`, see `tools/UNITY_AGENT.md`). View refs stay optional and null-tolerant so an
   older scene keeps working.
4. **UI.** Extend or add a `*Setup.cs` builder under `S/Scripts/Editor/`. It must be idempotent by
   name lookup, its `[MenuItem]` root lives under `Festival Derby/...`, and its doc comment states
   both what it does and what it does NOT do (the garage builder's `:34` is the model).
5. **Juice.** Put the `HudJuice` component on the screen root, point `_scaleTarget` at the window
   (NOT the root — the root of most HUDs is the full-screen canvas and scaling it is wrong) and
   list the per-block rectangles for the stagger. The global switch is `HudJuiceSettings.Enabled`,
   read from `GameConfig` by `GameInitializeState`; treat it as read-only from the HUD.
6. **Captures.** Screenshot at 16:9 and 20:9 and compare each to the mockup — at least two
   refinement passes, then a defects pass. List remaining diffs in the feature doc.
7. **Loc.** Keys via `Loc.GetOr` in the view/mediator; run the static-label sweep for fixed labels.
8. **Commit** per work unit (`git commit --only`, Conventional, no BOM, no AI attribution).

## Output contract

Return: builder path and its MenuItem path, view/mediator paths, the `GameView.Huds` entry, capture
paths per aspect, the remaining diffs versus the mockup, the kill-switch that keeps the screen off,
and the commit hash.

## References

- `references/recipe.md` — worked examples and line evidence.
- `../hud-premium-restyle/SKILL.md` — palette, sprite set and the shared builder helpers.
- `Documentation/Knowledge/hud-from-mockup-pipeline.md` — the durable lesson, in the Knowledge index.
- `Documentation/Design/UI_JUICE_GUIDE.md` — juice vocabulary and the in-game HUD checklist.
