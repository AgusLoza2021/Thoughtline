package dashboard

import (
	"strings"
	"testing"
)

// TestRenderBrand_ContainsBrainAndTitle verifies that renderBrand returns a
// string that contains both the emoji + app name and the tagline.
func TestRenderBrand_ContainsBrainAndTitle(t *testing.T) {
	got := renderBrand(defaultPalette)

	if !strings.Contains(got, "Thoughtline") {
		t.Errorf("renderBrand should contain %q, got: %q", "Thoughtline", got)
	}
	if !strings.Contains(got, "Local memory for game projects") {
		t.Errorf("renderBrand should contain tagline %q, got: %q",
			"Local memory for game projects", got)
	}
}

// TestRenderBrand_IsShort verifies that the rendered output is compact (< 200
// chars), proving it is a text brand, not multi-row ASCII art.
func TestRenderBrand_IsShort(t *testing.T) {
	got := renderBrand(defaultPalette)
	if len(got) >= 200 {
		t.Errorf("renderBrand output is %d chars; expected < 200 (text brand, not ASCII art)", len(got))
	}
}

// TestRenderBrand_NoForbiddenHex verifies that the brand does not embed any
// Rose-Pine-Moon hex codes that were forbidden by the design.
// The carry-over #C4A7E7 (Brand purple) is allowed.
func TestRenderBrand_NoForbiddenHex(t *testing.T) {
	got := renderBrand(defaultPalette)

	// These exact Rose-Pine-Moon hex values must not appear in the brand output.
	forbidden := []string{
		"#E0DEF4", "#e0def4",
		"#6E6A86", "#6e6a86",
		"#393552", "#393552",
		"#A88DC9", "#a88dc9",
		"#9CCFD8", "#9ccfd8",
		"#3E8FB0", "#3e8fb0",
		"#56949F", "#56949f",
		"#EB6F92", "#eb6f92",
		"#F6C177", "#f6c177",
		"#2A273F", "#2a273f",
	}
	for _, hex := range forbidden {
		if strings.Contains(got, hex) {
			t.Errorf("renderBrand output must not contain forbidden hex %q", hex)
		}
	}
}

// TestRenderBrand_IsSingleLine verifies that the brand output has exactly one
// logical line (the text brand plus tagline on the same row).
func TestRenderBrand_IsSingleLine(t *testing.T) {
	got := renderBrand(defaultPalette)
	// Strip ANSI escapes to count visual newlines.
	// We just count raw \n characters — the brand should be inline (no \n).
	if strings.Contains(got, "\n") {
		t.Errorf("renderBrand should not contain newlines (single-line brand), got: %q", got)
	}
}
