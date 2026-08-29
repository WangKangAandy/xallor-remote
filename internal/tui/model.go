package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type tab int

const (
	tabHome tab = iota
	tabGrant
	tabPeers
	tabExec
)

var tabNames = []string{"本机", "授权", "对方", "执行"}

type statusMsg map[string]any
type grantMsg string
type peersMsg []string
type execChunk string
type execDone struct{ err error }
type errMsg string

type model struct {
	tab    tab
	status map[string]any
	grant  string
	peers  []string
	err    string
	note   string
	input  textinput.Model
	output string
	busy   bool
}

func newModel() model {
	ti := textinput.New()
	ti.Placeholder = "命令（回车执行）"
	ti.Width = 56
	return model{input: ti}
}

func (m model) Init() tea.Cmd {
	return refreshAll()
}

type tickRefresh struct{}

func refreshAll() tea.Cmd {
	return tea.Batch(cmdStatus(), cmdGrant(), cmdPeers())
}

func cmdStatus() tea.Cmd {
	return func() tea.Msg {
		res, err := rpc("status", nil)
		if err != nil {
			return errMsg(err.Error())
		}
		return statusMsg(res)
	}
}

func cmdGrant() tea.Cmd {
	return func() tea.Msg {
		res, err := rpc("grant.show", nil)
		if err != nil {
			return errMsg(err.Error())
		}
		g, _ := res["grant"].(string)
		return grantMsg(g)
	}
}

func cmdPeers() tea.Cmd {
	return func() tea.Msg {
		res, err := rpc("peer.list", nil)
		if err != nil {
			return errMsg(err.Error())
		}
		return peersMsg(peerIDs(res))
	}
}

func cmdAction(method string, params map[string]any, after tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		if _, err := rpc(method, params); err != nil {
			return errMsg(err.Error())
		}
		return after()
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.tab == tabExec {
			switch {
			case msg.String() == "ctrl+c":
				return m, tea.Quit
			case msg.Type == tea.KeyTab:
				m.tab = tabHome
				m.input.Blur()
				return m, nil
			case msg.Type == tea.KeyEnter && !m.busy:
				cmd := m.input.Value()
				if cmd == "" {
					return m, nil
				}
				m.busy = true
				m.output = ""
				m.note = "执行中…"
				return m, runExecCmd(cmd)
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.tab = (m.tab + 1) % 4
			if m.tab == tabExec {
				return m, m.input.Focus()
			}
			return m, nil
		case "left", "h":
			m.tab = (m.tab + 3) % 4
			if m.tab == tabExec {
				return m, m.input.Focus()
			}
			return m, nil
		case "r":
			m.note = ""
			return m, refreshAll()
		case "i":
			if m.tab == tabGrant {
				return m, cmdAction("grant.issue", nil, cmdGrant())
			}
		case "n":
			if m.tab == tabGrant {
				return m, cmdAction("grant.rotate", nil, cmdGrant())
			}
		case "o":
			if m.tab == tabGrant {
				return m, cmdAction("inbound.set", map[string]any{"enabled": true}, refreshAll())
			}
		case "f":
			if m.tab == tabGrant {
				return m, cmdAction("inbound.set", map[string]any{"enabled": false}, refreshAll())
			}
		}
	case statusMsg:
		m.status = map[string]any(msg)
		m.err = ""
	case grantMsg:
		m.grant = string(msg)
		m.err = ""
	case peersMsg:
		m.peers = []string(msg)
		m.err = ""
	case execChunk:
		m.output += string(msg)
		if len(m.output) > 8000 {
			m.output = m.output[len(m.output)-8000:]
		}
		return m, waitExec()
	case execDone:
		m.busy = false
		if msg.err != nil {
			m.note = msg.err.Error()
		} else {
			m.note = "完成"
		}
	case errMsg:
		m.err = string(msg)
		m.busy = false
	}
	return m, nil
}
