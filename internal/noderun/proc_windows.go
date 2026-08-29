//go:build windows

package noderun

import "syscall"

func procAttr() *syscall.SysProcAttr { return nil }
