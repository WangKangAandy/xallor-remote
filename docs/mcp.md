# MCP 头

SSOT：mcp 配置与 tools。ensure 见 [model.md](model.md)。环境变量名见 [decisions.md](decisions.md)。

---

## 1. 它是什么

Cursor 等 spawn 的短驻 stdio 进程：`npx xallor-remote-mcp`。ensure Go Runtime；经 IPC 发任务；不持有到 Relay 的长期 WSS；不签发 grant。

---

## 2. 配置

单台覆盖（Runtime 收编进 peers）。Relay URL 可省略，默认官方 hosted。

```json
{
  "mcpServers": {
    "xallor-remote": {
      "command": "npx",
      "args": ["-y", "xallor-remote-mcp"],
      "env": {
        "XALLOR_REMOTE_DEVICE_ID": "dev_windows_gpu",
        "XALLOR_REMOTE_DEVICE_GRANT": "xr_grant_…"
      }
    }
  }
}
```

自托管时再加 `XALLOR_REMOTE_RELAY_URL`。多台用 `xallor-remote peer add`。不要写 IP / SSH。

可选：`XALLOR_REMOTE_TLS_INSECURE`（仅本机调试）、`XALLOR_REMOTE_LOG_LEVEL`、`XALLOR_REMOTE_STREAM_MODE`。

缺 grant 要在第一条 tool call 报错。不要把 mcp.json 提交进 git。

---

## 3. Tools（名字冻结）

| Tool | 作用 |
| --- | --- |
| `remote.devices` | peers + 在线状态 |
| `remote.device_info` | 单台；只配一台可省 id |
| `remote.exec` | **流式**；cwd 默认对方 workspace |
| `remote.exec_cancel` | exec_id，幂等 |
| `remote.read` / `remote.write` | 路径原样传递 |
| `remote.processes` | 只读 |

不提供签发 grant / 开入站。`exec` 结果含 `exit_code`、`duration_ms`、`truncated`、`exec_id`。

---

## 4. 流式

内部事件见 [protocol.md](protocol.md)。客户端支持 notifications 则边推；否则边收边攒——**Runtime↔Relay 必须是流**。
