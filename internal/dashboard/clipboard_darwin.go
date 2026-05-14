//go:build darwin

package dashboard

import (
	"bytes"
	"os/exec"
)

// execCommand is the hook for subprocess creation. Replaced by a shim in tests.
var execCommand = exec.Command

// copyToClipboardImpl pipes s into pbcopy on macOS.
func copyToClipboardImpl(s string) error {
	cmd := execCommand("pbcopy")
	cmd.Stdin = bytes.NewBufferString(s)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
