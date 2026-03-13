//go:build windows

package db

import "os/exec"

// setProcGroup is a no-op on Windows; process group isolation is not needed
// because the Wails app owns its own console window.
func setProcGroup(cmd *exec.Cmd) {}
