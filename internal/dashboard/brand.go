package dashboard

import "github.com/charmbracelet/lipgloss"

// renderBrand returns the single-line text brand for Thoughtline:
//
//	🧠 Thoughtline  Local memory for game projects
//
// The app name is rendered in p.Brand (purple), the tagline in p.Muted
// (gray). This replaces the 5-row block-letter ASCII art that lived in
// logo.go — the new design uses a compact inline mark instead.
func renderBrand(p palette) string {
	name := lipgloss.NewStyle().Foreground(p.Brand).Render("🧠 Thoughtline")
	tagline := lipgloss.NewStyle().Foreground(p.Muted).Render("Local memory for game projects")
	return name + "  " + tagline
}
