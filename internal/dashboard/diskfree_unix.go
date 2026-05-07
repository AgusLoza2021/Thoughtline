//go:build !windows

package dashboard

import (
	"fmt"
	"syscall"
)

// freeDiskPct returns the percentage of free disk space on the filesystem
// that contains the given path. Returns (0, err) on failure.
func freeDiskPct(path string) (int, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("statfs %q: %w", path, err)
	}
	if stat.Blocks == 0 {
		return 0, fmt.Errorf("statfs %q: zero block count", path)
	}
	pct := int(stat.Bavail * 100 / stat.Blocks)
	if pct > 100 {
		pct = 100
	}
	return pct, nil
}
