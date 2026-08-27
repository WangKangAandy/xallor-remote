//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
)

func detach(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}

