# Screenshots & demos

<!-- retired-v0.1.0 -->
> **Retired — this page documents the v0.1.0 MCP server.** That server is
> unmaintained, so the recording instructions below are not a supported path.
> For what this project is now — a gamedev memory vocabulary that runs on
> [Engram](https://github.com/Gentleman-Programming/engram) — read the
> [README](../../README.md), the [memory domain](../design/memory-domain.md) and the
> [tag conventions](../design/tag-conventions.md).

---

> **Legacy — the v0.1.0 server.** Everything below documents `thoughtline`, the
> retired server, and its `tl_*` tools. It is a record of how the project worked,
> not instructions to follow.

Drop launch screenshots and short GIFs here:

| File | What it should show |
|------|---------------------|
| `screenshot-brand.png`  | Overview tab, default `brand` theme. Animated cube visible on the left, hero pill, status bar with MEM/SESSIONS, full layout. |
| `screenshot-zbrush.png` | Same Overview, `--theme zbrush`. The warm-amber ZBrush look — the differentiator vs other TUIs. |
| `screenshot-search.gif` | 5–10s GIF: open search tab, type a query, hit enter, results render, hit enter on a result, viewport scrolls the full memory. |
| `screenshot-mono.png`   | `--theme mono` — useful for slide decks and 1-bit screenshots. Optional. |

## Recording tips

- **Easy mode (recommended): use [vhs](https://github.com/charmbracelet/vhs)** with the ready-made tape file at [`scripts/demo.tape`](../../scripts/demo.tape). One command from the repo root:
  ```
  vhs scripts/demo.tape
  ```
  vhs is reproducible (the `.tape` is committed code), uses the real keybindings, and outputs straight to `docs/media/demo.gif`. Edit the tape's `Sleep` values and `Type` lines to retune the choreography.
- **Manual alternatives**: [t-rec](https://github.com/sassman/t-rec-rs) (Mac/Linux) or [terminalizer](https://github.com/faressoft/terminalizer) (cross-platform). Cap to 8 fps and ≤ 2 MB.
- Resize your terminal to ~120×35 for landscape screenshots and 100×40 for the GIF.
- Use a dark terminal background with no transparency (Windows Terminal default works fine).
- Don't include personal usernames or absolute paths in the frame — set `THOUGHTLINE_DB=/tmp/demo.db` (or `%TEMP%\thoughtline-demo.db` on Windows) first and seed a few synthetic memories.
