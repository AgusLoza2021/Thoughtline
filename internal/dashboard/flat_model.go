package dashboard

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// minWidth and minHeight define the smallest terminal Thoughtline supports.
// Below these dimensions the TUI shows a resize warning instead of content.
const (
	minWidth  = 80
	minHeight = 24
)

// flatModel is the lightweight tea.Model that drives the new flat-screen TUI.
// It owns the tab layer, the screen stack for overlays, palette, terminal size,
// and a global status message banner. All real work (loading data, rendering,
// key handling) happens inside the Screen at the top of the stack or the active
// tab — flatModel just routes messages, intercepts navigation keys, and reacts
// to push/pop commands.

// tabKey identifies one of the six top-level tabs in the new memory-workspace
// TUI. The legacy multi-tab Model used a tabKey enum in tabs.go which was
// deleted in commit 5; this is the new identifier.
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

// focusInputer is the optional interface a Screen implements to allow the
// global '/' Quick Action to programmatically focus its text input. Only
// SearchScreen (and future screens with a search box) implement it.
type focusInputer interface {
	FocusInput()
}

// statusMessage is a time-bounded banner rendered at the bottom of the screen.
// It is used by Quick Actions (e.g., 's' shows a save hint) and by system
// notifications (e.g., clipboard result, promoted confirmation).
type statusMessage struct {
	Text    string
	Level   statusLevel // statusOK / statusWARN / statusERR / statusInfo
	Expires time.Time
}

// statusClearMsg is sent after a status message's TTL expires to clear it.
type statusClearMsg struct{}

type flatModel struct {
	storage  *storage.Storage
	cfg      Config
	pal      palette
	width    int
	height   int
	stack    []Screen
	Quitting bool

	// activeTab is the currently-rendered tab.
	activeTab tabKey
	// tabs holds one Screen per tab. Populated by newFlatModel; nil entries
	// are tolerated by stub-phase tests.
	tabs [6]Screen

	// status is the global footer status message (expires after TTL).
	status statusMessage
}

// newFlatModel constructs the flat TUI model with the six-tab workspace.
// Tab 0 (Home) is the initial active tab and receives an OnFocus call.
func newFlatModel(st *storage.Storage, cfg Config) flatModel {
	m := flatModel{
		storage:   st,
		cfg:       cfg,
		pal:       defaultPalette,
		activeTab: TabHome,
	}
	// Populate tabs.
	m.tabs[TabHome] = NewHomeScreen(st)
	m.tabs[TabMemories] = NewMemoriesScreen(st)
	m.tabs[TabSearch] = newSearchScreen(st, cfg.Project)
	m.tabs[TabInbox] = NewInboxScreen(st)
	m.tabs[TabSessions] = NewSessionsScreen(st)
	m.tabs[TabHelp] = NewHelpScreen()
	return m
}

// Init calls Init on the active tab and returns its command.
func (m flatModel) Init() tea.Cmd {
	if m.tabs[m.activeTab] == nil {
		return nil
	}
	return tea.Batch(m.tabs[m.activeTab].Init(), m.tabs[m.activeTab].OnFocus())
}

