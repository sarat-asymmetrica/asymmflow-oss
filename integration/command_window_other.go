//go:build !windows

package integration

import "os/exec"

// suppressCommandWindow is a no-op off Windows (no console window is created).
func suppressCommandWindow(cmd *exec.Cmd) {}
