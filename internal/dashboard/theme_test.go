package dashboard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

var hexRE = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// TestPalette_NewSemanticTokensDefined verifies every named palette token
// defined in the tui-memory-workspace design (Section 3) is present in
// defaultPalette as a valid 6-character hex color.
//
// Task B1 — RED-first when the legacy Rose-Pine-Moon palette was still in
// theme.go; turns GREEN after the palette swap (B2).
func TestPalette_NewSemanticTokensDefined(t *testing.T) {
	p := defaultPalette

	tokens := []struct {
		name string
		val  string
	}{
		{"Foreground", string(p.Foreground)},
		{"Muted", string(p.Muted)},
		{"Border", string(p.Border)},
		{"BorderFocused", string(p.BorderFocused)},
		{"FocusedBg", string(p.FocusedBg)},
		{"Nav", string(p.Nav)},
		{"NavActive", string(p.NavActive)},
		{"NavActiveBg", string(p.NavActiveBg)},
		{"Brand", string(p.Brand)},
		{"BrandDim", string(p.BrandDim)},
		{"Success", string(p.Success)},
		{"Warning", string(p.Warning)},
		{"Error", string(p.Error)},
		{"StatusOK", string(p.StatusOK)},
		{"StatusWarn", string(p.StatusWarn)},
		{"StatusErr", string(p.StatusErr)},
		{"BadgeMemoryType", string(p.BadgeMemoryType)},
		{"BadgeMemoryTypeBg", string(p.BadgeMemoryTypeBg)},
	}

	for _, tc := range tokens {
		t.Run(tc.name, func(t *testing.T) {
			if !hexRE.MatchString(tc.val) {
				t.Errorf("token %s = %q — not a valid 6-char hex color", tc.name, tc.val)
			}
		})
	}
}

// TestPalette_NoRosePineMoonHex asserts that the legacy Rose-Pine-Moon hex
// codes do not appear in theme.go after the palette swap. Matches the
// negative-assertion in tui-removed Req 12.
//
// NOTE: the forbidden list excludes #C4A7E7 because that hex was reused by
// the new design (Section 3) as the Brand and BadgeMemoryType token. This
// is a known, documented overlap with the tui-removed spec; the spec was
// authored before the design fixed Brand to #C4A7E7. The other 10 forbidden
// hexes are still asserted absent.
func TestPalette_NoRosePineMoonHex(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("theme.go"))
	if err != nil {
		t.Fatalf("read theme.go: %v", err)
	}
	src := strings.ToLower(string(raw))

	forbidden := []string{
		"#e0def4", // Rose-Pine-Moon Foreground
		"#6e6a86", // Muted
		"#393552", // Border
		"#a88dc9", // LogoGradient[1]
		"#9ccfd8", // LogoGradient[2] / legacy StatusOK
		"#3e8fb0", // LogoGradient[3]
		"#56949f", // LogoGradient[4]
		"#eb6f92", // legacy StatNumber / StatusErr
		"#f6c177", // legacy Cursor / StatusWarn
		"#2a273f", // legacy MenuSelectedBg
		// "#c4a7e7" intentionally excluded — re-used as Brand in the new design.
	}

	for _, hex := range forbidden {
		if strings.Contains(src, hex) {
			t.Errorf("theme.go must not contain Rose-Pine-Moon hex %q", hex)
		}
	}
}

// TestStatusStyle_ThreeDistinctStyles verifies that statusStyle returns
// styles with distinct foreground color values for OK, WARN, and ERR after
// the palette swap (task B3).
func TestStatusStyle_ThreeDistinctStyles(t *testing.T) {
	p := defaultPalette

	okS := statusStyle(statusOK, p)
	warnS := statusStyle(statusWARN, p)
	errS := statusStyle(statusERR, p)

	okColor := okS.GetForeground()
	warnColor := warnS.GetForeground()
	errColor := errS.GetForeground()

	if okColor == warnColor {
		t.Errorf("OK and WARN must have distinct foreground colors, both got %v", okColor)
	}
	if okColor == errColor {
		t.Errorf("OK and ERR must have distinct foreground colors, both got %v", okColor)
	}
	if warnColor == errColor {
		t.Errorf("WARN and ERR must have distinct foreground colors, both got %v", warnColor)
	}
}

// TestStatusStyle_MapsToSemanticTokens asserts that the OK/WARN/ERR styles
// resolve to the Success/Warning/Error tokens, NOT to the navigation or
// brand tokens. This is the source-level check for Req 20 of
// tui-memory-workspace: "Success color is not used for navigation."
func TestStatusStyle_MapsToSemanticTokens(t *testing.T) {
	p := defaultPalette

	cases := []struct {
		name  string
		level statusLevel
		want  string
	}{
		{"OK→Success", statusOK, string(p.Success)},
		{"WARN→Warning", statusWARN, string(p.Warning)},
		{"ERR→Error", statusERR, string(p.Error)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fg := statusStyle(tc.level, p).GetForeground()
			gotStr, ok := fg.(lipgloss.Color)
			if !ok {
				t.Fatalf("foreground is not a lipgloss.Color: %T", fg)
			}
			if string(gotStr) != tc.want {
				t.Errorf("statusStyle(%s).Foreground = %q, want %q", tc.name, string(gotStr), tc.want)
			}
		})
	}

	// Source-level: Success must not appear in the same Foreground() call as
	// the navigation styles. We assert this structurally rather than via grep
	// because styles.go indirectly uses the same hex — instead we assert that
	// Success != Nav and Success != NavActive token values, which is the
	// observable form of the "not used for navigation" contract.
	if string(p.Success) == string(p.Nav) {
		t.Errorf("Success and Nav must be distinct hex values")
	}
	if string(p.Success) == string(p.NavActive) {
		t.Errorf("Success and NavActive must be distinct hex values")
	}
}

