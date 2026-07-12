//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func suppressCommandWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
