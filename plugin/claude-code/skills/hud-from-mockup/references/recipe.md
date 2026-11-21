# HUD builder recipe

Generic skeleton for an idempotent uGUI builder. Adapt names to the project.

## Shape of a builder

```csharp
#if UNITY_EDITOR
using UnityEditor;
using UnityEngine;
using UnityEngine.UI;

/// Builds the XyzScreen hierarchy under the open scene's canvas.
/// Does: create/re-wire nodes by name, assign serialized references.
/// Does NOT: enable the feature, touch other screens, hand-tuned nodes outside XyzScreen.
public static class XyzScreenBuilder
{
    const float RefWidth = 1672f;      // the mockup's own width in pixels
    static float _k;                    // canvasWidth / RefWidth, computed once per run

    [MenuItem("Tools/UI/Build Xyz Screen")]
    public static void Build()
    {
        var canvas = Object.FindFirstObjectByType<Canvas>();
        _k = ((RectTransform)canvas.transform).rect.width / RefWidth;

        var root = Node(canvas.transform, "XyzScreen");
        var header = Node(root, "Header");
        Place(header, x: 836, y: 90, w: 600, h: 120);   // mockup pixels
        // ... blocks -> slots
        EditorUtility.SetDirty(root.gameObject);
    }

    // Find by name in the DIRECT parent only; create if missing.
    static RectTransform Node(Transform parent, string name)
    {
        var t = parent.Find(name);
        if (t == null)
        {
            var go = new GameObject(name, typeof(RectTransform));
            go.transform.SetParent(parent, false);
            t = go.transform;
        }
        return (RectTransform)t;
    }

    // Absolute rect in mockup pixels, anchored top-left, scaled once by _k.
    static void Place(RectTransform rt, float x, float y, float w, float h)
    {
        rt.anchorMin = rt.anchorMax = new Vector2(0f, 1f);
        rt.pivot = new Vector2(0.5f, 0.5f);
        rt.sizeDelta = new Vector2(w * _k, h * _k);
        rt.anchoredPosition = new Vector2(x * _k, -y * _k);
    }

    // Resolve serialized references by path; log loudly so a partial build is never silently null.
    static T Find<T>(Transform root, string path) where T : Component
    {
        var t = root.Find(path);
        if (t == null) { Debug.LogError($"[XyzScreenBuilder] missing node: {path}"); return null; }
        return t.GetComponent<T>();
    }
}
#endif
```

## Principles

- **Reference space.** Author numbers in the mockup's pixels so each one can be checked against the
  image; convert once with a single scale factor.
- **Group the tree.** Screen → block → slot. List slot rectangles so an animation component can
  stagger them later.
- **Wiring.** Assign view fields through `Find<T>`; a missing node logs an error instead of leaving a
  silent null.
- **Render rigs live outside the Canvas.** A camera or light cannot sit under a Canvas; give a 3D
  preview its own root object and show it through a RenderTexture.
- **Canvas scaler.** Pick the match value for the aspect range you support and test both ends.

## Captures

- Capture the Game View at 16:9 and 20:9; compare each to the brief.
- Screen Space - Overlay UI does not appear in camera captures; switch the canvas to Screen Space -
  Camera for those, or screenshot the Game View.
- With the Editor in the background the player loop can freeze and captures come back stale; set
  `Application.runInBackground = true` for live captures.
- TextMeshPro built while its object was inactive can bake a solid bar. Call `ForceMeshUpdate(true)`
  once when the screen opens, not every frame.

## Localization

- Runtime-written labels go through the localization API in code.
- Fixed labels are attached by a one-time sweep. Never put a localized-text component on a label a
  controller also writes: two writers on one string is the bug.
- Check new punctuation and glyphs against the font before shipping (some display fonts are
  caps-only or lack symbols).

## Delegation prompts that worked

- **Explorer (read-only):** "Map the existing screen: scene node, view, controller, builder, data
  source, loc keys, kill-switch. Return what exists versus what the brief needs."
- **Builder writer:** "Write an idempotent builder (find-before-create, menu item, doc comment says
  what it does and does NOT do). Positions in mockup pixels scaled once. Do not enable the feature.
  Iterate with your own captures at 16:9 and 20:9."
- **Polish writer:** "From these captures list defects; find the root cause before fixing. Rerun the
  builder and captures; report evidence."
