//go:build linux

package dashboard

import (
	"bytes"
	"os/exec"
)

// execLookPath is the hook for exec.LookPath used in the fallback chain.
// Replaced by a shim in tests.
var execLookPath = exec.LookPath

// execCommand is the hook for subprocess creation. Replaced by a shim in tests.
var execCommand = exec.Command

// copyToClipboardImpl tries wl-copy (Wayland) first, then xclip (X11).
// Returns ErrClipboardUnavailable if neither is found.
func copyToClipboardImpl(s string) error {
	if _, err := execLookPath("wl-copy"); err == nil {
		cmd := execCommand("wl-copy")
		cmd.Stdin = bytes.NewBufferString(s)
		return cmd.Run()
	}
	if _, err := execLookPath("xclip"); err == nil {
		cmd := execCommand("xclip", "-selection", "clipboard")
		cmd.Stdin = bytes.NewBufferString(s)
		return cmd.Run()
	}
	return ErrClipboardUnavailable
}
