package noderun

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"sync"
	"time"

	"github.com/WangKangAandy/xallor-remote/internal/protocol"
)

func (d *Daemon) runLocalExec(link *wsConn, msg protocol.Message) {
	p, err := protocol.ParseExecPayload(msg.Payload)
	if err != nil {
		_ = link.send(protocol.Nack(msg.ExecID, protocol.AgentError))
		return
	}
	_, _, _, workspace, _, inbound, _ := d.store.Snapshot()
	if !inbound {
		_ = link.send(protocol.Nack(msg.ExecID, protocol.InboundDisabled))
		return
	}
	cwd := workspace
	if p.Cwd != "" {
		cwd = filepath.Join(workspace, p.Cwd)
	}
	ctx := d.ctx
	if p.TimeoutMS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(p.TimeoutMS)*time.Millisecond)
		defer cancel()
	}
	ctx, cancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.local[msg.ExecID] = cancel
	d.mu.Unlock()
	defer func() {
		cancel()
		d.mu.Lock()
		delete(d.local, msg.ExecID)
		d.mu.Unlock()
	}()

	cmd := buildCmd(ctx, p.Command, cwd)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	start := time.Now()
	if err := cmd.Start(); err != nil {
		_ = link.send(protocol.Nack(msg.ExecID, protocol.AgentError))
		return
	}
	var wg sync.WaitGroup
	pump := func(r io.Reader, typ string) {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				_ = link.send(protocol.Message{Type: typ, ExecID: msg.ExecID, Data: string(buf[:n])})
			}
			if err != nil {
				return
			}
		}
	}
	wg.Add(2)
	go pump(stdout, protocol.TypeStdout)
	go pump(stderr, protocol.TypeStderr)
	err = cmd.Wait()
	wg.Wait()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = 1
		}
	}
	status := protocol.ExitCompleted
	if ctx.Err() == context.Canceled {
		status = protocol.ExitCancelled
	} else if ctx.Err() == context.DeadlineExceeded {
		status = protocol.ExitTimeout
	}
	dur := time.Since(start).Milliseconds()
	_ = link.send(protocol.Message{
		Type: protocol.TypeExit, ExecID: msg.ExecID, ExitCode: &code,
		DurationMS: &dur, Truncated: protocol.Bool(false), Status: status,
	})
}

func buildCmd(ctx context.Context, command, cwd string) *exec.Cmd {
	var cmd *exec.Cmd
	if stdruntime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-lc", command)
	}
	cmd.Dir = cwd
	cmd.Env = os.Environ()
	return cmd
}
