//go:build !windows

package db

import (
	"os/exec"
	"syscall"
)

// setProcGroup puts cmd into its own process group so that a SIGINT sent to
// the terminal (Ctrl-C) is not forwarded to the SurrealDB child process.
// Our signal handler in app.go will stop it gracefully via Manager.Stop().
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
