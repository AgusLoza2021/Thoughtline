package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_TabCycleForward(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)
	if m.tab != tabOverview {
		t.Fatalf("expected to start on Overview, got %s", m.tab)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	final := updated.(Model)
	if final.tab != tabBrowse {
		t.Errorf("tab should advance to Browse, got %s", final.tab)
	}
}

func TestModel_TabCycleWrapsAround(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)
	for i := 0; i < len(allTabs); i++ {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(Model)
	}
	if m.tab != tabOverview {
		t.Errorf("after %d tab presses we should wrap to Overview, got %s", len(allTabs), m.tab)
	}
}

func TestModel_DigitJumpsToTab(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	final := updated.(Model)
	if final.tab != legacyTabSearch {
		t.Errorf("'3' should jump to Search, got %s", final.tab)
	}
}

func TestModel_HelpToggle(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(Model)
	if m.tab != legacyTabHelp {
		t.Errorf("? should jump to Help, got %s", m.tab)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(Model)
	if m.tab != tabOverview {
		t.Errorf("? from Help should return to Overview, got %s", m.tab)
	}
}

func TestModel_HelpViewMentionsKeybindings(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)
	m.tab = legacyTabHelp
	out := m.View()
	for _, want := range []string{"Keybindings", "tab", "search", "refresh", "quit"} {
		if !strings.Contains(strings.ToLower(out), strings.ToLower(want)) {
			t.Errorf("help view should mention %q\n%s", want, out)
		}
	}
}

func TestModel_TickRefreshesOnlyOnOverview(t *testing.T) {
	m, _ := newTestModel(t)
	m = drainInit(t, m)

	// Tick on overview triggers a stats reload.
	updated, cmd := m.Update(tickMsg{})
	if !updated.(Model).loading {
		t.Errorf("tick on overview should set loading=true")
	}
	if cmd == nil {
		t.Errorf("tick should always re-arm the ticker (non-nil cmd)")
	}

	// Switch to Sessions tab; ticks should NOT re-trigger a load there.
	m = updated.(Model)
	m.tab = tabSessions
	m.loading = false
	updated, _ = m.Update(tickMsg{})
	if updated.(Model).loading {
		t.Errorf("tick on non-overview tab should not set loading")
	}
}

func TestNextTab_WrapsBothDirections(t *testing.T) {
	if got := nextTab(tabOverview, -1); got != legacyTabHelp {
		t.Errorf("overview -1 = %s, want Help", got)
	}
	if got := nextTab(legacyTabHelp, +1); got != tabOverview {
		t.Errorf("help +1 = %s, want Overview", got)
	}
}

func TestRoadmap_LoadedFromYAML(t *testing.T) {
	got := Roadmap()
	if len(got) < 7 {
		t.Fatalf("expected at least 7 milestones from yaml, got %d", len(got))
	}
	if got[0].ID != "M0" || got[0].Status != "done" {
		t.Errorf("first milestone should be M0/done, got %+v", got[0])
	}
}
