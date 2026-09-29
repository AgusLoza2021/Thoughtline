---
description: Tell the user how to open the thoughtline TUI dashboard.
---

<!-- retired-v0.1.0 -->
> **Retired — this file belongs to the v0.1.0 MCP server.** That server is
> unmaintained, so nothing below is a supported path, a tool to call, or a command to
> run. The plugin's record starts at the [plugin README](../README.md); what
> this project is now is the [repository README](../../../README.md).

The thoughtline TUI is a separate process from Claude Code. It is not something you can launch from inside this conversation.

Tell the user to run, in a fresh terminal:

```
thoughtline ui
```

Mention the available flags:

- `--theme {brand|zbrush|mono}` — pick a palette
- `--no-splash` — skip the intro animation
- `--no-update-check` — skip the GitHub release lookup

And the runtime hotkeys: `[t]` cycles theme, `[r]` refreshes, `[/]` searches, `[?]` shows help, `[q]` quits.

Do not try to run `thoughtline ui` yourself via the Bash tool — it needs an interactive terminal you do not have.
