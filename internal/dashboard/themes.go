package dashboard

import "github.com/charmbracelet/lipgloss"

// Theme is the legacy multi-theme container used by the soon-to-be-deleted
// legacy Model (model.go / view.go / update.go). The active TUI no longer
// honors theme cycling — styles.go is now driven by the single semantic
// palette in theme.go. ApplyTheme is retained only as a no-op shim so the
// legacy Model continues to compile until it is deleted in a follow-up
// commit. Once that happens this entire file goes with it.
//
// Theme was previously used as the source-of-truth for styles.go; the
// multi-theme infrastructure was decoupled in commit 3 of the
// tui-memory-workspace change.
//
// Colors use AdaptiveColor so a single theme renders well on both light
// and dark terminal backgrounds. The "ZBrush" theme breaks that
// convention deliberately: it is a dark-only theme, so its Light side
// just keeps the dark values (we want the same warm look regardless
// of the user's terminal bg).
type Theme struct {
	Name string

	// Identity
	Brand      lipgloss.AdaptiveColor // primary brand color (header pills, key glyphs)
	BrandSolid lipgloss.Color         // single hex for components that can't take adaptive

	// Accent + semantic
	Accent  lipgloss.AdaptiveColor
	Success lipgloss.AdaptiveColor
	Warning lipgloss.AdaptiveColor
	Danger  lipgloss.AdaptiveColor

	// Surfaces and text
	Text        lipgloss.AdaptiveColor
	TextOnBrand lipgloss.Color
	Muted       lipgloss.AdaptiveColor
	Subtle      lipgloss.AdaptiveColor
	Border      lipgloss.AdaptiveColor
	Surface     lipgloss.AdaptiveColor
}

// ThemeBrand — the original violet+cyan SaaS-tech palette. Default.
var ThemeBrand = Theme{
	Name:        "brand",
	Brand:       lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#A78BFA"},
	BrandSolid:  lipgloss.Color("#7C3AED"),
	Accent:      lipgloss.AdaptiveColor{Light: "#0891B2", Dark: "#22D3EE"},
	Success:     lipgloss.AdaptiveColor{Light: "#047857", Dark: "#34D399"},
	Warning:     lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"},
	Danger:      lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"},
	Text:        lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"},
	TextOnBrand: lipgloss.Color("#FFFFFF"),
	Muted:       lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"},
	Subtle:      lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"},
	Border:      lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"},
	Surface:     lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#1F2937"},
}

// ThemeZBrush — warm tactile palette inspired by Pixologic ZBrush.
// Warm grays tending to brown, signature amber/orange accents, cream
// text. Dark-only by design (Light values mirror Dark).
var ThemeZBrush = Theme{
	Name:        "zbrush",
	Brand:       lipgloss.AdaptiveColor{Light: "#D68A3C", Dark: "#D68A3C"},
	BrandSolid:  lipgloss.Color("#D68A3C"),
	Accent:      lipgloss.AdaptiveColor{Light: "#E8A858", Dark: "#E8A858"},
	Success:     lipgloss.AdaptiveColor{Light: "#B5A642", Dark: "#B5A642"},
	Warning:     lipgloss.AdaptiveColor{Light: "#E8A858", Dark: "#E8A858"},
	Danger:      lipgloss.AdaptiveColor{Light: "#C25450", Dark: "#C25450"},
	Text:        lipgloss.AdaptiveColor{Light: "#E8DCC4", Dark: "#E8DCC4"},
	TextOnBrand: lipgloss.Color("#1A1A1A"),
	Muted:       lipgloss.AdaptiveColor{Light: "#A89F8C", Dark: "#A89F8C"},
	Subtle:      lipgloss.AdaptiveColor{Light: "#7A7163", Dark: "#7A7163"},
	Border:      lipgloss.AdaptiveColor{Light: "#4A443D", Dark: "#4A443D"},
	Surface:     lipgloss.AdaptiveColor{Light: "#2D2A26", Dark: "#2D2A26"},
}

// ThemeMono — minimalist grayscale. Useful for screenshots, slides,
// and low-color terminals.
var ThemeMono = Theme{
	Name:        "mono",
	Brand:       lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"},
	BrandSolid:  lipgloss.Color("#000000"),
	Accent:      lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"},
	Success:     lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"},
	Warning:     lipgloss.AdaptiveColor{Light: "#374151", Dark: "#D1D5DB"},
	Danger:      lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"},
	Text:        lipgloss.AdaptiveColor{Light: "#111827", Dark: "#F9FAFB"},
	TextOnBrand: lipgloss.Color("#FFFFFF"),
	Muted:       lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"},
	Subtle:      lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"},
	Border:      lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#4B5563"},
	Surface:     lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#1F2937"},
}

// AllThemes is the cycle order used by the [t] hotkey.
var AllThemes = []Theme{ThemeBrand, ThemeZBrush, ThemeMono}

// ThemeByName returns the theme with the matching Name, or ThemeBrand
// when no match is found. Comparison is case-insensitive on the caller
// side — pass the name lowercased.
func ThemeByName(name string) Theme {
	for _, t := range AllThemes {
		if t.Name == name {
			return t
		}
	}
	return ThemeBrand
}

// nextTheme returns the next theme in AllThemes after the current one,
// wrapping around. Used by the [t] hotkey on the legacy Model.
func nextTheme(current Theme) Theme {
	for i, t := range AllThemes {
		if t.Name == current.Name {
			return AllThemes[(i+1)%len(AllThemes)]
		}
	}
	return AllThemes[0]
}

// ApplyTheme is a legacy no-op shim retained so the legacy Model's update path
// (m.theme = nextTheme(m.theme); ApplyTheme(m.theme)) keeps compiling. Styles
// are no longer rebuilt — the active TUI uses a single semantic palette built
// once at init time in styles.go via rebuildStyles(defaultPalette).
//
// This function will be deleted alongside the legacy Model in a follow-up
// commit of the tui-memory-workspace change.
func ApplyTheme(_ Theme) {
	// no-op: see styles.go and theme.go
}
