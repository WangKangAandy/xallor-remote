package noderun

import (
	"context"
	"log/slog"
	stdruntime "runtime"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/WangKangAandy/xallor-remote/internal/identity"
	"github.com/WangKangAandy/xallor-remote/internal/ipc"
	"github.com/WangKangAandy/xallor-remote/internal/protocol"
)

type wsConn struct {
	mu   sync.Mutex
	ctx  context.Context
	conn *websocket.Conn
}

func (w *wsConn) send(m protocol.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return wsjson.Write(w.ctx, w.conn, m)
}

type clientLink struct {
	target string
	ws     *wsConn
	mu     sync.Mutex
}

type Daemon struct {
	log    *slog.Logger
	store  *identity.Store
	ctx    context.Context
	cancel context.CancelFunc

	mu      sync.Mutex
	device  *wsConn
	clients map[string]*clientLink
	local   map[string]context.CancelFunc
}

func New(store *identity.Store, log *slog.Logger) *Daemon {
	ctx, cancel := context.WithCancel(context.Background())
	return &Daemon{
		log: log, store: store, ctx: ctx, cancel: cancel,
		clients: map[string]*clientLink{}, local: map[string]context.CancelFunc{},
	}
}

func (d *Daemon) Stop() { d.cancel() }

func (d *Daemon) Run() error {
	go d.keepDevice()
	ln, err := ipc.Listen()
	if err != nil {
		return err
	}
	defer ln.Close()
	d.log.Info("runtime ipc up", "device", d.store.DeviceID)
	for {
		c, err := ln.Accept()
		if err != nil {
			if d.ctx.Err() != nil {
				return nil
			}
			return err
		}
		go d.serveIPC(ipc.NewConn(c))
	}
}

func (d *Daemon) keepDevice() {
	backoff := time.Second
	for d.ctx.Err() == nil {
		if err := d.connectDevice(); err != nil {
			d.log.Warn("device reconnect", "err", err)
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

func (d *Daemon) connectDevice() error {
	id, secret, grant, ws, relay, inbound, _ := d.store.Snapshot()
	ctx, cancel := context.WithCancel(d.ctx)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, relay, nil)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	link := &wsConn{ctx: ctx, conn: conn}
	inb := inbound && grant != ""
	hello := protocol.Message{
		Type: protocol.TypeHelloDevice, DeviceID: id, Secret: secret,
		OS: stdruntime.GOOS, Arch: stdruntime.GOARCH, AgentVersion: protocol.Version,
		Workspace: ws, Inbound: protocol.Bool(inb),
	}
	if err := link.send(hello); err != nil {
		return err
	}
	var ack protocol.Message
	if err := wsjson.Read(ctx, conn, &ack); err != nil {
		return err
	}
	if ack.Type != protocol.TypeHelloOK {
		return errHello(ack.Code)
	}
	if grant != "" {
		_ = link.send(protocol.Message{Type: protocol.TypeGrantRotate, NewGrantHash: protocol.SHA256Hex(grant)})
		_ = link.send(protocol.Message{Type: protocol.TypeInboundSet, Inbound: protocol.Bool(true)})
	}
	d.mu.Lock()
	d.device = link
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		if d.device == link {
			d.device = nil
		}
		d.mu.Unlock()
		d.killAllLocal()
	}()

	t := time.NewTicker(protocol.HeartbeatEvery)
	defer t.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = link.send(protocol.Message{Type: protocol.TypeHeartbeat})
			}
		}
	}()
	for {
		var msg protocol.Message
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			return err
		}
		d.onDeviceMsg(link, msg)
	}
}

func errHello(code string) error {
	return errStr("hello: " + code)
}

type errStr string

func (e errStr) Error() string { return string(e) }

func (d *Daemon) onDeviceMsg(link *wsConn, msg protocol.Message) {
	switch msg.Type {
	case protocol.TypeHeartbeat:
	case protocol.TypeInvoke:
		if msg.Op == protocol.OpCancel {
			d.mu.Lock()
			cancel := d.local[msg.ExecID]
			d.mu.Unlock()
			if cancel != nil {
				cancel()
			}
			return
		}
		if msg.Op == protocol.OpExec {
			go d.runLocalExec(link, msg)
		}
	}
}

func (d *Daemon) deviceOnline() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.device != nil
}

func (d *Daemon) pushGrant(grant string) {
	d.mu.Lock()
	link := d.device
	d.mu.Unlock()
	if link == nil {
		return
	}
	_ = link.send(protocol.Message{Type: protocol.TypeGrantRotate, NewGrantHash: protocol.SHA256Hex(grant)})
	_ = link.send(protocol.Message{Type: protocol.TypeInboundSet, Inbound: protocol.Bool(true)})
}

func (d *Daemon) killAllLocal() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.local {
		c()
	}
}
