package dashboard

import "github.com/charmbracelet/lipgloss"

// styles.go owns the lipgloss style values used by every Screen. As of the
// tui-memory-workspace refactor (commit 3), styles depend on EXACTLY ONE
// source of truth: the single semantic `defaultPalette` declared in
// theme.go. The legacy multi-theme machinery has been decoupled — see
// themes.go for the no-op shim retained so the legacy Model continues to
// compile until it is deleted in a follow-up commit. The legacy shims no
// longer drive the style vars below.
//
// The package-level style vars below are populated once at init time via
// rebuildStyles(defaultPalette). The legacy Model continues to render
// correctly because it reads these same vars (which never change again
// once init has run).

// Style instances. Read by view.go (legacy), workstation_screen.go, and the
// upcoming Home / Memories / Inbox / Search / Sessions / Help tab screens.
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

// rebuildStyles populates every lipgloss style var above from the given
// palette. Called once from init() against defaultPalette. No multi-theme
// runtime swap path — the package has one palette and it does not change.
func rebuildStyles(p palette) {
	// Header
	brandPillStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Foreground).
		Background(p.Border).
		Padding(0, 1)

	versionPillStyle = lipgloss.NewStyle().
		Foreground(p.Foreground).
		Background(p.Border).
		Padding(0, 1)

	breadcrumbSepStyle = lipgloss.NewStyle().
		Foreground(p.Muted).
		SetString(" › ")

	headerMetaStyle = lipgloss.NewStyle().Foreground(p.Muted)
	headerKeyStyle = lipgloss.NewStyle().Foreground(p.StatusOK)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
		Foreground(p.Foreground).
		Padding(0, 1)
	statusOnlineStyle = lipgloss.NewStyle().Bold(true).Foreground(p.StatusOK)
	statusMetricStyle = lipgloss.NewStyle().Foreground(p.Muted)
	statusValueStyle = lipgloss.NewStyle().Bold(true).Foreground(p.Foreground)
	statusUpdateStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Foreground).
		Background(p.StatusWarn).
		Padding(0, 1)
	statusSepStyle = lipgloss.NewStyle().Foreground(p.Border).SetString(" · ")

	// Title / subtle
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(p.Tag)
	subtleStyle = lipgloss.NewStyle().Foreground(p.Muted)

	// Panels
	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Border).
		Padding(0, 1)
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(p.StatusOK)

	// Stats
	statsNumberStyle = lipgloss.NewStyle().Bold(true).Foreground(p.Tag)
	statsLabelStyle = lipgloss.NewStyle().Foreground(p.Muted)

	// Roadmap
	doneStyle = lipgloss.NewStyle().Foreground(p.StatusOK)
	nextStyle = lipgloss.NewStyle().Bold(true).Foreground(p.StatusOK)
	inProgressStyle = lipgloss.NewStyle().Bold(true).Foreground(p.StatusWarn)
	deferredStyle = lipgloss.NewStyle().Foreground(p.Muted).Faint(true)

	errStyle = lipgloss.NewStyle().Bold(true).Foreground(p.StatusErr)

	// Tabs
	tabActiveStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Tag).
		Underline(true).
		Padding(0, 1)
	tabInactiveStyle = lipgloss.NewStyle().Foreground(p.Muted).Padding(0, 1)
	tabSepStyle = lipgloss.NewStyle().Foreground(p.Border).SetString("│")

	// Footer
	footerKeyStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Foreground).
		Background(p.Tag).
		Padding(0, 1)
	footerLabelStyle = lipgloss.NewStyle().Foreground(p.Muted)
	footerSepStyle = lipgloss.NewStyle().Foreground(p.Border).SetString(" · ")
	footerStyle = lipgloss.NewStyle().MarginTop(1)

	// Help
	keyStyle = lipgloss.NewStyle().Bold(true).Foreground(p.Tag)
	hintStyle = lipgloss.NewStyle().Foreground(p.StatusOK)

	// Cube
	cubeStyle = lipgloss.NewStyle().Foreground(p.Tag)
}

func init() {
	rebuildStyles(defaultPalette)
}