// Update routes messages through the seven-step dispatch ladder documented in
// the tui-memory-workspace design (Section 5, Section 2b(B)).
//
// Ladder order:
//
//	(1) tea.WindowSizeMsg  — fan out to all tabs + stack; store size; return
//	(2) ctrl+c             — quit unconditionally
//	(3) 'q' quit           — when stack empty AND !inputFocused (AND not under-min)
//	(4) stack non-empty    — dispatch to top-of-stack; esc pops with originator handling
//	(5) under-min guard    — freeze all navigation below min viewport (q/ctrl+c handled above)
//	(6) InputFocused guard — pass key directly to active tab Screen
//	(7) tab intercept      — '1'-'6', tab, shift+tab
//	(8) Quick Actions      — 's', '/', 'm', 'i'
//	(9) fall-through       — dispatch to tabs[activeTab]
func (m flatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── (1) Window resize — fan out to all tabs + stack ─────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		var cmds []tea.Cmd
		for i, tab := range m.tabs {
			if tab == nil {
				continue
			}
			updated, cmd := tab.Update(msg)
			m.tabs[i] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		for i, s := range m.stack {
			updated, cmd := s.Update(msg)
			m.stack[i] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	// ── Push/pop commands from child screens ────────────────────────────────
	case pushScreenCmd:
		m.stack = append(m.stack, msg.screen)
		return m, msg.screen.Init()

	case popScreenCmd:
		if len(m.stack) > 0 {
			popped := m.stack[len(m.stack)-1]
			m.stack = m.stack[:len(m.stack)-1]
			if orig, ok := popped.(originator); ok {
				cmd := m.setActiveTab(orig.OriginatingTab())
				return m, cmd
			}
		}
		return m, nil

	// ── Status TTL expiry ────────────────────────────────────────────────────
	case statusClearMsg:
		m.status = statusMessage{}
		return m, nil

	// ── Key messages run through the full ladder ─────────────────────────────
	case tea.KeyMsg:

		// ── (2) ctrl+c quits unconditionally ────────────────────────────────
		if msg.Type == tea.KeyCtrlC {
			m.Quitting = true
			return m, tea.Quit
		}

		// ── (3) 'q' quit when stack empty AND not focused ───────────────────
		if msg.String() == "q" && len(m.stack) == 0 && !m.currentInputFocused() {
			m.Quitting = true
			return m, tea.Quit
		}

		// ── (4) stack non-empty — dispatch to top-of-stack ──────────────────
		if len(m.stack) > 0 {
			top := m.stack[len(m.stack)-1]

			// esc pops the overlay and reads the originating tab.
			if msg.Type == tea.KeyEsc {
				m.stack = m.stack[:len(m.stack)-1]
				if orig, ok := top.(originator); ok {
					cmd := m.setActiveTab(orig.OriginatingTab())
					return m, cmd
				}
				return m, nil
			}

			// All other keys go to the top of the stack.
			updated, cmd := top.Update(msg)
			m.stack[len(m.stack)-1] = updated
			return m, cmd
		}

		// ── (5) under-min guard — freeze navigation ──────────────────────────
		// ctrl+c (step 2) and q quit (step 3) already handled above.
		// 'q' under-min quits (handled in step 3: stack is empty and not focused).
		// All other nav keys are frozen when under-min.
		if m.width > 0 && m.height > 0 && (m.width < minWidth || m.height < minHeight) {
			return m, nil
		}

		// ── (6) InputFocused guard ────────────────────────────────────────────
		if m.currentInputFocused() {
			tab := m.tabs[m.activeTab]
			if tab != nil {
				updated, cmd := tab.Update(msg)
				m.tabs[m.activeTab] = updated
				return m, cmd
			}
			return m, nil
		}

		// ── (7) Tab intercept: '1'-'6', tab, shift+tab ───────────────────────
		switch msg.Type {
		case tea.KeyTab:
			cmd := m.cycleTab(+1)
			return m, cmd
		case tea.KeyShiftTab:
			cmd := m.cycleTab(-1)
			return m, cmd
		}

		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case '1':
				return m, m.setActiveTab(TabHome)
			case '2':
				return m, m.setActiveTab(TabMemories)
			case '3':
				return m, m.setActiveTab(TabSearch)
			case '4':
				return m, m.setActiveTab(TabInbox)
			case '5':
				return m, m.setActiveTab(TabSessions)
			case '6':
				return m, m.setActiveTab(TabHelp)
			}
		}

		// ── (8) Quick Actions ─────────────────────────────────────────────────
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 's':
				m.status = statusMessage{
					Text:    "Save: use 'tl save ...' (CLI) or tl_save (MCP). In-TUI save coming in a follow-up change.",
					Level:   statusInfo,
					Expires: time.Now().Add(5 * time.Second),
				}
				return m, statusClearAfter(5 * time.Second)

			case '/':
				cmd := m.setActiveTab(TabSearch)
				// Call FocusInput if the Search screen implements it.
				if fi, ok := m.tabs[TabSearch].(focusInputer); ok {
					fi.FocusInput()
				}
				return m, cmd

			case 'm':
				return m, m.setActiveTab(TabMemories)

			case 'i':
				return m, m.setActiveTab(TabInbox)
			}
		}
	}

	// ── (9) fall-through: dispatch to active tab ──────────────────────────────
	tab := m.tabs[m.activeTab]
	if tab == nil {
		return m, nil
	}
	updated, cmd := tab.Update(msg)
	m.tabs[m.activeTab] = updated
	return m, cmd
}

// View renders the active tab (or the top-of-stack overlay), gated by the
// minimum-size guard. Also renders the global status message when not expired.
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

	// Render from stack overlay if present.
	if len(m.stack) > 0 {
		top := m.stack[len(m.stack)-1]
		content := top.View(m.width, m.height, m.pal)
		return m.appendStatus(content)
	}

	// Render active tab.
	tab := m.tabs[m.activeTab]
	if tab == nil {
		return ""
	}
	content := tab.View(m.width, m.height, m.pal)
	return m.appendStatus(content)
}

// appendStatus appends the status message line to the rendered content when
// the message has not yet expired.
func (m flatModel) appendStatus(content string) string {
	if m.status.Text == "" || time.Now().After(m.status.Expires) {
		return content
	}
	style := statusStyle(m.status.Level, m.pal)
	return content + "\n" + style.Render(m.status.Text)
}

// setActiveTab updates activeTab, calls OnFocus on the newly-active tab, and
// returns the resulting tea.Cmd. Safe to call when tabs[t] is nil.
func (m *flatModel) setActiveTab(t tabKey) tea.Cmd {
	m.activeTab = t
	if m.tabs[t] == nil {
		return nil
	}
	return m.tabs[t].OnFocus()
}

// currentInputFocused returns true when the currently-active input owner
// (top of stack if non-empty, else active tab) reports InputFocused() == true.
func (m flatModel) currentInputFocused() bool {
	if len(m.stack) > 0 {
		if f, ok := m.stack[len(m.stack)-1].(inputFocuser); ok {
			return f.InputFocused()
		}
	}
	if f, ok := m.tabs[m.activeTab].(inputFocuser); ok {
		return f.InputFocused()
	}
	return false
}

// cycleTab advances (direction +1) or retreats (direction -1) through the six
// tabs, wrapping at both ends. Returns the OnFocus cmd for the new tab.
func (m *flatModel) cycleTab(direction int) tea.Cmd {
	next := (int(m.activeTab) + direction + 6) % 6
	return m.setActiveTab(tabKey(next))
}

// statusClearAfter returns a tea.Cmd that sends statusClearMsg after the given
// duration. Used to automatically expire the global status banner.
func statusClearAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return statusClearMsg{}
	})
}

// focusTop calls OnFocus on the top of the stack (used internally when a
// child screen is popped off via popScreenCmd).
func (m flatModel) focusTop() tea.Cmd {
	if len(m.stack) == 0 {
		return nil
	}
	return m.stack[len(m.stack)-1].OnFocus()
}
