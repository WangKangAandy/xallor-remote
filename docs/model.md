# 产品模型：一个 Node

SSOT：安装形态、幂等、双向、本机进程。选型见 [decisions.md](decisions.md)。

---

## 1. 一台电脑

用户装的是 **XallorRemote**，不是三套互不相认的软件。

```text
本机 Runtime（Go，长驻，一个）
  ├── 数据：identity、inbound、issued grant、peers、policy、workspace
  ├── IPC：Windows named pipe / Unix socket（见 decisions）
  ├── 连接 A：hello_device → Relay
  └── 连接 B：hello_client → Relay
        ▲
   MCP（TS 短驻）   CLI / TUI / GUI
```

Relay 默认官方 hosted（`wss://relay.xallorremote.com`），可改 URL 自托管。

---

## 2. 通用包与幂等

- MCP 包：`xallor-remote-mcp`，跨 OS 同一包。ensure 拉起本 OS 的 Go Runtime。
- 第一次被 Cursor 拉起：ensure → identity 已有则不重建 → **不** grant issue → 经 Runtime 干活。
- `xallor-remote mcp merge-config` 只幂等追加本产品条目。

| 再做一次 | 必须 | 禁止 |
| --- | --- | --- |
| 安装 / 开 MCP | 仍一个 Runtime，identity 不变 | 第二套服务、新 device_id |
| 小版本升级 | 换二进制，留数据目录 | 当全新安装 |
| Runtime 已在跑 | 不杀，除非强制重启 | 每次开 Cursor 闪断入站 |
| 用户改过 workspace / 策略 | 保留 | 恢复出厂 |
| mcp.json 已有本 server | 不重复插入 | 每次保存再追加一条 |

破坏性只有 `xallor-remote reset --yes`。

---

## 3. 双向

```text
B 执行 grant issue  →  A peer add B  →  A 控 B
A 执行 grant issue  →  B peer add A  →  B 控 A
```

两份 grant。入站默认关。ensure 后 Runtime 保持 `hello_device`（可 online 且 inbound_disabled）。

无桌面 Linux：`xallor-remote peer add` + `exec`，或本机也跑 MCP。

---

## 4. 控制路径

```text
Cursor ──stdio──► MCP ──IPC──► Runtime ──WSS──► Relay ──► 对方 Runtime
人     ──CLI/TUI/GUI ────────► Runtime ──WSS──► Relay ──► 对方 Runtime
```

mcp.json 单台 `XALLOR_REMOTE_DEVICE_ID` + `GRANT` 由 Runtime 收编进 `peers.json`。MCP 不持有长期 WSS。
