# ADR 0003 — Store memory data in user data dir, not user cache dir

- **Status**: Accepted
- **Date**: 2026-05-05
- **Supersedes**: —
- **Related**: [ADR 0001](0001-architecture-baseline.md)

## Context

`cmd/thoughtline/main.go::resolveDBPath()` resolves the default location of `thoughtline.db` by calling Go's `os.UserCacheDir()`. That returns:

- Linux: `$XDG_CACHE_HOME` or `~/.cache`
- macOS: `~/Library/Caches`
- Windows: `%LOCALAPPDATA%`

The package-level docstring on `main.go` claims the defaults are `$XDG_DATA_HOME` / `~/Library/Application Support` / `%LOCALAPPDATA%`. **Comment and code disagree on macOS and Linux.** The comment is what we want; the code is what ships.

This is not just a cosmetic mismatch. **Cache directories are by convention disposable.** OS-level cleanup tools (CleanMyMac, BleachBit, `~/.cache` clearing scripts, fresh-Mac restore flows, container image rebuilds) are entitled to wipe their contents and routinely do. If a user runs one of those tools after a year of Thoughtline use, they lose every memory their AI ever saved. That is not the contract a memory system should ship with.

The right destination on each platform is the **user data** directory:

- Linux: `$XDG_DATA_HOME` (default `~/.local/share`)
- macOS: `~/Library/Application Support`
- Windows: `%LOCALAPPDATA%` — the same path `os.UserCacheDir()` returns on Windows. Windows doesn't distinguish cache vs data on disk; the contract is set by the application, and our app declares this is data.

## Decision

**Resolve the default home directory from a platform-specific data dir, not the cache dir. Auto-migrate the DB on first launch after the change so existing users don't lose anything.**

Concretely:

1. Add `dataDir()` helper in `cmd/thoughtline/main.go` that returns:
   - **Windows**: `%LOCALAPPDATA%` (read from env, falling back to `os.UserCacheDir()` which is the same value).
   - **macOS**: `~/Library/Application Support`.
   - **Linux / BSDs**: `$XDG_DATA_HOME` else `~/.local/share`.
2. `resolveDBPath()` uses `dataDir()` for the default home. The precedence stays:
   1. `THOUGHTLINE_DB` (full path) wins.
   2. `THOUGHTLINE_HOME` directory wins over the default.
   3. Fall back to `dataDir() / thoughtline / thoughtline.db`.
3. **One-time auto-migration.** Before deciding the default path is missing, check whether the **old** path (`os.UserCacheDir() / thoughtline /`) contains a `thoughtline.db`. If yes and the new path doesn't, move the entire `thoughtline` subdirectory from old to new. Log to stderr that the move happened. This runs on every default-path resolution but is a no-op once the new path exists.
4. Migration only applies to the **default** path, never when `THOUGHTLINE_HOME` or `THOUGHTLINE_DB` is set — those are explicit user overrides and we don't second-guess them.
5. Update `main.go` package docstring, [INSTALLATION.md](../INSTALLATION.md), and [ARCHITECTURE.md](../ARCHITECTURE.md) to reflect the new defaults.

## Consequences

### Positive

- Memory data is no longer at risk from cache-cleaning tools.
- Existing users transparently keep their data on first launch after upgrading.
- The commented intent and the code finally agree.
- On macOS, `~/Library/Application Support/thoughtline/thoughtline.db` shows up in Time Machine backups by default. That's the right behavior for a memory system.

### Negative

- One-time silent file move on first launch. The stderr log line ("moved DB from old cache path to new data path") makes it visible but it's not a confirmation prompt. We accept this — the migration is conservative (only when source exists and destination doesn't) and the alternative (prompt) is worse UX for a tool that runs as a background MCP server.
- A user who manually deleted the cache dir between an old build and the new build won't have anything to migrate. They get a fresh DB at the new location. This is the same outcome they had before; we just didn't pretend to recover.
- A user who runs both an old `thoughtline` (still pointing at cache) and the new build (pointing at data) on the same machine will end up with two divergent databases. The migration handles this only if old exists and new doesn't, so once new exists, old becomes orphan. This is acceptable: nobody runs two builds intentionally.

### Neutral

- No schema change. No data format change. Just the directory.
- Windows users see no observable change — the path was already `%LOCALAPPDATA%`.

## Alternatives considered

**Comment-only fix (update docstring to say "Caches", change nothing else).** Cheapest. Rejected because it institutionalizes a real bug behind cosmetic accuracy.

**Use `os.UserConfigDir()` instead.** Returns `~/Library/Application Support` on macOS (right), but `~/.config` on Linux (wrong — that's config not data) and `%AppData%` (Roaming) on Windows (wrong — Thoughtline data is machine-local, not roaming). Doesn't match what we want on two of three platforms.

**Use `XDG_STATE_HOME` on Linux.** XDG defines state for "data that should persist between (application) restarts but isn't important enough to back up." Memories are explicitly important enough to back up — that's the whole point. `XDG_DATA_HOME` is the right semantic match.

## Implementation note

The migration helper is called from `resolveDBPath()` only when neither override is set. It does not log on no-op runs (when there's nothing to move), so steady-state startup stays quiet.
