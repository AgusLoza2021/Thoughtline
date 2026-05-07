//go:build windows

package dashboard

import (
	"os"
	"testing"
)

func TestFreeDiskPct_Windows(t *testing.T) {
	pct, err := freeDiskPct(os.TempDir())
	if err != nil {
		t.Fatalf("freeDiskPct: %v", err)
	}
	if pct < 0 || pct > 100 {
		t.Errorf("freeDiskPct(%q) = %d, want value in [0, 100]", os.TempDir(), pct)
	}
}
