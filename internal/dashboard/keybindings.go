package dashboard

// keybindings.go is the single source of truth for documented hotkeys in the
// flat workspace TUI. The Help tab renders this registry; the drift test in
// keybindings_test.go guards against the registry going out of sync with the
// actual key handling wired in flat_model.go's Update ladder.
//
// Per design Section 2b(J): documentation is data, not strings sprinkled in
// view code. To add or change a hotkey, update both this file AND the
// corresponding case in flat_model.go (or the active Screen). The drift test
// will fail loudly when only one side moves.

// Keybind is one documented key-to-action pairing.
type Keybind struct {
	// Keys is the human-readable key spec (e.g. "1-6", "tab / shift+tab",
	// "ctrl+c"). It is what the Help tab renders.
	Keys string
	// Desc is the action description.
	Desc string
}

// KeybindGroup is a labeled section of the Help tab keybindings list.
type KeybindGroup struct {
	Title    string
	Bindings []Keybind
}

// Keybindings is the registry consumed by HelpScreen and the drift test.
// Order matters — the Help tab renders groups top-to-bottom in this order.
var Keybindings = []KeybindGroup{
	{Title: "Navigation", Bindings: []Keybind{
		{Keys: "1-6", Desc: "jump to tab"},
		{Keys: "tab / shift+tab", Desc: "cycle tabs"},
		{Keys: "esc", Desc: "back / pop overlay"},
		{Keys: "q / ctrl+c", Desc: "quit"},
	}},
	{Title: "Quick Actions", Bindings: []Keybind{
		{Keys: "s", Desc: "save (CLI/MCP for now)"},
		{Keys: "/", Desc: "search"},
		{Keys: "m", Desc: "memories"},
		{Keys: "i", Desc: "inbox"},
	}},
	{Title: "Memories tab", Bindings: []Keybind{
		{Keys: "↑↓", Desc: "select row"},
		{Keys: "n / p", Desc: "next / prev page"},
		{Keys: "f", Desc: "cycle filter focus"},
		{Keys: "c", Desc: "clear filters"},
		{Keys: "enter", Desc: "open detail"},
	}},
	{Title: "Detail view", Bindings: []Keybind{
		{Keys: "↑↓ / PgUp / PgDn", Desc: "scroll"},
		{Keys: "C", Desc: "copy to clipboard"},
	}},
	{Title: "Inbox", Bindings: []Keybind{
		{Keys: "A", Desc: "accept (promote as-is)"},
		{Keys: "E", Desc: "edit then promote"},
		{Keys: "R", Desc: "reject"},
	}},
}

// wiredHotkeys is the canonical set of hotkey tokens handled by the
// flatModel.Update ladder (tab/quick-action layer). It is hand-curated and
// asserted by the drift test against the Keybindings registry: every token
// here MUST appear in at least one Keybind.Keys entry, and every key listed
// in Keybindings.Navigation/Quick Actions MUST appear here.
//
// We use a hand-curated allowlist rather than parsing flat_model.go source
// because the latter would produce false positives (e.g. case 'q' appears
// inside InputFocused branches but is functionally one logical handler).
//
// Per-screen hotkeys (Memories ↑↓, Detail C, Inbox A/E/R, etc.) are wired
// inside the respective Screen.Update methods, not the flatModel ladder, so
// they are documented in Keybindings but NOT listed here.
var wiredHotkeys = []string{
	"1", "2", "3", "4", "5", "6", // tab digit jumps
	"tab", "shift+tab", // tab cycle
	"esc",              // overlay pop
	"q", "ctrl+c",      // quit
	"s", "/", "m", "i", // Quick Actions
}
