---
name: hud-from-mockup
description: "Trigger: build a Unity HUD/popup/menu screen from a mockup or reference image, rebuild a screen like this image, redesign a HUD, generate Unity UI quickly. Pipeline for uGUI: layout brief first, then an idempotent Editor builder as the source of truth, captures, juice, localization."
metadata:
  version: "2.0"
---

# Unity HUD from a mockup

A pipeline to turn a reference image (or an existing screen in the project) into a Unity uGUI screen
that can be rebuilt at any time by re-running one Editor script.

## Activation contract

Load when the user hands over a mockup or reference and asks to build, rebuild or restyle a screen.
Read `references/recipe.md` for the builder skeleton before writing code. For a pure reskin of an
existing screen use the `hud-premium-restyle` skill instead.

## Reference, never copy

A reference teaches **structure**: hierarchy, spacing rhythm, hit-target sizes, grouping, and which
state each element shows. It is never a source of pixels.

- Safe references: the user's own mockups and screens that already exist in the project.
- A screenshot of someone else's game is a mood board only. Never trace, slice or redraw it 1:1, and
  never reuse its icons, fonts, names or copy. Reduce it to a written **layout brief** (zones,
  priorities, states) and build from the brief, not from the image.
- To use an existing screen as the example, read its builder for the **pattern** (find-before-create,
  reference space, helpers) and write a new builder for the new screen. Do not duplicate its node
  tree or layout numbers; extract shared helpers instead of pasting them.
- Art comes from the project's own sprite folders or is generated new. Do not commit the reference
  image unless the user made it for this project.

## Hard rules

- **The builder is the source of truth.** Edit the Editor script and re-run it. Hand edits in the
  Hierarchy get overwritten on the next run.
- **Find before create.** Look a node up by name **in its direct parent** before creating it, so a
  rerun re-wires instead of duplicating. Keep the lookup local: sub-blocks reuse names like `Icon`
  or `Value`, and a deep search re-parents one slot's child into another.
- **Never enable a feature from the builder.** It authors the screen; a separate flag or kill-switch
  turns it on.
- **Author in one coordinate space.** Write positions in the mockup's own pixels and scale once into
  the canvas. A constant calibrated on a single aspect ratio breaks every other aspect.
- **Text goes through the project's localization**, never a literal string in a view.
- **Every UI animation restores its state in `OnDisable`, idempotently.** A persistent HUD with a
  tween left mid-flight (scale 0, alpha 0) stays invisible forever with no log.
- **Reuse art before drawing any.** Check existing sprites and atlases first.
- Do not build the player or run Editor actions on a scene someone else is editing; commit with
  Conventional Commits.

## Decision gates

| Question | Answer | Route |
|---|---|---|
| Scope | Reskin, same nodes | `hud-premium-restyle` |
| Scope | New rules, stats or economy | pure logic + tests first, then the view |
| Existing screen | Built by an Editor builder | change the builder, rerun it, save the scene |
| Existing screen | Hand-tuned in the scene | ask before running any builder over it |
| Art | Exists | reuse it |
| Art | Missing | generate new, view every image before installing |

## Execution steps

1. **Brief.** Read the reference; write the layout brief (zones, priorities, states). Classify each
   element as pure UI, rules/data, or art. Ask the user ONE product question at a time and wait.
2. **Art gap.** Generate or reuse sprites; keep them in a folder that is part of the sprite atlas.
3. **Logic.** Keep rules in plain C# classes with tests; view references stay optional and
   null-tolerant so an older scene keeps loading.
4. **Builder.** Write the idempotent Editor builder (see recipe). Its doc comment states what it
   does AND what it does NOT do.
5. **Juice.** Add entrance/press animation as a component on the screen root, scale the window (not
   the full-screen root), and make it restore state on disable.
6. **Captures.** Screenshot at 16:9 and 20:9, compare to the brief/mockup, do at least two
   refinement passes, then a defects pass (duplicates, stray elements, wrong states, gaps).
7. **Localization.** Keys for every runtime-written label; a sweep for fixed labels.
8. **Commit** per work unit: builder + view + docs together.

## Output contract

Return: the builder path and its menu path, the view/controller paths, the kill-switch that keeps
the screen off, capture paths per aspect, remaining diffs versus the brief, and the commit hash.
