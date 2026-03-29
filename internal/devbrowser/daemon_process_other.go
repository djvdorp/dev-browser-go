//go:build !unix

package devbrowser

import "os/exec"

func configureDaemonProcess(cmd *exec.Cmd) {}
