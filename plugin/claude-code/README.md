# Thoughtline — Claude Code plugin

Persistent, project-aware memory for [Claude Code](https://claude.com/claude-code). Survives across sessions and compactions. Local-first — no cloud, no telemetry, your `thoughtline.db` lives in your user cache directory.

This plugin wires Claude Code into the [Thoughtline MCP server](https://github.com/AgusLoza2021/Thoughtline):

- **MCP tools** — `tl_save`, `tl_search`, `tl_context`, `tl_get_observation`, `tl_session_start`, `tl_session_summary`, `tl_update`, `tl_delete`, `tl_stats`
- **Hooks** — `SessionStart` and post-compaction hooks inject the active-protocol markdown so Claude knows the memory tools exist and when to use them
- **Skill** — `thoughtline-memory`, the protocol Claude follows for proactive saves and recalls
- **Slash commands** — `/tl-recent`, `/tl-search`, `/tl-stats`, `/tl-ui`, `/tl-export`
- **Subagent** — `tl-archivist` for memory hygiene (dedup, prune, audit)

## Prerequisites

The plugin shells out to the `thoughtline` binary, so it has to be on your `PATH` first:

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
```

Or grab a prebuilt binary from [the latest release](https://github.com/AgusLoza2021/Thoughtline/releases/latest) and put it on your PATH.

Verify:

```bash
thoughtline version
```

## Install

Once Claude Code's plugin support ships, install via:

```bash
claude plugin install github:AgusLoza2021/Thoughtline/plugin/claude-code
```

Until then, you can clone-and-link manually — drop the plugin in your Claude Code plugins directory and restart.

## What the plugin does

When Claude Code starts a session in your repo:

1. The `SessionStart` hook calls `thoughtline protocol --event session-start` which prints the active-protocol markdown to stdout.
2. Claude Code captures that as `additionalContext` for the session — Claude now knows the `tl_*` tools exist, when to call `tl_save`, and how to format `topic_key`s.
3. On compaction, the `compact` matcher fires `thoughtline protocol --event post-compaction` so Claude can recover state.

The protocol is one source of truth: **the binary**. The bash scripts that used to live here are gone. This means Windows users do not need WSL or Git Bash.

## Files

```
plugin/claude-code/
├── .claude-plugin/
│   └── plugin.json         # plugin manifest (name, version, keywords, repo)
├── LICENSE                 # MIT
├── README.md               # this file
├── .mcp.json               # MCP server registration
├── hooks/
│   └── hooks.json          # SessionStart + compact hooks
├── skills/
│   └── memory/
│       └── SKILL.md        # the protocol Claude follows
├── commands/               # slash commands (/tl-recent, /tl-search, ...)
├── agents/                 # custom subagents (tl-archivist)
└── examples/               # sample memories + transcripts
```

## Configuration

The plugin's `.mcp.json` registers the server as `tl` and runs `thoughtline serve`. To use a custom binary path, set `THOUGHTLINE_BIN` (planned) or edit `.mcp.json`.

To use a custom database location, set `THOUGHTLINE_DB` or `THOUGHTLINE_HOME`. See the [main README](https://github.com/AgusLoza2021/Thoughtline#configuration) for the full list.

## Uninstall

```bash
claude plugin remove thoughtline
```

Your data (`thoughtline.db`) is **not** removed. Delete it manually from `$THOUGHTLINE_HOME` or your user cache directory if you want a clean slate.

## License

MIT — see [LICENSE](./LICENSE).
