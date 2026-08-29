//go:build windows

package noderun

import (
	"os/exec"
	"strconv"
)

func killTree(pid int) {
	_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
}
