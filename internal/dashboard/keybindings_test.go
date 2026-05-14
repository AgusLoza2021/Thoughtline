package dashboard

import (
	"strings"
	"testing"
)

// N1 — Keybindings slice is non-empty and exposes the expected group titles.
func TestKeybindings_RegistryStructure(t *testing.T) {
	if len(Keybindings) == 0 {
		t.Fatal("Keybindings is empty — Help tab would render nothing")
	}

	wantGroups := []string{"Navigation", "Quick Actions", "Memories tab", "Detail view", "Inbox"}
	got := make(map[string]bool)
	for _, g := range Keybindings {
		got[g.Title] = true
	}
	for _, w := range wantGroups {
		if !got[w] {
			t.Errorf("missing expected group %q in Keybindings", w)
		}
	}
}

// N2 — every Keybind has non-empty Keys and Desc (no skeleton entries).
func TestKeybindings_NoEmptyEntries(t *testing.T) {
	for _, g := range Keybindings {
		if g.Title == "" {
			t.Errorf("group has empty title")
		}
		if len(g.Bindings) == 0 {
			t.Errorf("group %q has no bindings", g.Title)
		}
		for _, kb := range g.Bindings {
			if kb.Keys == "" {
				t.Errorf("group %q has a binding with empty Keys (desc=%q)", g.Title, kb.Desc)
			}
			if kb.Desc == "" {
				t.Errorf("group %q binding %q has empty Desc", g.Title, kb.Keys)
			}
		}
	}
}

// N3 — HelpScreen.View renders every group title from Keybindings.
func TestHelpScreen_RendersAllGroupTitles(t *testing.T) {
	h := NewHelpScreen()
	out := h.View(120, 30, defaultPalette)
	for _, g := range Keybindings {
		if !strings.Contains(out, g.Title) {
			t.Errorf("HelpScreen.View missing group title %q in output:\n%s", g.Title, out)
		}
	}
}

// N4 — HelpScreen.View renders at least one roadmap entry.
func TestHelpScreen_RendersRoadmap(t *testing.T) {
	h := NewHelpScreen()
	out := h.View(120, 30, defaultPalette)
	if !strings.Contains(out, "Roadmap") {
		t.Errorf("HelpScreen.View missing 'Roadmap' header:\n%s", out)
	}
	rm := Roadmap()
	if len(rm) == 0 {
		t.Skip("no roadmap milestones embedded — nothing to assert")
	}
	// Assert at least one milestone Name appears.
	found := false
	for _, m := range rm {
		if strings.Contains(out, m.Name) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("HelpScreen.View contains no milestone Name from roadmap.yaml:\n%s", out)
	}
}

// N5+N6 — drift test.
//
// Strategy: maintain a hand-curated `wiredHotkeys` allowlist in
// keybindings.go that enumerates every token handled by flatModel.Update's
// global ladder (digits, tab/shift+tab, esc, q, ctrl+c, and the Quick
// Actions s, /, m, i). The test asserts every token in `wiredHotkeys`
// appears in at least one Keybind.Keys entry of the Navigation or
// Quick Actions groups.
//
// We deliberately do NOT scan flat_model.go source — that would produce
// false positives because `case 'q'` can appear inside InputFocused
// branches that are functionally one logical handler. Per-screen hotkeys
// (Memories ↑↓, Detail C, Inbox A/E/R, etc.) live inside the respective
// Screen.Update methods, not the flatModel ladder, so they are documented
// in Keybindings but NOT listed in wiredHotkeys.
func TestKeybindings_DriftAgainstWiredHotkeys(t *testing.T) {
	// Build a haystack from all Navigation + Quick Actions Keys fields,
	// lower-cased for case-insensitive token matching.
	var haystack strings.Builder
	for _, g := range Keybindings {
		if g.Title != "Navigation" && g.Title != "Quick Actions" {
			continue
		}
		for _, kb := range g.Bindings {
			haystack.WriteString(strings.ToLower(kb.Keys))
			haystack.WriteString(" ")
		}
	}
	hay := haystack.String()

	for _, token := range wiredHotkeys {
		needle := strings.ToLower(token)
		// Map runtime token to its renderable form in Keys.
		// "1".."6" are documented as "1-6" range. "ctrl+c" / "shift+tab" /
		// "tab" / "esc" / "q" / "/" / "s" / "m" / "i" appear verbatim.
		matched := false
		switch needle {
		case "1", "2", "3", "4", "5", "6":
			// Range notation "1-6" covers all six digits.
			if strings.Contains(hay, "1-6") || strings.Contains(hay, needle) {
				matched = true
			}
		default:
			if strings.Contains(hay, needle) {
				matched = true
			}
		}
		if !matched {
			t.Errorf("drift: wired hotkey %q is not documented in Keybindings Navigation/Quick Actions", token)
		}
	}
}
