package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStyles_DecoupledFromTheme asserts that styles.go (the file, not the
// resulting compiled symbols) no longer references the legacy multi-theme
// machinery. This is a source-level guard so a future contributor cannot
// silently reintroduce a coupling we deliberately removed in commit 3 of
// tui-memory-workspace.
func TestStyles_DecoupledFromTheme(t *testing.T) {
	path := filepath.Join("styles.go")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	src := string(raw)

	// Forbidden identifiers — these belong to the legacy multi-theme world
	// and must not appear anywhere in styles.go after the decoupling.
	forbidden := []string{
		// "ApplyTheme" intentionally allowed only inside comment markers
		// describing the decoupling. We assert it does not appear as a CALL.
		"ApplyTheme(",
		"ThemeBrand",
		"ThemeZBrush",
		"ThemeMono",
		"nextTheme",
		"currentTheme",
	}
	for _, tok := range forbidden {
		if strings.Contains(src, tok) {
			t.Errorf("styles.go must not reference %q after the decoupling (commit 3)", tok)
		}
	}

	// Positive assertion: styles.go must mention the new build entry point
	// so a reader lands on the right symbol when investigating.
	if !strings.Contains(src, "rebuildStyles") {
		t.Errorf("styles.go must define rebuildStyles(palette) as the single build path")
	}
}

// TestStyles_AllStyleVarsPopulated asserts that every package-level lipgloss
// style var has been initialised by the package init(). A style with zero
// content renders as the empty string when used with .Render("x") — we use
// the actual render output as a non-empty smoke check.
func TestStyles_AllStyleVarsPopulated(t *testing.T) {
	cases := []struct {
		name  string
		input string
		style stylish
	}{
		{"brandPillStyle", "x", brandPillStyle},
		{"versionPillStyle", "x", versionPillStyle},
		{"headerMetaStyle", "x", headerMetaStyle},
		{"headerKeyStyle", "x", headerKeyStyle},
		{"statusBarStyle", "x", statusBarStyle},
		{"statusOnlineStyle", "x", statusOnlineStyle},
		{"statusMetricStyle", "x", statusMetricStyle},
		{"statusValueStyle", "x", statusValueStyle},
		{"statusUpdateStyle", "x", statusUpdateStyle},
		{"titleStyle", "x", titleStyle},
		{"subtleStyle", "x", subtleStyle},
		{"panelStyle", "x", panelStyle},
		{"panelTitleStyle", "x", panelTitleStyle},
		{"statsNumberStyle", "x", statsNumberStyle},
		{"statsLabelStyle", "x", statsLabelStyle},
		{"doneStyle", "x", doneStyle},
		{"nextStyle", "x", nextStyle},
		{"inProgressStyle", "x", inProgressStyle},
		{"deferredStyle", "x", deferredStyle},
		{"errStyle", "x", errStyle},
		{"tabActiveStyle", "x", tabActiveStyle},
		{"tabInactiveStyle", "x", tabInactiveStyle},
		{"footerKeyStyle", "x", footerKeyStyle},
		{"footerLabelStyle", "x", footerLabelStyle},
		{"keyStyle", "x", keyStyle},
		{"hintStyle", "x", hintStyle},
		{"cubeStyle", "x", cubeStyle},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.style.Render(tc.input)
			if got == "" {
				t.Errorf("style %s rendered empty — likely uninitialised by rebuildStyles", tc.name)
			}
		})
	}
}

// stylish is the narrow contract we need from lipgloss.Style for the table
// above. Kept here so the test file does not need to import lipgloss
// directly.
type stylish interface {
	Render(strs ...string) string
}
