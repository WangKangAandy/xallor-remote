package noderun

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/WangKangAandy/xallor-remote/internal/identity"
	"github.com/WangKangAandy/xallor-remote/internal/ipc"
	"github.com/WangKangAandy/xallor-remote/internal/protocol"
)

func (d *Daemon) serveIPC(c *ipc.Conn) {
	defer c.Close()
	for {
		req, err := c.ReadRequest()
		if err != nil {
			return
		}
		switch req.Method {
		case "status":
			id, _, grant, ws, relay, inbound, peers := d.store.Snapshot()
			_ = c.WriteFrame(ipc.OK(req.ID, map[string]any{
				"device_id": id, "workspace": ws, "relay": relay,
				"inbound": inbound, "has_grant": grant != "",
				"online": d.deviceOnline(), "version": protocol.Version,
				"peers": peers,
			}))
		case "grant.issue":
			g, err := d.store.IssueGrant()
			if err != nil {
				_ = c.WriteFrame(ipc.Fail(req.ID, protocol.AgentError, err.Error()))
				continue
			}
			d.pushGrant(g)
			_ = c.WriteFrame(ipc.OK(req.ID, map[string]string{"grant": g}))
		case "grant.show":
			_, _, g, _, _, _, _ := d.store.Snapshot()
			_ = c.WriteFrame(ipc.OK(req.ID, map[string]string{"grant": g}))
		case "peer.add":
			var p struct {
				DeviceID string `json:"device_id"`
				Grant    string `json:"grant"`
			}
			_ = json.Unmarshal(req.Params, &p)
			if p.DeviceID == "" || p.Grant == "" {
				_ = c.WriteFrame(ipc.Fail(req.ID, protocol.AgentError, "需要 device_id 与 grant"))
				continue
			}
			if err := d.store.AddPeer(p.DeviceID, p.Grant); err != nil {
				_ = c.WriteFrame(ipc.Fail(req.ID, protocol.AgentError, err.Error()))
				continue
			}
			_ = c.WriteFrame(ipc.OK(req.ID, map[string]string{"ok": "1"}))
		case "exec":
			d.ipcExec(c, req)
		default:
			_ = c.WriteFrame(ipc.Fail(req.ID, protocol.AgentError, "未知方法"))
		}
	}
}

func (d *Daemon) ipcExec(c *ipc.Conn, req ipc.Request) {
	var p struct {
		DeviceID  string `json:"device_id"`
		Command   string `json:"command"`
		Cwd       string `json:"cwd"`
		TimeoutMS int    `json:"timeout_ms"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil || p.Command == "" {
		_ = c.WriteFrame(ipc.Fail(req.ID, protocol.AgentError, "需要 command"))
		return
	}
	peer, ok := d.store.Peer(p.DeviceID)
	if !ok {
		_ = c.WriteFrame(ipc.Fail(req.ID, protocol.UnknownDevice, ipc.Human(protocol.UnknownDevice)))
		return
	}
	link, cl, err := d.ensureClient(peer)
	if err != nil {
		_ = c.WriteFrame(ipc.Fail(req.ID, protocol.RelayError, ipc.Human(protocol.RelayError)))
		return
	}
	cl.mu.Lock()
	defer cl.mu.Unlock()
	execID, err := protocol.NewExecID()
	if err != nil {
		_ = c.WriteFrame(ipc.Fail(req.ID, protocol.AgentError, err.Error()))
		return
	}
	payload, _ := json.Marshal(protocol.ExecPayload{Command: p.Command, Cwd: p.Cwd, TimeoutMS: p.TimeoutMS})
	if err := link.send(protocol.Message{Type: protocol.TypeInvoke, ExecID: execID, Op: protocol.OpExec, Payload: payload}); err != nil {
		_ = c.WriteFrame(ipc.Fail(req.ID, protocol.RelayError, ipc.Human(protocol.RelayError)))
		return
	}
	d.readClientLoop(c, req.ID, execID, link)
}

func (d *Daemon) ensureClient(peer identity.Peer) (*wsConn, *clientLink, error) {
	d.mu.Lock()
	if cl, ok := d.clients[peer.DeviceID]; ok && cl.ws != nil {
		d.mu.Unlock()
		return cl.ws, cl, nil
	}
	d.mu.Unlock()

	_, _, _, _, relay, _, _ := d.store.Snapshot()
	ctx, cancel := context.WithCancel(d.ctx)
	conn, _, err := websocket.Dial(ctx, relay, nil)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	link := &wsConn{ctx: ctx, conn: conn}
	hello := protocol.Message{Type: protocol.TypeHelloClient, TargetID: peer.DeviceID, Grant: peer.Grant}
	if err := link.send(hello); err != nil {
		cancel()
		return nil, nil, err
	}
	var ack protocol.Message
	if err := wsjson.Read(ctx, conn, &ack); err != nil {
		cancel()
		return nil, nil, err
	}
	if ack.Type != protocol.TypeHelloOK {
		cancel()
		return nil, nil, fmt.Errorf("%s", ack.Code)
	}
	cl := &clientLink{target: peer.DeviceID, ws: link}
	d.mu.Lock()
	d.clients[peer.DeviceID] = cl
	d.mu.Unlock()
	go func() {
		<-d.ctx.Done()
		cancel()
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()
	return link, cl, nil
}

func (d *Daemon) readClientLoop(ipcConn *ipc.Conn, reqID, execID string, link *wsConn) {
	for {
		var msg protocol.Message
		if err := wsjson.Read(link.ctx, link.conn, &msg); err != nil {
			_ = ipcConn.WriteFrame(ipc.Fail(reqID, protocol.RelayError, ipc.Human(protocol.RelayError)))
			return
		}
		if msg.ExecID != "" && msg.ExecID != execID {
			continue
		}
		switch msg.Type {
		case protocol.TypeStdout:
			_ = ipcConn.WriteFrame(ipc.Event(reqID, "stdout", msg.Data))
		case protocol.TypeStderr:
			_ = ipcConn.WriteFrame(ipc.Event(reqID, "stderr", msg.Data))
		case protocol.TypeExit:
			code := 0
			if msg.ExitCode != nil {
				code = *msg.ExitCode
			}
			dur := int64(0)
			if msg.DurationMS != nil {
				dur = *msg.DurationMS
			}
			trunc := false
			if msg.Truncated != nil {
				trunc = *msg.Truncated
			}
			_ = ipcConn.WriteFrame(ipc.OK(reqID, map[string]any{
				"exit_code": code, "duration_ms": dur, "truncated": trunc,
				"status": msg.Status, "exec_id": execID,
			}))
			return
		case protocol.TypeInvokeNack, protocol.TypeError:
			_ = ipcConn.WriteFrame(ipc.Fail(reqID, msg.Code, ipc.Human(msg.Code)))
			return
		}
	}
}
