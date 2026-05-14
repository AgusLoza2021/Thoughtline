package dashboard

import "time"

// Config carries the wiring the dashboard needs from main(). DBPath and
// Version come from the binary's runtime; Project is the auto-detected
// default ("" means "show all projects").
type Config struct {
	Version string
	DBPath  string
	Project string

	// ThemeName is preserved for CLI flag backwards compatibility. The
	// active TUI ignores it and renders the single semantic palette.
	ThemeName string

	// Splash and SplashDuration drive the legacy intro animation. The
	// new flat dashboard ignores them. Retained so --no-splash and
	// --splash-ms keep parsing until the CLI flags are removed (commit 13
	// of the tui-memory-workspace change).
	Splash         bool
	SplashDuration time.Duration

	// CheckUpdates toggles the GitHub release lookup. Default true in
	// the binary, false in tests so we don't hit the network.
	CheckUpdates bool
}
