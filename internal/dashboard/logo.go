package dashboard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// thoughtlineLogo is the hardcoded 5-row block-letter ASCII art for
// "THOUGHTLINE". Each row is ≤70 runes wide. The gradient renderer applies
// palette.LogoGradient[i] to row i via lipgloss.
//
// Designed as a compact block-letter style (similar to figlet "ANSI Regular"
// but trimmed to 5 rows to fit the 70-col budget for 11 characters).
var thoughtlineLogo = []string{
	` _____ _  _ ___  _   _  ___  _  _ _____ _    ___ _  _ ___`,
	`|_   _| || | _ \| | | |/ __|| || |_   _| |  |_ _| \| | __|`,
	`  | | | __ |   /| |_| | (_ || __ | | | | |__ | ||  ` + "`" + ` | _|`,
	`  |_| |_||_|_|_\ \___/ \___||_||_| |_| |____|___|_|\_|___|`,
	`           game-dev memory that survives                    `,
}

// renderLogo applies the 5-color gradient from p.LogoGradient to each row of
// thoughtlineLogo and joins them vertically.
//
// If termWidth < 70 the gradient art is replaced by a single-color literal
// "THOUGHTLINE" in the first gradient color (design decision J).
func renderLogo(p palette, termWidth int) string {
	if termWidth < 70 {
		return lipgloss.NewStyle().
			Foreground(p.LogoGradient[0]).
			Render("THOUGHTLINE")
	}

	rows := make([]string, len(thoughtlineLogo))
	for i, row := range thoughtlineLogo {
		colorIdx := i
		if colorIdx >= len(p.LogoGradient) {
			colorIdx = len(p.LogoGradient) - 1
		}
		rows[i] = lipgloss.NewStyle().
			Foreground(p.LogoGradient[colorIdx]).
			Render(row)
	}
	return strings.Join(rows, "\n")
}
