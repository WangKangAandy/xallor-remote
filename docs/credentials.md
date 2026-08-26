# 设备 ID 与授权码

SSOT：三套秘密。名字前缀见 [decisions.md](decisions.md)。

---

## 1. 用户看见什么

`xallor-remote start` 之后：

```text
Relay:     wss://relay.xallorremote.com
Device ID: dev_windows_gpu
Workspace: C:\Users\<you>\XallorRemote\workspace
入站:      关（还没有授权码）
```

需要给别人控：`xallor-remote grant issue`

```text
授权码: xr_grant_8f3c…    ← 给对方 peer add / mcp 覆盖
```

短配对码 v0 不做。反向再签发一张，互不替代。

---

## 2. 三套秘密

| 名字 | 谁拿着 | 寿命 | 用途 |
| --- | --- | --- | --- |
| 设备 ID | 可见 | 直到撤机 | 路由 |
| 授权码 grant | 控制端 | 直到轮换/吊销 | 允许指挥这台 |
| 设备密钥 secret | 仅本机 Runtime | 直到重装/撤机 | 证明上线的是这台 |

---

## 3. 规则

授权码：高熵、前缀 `xr_grant_`；Relay 只存哈希；日志不得打全文；禁止放进 URL。v0 一台设备一个有效 grant，轮换即作废。多控制端可共享这一份。

设备 ID：同一 Relay 唯一；不是密钥。

`xallor-remote grant rotate`：旧码立刻 `unauthorized`。`revoke`：踢线。只杀进程 → `device_offline`。泄漏 grant 须立刻 rotate。

多 peer：`xallor-remote peer add`；多台时 exec 必带 device_id。

---

## 4. 威胁

| 威胁 | 缓解 |
| --- | --- |
| 只装 MCP 就被控 | 默认无 grant、入站关 |
| grant 进 git / 聊天 | 日志打码；MCP 不签发 |
| 冒充设备上线 | secret 不出盒 |
| 伪造 ID 打别人 | grant 必须匹配该 ID |
| Relay 被读流量 | TLS；不落 stdout；E2E 以后 |
| 模型乱开入站 | 无此类 MCP tool |
| 官方 Relay 依赖 | 可随时改 URL 自托管 |
