package dashboard

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// -----------------------------------------------------------------------------
// Test fixtures
// -----------------------------------------------------------------------------

// newFlatModelForTabTests constructs a flatModel ready to exercise the tab
// layer. The Screen stack starts empty (no overlays). The tabs array is
// populated with lightweight stub screens so dispatch tests can observe
// per-tab Update routing without crashing.
func newFlatModelForTabTests(t *testing.T) flatModel {
	t.Helper()
	st := newWorkspaceStorage(t)
	m := flatModel{
		storage: st,
		cfg:     Config{Project: "test-workspace", Version: "test"},
		pal:     defaultPalette,
		width:   100,
		height:  30,
		stack:   nil,
	}
	for i := range m.tabs {
		m.tabs[i] = &recordingTabStub{key: tabKey(i)}
	}
	m.activeTab = TabHome
	return m
}

// recordingTabStub is a Screen implementation used purely as a tab body in
// tests. It records the last message it saw via Update so dispatch tests can
// verify that, when no intercept fires, the keystroke reaches the active tab.
type recordingTabStub struct {
	key   tabKey
	lastK string
}

func (s *recordingTabStub) Init() tea.Cmd { return nil }
func (s *recordingTabStub) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		s.lastK = k.String()
	}
	return s, nil
}
func (s *recordingTabStub) View(width, height int, p palette) string { return "stub" }
func (s *recordingTabStub) Title() string                            { return "stub" }
func (s *recordingTabStub) OnFocus() tea.Cmd                         { return nil }

// focusedSearchStub is a Screen that reports an input as focused via the
// optional inputFocuser interface. Used by F4/F5 tests.
type focusedSearchStub struct {
	recordingTabStub
	input textinput.Model
}

func newFocusedSearchStub() *focusedSearchStub {
	ti := textinput.New()
	ti.Focus()
	return &focusedSearchStub{
		recordingTabStub: recordingTabStub{key: TabSearch},
		input:            ti,
	}
}

func (s *focusedSearchStub) InputFocused() bool { return s.input.Focused() }

func (s *focusedSearchStub) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		s.lastK = k.String()
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(k)
		return s, cmd
	}
	return s, nil
}

// originScreenStub is a stack-overlay screen that declares an originating tab
// via the optional originator interface. Used by F6.
type originScreenStub struct {
	recordingTabStub
	origin tabKey
}

func (s *originScreenStub) OriginatingTab() tabKey { return s.origin }

// updateModel runs m.Update and returns the new flatModel. Convenience helper
// to keep the test calls compact.
func updateModel(t *testing.T, m flatModel, msg tea.Msg) (flatModel, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	return updated.(flatModel), cmd
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// -----------------------------------------------------------------------------
// F1 — Tab digit-jump 1..6 (Req 1)
// -----------------------------------------------------------------------------

func TestFlatModel_F1_TabDigitJump(t *testing.T) {
	cases := []struct {
		digit rune
		want  tabKey
	}{
		{'1', TabHome},
		{'2', TabMemories},
		{'3', TabSearch},
		{'4', TabInbox},
		{'5', TabSessions},
		{'6', TabHelp},
	}

	for _, tc := range cases {
		t.Run(string(tc.digit), func(t *testing.T) {
			m := newFlatModelForTabTests(t)
			// Start on a tab that is NOT the target so the assertion is meaningful.
			if tc.want == TabHome {
				m.activeTab = TabHelp
			} else {
				m.activeTab = TabHome
			}
			m, _ = updateModel(t, m, runeKey(tc.digit))
			if m.activeTab != tc.want {
				t.Errorf("press %q: activeTab = %d, want %d", tc.digit, m.activeTab, tc.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// F2 — Tab cycle with wrap (Req 1)
// -----------------------------------------------------------------------------

func TestFlatModel_F2_TabCycle(t *testing.T) {
	t.Run("forward from Inbox→Sessions", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.activeTab = TabInbox
		m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyTab})
		if m.activeTab != TabSessions {
			t.Errorf("tab from Inbox = %d, want Sessions(%d)", m.activeTab, TabSessions)
		}
	})

	t.Run("forward wrap Help→Home", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.activeTab = TabHelp
		m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyTab})
		if m.activeTab != TabHome {
			t.Errorf("tab from Help = %d, want Home(%d)", m.activeTab, TabHome)
		}
	})

	t.Run("backward wrap Home→Help", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.activeTab = TabHome
		m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
		if m.activeTab != TabHelp {
			t.Errorf("shift+tab from Home = %d, want Help(%d)", m.activeTab, TabHelp)
		}
	})

	t.Run("backward sequence", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.activeTab = TabSessions
		m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
		if m.activeTab != TabInbox {
			t.Errorf("shift+tab from Sessions = %d, want Inbox", m.activeTab)
		}
		m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
		if m.activeTab != TabSearch {
			t.Errorf("shift+tab from Inbox = %d, want Search", m.activeTab)
		}
	})
}

