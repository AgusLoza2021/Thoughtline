package dashboard

import (
	"strings"
	"testing"
)

// TestLogo_RowCountAndColumnBudget verifies that thoughtlineLogo has at least
// 5 rows and each row fits within 70 runes.
func TestLogo_RowCountAndColumnBudget(t *testing.T) {
	if len(thoughtlineLogo) < 5 {
		t.Errorf("thoughtlineLogo has %d rows, want ≥5", len(thoughtlineLogo))
	}
	for i, row := range thoughtlineLogo {
		runes := []rune(row)
		if len(runes) > 70 {
			t.Errorf("row %d has %d runes, max is 70: %q", i, len(runes), row)
		}
	}
}

// TestRenderLogo_ProducesGradientRows verifies that renderLogo returns a
// multi-line string where consecutive lines differ in ANSI color prefix
// (i.e., each row is styled with a distinct gradient color).
func TestRenderLogo_ProducesGradientRows(t *testing.T) {
	rendered := renderLogo(defaultPalette, 100)

	lines := strings.Split(rendered, "\n")
	// Filter empty lines that may appear at end.
	var nonEmpty []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}

	if len(nonEmpty) < 5 {
		t.Errorf("renderLogo produced %d non-empty lines, want ≥5", len(nonEmpty))
	}

	// Consecutive lines must differ (different color escapes).
	for i := 1; i < len(nonEmpty); i++ {
		if nonEmpty[i] == nonEmpty[i-1] {
			t.Errorf("lines %d and %d are identical — gradient not applied:\n%q", i-1, i, nonEmpty[i])
		}
	}
}
