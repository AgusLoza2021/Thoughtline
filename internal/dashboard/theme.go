package dashboard

import "github.com/charmbracelet/lipgloss"

// palette is the single fixed colour palette for Thoughtline's TUI.
// Based on Rose-Pine Moon, adapted for gamedev "warm/dark/punchy" aesthetics.
// No runtime theme-switching; this is the one and only palette.
type palette struct {
	Foreground    lipgloss.Color
	Muted         lipgloss.Color
	Border        lipgloss.Color
	LogoGradient  [5]lipgloss.Color
	StatNumber    lipgloss.Color
	Cursor        lipgloss.Color
	MenuSelectedBg lipgloss.Color
	StatusOK      lipgloss.Color
	StatusWarn    lipgloss.Color
	StatusErr     lipgloss.Color
	Tag           lipgloss.Color
}

// defaultPalette is the one palette value used across all screens.
var defaultPalette = palette{
	Foreground:    lipgloss.Color("#E0DEF4"),
	Muted:         lipgloss.Color("#6E6A86"),
	Border:        lipgloss.Color("#393552"),
	LogoGradient: [5]lipgloss.Color{
		lipgloss.Color("#C4A7E7"), // mauve  (top)
		lipgloss.Color("#A88DC9"), // lavender
		lipgloss.Color("#9CCFD8"), // blue-cyan
		lipgloss.Color("#3E8FB0"), // teal
		lipgloss.Color("#56949F"), // green-teal (bottom)
	},
	StatNumber:    lipgloss.Color("#EB6F92"),
	Cursor:        lipgloss.Color("#F6C177"),
	MenuSelectedBg: lipgloss.Color("#2A273F"),
	StatusOK:      lipgloss.Color("#9CCFD8"),
	StatusWarn:    lipgloss.Color("#F6C177"),
	StatusErr:     lipgloss.Color("#EB6F92"),
	Tag:           lipgloss.Color("#C4A7E7"),
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
