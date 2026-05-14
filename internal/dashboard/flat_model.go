package dashboard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// flatModel is the lightweight tea.Model that drives the new flat-screen TUI.
// It owns only the screen stack, palette, terminal size, and a Quitting flag.
// All real work (loading data, rendering, key handling) happens inside the
// Screen at the top of the stack — flatModel just routes messages and
// reacts to push/pop commands.
//
// The legacy Model (in model.go) is left untouched so its test suite stays
// green; flatModel is the entry point used by Run() and is what the user
// sees when they invoke `thoughtline ui`.
// tabKey identifies one of the six top-level tabs in the new memory-workspace
// TUI. The legacy multi-tab Model used a tabKey enum in tabs.go which is being
// deleted in a later commit; this is the new identifier.
type tabKey int

const (
	TabHome tabKey = iota
	TabMemories
	TabSearch
	TabInbox
	TabSessions
	TabHelp
)

// tabDef describes one tab in the flat workspace.
type tabDef struct {
	Key    tabKey
	Label  string
	Hotkey rune
}

// defaultTabs is the ordered list of tabs presented in the workspace.
var defaultTabs = [6]tabDef{
	{TabHome, "Home", '1'},
	{TabMemories, "Memories", '2'},
	{TabSearch, "Search", '3'},
	{TabInbox, "Inbox", '4'},
	{TabSessions, "Sessions", '5'},
	{TabHelp, "Help", '6'},
}

// inputFocuser is the optional interface a Screen implements to signal whether
// it currently owns a focused text input/textarea. The Update ladder in
// flatModel uses this (via interface assertion) to suppress global hotkeys.
type inputFocuser interface {
	InputFocused() bool
}

// originator is the optional interface a pushed Screen implements to declare
// which tab it was launched from. On esc-pop, flatModel reads this and sets
// activeTab accordingly so the user returns to the right tab.
type originator interface {
	OriginatingTab() tabKey
}

type flatModel struct {
	storage  *storage.Storage
	cfg      Config
	pal      palette
	width    int
	height   int
	stack    []Screen
	Quitting bool

	// activeTab is the currently-rendered tab. Tab dispatch and rendering
	// against `tabs` is implemented in a follow-up commit (Group G); this
	// field exists now so the RED tests for the tab layer can compile.
	activeTab tabKey
	// tabs holds one Screen per tab. Populated in a follow-up commit; nil
	// entries are tolerated by the stub-phase tests.
	tabs [6]Screen
}

// newFlatModel constructs the flat TUI model with the welcome dashboard
// (engram-style menu) as the root of the navigation stack. Selecting an
// action from the welcome menu pushes the v2 WorkstationScreen with the
// matching section pre-focused. esc/q in the workstation pops back to
// the welcome screen.
func newFlatModel(st *storage.Storage, cfg Config) flatModel {
	root := NewDashboardScreen(st, cfg.Project, cfg.Version)
	return flatModel{
		storage: st,
		cfg:     cfg,
		pal:     defaultPalette,
		stack:   []Screen{root},
	}
}

// Init returns the root screen's initial command.
func (m flatModel) Init() tea.Cmd {
	if len(m.stack) == 0 {
		return nil
	}
	return m.stack[0].Init()
}

// Update routes messages to the active (top-of-stack) screen, intercepting
// only the events flatModel itself owns: window resizing, global quit
// shortcuts, and push/pop screen commands emitted by child screens.
func (m flatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// fall through to also let the active screen react if it wants

	case tea.KeyMsg:
		// ctrl+c always quits, regardless of which screen is active.
		if msg.String() == "ctrl+c" {
			m.Quitting = true
			return m, tea.Quit
		}
		// esc pops the stack down to the dashboard. esc is a non-typeable
		// key so it's safe to intercept globally.
		if msg.String() == "esc" && len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			return m, m.focusTop()
		}
		// 'q' is intentionally NOT handled here. Each screen owns its own
		// 'q' semantics — most pop, the welcome dashboard quits, and
		// SearchScreen lets it through as a typeable character (otherwise
		// queries containing the letter q would be impossible).

	case pushScreenCmd:
		m.stack = append(m.stack, msg.screen)
		return m, msg.screen.Init()

	case popScreenCmd:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
			return m, m.focusTop()
		}
		return m, nil
	}

	// Forward everything else to the top screen.
	if len(m.stack) == 0 {
		return m, nil
	}
	top := m.stack[len(m.stack)-1]
	updated, cmd := top.Update(msg)
	m.stack[len(m.stack)-1] = updated
	return m, cmd
}

// View renders the top screen, gated by the same minimum-size guard the
// legacy Model uses.
func (m flatModel) View() string {
	if m.Quitting {
		return ""
	}
	if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
		return fmt.Sprintf(
			"Terminal too small (%dx%d). Please resize to at least %dx%d.\n",
			m.width, m.height, minWidth, minHeight,
		)
	}
	if len(m.stack) == 0 {
		return ""
	}
	return m.stack[len(m.stack)-1].View(m.width, m.height, m.pal)
}

// focusTop calls OnFocus on the top of the stack so it can refresh data
// when a child screen is popped off.
func (m flatModel) focusTop() tea.Cmd {
	if len(m.stack) == 0 {
		return nil
	}
	return m.stack[len(m.stack)-1].OnFocus()
}
