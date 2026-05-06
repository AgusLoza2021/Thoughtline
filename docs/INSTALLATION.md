# Installation

Thoughtline ships as a single Go binary (`thoughtline` / `thoughtline.exe`) backed by a single SQLite file. **No Node, no Python, no Docker**. Pick your platform below.

> Looking for a feature comparison vs [Engram](https://github.com/Gentleman-Programming/engram) and [claude-mem](https://github.com/thedotmack/claude-mem)? See [COMPARISON.md](COMPARISON.md).

---

## Windows — one command (auto-installer)

The recommended path. Idempotent, no admin/UAC, registers the MCP server in `~/.claude.json` (which the Claude Code CLI **and** the VS Code extension both read).

**From PowerShell:**

```powershell
irm https://raw.githubusercontent.com/AgusLoza2021/Thoughtline/main/scripts/install.ps1 | iex
```

**From classic `cmd.exe`:**

```cmd
powershell -NoProfile -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/AgusLoza2021/Thoughtline/main/scripts/install.ps1 | iex"
```

What the script does:

1. Pre-flight (PowerShell ≥ 5.1, Windows)
2. Detects Go; if missing, installs it via `winget install GoLang.Go`
3. Runs `go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest`
4. Adds `%USERPROFILE%\go\bin` to your **user** PATH
5. Registers `thoughtline` as an MCP server in `%USERPROFILE%\.claude.json` (merge, not overwrite)
6. Verifies with `thoughtline version`

After it finishes, **restart Claude Code** (CLI or VS Code extension) so it picks up the new MCP entry.

### Flags

```powershell
.\scripts\install.ps1 -Version v0.0.1   # pin a specific tag
.\scripts\install.ps1 -SkipPathSetup    # don't touch user PATH
.\scripts\install.ps1 -SkipMcp          # don't edit ~/.claude.json
.\scripts\install.ps1 -Force            # overwrite an existing thoughtline MCP entry
```

See [`scripts/install.ps1`](../scripts/install.ps1) for the source.

---

## macOS

### Option A — Homebrew _(planned, not yet published)_

```bash
brew install AgusLoza2021/tap/thoughtline
```

This will land once the v0.0.1 release is cut and the Homebrew tap is published. Until then, use Option B.

### Option B — Go install

Requires [Go 1.25+](https://go.dev/dl/).

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
```

`go install` drops the binary in `$(go env GOPATH)/bin` (default `~/go/bin`). Make sure that directory is on your `PATH`:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc   # or ~/.bashrc
source ~/.zshrc
```

---

## Linux

Same as macOS Option B — `go install`. Prebuilt `tar.gz` archives will land in [GitHub Releases](https://github.com/AgusLoza2021/Thoughtline/releases) once v0.0.1 ships (built by [`.goreleaser.yaml`](../.goreleaser.yaml) for `amd64` and `arm64`).

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
```

---

## Build from source

For contributors or people on architectures we don't ship binaries for.

```bash
git clone https://github.com/AgusLoza2021/Thoughtline.git
cd Thoughtline
go build -o thoughtline ./cmd/thoughtline    # or `go install ./cmd/thoughtline`
./thoughtline version
```

The binary is **CGO-free** (we use [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite), a pure-Go SQLite). No C toolchain needed.

---

## Register the MCP server

If you used the **Windows auto-installer**, this is already done. Skip to [Verify](#verify).

If you installed manually, add Thoughtline to your AI client's MCP config.

### Claude Code (CLI + VS Code extension share `~/.claude.json`)

Add this block to `~/.claude.json` (or `%USERPROFILE%\.claude.json` on Windows):

```json
{
  "mcpServers": {
    "thoughtline": {
      "type": "stdio",
      "command": "thoughtline",
      "args": ["serve"]
    }
  }
}
```

If the file already has other `mcpServers` entries, merge — don't overwrite.

### Cursor

See [docs/integrations/cursor.md](integrations/cursor.md).

### Zed

See [docs/integrations/zed.md](integrations/zed.md).

### JetBrains Rider (Unity workflows)

See [docs/integrations/rider-unity.md](integrations/rider-unity.md).

### Other MCP clients

Anything that speaks MCP over stdio works. The launch command is always:

```
thoughtline serve
```

---

## Install the Claude Code plugin (optional, but recommended)

The [official plugin](../plugin/claude-code/) bundles MCP server registration, slash commands (`/tl-recent`, `/tl-search`, `/tl-stats`, `/tl-ui`, `/tl-export`), the `tl-archivist` subagent for memory hygiene, and the `thoughtline-memory` skill that tells Claude when to save and recall.

```bash
claude plugin install github:AgusLoza2021/Thoughtline/plugin/claude-code
```

The plugin shells out to `thoughtline protocol` instead of bash scripts, so Windows users do not need WSL or Git Bash.

See [plugin/claude-code/README.md](../plugin/claude-code/README.md) for the full plugin reference.

---

## Configuration

Thoughtline reads three environment variables. All are optional.

| Variable | Default | Purpose |
|---|---|---|
| `THOUGHTLINE_HOME` | _platform-specific, see below_ | Directory for the SQLite database. The DB file inside is always `thoughtline.db`. |
| `THOUGHTLINE_DB` | _(unset)_ | Full file path for the SQLite database. **Wins over `THOUGHTLINE_HOME` if both are set.** Useful for tests or demo recordings. |
| `THOUGHTLINE_PROJECT` | basename of the working directory at startup, falling back to `default` | Default project identifier when a tool call omits `project`. |

**Platform-specific defaults** for the database (per [ADR 0003](decisions/0003-state-dir-not-cache-dir.md) — user data dir, not cache dir, so OS cleanup tools won't wipe it):

| Platform | Default path |
|---|---|
| Windows | `%LOCALAPPDATA%\thoughtline\thoughtline.db` |
| macOS | `~/Library/Application Support/thoughtline/thoughtline.db` |
| Linux | `${XDG_DATA_HOME:-~/.local/share}/thoughtline/thoughtline.db` |

> **Upgrading from a build that pre-dated ADR 0003?** On first launch, Thoughtline auto-migrates the database from the legacy cache-dir location to the new data-dir location. A one-line note is printed to stderr; nothing else changes. If you set `THOUGHTLINE_HOME` or `THOUGHTLINE_DB` explicitly, migration is bypassed (those overrides are respected as-is).

### Examples

```bash
# Sandbox the demo DB during a screen recording
export THOUGHTLINE_DB=/tmp/thoughtline-demo.db
export THOUGHTLINE_PROJECT=thoughtline-demo
thoughtline ui
```

```powershell
# Windows equivalent
$env:THOUGHTLINE_DB = "$env:TEMP\thoughtline-demo.db"
$env:THOUGHTLINE_PROJECT = "thoughtline-demo"
thoughtline ui
```

---

## Verify

```bash
thoughtline version
thoughtline help
thoughtline ui                  # opens the dashboard
```

To verify Claude Code sees it: open the Claude Code CLI and run `/mcp` (or check the MCP panel in the VS Code extension). You should see `thoughtline` listed with green status.

---

## Update

### Windows installer path

```powershell
.\scripts\install.ps1                       # latest
.\scripts\install.ps1 -Version v0.1.0       # pin a tag
```

### Manual / `go install`

```bash
go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@latest
```

Restart your AI client after upgrading.

---

## Uninstall

Remove the binary:

```bash
# macOS / Linux
rm "$(go env GOPATH)/bin/thoughtline"

# Windows (PowerShell)
Remove-Item "$env:USERPROFILE\go\bin\thoughtline.exe"
```

Remove the MCP entry from `~/.claude.json` (delete the `thoughtline` key under `mcpServers`).

Optionally delete the database (this is the only place your memories live):

```bash
# macOS
rm -rf "~/Library/Application Support/thoughtline"

# Linux
rm -rf "$XDG_DATA_HOME/thoughtline"

# Windows
Remove-Item "$env:LOCALAPPDATA\thoughtline" -Recurse
```

---

## Troubleshooting

**`thoughtline: command not found` after install**
Your shell hasn't seen the new PATH entry yet. Open a fresh terminal, or run `source ~/.zshrc` / restart PowerShell.

**Claude Code says the MCP server "failed to start"**
Run `thoughtline serve` manually from a terminal. If it crashes, the error message tells you why (most often: locked SQLite file, or `THOUGHTLINE_DB` points to a path the user can't write to). If it stays alive and waits silently, that's correct behavior — `serve` reads JSON-RPC over stdin.

**Windows installer says "winget: command not found"**
You're on a Windows version older than 10 1809 or winget got disabled. Install Go manually from <https://go.dev/dl/> and re-run the installer; it will skip the winget step and proceed.

**TUI looks broken in classic `cmd.exe`**
Use **Windows Terminal** or **PowerShell 7+**. Bubbletea relies on full ANSI support that legacy `conhost` doesn't have.

**`go install` fails with `module not found`**
Confirm you typed the import path exactly: `github.com/AgusLoza2021/Thoughtline/cmd/thoughtline`. Capitalization on `Thoughtline` matters because Go modules are case-sensitive.

**Two MCP servers writing to the same DB**
Don't. SQLite handles concurrent reads fine but Thoughtline assumes a single writer. If you must, use separate `THOUGHTLINE_DB` paths per process.

---

## Next

- [README.md](../README.md) — project overview and quickstart
- [COMPARISON.md](COMPARISON.md) — vs Engram, vs claude-mem
- [ARCHITECTURE.md](ARCHITECTURE.md) — internals (storage, MCP surface, dashboard)
- [docs/integrations/](integrations/) — per-editor setup
- [plugin/claude-code/README.md](../plugin/claude-code/README.md) — Claude Code plugin reference
