package dashboard

import (
	"errors"
	"testing"
)

func TestDiskStatusLevel_Thresholds(t *testing.T) {
	// >= 10% free with no error → OK
	if got := diskStatusLevel(25, nil); got != statusOK {
		t.Errorf("diskStatusLevel(25, nil) = %v, want statusOK", got)
	}
	// >= 10% boundary
	if got := diskStatusLevel(10, nil); got != statusOK {
		t.Errorf("diskStatusLevel(10, nil) = %v, want statusOK", got)
	}
	// < 10% free → WARN
	if got := diskStatusLevel(8, nil); got != statusWARN {
		t.Errorf("diskStatusLevel(8, nil) = %v, want statusWARN", got)
	}
	// 0% free → WARN (no error, just low disk)
	if got := diskStatusLevel(0, nil); got != statusWARN {
		t.Errorf("diskStatusLevel(0, nil) = %v, want statusWARN", got)
	}
	// DB error → ERR regardless of pct
	dbErr := errors.New("db down")
	if got := diskStatusLevel(0, dbErr); got != statusERR {
		t.Errorf("diskStatusLevel(0, err) = %v, want statusERR", got)
	}
	if got := diskStatusLevel(50, dbErr); got != statusERR {
		t.Errorf("diskStatusLevel(50, err) = %v, want statusERR", got)
	}
}

func TestFormatStatusLine_OK(t *testing.T) {
	line := formatStatusLine(25, nil, defaultPalette)
	// Must contain "MEM: OK" and the percentage.
	if !containsStr(line, "OK") {
		t.Errorf("formatStatusLine(25, nil) must contain OK, got %q", line)
	}
	if !containsStr(line, "25%") {
		t.Errorf("formatStatusLine(25, nil) must contain 25%%, got %q", line)
	}
}

func TestFormatStatusLine_WARN(t *testing.T) {
	line := formatStatusLine(8, nil, defaultPalette)
	if !containsStr(line, "WARN") {
		t.Errorf("formatStatusLine(8, nil) must contain WARN, got %q", line)
	}
}

func TestFormatStatusLine_ERR(t *testing.T) {
	line := formatStatusLine(0, errors.New("fail"), defaultPalette)
	if !containsStr(line, "ERR") {
		t.Errorf("formatStatusLine(0, err) must contain ERR, got %q", line)
	}
}

// containsStr is a simple helper to avoid importing strings in this test file.
func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
