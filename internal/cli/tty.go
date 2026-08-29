package cli

import (
	"os"

	"golang.org/x/term"
)

func stdinIsTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
