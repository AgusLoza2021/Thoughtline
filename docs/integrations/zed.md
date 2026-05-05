# Zed

[Zed](https://zed.dev) added MCP support to its assistant in late 2025. Wire Thoughtline as a context server.

## 1. Install the binary

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
thoughtline version
```

Or a release binary from [the latest release](https://github.com/AgusLoza2021/Thoughtline/releases/latest).

## 2. Register Thoughtline

Open `Zed → Settings → Open Settings (JSON)` (or `cmd+,` then "Open settings file") and add the entry under `assistant.context_servers`:

```jsonc
{
  "assistant": {
    "context_servers": {
      "thoughtline": {
        "source": "custom",
        "command": "thoughtline",
        "args": []
      }
    }
  }
}
```

Reload the window (`cmd/ctrl+shift+P → reload window`). The `tl_*` tools will appear in the assistant's tool list.

## 3. Tell the agent the protocol

Zed's assistant doesn't auto-load skill files. Drop the protocol into your project at `.zed/rules.md` or include it in your top-level CLAUDE.md / RULES.md (whichever convention your team uses). The protocol text is the same as the Cursor doc — see `cursor.md` for the boilerplate.

## Tagging tip for Zed users

Tag memories with `tool:zed` and pass `agent_label: "zed"` when calling `tl_session_start`:

```jsonc
{ "agent_label": "zed" }
```

That lets the dashboard's session list group activity by client.
