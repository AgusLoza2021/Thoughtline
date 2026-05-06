# Thoughtline TUI demo recorder for vhs.
# https://github.com/charmbracelet/vhs
#
# Usage:
#   1. Install vhs:   `go install github.com/charmbracelet/vhs@latest`
#                     (vhs needs ttyd + ffmpeg on PATH; vhs --version will
#                     tell you if anything's missing)
#   2. Seed a demo database with a handful of synthetic memories so the
#      Browse/Search tabs render real content. Quickest path:
#         set THOUGHTLINE_DB=%TEMP%\thoughtline-demo.db
#         (use Claude Code or another MCP client to call tl_save a few times,
#          or run a small seeding script — see docs/media/README.md)
#   3. From the repo root:
#         vhs scripts/demo.tape
#      Output lands at docs/media/demo.gif.
#
# Keybindings exercised (see internal/dashboard/update.go):
#   1-6  jump to tab (Overview, Browse, Search, Sessions, Tags, Help)
#   tab/shift+tab  cycle tabs
#   /    enter the search input
#   t    cycle theme (brand -> zbrush -> mono)
#   r    refresh
#   ?    toggle help
#   q    quit

Output docs/media/demo.gif

Set FontSize 14
Set Width 1280
Set Height 720
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms
Set PlaybackSpeed 1.0

# Use the seeded demo DB; skip update check so the network call doesn't slow the splash.
Env THOUGHTLINE_DB "${TEMP}/thoughtline-demo.db"
Env THOUGHTLINE_PROJECT "thoughtline-demo"

Type "thoughtline ui --no-update-check"
Enter
Sleep 2500ms   # splash + first render

# --- Overview: hero cube, status bar, badges ---
Sleep 1500ms

# --- Browse tab ---
Type "2"
Sleep 2000ms

# --- Search tab ---
Type "3"
Sleep 800ms
Type "/"
Sleep 300ms
Type "render"
Sleep 600ms
Enter
Sleep 1800ms
Enter            # open the top result
Sleep 2500ms
Escape           # back to results
Sleep 500ms

# --- Theme switcher: brand -> zbrush -> mono -> brand ---
Type "1"         # overview again so the cube shows the theme change
Sleep 700ms
Type "t"
Sleep 1500ms
Type "t"
Sleep 1500ms
Type "t"
Sleep 1000ms

# --- Help ---
Type "?"
Sleep 1500ms
Type "?"
Sleep 500ms

Type "q"
