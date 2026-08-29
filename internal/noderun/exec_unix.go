//go:build !windows

package noderun

func wrapCommand(command string) string { return command }
