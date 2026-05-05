# Screenshots & demos

Drop launch screenshots and short GIFs here:

| File | What it should show |
|------|---------------------|
| `screenshot-brand.png`  | Overview tab, default `brand` theme. Animated cube visible on the left, hero pill, status bar with MEM/SESSIONS, full layout. |
| `screenshot-zbrush.png` | Same Overview, `--theme zbrush`. The warm-amber ZBrush look — the differentiator vs other TUIs. |
| `screenshot-search.gif` | 5–10s GIF: open search tab, type a query, hit enter, results render, hit enter on a result, viewport scrolls the full memory. |
| `screenshot-mono.png`   | `--theme mono` — useful for slide decks and 1-bit screenshots. Optional. |

## Recording tips

- Resize your terminal to ~120×35 for landscape screenshots and 100×40 for the GIF.
- Use a dark terminal background with no transparency (Windows Terminal default works fine).
- For the GIF: [t-rec](https://github.com/sassman/t-rec-rs) (Mac/Linux) or [terminalizer](https://github.com/faressoft/terminalizer) (cross-platform). Cap to 8 fps and ≤ 2 MB.
- Don't include personal usernames or absolute paths in the frame — set `THOUGHTLINE_DB=/tmp/demo.db` first and seed a few synthetic memories.
