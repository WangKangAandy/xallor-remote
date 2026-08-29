# XallorRemote

**XallorRemote**：每台电脑一个 Node。凭设备 ID + 授权码，经中转把任务发到对方 Windows / Linux，实时回流。同一套可反向互控。无公网 IP、无 SSH。

计划书从 [docs/README.md](docs/README.md) 进入。给代理的铁律与 PowerShell 教训：[AGENTS.md](AGENTS.md)。Relay：[docs/dataplane.md](docs/dataplane.md)。头与栈：[docs/heads.md](docs/heads.md)、[docs/stack.md](docs/stack.md)。

默认官方中转：`wss://api.xallor.com/remote`。人侧安装优先 **一个 npm 包** `xallor-remote-mcp`（内含 Runtime）；见 [docs/mcp.md](docs/mcp.md) / [docs/stack.md](docs/stack.md)。

本机自托管试跑（两台或两个终端）：

```text
go run ./cmd/xallor-remote relay --listen 127.0.0.1:8443
$env:XALLOR_REMOTE_RELAY_URL="ws://127.0.0.1:8443"
go run ./cmd/xallor-remote start
```

另一台同样设 Relay URL 后 `grant issue` / `peer add` / `exec -- whoami`。
