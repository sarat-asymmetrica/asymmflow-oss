//go:build !windows

package main

import "os/exec"

func suppressCommandWindow(cmd *exec.Cmd) {
}
