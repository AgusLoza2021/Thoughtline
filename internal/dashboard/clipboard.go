package dashboard

import "errors"

// ErrClipboardUnavailable is returned when the platform clipboard backend
// (or its fallback chain) cannot be found or invoked.
var ErrClipboardUnavailable = errors.New("clipboard: no backend available")

// errClipboardUnavailable is a package-level alias for test access.
var errClipboardUnavailable = ErrClipboardUnavailable

// CopyToClipboard sends s to the OS clipboard.
// Returns a non-nil error when the platform-specific backend is unavailable.
// Implemented per build tag in clipboard_{windows,darwin,linux}.go.
// The function signature is the same on all platforms; only the body differs.
func CopyToClipboard(s string) error {
	return copyToClipboardImpl(s)
}
