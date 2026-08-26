# 协议合同（v0）

SSOT：消息语义、错误码。时序见 [architecture.md](architecture.md)。JSON 字段名实现时可加前缀，**type 与 code 不得改义**。

本文不是加密规格，也不是完整 JSON Schema。

---

## 1. 两条连接

| role | 谁发起 | 证明 | 用途 |
| --- | --- | --- | --- |
| `device` | 被控 Runtime | `device_id` + `device_secret` | 上线、收 invoke、回事件 |
| `client` | 控制侧 Runtime | 目标 `device_id` + **该目标的 grant** | 发 invoke、收事件 |

同一 Runtime 可以同时有一条 `device` 连接和多条 `client` 连接（每个 peer 一条，或一条多路复用；实现自选，语义是按 target `device_id` 路由）。

---

## 2. 控制面消息

| type | 方向 | 必带 | 含义 |
| --- | --- | --- | --- |
| `hello_device` | Runtime → Relay | device_id, secret, os, arch, agent_version, workspace, inbound | 上线 |
| `hello_client` | Runtime → Relay | target_device_id, grant | 我要控这台 |
| `heartbeat` | 双向 | — | 保活 |
| `invoke` | client → Relay → device | exec_id, op, payload | op = exec / read / write / processes / info / cancel |
| `invoke_nack` | 反向 | exec_id, code | 未开跑 |
| `revoke` | 管理 | device_id 或 grant_id | 作废 |

`invoke.exec` payload：`command`、`cwd?`、`timeout_ms?`。  
`cancel` payload：`exec_id`。  
`read` / `write`：路径与区间，见 [mcp.md](mcp.md) 与 [runtime.md](runtime.md)。

---

## 3. 数据面消息

| type | 方向 | 必带 |
| --- | --- | --- |
| `stdout` / `stderr` | device → client | exec_id, data（UTF-8 文本） |
| `exit` | device → client | exec_id, exit_code, duration_ms, truncated |
| `error` | 任一跳 | exec_id?, code | 通道失败，**不是** 命令非零退出 |
| `progress` | 可选 | 文件百分比；v0 可不上 |

非零退出只走 `exit`。

`exec_id`：控制侧 Runtime 生成 UUID，全程携带。

---

## 4. 错误码

| code | 何时 |
| --- | --- |
| `unauthorized` | grant 错或已吊销 |
| `unknown_device` | ID 不存在 |
| `device_offline` | 登记过但无 device 连接 |
| `inbound_disabled` | 在线但未 `grant issue` / 入站关（有别于码错） |
| `policy_deny` | Runtime 策略拒绝 |
| `approval_timeout` | 要确认但没人点 |
| `approval_required` | 仅 nack：还在等确认（可选） |
| `exec_timeout` | 超时被杀 |
| `cancelled` | 主动取消 |
| `truncated` | 可与成功 exit 同时出现 |
| `relay_error` | 中转故障或重启丢 inflight |
| `agent_error` | Runtime 内部失败 |
| `quota_exceeded` | 官方 Relay 硬顶；自托管默认可关 |
| `workspace_missing` | workspace 目录不可用 |

MCP / CLI 对外必须用这些 code，再附一句人话。禁止只回 HTTP 502。

---

## 5. 大小（数字只在这里改）

| 项 | v0 |
| --- | --- |
| 单次 stdout+stderr | 1–4 MiB 后 truncated |
| read 默认 / 上限 | 64 KiB / 1 MiB |
| 每设备并发 exec | 2–4 |
| 排队 | 不排队，超额 nack |
| heartbeat 超时 | 30–60s → offline |