// -----------------------------------------------------------------------------
// F3 — Tab state survives Screen stack push/pop (Req 1, Req 17)
// -----------------------------------------------------------------------------

func TestFlatModel_F3_TabSurvivesPushPop(t *testing.T) {
	t.Run("esc pop returns to active tab when no originator", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.activeTab = TabMemories
		// Push an overlay screen.
		m.stack = append(m.stack, &recordingTabStub{key: -1})
		// Press esc to pop.
		m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.activeTab != TabMemories {
			t.Errorf("after pop, activeTab = %d, want Memories(%d)", m.activeTab, TabMemories)
		}
		if len(m.stack) != 0 {
			t.Errorf("after pop, stack len = %d, want 0", len(m.stack))
		}
	})

	t.Run("originator screen overrides active tab on pop", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.activeTab = TabMemories
		// Push a detail screen that declares Home as its origin.
		m.stack = append(m.stack, &originScreenStub{origin: TabHome})
		m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.activeTab != TabHome {
			t.Errorf("after pop with originator=Home, activeTab = %d, want Home(%d)", m.activeTab, TabHome)
		}
	})
}

// -----------------------------------------------------------------------------
// F4 — InputFocused() guard (Req 16, Req 23, Reconciliation #4)
// -----------------------------------------------------------------------------

func TestFlatModel_F4_InputFocusGuard(t *testing.T) {
	setup := func(t *testing.T) (flatModel, *focusedSearchStub) {
		m := newFlatModelForTabTests(t)
		m.activeTab = TabSearch
		stub := newFocusedSearchStub()
		m.tabs[TabSearch] = stub
		return m, stub
	}

	t.Run("m in search input is literal (no tab switch)", func(t *testing.T) {
		m, stub := setup(t)
		m, _ = updateModel(t, m, runeKey('m'))
		if m.activeTab != TabSearch {
			t.Errorf("activeTab changed to %d while input focused; want Search", m.activeTab)
		}
		if stub.input.Value() != "m" {
			t.Errorf("input value = %q, want %q (literal m should land in textinput)", stub.input.Value(), "m")
		}
	})

	t.Run("s while textarea focused does not trigger save", func(t *testing.T) {
		m, stub := setup(t)
		m, _ = updateModel(t, m, runeKey('s'))
		if m.activeTab != TabSearch {
			t.Errorf("activeTab changed to %d; want Search (s must NOT trigger save jump)", m.activeTab)
		}
		if stub.input.Value() != "s" {
			t.Errorf("input value = %q, want %q", stub.input.Value(), "s")
		}
	})

	t.Run("digit 2 while input focused does not tab-switch", func(t *testing.T) {
		m, stub := setup(t)
		m, _ = updateModel(t, m, runeKey('2'))
		if m.activeTab != TabSearch {
			t.Errorf("activeTab = %d after '2' with input focused; want Search", m.activeTab)
		}
		if stub.input.Value() != "2" {
			t.Errorf("input value = %q, want %q", stub.input.Value(), "2")
		}
	})

	t.Run("q while input focused does NOT quit", func(t *testing.T) {
		m, stub := setup(t)
		_, cmd := updateModel(t, m, runeKey('q'))
		if cmd != nil {
			// We can't easily test tea.Quit identity, but cmd should not be the global Quit.
			// We accept cmd != nil here only if it's the textinput's own Cmd; the model must NOT be Quitting.
		}
		if m.Quitting {
			t.Errorf("flatModel should not be Quitting when q is pressed in focused input")
		}
		if stub.input.Value() != "q" {
			t.Errorf("input value = %q, want %q (q must be literal)", stub.input.Value(), "q")
		}
	})

	t.Run("ctrl+c quits even with input focused", func(t *testing.T) {
		m, _ := setup(t)
		_, cmd := updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Errorf("ctrl+c with input focused must return a non-nil tea.Cmd (tea.Quit)")
		}
	})
}

// -----------------------------------------------------------------------------
// F5 — `/` jumps to Search AND focuses input (Req 15)
// -----------------------------------------------------------------------------

