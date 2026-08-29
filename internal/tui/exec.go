package tui

import tea "github.com/charmbracelet/bubbletea"

var execCh chan tea.Msg

func runExecCmd(command string) tea.Cmd {
	execCh = make(chan tea.Msg, 16)
	go func() {
		err := streamExec(command, func(s string) {
			execCh <- execChunk(s)
		})
		execCh <- execDone{err: err}
		close(execCh)
	}()
	return waitExec()
}

func waitExec() tea.Cmd {
	return func() tea.Msg {
		if execCh == nil {
			return execDone{}
		}
		msg, ok := <-execCh
		if !ok {
			return execDone{}
		}
		return msg
	}
}
