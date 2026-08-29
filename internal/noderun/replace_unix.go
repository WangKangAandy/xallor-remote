//go:build !windows

package noderun

import "os"

func replaceFile(tmp, dest string) error {
	return os.Rename(tmp, dest)
}
