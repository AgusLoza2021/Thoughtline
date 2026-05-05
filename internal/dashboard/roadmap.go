// Package dashboard implements the `thoughtline ui` Bubbletea TUI.
//
// It opens the same SQLite store the MCP server uses, calls storage.Stats,
// and renders a multi-tab dashboard so a developer can see at a glance
// what's stored, browse it, search it, and review session activity.
package dashboard

import (
	_ "embed"
	"sync"

	"gopkg.in/yaml.v3"
)

// Milestone is a roadmap entry rendered in the Overview panel of the TUI.
type Milestone struct {
	ID     string `yaml:"id"`     // "M0", "M1", ...
	Name   string `yaml:"name"`   // short title
	Status string `yaml:"status"` // "done", "in-progress", "next", "deferred"
}

//go:embed roadmap.yaml
var roadmapYAML []byte

type roadmapDoc struct {
	Milestones []Milestone `yaml:"milestones"`
}

var (
	roadmapOnce  sync.Once
	roadmapCache []Milestone
)

// Roadmap returns the parsed milestone list, embedded from roadmap.yaml at
// build time. The first call parses; subsequent calls return the cached
// slice. Tests that mutate the slice should copy first — Roadmap shares
// memory by design (cheap, read-only data).
func Roadmap() []Milestone {
	roadmapOnce.Do(func() {
		var doc roadmapDoc
		if err := yaml.Unmarshal(roadmapYAML, &doc); err != nil {
			// Should never happen — roadmap.yaml is checked in and embedded.
			// On failure we fall back to a hardcoded sentinel so the TUI
			// stays usable rather than crashing on startup.
			roadmapCache = []Milestone{{ID: "??", Name: "(roadmap parse failed)", Status: "next"}}
			return
		}
		roadmapCache = doc.Milestones
	})
	return roadmapCache
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
