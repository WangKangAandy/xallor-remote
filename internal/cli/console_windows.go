//go:build windows

package cli

import "golang.org/x/sys/windows"

func initConsole() {
	const utf8 = 65001
	_ = windows.SetConsoleOutputCP(utf8)
	_ = windows.SetConsoleCP(utf8)
}
