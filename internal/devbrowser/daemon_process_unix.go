//go:build unix

package devbrowser

import (
	"os/exec"
	"syscall"
)

func configureDaemonProcess(cmd *exec.Cmd) {
	// Detach the daemon from the caller's session/process group so it survives
	// short-lived CLI invocations and agent runners that clean up child groups.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
