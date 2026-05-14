package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// helpers.go holds pure rendering / formatting helpers shared across multiple
// Screens. The functions in this file were originally defined alongside
// WorkstationScreen in workstation_screen.go; they are extracted verbatim
// here (identical signatures, identical behavior) so the WorkstationScreen
// file can be deleted in a follow-up commit without breaking the screens
// that currently call into these helpers.
//
// IMPORTANT: do not modify behavior in this extraction. Any refactor of
// these helpers belongs to a separate change.

// paneBox wraps body in a normal-border lipgloss style sized to width x height
// and padded with one column on each side. Used for the three-pane workstation
// layout and reused by the new Home cards.
func paneBox(body string, width, height int, p palette) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(p.Border).
		Width(width - 2).
		Height(height - 2).
		Padding(0, 1)
	return style.Render(body)
}

// centerString renders s in muted color, vertically and horizontally centered
// inside a viewport of width x height.
func centerString(s string, width, height int, p palette) string {
	muted := lipgloss.NewStyle().Foreground(p.Muted)
	pad := strings.Repeat("\n", height/2-1)
	return pad + lipgloss.PlaceHorizontal(width, lipgloss.Center, muted.Render(s))
}

// clampCursor clamps c into [0, n-1]. Returns 0 when n <= 0 (empty list).
func clampCursor(c, n int) int {
	if c >= n {
		c = n - 1
	}
	if c < 0 {
		c = 0
	}
	return c
}

// truncate truncates s to at most n runes, appending "…" when truncation
// happens. Previously defined in view.go; moved here so it is available to
// all screens after view.go is removed.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

// truncateLeft truncates s from the right so the visible width fits within
// max, appending an ellipsis when truncation happens. min width clamped to 4.
func truncateLeft(s string, max int) string {
	if max < 4 {
		max = 4
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max < 3 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

// wrap breaks s into hard-wrapped lines of at most width characters, joined
// with newlines. Width is clamped to a minimum of 8.
func wrap(s string, width int) string {
	if width < 8 {
		width = 8
	}
	var out strings.Builder
	for len(s) > width {
		out.WriteString(s[:width])
		out.WriteString("\n")
		s = s[width:]
	}
	out.WriteString(s)
	return out.String()
}

// relTime returns a short human-readable description of how long ago t was.
// Zero time renders as "—". Buckets: just-now / Nm / Nh / Nd / YYYY-MM-DD.
func relTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// formatLastSave returns relTime() of the most recent memory in items, or "" if
// the list is empty.
func formatLastSave(items []storage.SearchResult) string {
	if len(items) == 0 {
		return ""
	}
	return relTime(items[0].UpdatedAt)
}

// padRight pads s on the right with spaces so the resulting string is exactly
// n columns wide. Strings already wider than n are returned unchanged.
//
// padRight previously lived in view.go (legacy Model rendering). It is moved
// here so the helpers package is the single source of truth and view.go can
// be deleted alongside the rest of the legacy Model in a follow-up commit.
func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
