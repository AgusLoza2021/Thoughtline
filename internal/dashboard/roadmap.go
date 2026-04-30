// Package dashboard implements the `thoughtline ui` Bubbletea TUI.
//
// It opens the same SQLite store the MCP server uses, calls storage.Stats,
// and renders a four-panel dashboard so a developer can see at a glance
// what's stored and how the project is progressing.
package dashboard

// Milestone is a roadmap entry rendered in the bottom panel of the TUI.
type Milestone struct {
	ID     string // "M0", "M1", ...
	Name   string // short title
	Status string // "done", "next", "deferred"
}

// Roadmap returns the hardcoded milestone list. Updated by hand alongside
// docs/PROGRESS.md when a milestone changes state. Kept here (not derived
// from git tags or a JSON file) because it's load-bearing UX content that
// needs review at PR time, not runtime mystery.
func Roadmap() []Milestone {
	return []Milestone{
		{"M0", "Bootstrap", "done"},
		{"M1", "Save", "done"},
		{"M2", "Search", "done"},
		{"M3", "Context / Update / Delete", "done"},
		{"M4", "Sessions", "done"},
		{"M5", "Dashboard (TUI + tl_stats)", "done"},
		{"M6", "Smarts (embeddings)", "deferred"},
	}
}

// StatusGlyph returns a one-character emoji-free marker for a milestone
// status — keeps the TUI rendering portable across terminals that handle
// emoji unevenly.
func StatusGlyph(status string) string {
	switch status {
	case "done":
		return "✓"
	case "next":
		return "→"
	case "deferred":
		return "·"
	case "in-progress":
		return "*"
	default:
		return "?"
	}
}
