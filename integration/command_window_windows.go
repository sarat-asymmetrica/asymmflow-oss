//go:build windows

package integration

import (
	"os/exec"
	"syscall"
)

// suppressCommandWindow hides the console window that Windows would otherwise
// pop for a child process (e.g. `pandoc --version` during tool validation).
// Must be called after exec.Command and before Run/Start/Output.
func suppressCommandWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
