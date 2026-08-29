//go:build windows

package noderun

import (
	"os"

	"golang.org/x/sys/windows"
)

func startJob(proc *os.Process) (kill func(), closeFn func()) {
	nop := func() {}
	if proc == nil {
		return nop, nop
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return func() { killTree(proc.Pid) }, nop
	}
	p, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.PROCESS_SET_INFORMATION,
		false,
		uint32(proc.Pid),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		return func() { killTree(proc.Pid) }, nop
	}
	if err := windows.AssignProcessToJobObject(job, p); err != nil {
		_ = windows.CloseHandle(p)
		_ = windows.CloseHandle(job)
		return func() { killTree(proc.Pid) }, nop
	}
	_ = windows.CloseHandle(p)
	return func() { _ = windows.TerminateJobObject(job, 1) }, func() { _ = windows.CloseHandle(job) }
}
