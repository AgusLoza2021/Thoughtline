package dashboard

import "fmt"

// diskStatusLevel maps a disk free-percentage and optional DB error to one of
// the three status levels shown in the dashboard header.
//
// Rules (from tui-dashboard Req 11):
//   - dbErr non-nil  → statusERR
//   - pct < 10       → statusWARN
//   - otherwise      → statusOK
func diskStatusLevel(pct int, dbErr error) statusLevel {
	if dbErr != nil {
		return statusERR
	}
	if pct < 10 {
		return statusWARN
	}
	return statusOK
}

// formatStatusLine returns the styled header status string based on the disk
// percentage and any DB error. Formats match spec Req 11:
//
//	OK   → "MEM: OK 25%"
//	WARN → "MEM: WARN"
//	ERR  → "MEM: ERR"
func formatStatusLine(pct int, dbErr error, p palette) string {
	level := diskStatusLevel(pct, dbErr)
	style := statusStyle(level, p)
	switch level {
	case statusOK:
		return style.Render(fmt.Sprintf("MEM: OK %d%%", pct))
	case statusWARN:
		return style.Render("MEM: WARN")
	default: // statusERR
		return style.Render("MEM: ERR")
	}
}
