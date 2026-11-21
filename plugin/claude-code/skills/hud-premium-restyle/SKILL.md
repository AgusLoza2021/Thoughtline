---
name: hud-premium-restyle
description: "Trigger: restyle a HUD, reskin, premium-ify, restyle a prefab, garage kit, match the garage look. Covers the palette, the sprite set and the shared builder helpers, plus the honest gap where a shared styler library does not exist here yet."
metadata:
  version: "1.0"
---

# HUD kit restyle (garage language)

Read this **before** writing a new restyle tool, because the first thing it has to say is what
does not exist here. This repo is public: restyle from our own kit and our own screens, never from
another game's art (see "Reference, never copy" in `../hud-from-mockup/SKILL.md`).

## The honest starting point

There is **no shared styler library yet** (no `PremiumKitStyler`-style class). What exists is:

- **One builder that is the reference for the language**: `S/Scripts/Editor/GarageHudSetup.cs`
  (`S = Assets/GorodiskiGames/Survivalio`). Its palette constants (`:52-65`), reference-space
  scaling (`:70-72`) and helpers are the working kit.
- **A separate prefab builder**: `S/Scripts/Editor/ConsentHudSetup.cs`, which edits the Consent
  prefab in `S/Resources/Prefabs/ConsentHud.prefab` instead of the scene.
- **A sprite set** already in the repo (below).

So "restyle a HUD to the kit" today means either (a) extend `GarageHudSetup`-style code for the
target HUD, copying the palette constants and calling the same helpers if you are inside that class,
or (b) **extract a `GarageKitStyler`** from those helpers and have both builders call it. (b) is the
better end state and is a refactor that needs the owner's go-ahead — do not silently fork the
helpers into a second builder, because then the two drift on the first palette change.

Do not invent sprites to fill a gap — reuse what is listed below or generate new art through
`hud-from-mockup`.

## The kit (locked: no new art, no new fonts, no new colors without the owner)

Sprites, all under `S/ResourcesStatic/Textures/`:

| Role | Sprites | Folder |
|---|---|---|
| Panel fill / frame / header bar | `PanelFill9`, `PanelFrame9`, `PanelHeaderBar9` | `MenuPanels/` |
| Sunken wells and inputs | `ScrollWell9`, `InputField9` | `MenuPanels/` |
| Day cell | `DayCellFrame9` | `MenuPanels/` |
| Primary CTA | `ButtonWide9_Yellow` | `ButtonIcons/` |
| Neutral / secondary | `ButtonGrey`, `ButtonYellow`, `ButtonGreen`, `ButtonRed` (tinted variants) | `ButtonIcons/` |
| Close | `CloseCircle`, `CloseIcon` | `ButtonIcons/` |
| Tabs | `MenuButtonActive`, `MenuButtonInactive` | `ButtonIcons/` |
| Badge | `BadgeDot` | `ButtonIcons/` |
| Slots | `SlotA`, `SlotC` | `Slots/` |
| Shapes (bars, dividers) | `Circle`, `Square`, `SquareFrame` | `BaseShapes/` |
| Sheet header | `HeaderGrey` | `Headers/` |
| Feature icons | `ArmoryIcon`, `BattleIcon`, `EquipmentIcon`, `RankingIcon`, `ShopIcon`, `LockedTabIcon` | `ButtonIcons/` |
| Screen art (not atlased, not 9-sliced) | `bg_garage`, `avatar_driver`, `icon_car`, `icon_checkered`, `icon_coin`, ... | `S/Art/UI/Garage/` |

Font: **`S/Fonts/DonJose_Black SDF Underlay.asset`** is the only font source (caps-only; see
`DonJose_FONT_INFO.md`). `MakeText` binds it and never a second TMP style.

Palette, read off the reference mockup (`GarageHudSetup.cs:52-65`). These are the values; no new
color is added without the owner:

| Name | Value | Use |
|---|---|---|
| `_panelFill` | `(0.07, 0.08, 0.11, 0.96)` | screen plate |
| `_cardFill` | `(0.10, 0.11, 0.14, 1)` | card |
| `_slotFill` | `(0.06, 0.06, 0.08, 0.92)` | slot under a sprite |
| `_tileFill` | `(0.09, 0.10, 0.13, 0.96)` | selectable tile |
| `_wellFill` | `(0.20, 0.22, 0.26, 1)` | scroll/sunken well |
| `_yellow` | `(1.00, 0.914, 0, 1)` → `#FFE900` | accent / selection |
| `_deep` | `(0.05, 0.05, 0.06, 1)` | behind everything |
| `_text` | `(0.95, 0.94, 0.88, 1)` | primary text |
| `_textMuted` | `(0.72, 0.73, 0.76, 1)` | secondary text |
| `_avatarTeal` | `(0.13, 0.62, 0.66, 1)` | driver avatar plate |
| `_overlayFill` | `(0.035, 0.04, 0.055, 0.99)` | overlay plate (a touch darker than the base) |
| `_headerFill` | `(0.09, 0.10, 0.13, 1)` | overlay header |
| `_unselectedTile` | `(0.13, 0.14, 0.17, 1)` | unselected tile |

