//go:build windows

package dashboard

import (
	"fmt"
	"syscall"
	"unsafe"
)

var getDiskFreeSpaceEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

// freeDiskPct returns the percentage of free disk space on the filesystem
// that contains the given path. Returns (0, err) on failure.
func freeDiskPct(path string) (int, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, fmt.Errorf("utf16 path: %w", err)
	}

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	r, _, callErr := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if r == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceExW: %w", callErr)
	}
	if totalBytes == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceExW: zero total bytes")
	}
	pct := int(freeBytesAvailable * 100 / totalBytes)
	if pct > 100 {
		pct = 100
	}
	return pct, nil
}
