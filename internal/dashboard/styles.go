package dashboard

import "github.com/charmbracelet/lipgloss"

// All package-level styles below are RECOMPUTED by ApplyTheme. They
// are declared as `var`s rather than constants because lipgloss styles
// are values, not constants, and we want to swap them at runtime when
// the user cycles themes via the [t] hotkey.
//
// Order of init: package init runs `ApplyTheme(ThemeBrand)` so the
// styles are valid even if no theme has been explicitly applied yet
// (e.g. during tests that call New() directly).

var currentTheme Theme

// Color shorthands kept as package-level vars for readability in
// view.go. They are rebuilt by ApplyTheme.
var (
	colBrand       lipgloss.AdaptiveColor
	colBrandSolid  lipgloss.Color
	colAccent      lipgloss.AdaptiveColor
	colSuccess     lipgloss.AdaptiveColor
	colWarning     lipgloss.AdaptiveColor
	colDanger      lipgloss.AdaptiveColor
	colText        lipgloss.AdaptiveColor
	colTextOnBrand lipgloss.Color
	colMuted       lipgloss.AdaptiveColor
	colSubtle      lipgloss.AdaptiveColor
	colBorder      lipgloss.AdaptiveColor
	colSurface     lipgloss.AdaptiveColor
)

// Style instances. These all read from the colors above so swapping
// the theme means rebuilding these structs.
var (
	// Header
	brandPillStyle     lipgloss.Style
	versionPillStyle   lipgloss.Style
	breadcrumbSepStyle lipgloss.Style
	headerMetaStyle    lipgloss.Style
	headerKeyStyle     lipgloss.Style

	// Status bar
	statusBarStyle    lipgloss.Style
	statusOnlineStyle lipgloss.Style
	statusMetricStyle lipgloss.Style
	statusValueStyle  lipgloss.Style
	statusUpdateStyle lipgloss.Style
	statusSepStyle    lipgloss.Style

	// Title (used in detail panes)
	titleStyle  lipgloss.Style
	subtleStyle lipgloss.Style

	// Panels
	panelStyle      lipgloss.Style
	panelTitleStyle lipgloss.Style

	// Stats
	statsNumberStyle lipgloss.Style
	statsLabelStyle  lipgloss.Style

	// Roadmap
	doneStyle       lipgloss.Style
	nextStyle       lipgloss.Style
	inProgressStyle lipgloss.Style
	deferredStyle   lipgloss.Style

	errStyle lipgloss.Style

	// Tabs
	tabActiveStyle   lipgloss.Style
	tabInactiveStyle lipgloss.Style
	tabSepStyle      lipgloss.Style

	// Footer
	footerKeyStyle   lipgloss.Style
	footerLabelStyle lipgloss.Style
	footerSepStyle   lipgloss.Style
	footerStyle      lipgloss.Style

	// Help / keybinding rows
	keyStyle  lipgloss.Style
	hintStyle lipgloss.Style

	// Cube / hero
	cubeStyle lipgloss.Style
)

// ApplyTheme rebuilds every package-level style from the given theme.
// Call this once at startup and again whenever the user cycles themes.
// Concurrency note: Bubbletea runs Update on a single goroutine, so we
// don't lock here.
func ApplyTheme(t Theme) {
	currentTheme = t

	colBrand = t.Brand
	colBrandSolid = t.BrandSolid
	colAccent = t.Accent
	colSuccess = t.Success
	colWarning = t.Warning
	colDanger = t.Danger
	colText = t.Text
	colTextOnBrand = t.TextOnBrand
	colMuted = t.Muted
	colSubtle = t.Subtle
	colBorder = t.Border
	colSurface = t.Surface

	// Header
	brandPillStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colTextOnBrand).
		Background(colBrandSolid).
		Padding(0, 1)

	versionPillStyle = lipgloss.NewStyle().
		Foreground(colText).
		Background(colSurface).
		Padding(0, 1)

	breadcrumbSepStyle = lipgloss.NewStyle().
		Foreground(colSubtle).
		SetString(" › ")

	headerMetaStyle = lipgloss.NewStyle().Foreground(colMuted)
	headerKeyStyle = lipgloss.NewStyle().Foreground(colAccent)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
		Foreground(colText).
		Background(colSurface).
		Padding(0, 1)
	statusOnlineStyle = lipgloss.NewStyle().Bold(true).Foreground(colSuccess)
	statusMetricStyle = lipgloss.NewStyle().Foreground(colMuted)
	statusValueStyle = lipgloss.NewStyle().Bold(true).Foreground(colText)
	statusUpdateStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colTextOnBrand).
		Background(colWarning).
		Padding(0, 1)
	statusSepStyle = lipgloss.NewStyle().Foreground(colBorder).SetString(" · ")

	// Title / subtle
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(colBrand)
	subtleStyle = lipgloss.NewStyle().Foreground(colMuted)

	// Panels
	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBorder).
		Padding(0, 1)
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colAccent)

	// Stats
	statsNumberStyle = lipgloss.NewStyle().Bold(true).Foreground(colBrand)
	statsLabelStyle = lipgloss.NewStyle().Foreground(colMuted)

	// Roadmap
	doneStyle = lipgloss.NewStyle().Foreground(colSuccess)
	nextStyle = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	inProgressStyle = lipgloss.NewStyle().Bold(true).Foreground(colWarning)
	deferredStyle = lipgloss.NewStyle().Foreground(colSubtle).Faint(true)

	errStyle = lipgloss.NewStyle().Bold(true).Foreground(colDanger)

	// Tabs
	tabActiveStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colBrand).
		Underline(true).
		Padding(0, 1)
	tabInactiveStyle = lipgloss.NewStyle().Foreground(colMuted).Padding(0, 1)
	tabSepStyle = lipgloss.NewStyle().Foreground(colBorder).SetString("│")

	// Footer
	footerKeyStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colTextOnBrand).
		Background(colBrand).
		Padding(0, 1)
	footerLabelStyle = lipgloss.NewStyle().Foreground(colMuted)
	footerSepStyle = lipgloss.NewStyle().Foreground(colBorder).SetString(" · ")
	footerStyle = lipgloss.NewStyle().MarginTop(1)

	// Help
	keyStyle = lipgloss.NewStyle().Bold(true).Foreground(colBrand)
	hintStyle = lipgloss.NewStyle().Foreground(colAccent)

	// Cube
	cubeStyle = lipgloss.NewStyle().Foreground(colBrand)
}

func init() {
	ApplyTheme(ThemeBrand)
}