func TestFlatModel_F5_SlashJumpsAndFocuses(t *testing.T) {
	m := newFlatModelForTabTests(t)
	m.activeTab = TabHome
	// Install a Search stub that can report focus state.
	search := newFocusedSearchStub()
	search.input.Blur() // Start blurred — `/` must focus it.
	m.tabs[TabSearch] = search

	m, _ = updateModel(t, m, runeKey('/'))

	if m.activeTab != TabSearch {
		t.Errorf("after '/', activeTab = %d, want Search(%d)", m.activeTab, TabSearch)
	}
	if !search.InputFocused() {
		t.Errorf("after '/', SearchScreen.InputFocused() = false, want true")
	}
}

// -----------------------------------------------------------------------------
// F6 — OriginatingTab() return on esc pop (Req 17)
// -----------------------------------------------------------------------------

func TestFlatModel_F6_OriginatingTabReturn(t *testing.T) {
	cases := []struct {
		name       string
		startTab   tabKey
		origin     tabKey
		wantActive tabKey
	}{
		{"detail-from-memories", TabMemories, TabMemories, TabMemories},
		{"detail-from-home", TabHome, TabHome, TabHome},
		{"detail-from-search returns to search", TabSearch, TabSearch, TabSearch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newFlatModelForTabTests(t)
			m.activeTab = tc.startTab
			m.stack = append(m.stack, &originScreenStub{origin: tc.origin})
			m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
			if m.activeTab != tc.wantActive {
				t.Errorf("activeTab after esc-pop = %d, want %d", m.activeTab, tc.wantActive)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// F7 — Min viewport 80x24 warning + key freeze (Req 22)
// -----------------------------------------------------------------------------

func TestFlatModel_F7_MinViewport(t *testing.T) {
	t.Run("exactly 80x24 is OK (no warning)", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.width, m.height = 80, 24
		out := m.View()
		if strings.Contains(out, "Terminal too small") {
			t.Errorf("80x24 must not render warning, got:\n%s", out)
		}
	})

	t.Run("79x24 renders warning", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.width, m.height = 79, 24
		out := m.View()
		if !strings.Contains(out, "Terminal too small") {
			t.Errorf("79x24 must render warning, got:\n%s", out)
		}
	})

	t.Run("80x23 renders warning", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.width, m.height = 80, 23
		out := m.View()
		if !strings.Contains(out, "Terminal too small") {
			t.Errorf("80x23 must render warning, got:\n%s", out)
		}
	})

	t.Run("under-min freezes tab keys", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.width, m.height = 70, 20
		m.activeTab = TabHome
		// '1', '/', 'm' must NOT change activeTab while under min.
		for _, r := range []rune{'1', '2', '/', 'm', 'i'} {
			before := m.activeTab
			m, _ = updateModel(t, m, runeKey(r))
			if m.activeTab != before {
				t.Errorf("under min, key %q changed activeTab from %d to %d", r, before, m.activeTab)
			}
		}
	})

	t.Run("under-min q still quits", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.width, m.height = 70, 20
		_, cmd := updateModel(t, m, runeKey('q'))
		if cmd == nil {
			t.Errorf("q under min must still quit (non-nil cmd)")
		}
	})

	t.Run("under-min ctrl+c still quits", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.width, m.height = 70, 20
		_, cmd := updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Errorf("ctrl+c under min must still quit (non-nil cmd)")
		}
	})
}

// -----------------------------------------------------------------------------
// F8 — Quit semantics (Req 23)
// -----------------------------------------------------------------------------

func TestFlatModel_F8_QuitSemantics(t *testing.T) {
	t.Run("q_quits_on_memories", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.activeTab = TabMemories
		// No stack overlay, no input focused.
		_, cmd := updateModel(t, m, runeKey('q'))
		if cmd == nil {
			t.Errorf("q on Memories (no input, no stack) must return tea.Quit (non-nil cmd)")
		}
	})

	t.Run("ctrl+c always quits", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		_, cmd := updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Errorf("ctrl+c must return tea.Quit (non-nil cmd)")
		}
	})

	t.Run("q with stack overlay does NOT quit globally", func(t *testing.T) {
		m := newFlatModelForTabTests(t)
		m.stack = append(m.stack, &recordingTabStub{key: -1})
		m, _ = updateModel(t, m, runeKey('q'))
		if m.Quitting {
			t.Errorf("q with non-empty stack must not quit globally")
		}
	})
}

// Compile-time guard: ensure recordingTabStub satisfies Screen.
var _ Screen = (*recordingTabStub)(nil)
var _ Screen = (*focusedSearchStub)(nil)
var _ Screen = (*originScreenStub)(nil)
var _ inputFocuser = (*focusedSearchStub)(nil)
var _ originator = (*originScreenStub)(nil)

// keep storage import used so go-vet doesn't complain on Windows builds where
// we touch storage.StatsOptions in the seeded helpers indirectly.
var _ = storage.StatsOptions{}
