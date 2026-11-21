---
name: hud-premium-restyle
description: "Trigger: restyle a Unity HUD, reskin, polish a screen, make UI consistent, shared UI kit. Restyle existing uGUI screens from one shared kit (palette, sprites, fonts, helpers) instead of per-screen forks."
metadata:
  version: "2.0"
---

# Unity HUD restyle from a shared kit

Use when the screen's nodes already exist and only the look changes. For a new screen from a
reference, use `hud-from-mockup` first.

## Rule: one kit, many screens

A consistent UI comes from **one** definition of the look that every builder calls. Do not paste
palette constants and helper methods into each screen's builder; they drift on the first change.

The kit has four parts:

| Part | What it holds |
|---|---|
| Palette | Named colors (background, surface, accent, text, danger), never raw hex in views |
| Sprites | 9-sliced panel, button, slot, header, icon set; one folder, part of the atlas |
| Fonts | The one or two fonts and the sizes for title / body / caption |
| Helpers | `MakeText`, `MakeButton`, `MakePanel` that apply the three parts above |

Keep the kit in one Editor class (or a ScriptableObject for runtime-readable tokens). If the project
has none yet, create it once, then have every builder call it. Extracting it is a refactor: tell the
user before moving existing builders onto it.

## Restyle steps

1. Inventory the screen: which nodes use raw colors, ad-hoc sprites or literal fonts.
2. Map each to a kit role. If no role fits, add the role to the kit; do not special-case the screen.
3. Rerun the screen's builder (or the restyle tool) so the change is repeatable, never hand-edited.
4. Buttons: use color-tint transitions with an explicit color block so press feedback exists by
   default; do not switch to no-transition without adding an animation.
5. Capture at 16:9 and 20:9 before and after; list remaining diffs.

## Guardrails

- No new art, fonts or colors outside the kit without the user's approval.
- Do not invent sprites to fill a gap: reuse the kit or generate new art through `hud-from-mockup`.
- Reference screens teach structure and the kit's language; never copy another game's art or layout.
- Restore any UI animation state in `OnDisable`; text through localization; features stay behind a
  kill-switch until the user turns them on.
