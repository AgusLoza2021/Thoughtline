package dashboard

import "github.com/charmbracelet/lipgloss"

// palette is the single semantic palette for Thoughtline's TUI. It encodes
// color *roles*, not decorative variants — every consumer reads a named
// token by purpose (Nav, Brand, Success, Warning, Error, Muted, ...) and
// receives a hex value chosen to satisfy that role across the supported
// terminals.
//
// Tokens are taken from the tui-memory-workspace design (Section 3), which
// derives from a Tokyo-Night-Storm dark palette with the gamedev-purple
// brand accent preserved. All foreground values verify for WCAG AA contrast
// on the canonical dark-terminal backgrounds (#1E1E1E VS Code dark+,
// #0C0C0C Windows Terminal default).
//
type palette struct {
	// Core surface roles
	Foreground lipgloss.Color
	Muted      lipgloss.Color
	Border     lipgloss.Color
	FocusedBg  lipgloss.Color

	// Focus / navigation
	BorderFocused lipgloss.Color
	Nav           lipgloss.Color
	NavActive     lipgloss.Color
	NavActiveBg   lipgloss.Color

	// Brand / memory-type accents
	Brand    lipgloss.Color
	BrandDim lipgloss.Color

	// Semantic state colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color

	// Status indicator aliases (kept distinct so future themes could split them)
	StatusOK   lipgloss.Color
	StatusWarn lipgloss.Color
	StatusErr  lipgloss.Color

	// Memory-type badge
	BadgeMemoryType   lipgloss.Color
	BadgeMemoryTypeBg lipgloss.Color

	// Legacy aliases kept until the screens that consume them are rewritten:
	//  - Tag        — used by detail rendering for tags and badges; aliased to Brand.
	//  - StatNumber — used by legacy stat cards; aliased to Brand for now.
	//  - Cursor     — used by legacy list cursor color; aliased to NavActive.
	//  - MenuSelectedBg — used by legacy menu; aliased to NavActiveBg.
	Tag            lipgloss.Color
	StatNumber     lipgloss.Color
	Cursor         lipgloss.Color
	MenuSelectedBg lipgloss.Color

}

// defaultPalette is the one and only palette value used across all screens.
// Per the tui-memory-workspace design (Section 3), it is dark-terminal-first
// and does NOT set a background — terminal transparency is preserved.
var defaultPalette = palette{
	// Core surface roles
	Foreground: lipgloss.Color("#E5E5E5"),
	Muted:      lipgloss.Color("#7A7A85"),
	Border:     lipgloss.Color("#3A3A4A"),
	FocusedBg:  lipgloss.Color("#2A2A3A"),

	// Focus / navigation
	BorderFocused: lipgloss.Color("#7AA2F7"),
	Nav:           lipgloss.Color("#7DCFFF"),
	NavActive:     lipgloss.Color("#7AA2F7"),
	NavActiveBg:   lipgloss.Color("#1F2A44"),

	// Brand / memory-type accents
	Brand:    lipgloss.Color("#C4A7E7"),
	BrandDim: lipgloss.Color("#8E7AB5"),

	// Semantic state colors
	Success: lipgloss.Color("#9ECE6A"),
	Warning: lipgloss.Color("#E0AF68"),
	Error:   lipgloss.Color("#F7768E"),

	// Status indicator aliases
	StatusOK:   lipgloss.Color("#9ECE6A"),
	StatusWarn: lipgloss.Color("#E0AF68"),
	StatusErr:  lipgloss.Color("#F7768E"),

	// Memory-type badge
	BadgeMemoryType:   lipgloss.Color("#C4A7E7"),
	BadgeMemoryTypeBg: lipgloss.Color("#2D2440"),

	// Legacy aliases — map onto the new tokens.
	Tag:            lipgloss.Color("#C4A7E7"),
	StatNumber:     lipgloss.Color("#C4A7E7"),
	Cursor:         lipgloss.Color("#7AA2F7"),
	MenuSelectedBg: lipgloss.Color("#1F2A44"),

}

// statusLevel encodes the three disk/DB health states shown in the header.
type statusLevel int

const (
	statusOK   statusLevel = iota
	statusWARN statusLevel = iota
	statusERR  statusLevel = iota
)

// statusStyle returns a lipgloss.Style with the foreground colour that matches
// the given statusLevel in the provided palette.
func statusStyle(l statusLevel, p palette) lipgloss.Style {
	switch l {
	case statusWARN:
		return lipgloss.NewStyle().Foreground(p.StatusWarn)
	case statusERR:
		return lipgloss.NewStyle().Foreground(p.StatusErr)
	default: // statusOK
		return lipgloss.NewStyle().Foreground(p.StatusOK)
	}
}
