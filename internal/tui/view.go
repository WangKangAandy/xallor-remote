package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	tabOn      = lipgloss.NewStyle().Bold(true).Underline(true)
	tabOff     = lipgloss.NewStyle().Faint(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	helpStyle  = lipgloss.NewStyle().Faint(true)
)

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("XallorRemote"))
	b.WriteString("\n")
	for i, name := range tabNames {
		style := tabOff
		if tab(i) == m.tab {
			style = tabOn
		}
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(style.Render(name))
	}
	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(errStyle.Render(m.err))
		b.WriteString("\n\n")
	}
	if m.note != "" {
		b.WriteString(m.note)
		b.WriteString("\n\n")
	}
	switch m.tab {
	case tabHome:
		if m.status != nil {
			b.WriteString(formatHome(m.status))
		}
		b.WriteString(helpStyle.Render("\nr 刷新  Tab 切换  q 离开"))
	case tabGrant:
		b.WriteString(formatGrant(m.grant))
		b.WriteString(helpStyle.Render("\n\ni 签发  n 换码  o 开入站  f 关入站"))
	case tabPeers:
		b.WriteString(formatPeers(m.peers))
		b.WriteString(helpStyle.Render("\n\n添加请用：xallor-remote peer add"))
	case tabExec:
		b.WriteString(m.input.View())
		b.WriteString("\n\n")
		b.WriteString(m.output)
		b.WriteString(helpStyle.Render("\n\n回车执行  Tab 切换"))
	}
	b.WriteString("\n")
	return b.String()
}
