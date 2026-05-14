package dashboard

// Config carries the wiring the dashboard needs from main(). DBPath and
// Version come from the binary's runtime; Project is the auto-detected
// default ("" means "show all projects").
//
// As of v0.2 the legacy ThemeName, Splash, and SplashDuration fields have
// been removed (commit 13 of the tui-memory-workspace SDD plan). The TUI
// renders a single semantic palette and has no splash animation.
type Config struct {
	Version string
	DBPath  string
	Project string

	// CheckUpdates toggles the GitHub release lookup. Default true in
	// the binary, false in tests so we don't hit the network.
	CheckUpdates bool
}