The owner's brief describes the language as **dark rounded panels, one yellow accent, white bold
uppercase text**. (`#F5C518` was quoted in the porting brief; the code actually uses `#FFE900` —
trust the code.)

## The shared helpers (what a new tool should call)

All private statics in `GarageHudSetup.cs`; if you extract a styler, these are the bodies that move:

- `Node(parent, name)` (`:1605`) — find-before-create, direct children only.
- `Place(rect, x, y, w, h)` (`:1753`) — absolute position in reference pixels, scaled by `_k`.
- `MakeText(rect, text, size, color, alignment, style)` (`:1791`) — clears a conflicting graphic
  first, sets the font, `fontSize = size * _k`, `enableAutoSizing = false`, `NoWrap`,
  `raycastTarget = false`. Sizes are authored in reference pixels, never in board pixels.
- `MakeImage(rect, spritePath, fallback, raycast)` — picks `Sliced` vs `Simple` from the sprite's
  border automatically, and records a missing sprite instead of silently drawing nothing.
- `MakeButton(rect, spritePath, fallback)` — a tinted `Image` as `targetGraphic` with
  `Transition.ColorTint` and `ColorBlock.defaultColorBlock`, **so press feedback exists by default**
  (color tint is the mechanism — do not switch a kit button to `Transition.None` without adding an explicit
  animation).
- `MakeLabel(parent, name, text, size, color, fromBottomRef, heightRef)` (`:1856`) — centred label,
  pinned to the parent's bottom when `fromBottomRef >= 0`. Text lives on its **own** node: two
  `Graphic`s cannot share one GameObject.
- `MakeChevron` — an arrow drawn as two rotated `Square` bars, because the display font has no arrow
  glyph and TMP renders a missing glyph as nothing at all, which reads as a broken button.

Structural rules these helpers encode:

- **Reference space** (`:70-72`): author in `1672x941`, scale once by `_k = 2868 / 1672`. Every
  number is checkable against the mockup image.
- **One Graphic per GameObject** — `ClearConflictingGraphic` enforces it, because a TMP and an Image
  on the same node fight over the mesh.
- **`OnDisable` restores state**, idempotently (AGENTS.md §7): a HUD is one persistent GameObject.

## The recipe

1. Inspect the target HUD first: scene node, `GameView.Huds` entry, view, mediator, builder, and
   whether the owner hand-tuned it. If the owner tunes it, ask before running anything over it.
2. Idempotent tool: find-before-create, `[MenuItem]` under `Festival Derby/...`, doc comment stating
   what it does and does NOT do. Re-running must be a no-op on an already-restyled screen.
3. Responsive: never bake a width for one canvas size. Compute from the real parent rect and
   recompute on `OnEnable` + `OnRectTransformDimensionsChange`. The canvas matches by **width**
   (2868); the visible height changes with the aspect (16:9 → 1613, 20:9 → ~1290).
4. Captures at 16:9 and 20:9, once per batch, after the whole set of restyles. Look at every PNG;
   check that the key numbers (prices, rewards, times) stay readable.
5. One writer per HUD. After the owner hand-tunes a screen, do **not** rerun the tool — it restyles
   over their tuning.

## Hard constraints

- **`Loc.GetOr(table, key, fallback)`** for all text — never a literal string.
- **DonJose Black is the only font**, and it is caps-only.
- **Scene overrides beat prefabs.** For a prefab HUD (ConsentHud), the instance in `Gameplay.unity`
  always wins over the prefab: check the instance before concluding a restyle "didn't take." For the
  garage menu (built directly in the scene) there is no prefab to diverge from.
- **New features ship disabled** behind a `GameConfig` kill-switch (AGENTS.md §7); the garage menu
  is OFF in production and flipped by `ToggleGarageMenu`.
- **Never build**, never run Editor actions on a scene another session owns (§13), no `git add -A`.
- Do not restyle ConsentHud's wording — it is a legal constraint.

## Excluded / do not touch

- Consent wording (legal); its button semantics were the owner's decision — leave them.
- Any screen the owner hand-tunes: ask first.
- The in-game HUD's joystick/drive overlays need `raycastTarget = false` on their `Image`s.

## Related

- `../hud-from-mockup/SKILL.md` — the pipeline that owns this kit.
- `Documentation/Knowledge/hud-from-mockup-pipeline.md`
- `Documentation/Design/UI_JUICE_GUIDE.md`
