package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Frame struct {
	ID      string          `json:"id,omitempty"`
	Event   string          `json:"event,omitempty"`
	Data    string          `json:"data,omitempty"`
	OK      *bool           `json:"ok,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Code    string          `json:"code,omitempty"`
	Message string          `json:"message,omitempty"`
}

type Conn struct {
	c  net.Conn
	mu sync.Mutex
	rd *bufio.Reader
}

func NewConn(c net.Conn) *Conn {
	return &Conn{c: c, rd: bufio.NewReader(c)}
}

func (c *Conn) ReadRequest() (Request, error) {
	line, err := c.rd.ReadBytes('\n')
	if err != nil {
		return Request{}, err
	}
	var r Request
	if err := json.Unmarshal(line, &r); err != nil {
		return Request{}, err
	}
	return r, nil
}

func (c *Conn) ReadFrame() (Frame, error) {
	line, err := c.rd.ReadBytes('\n')
	if err != nil {
		return Frame{}, err
	}
	var f Frame
	if err := json.Unmarshal(line, &f); err != nil {
		return Frame{}, err
	}
	return f, nil
}

func (c *Conn) WriteFrame(f Frame) error {
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err = c.c.Write(append(b, '\n'))
	return err
}

func (c *Conn) Close() error { return c.c.Close() }

func OK(id string, result any) Frame {
	raw, _ := json.Marshal(result)
	t := true
	return Frame{ID: id, OK: &t, Result: raw}
}

func Fail(id, code, message string) Frame {
	f := false
	return Frame{ID: id, OK: &f, Code: code, Message: message}
}

func Event(id, ev, data string) Frame {
	return Frame{ID: id, Event: ev, Data: data}
}

func Call(c *Conn, id, method string, params any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req := Request{ID: id, Method: method, Params: raw}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err = c.c.Write(append(b, '\n'))
	return err
}

func Human(code string) string {
	switch code {
	case "unauthorized":
		return "授权码无效或已吊销。"
	case "unknown_device":
		return "找不到这台设备。"
	case "device_offline":
		return "目标不在线。"
	case "inbound_disabled":
		return "对方尚未开放入站。"
	case "unknown_exec":
		return "没有这条任务。"
	case "relay_error":
		return "中转已断开，请稍后重试。"
	default:
		if code == "" {
			return "失败。"
		}
		return fmt.Sprintf("失败（%s）。", code)
	}
}
