package dashboard

import (
	"regexp"
	"testing"
)

var hexRE = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// TestPalette_AllTokensDefined verifies that all 14 named palette tokens are
// valid 6-character hex strings, and that LogoGradient has exactly 5 elements.
func TestPalette_AllTokensDefined(t *testing.T) {
	p := defaultPalette

	tokens := []struct {
		name string
		val  string
	}{
		{"Foreground", string(p.Foreground)},
		{"Muted", string(p.Muted)},
		{"Border", string(p.Border)},
		{"LogoGradient[0]", string(p.LogoGradient[0])},
		{"LogoGradient[1]", string(p.LogoGradient[1])},
		{"LogoGradient[2]", string(p.LogoGradient[2])},
		{"LogoGradient[3]", string(p.LogoGradient[3])},
		{"LogoGradient[4]", string(p.LogoGradient[4])},
		{"StatNumber", string(p.StatNumber)},
		{"Cursor", string(p.Cursor)},
		{"MenuSelectedBg", string(p.MenuSelectedBg)},
		{"StatusOK", string(p.StatusOK)},
		{"StatusWarn", string(p.StatusWarn)},
		{"StatusErr", string(p.StatusErr)},
		{"Tag", string(p.Tag)},
	}

	for _, tt := range tokens {
		if !hexRE.MatchString(tt.val) {
			t.Errorf("token %s = %q — not a valid 6-char hex color", tt.name, tt.val)
		}
	}

	// LogoGradient must have exactly 5 elements.
	if len(p.LogoGradient) != 5 {
		t.Errorf("LogoGradient len=%d, want 5", len(p.LogoGradient))
	}
}

// TestStatusStyle_ThreeDistinctColors verifies that statusStyle returns styles
// with distinct foreground color values for OK, WARN, and ERR.
func TestStatusStyle_ThreeDistinctStyles(t *testing.T) {
	p := defaultPalette

	okS := statusStyle(statusOK, p)
	warnS := statusStyle(statusWARN, p)
	errS := statusStyle(statusERR, p)

	// Extract the foreground color from each style via GetForeground.
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
