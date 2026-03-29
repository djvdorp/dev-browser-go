//go:build unix

package devbrowser

import (
	"os/exec"
	"testing"
)

func TestConfigureDaemonProcessDetachesSession(t *testing.T) {
	cmd := exec.Command("sleep", "1")
	if cmd.SysProcAttr != nil {
		t.Fatalf("expected nil SysProcAttr before configuration, got %#v", cmd.SysProcAttr)
	}

	configureDaemonProcess(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be configured")
	}
	if !cmd.SysProcAttr.Setsid {
		t.Fatalf("expected Setsid=true, got %#v", cmd.SysProcAttr)
	}
}
