//go:build windows

package dashboard

import (
	"bytes"
	"os/exec"
)

// execCommand is the hook for subprocess creation. In tests it is replaced
// by a shim; in production code it is exec.Command.
var execCommand = exec.Command

// copyToClipboardImpl pipes s into clip.exe on Windows.
func copyToClipboardImpl(s string) error {
	cmd := execCommand("cmd", "/c", "clip")
	cmd.Stdin = bytes.NewBufferString(s)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
