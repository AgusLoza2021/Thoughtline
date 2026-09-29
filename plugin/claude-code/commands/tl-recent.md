---
description: Show the most recent thoughtline memories for the active project.
---

<!-- retired-v0.1.0 -->
> **Retired — this file belongs to the v0.1.0 MCP server.** That server is
> unmaintained, so nothing below is a supported path, a tool to call, or a command to
> run. The plugin's record starts at the [plugin README](../README.md); what
> this project is now is the [repository README](../../../README.md).

Use the `tl_context` tool with the active project to fetch the most recent memories and sessions, then summarize them in a single concise list:

- Group by type (decisions, conventions, bugfixes, …)
- Show title, type, and relative time for each
- If a memory looks load-bearing for current work, surface it explicitly

If `tl_context` returns nothing, say so plainly — do not invent memories.
