//go:build windows

package noderun

func wrapCommand(command string) string {
	return "[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false); $OutputEncoding = [Console]::OutputEncoding; " + command
}
