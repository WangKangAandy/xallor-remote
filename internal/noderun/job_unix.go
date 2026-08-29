//go:build !windows

package noderun

import "os"

func startJob(proc *os.Process) (kill func(), closeFn func()) {
	pid := 0
	if proc != nil {
		pid = proc.Pid
	}
	return func() {
		if pid != 0 {
			killTree(pid)
		}
	}, func() {}
}
