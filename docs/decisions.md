# 产品决策（已拍板）

品牌与实现选型只在这里写全。范围仍见 [PRD.md](PRD.md)。

---

## 1. 品牌与名字

| 用途 | 取值 |
| --- | --- |
| 产品名 | **XallorRemote** |
| CLI / 单二进制 | `xallor-remote` |
| npm MCP 包 | `xallor-remote-mcp` |
| mcp.json server 键 | `xallor-remote` |
| 环境变量前缀 | `XALLOR_REMOTE_` |
| 授权码前缀 | `xr_grant_` |
| 本机数据 | Windows `%APPDATA%\XallorRemote\`；Linux `~/.config/xallor-remote/` |
| 默认 workspace | Windows `%USERPROFILE%\XallorRemote\workspace`；Linux `~/XallorRemote/workspace` |
| MCP tools | 仍用 `remote.*`（对模型是能力名，不是品牌名） |

不要用 `rdr`、`rdr-agent`、`Remote Device Runtime`、`XalloRemote` 当对外名字。

---

## 2. Relay：官方 hosted 为默认，自托管始终能换

主路径是「无公网 IP、无 SSH」。若 v0 只做自托管，用户仍要先搞一台可达的 VPS，五分钟故事不成立。

**拍板：**

- 客户端 **默认** `XALLOR_REMOTE_RELAY_URL=wss://relay.xallorremote.com`。
- v0 官方 Relay：**无账号**，只靠 device secret / grant；不做计费、不做组织。可做连接数与流量的硬顶，超限 `quota_exceeded`。
- **自托管一等公民**：`xallor-remote relay --listen :8443 --data <dir>`，改 URL 即切走。数据不锁死在云上。
- 官方域名未上线前，dogfood 用自托管 URL；**产品默认仍按官方 hosted 写**，不要改回「v0 只能自托管」。
- v1 才在 hosted 上加账号、设备清单、订阅。纯 grant 的 mcp.json **永久允许**。

---

## 3. 本机 IPC

**拍板：按操作系统用该平台惯用本地 IPC，产品层同一套 API，禁止对 localhost 开一个随便端口。**

| OS | 机制 |
| --- | --- |
| Windows | Named pipe `\\.\pipe\XallorRemote` |
| Linux / macOS | Unix socket：优先 `$XDG_RUNTIME_DIR/xallor-remote.sock`，否则 `~/.config/xallor-remote/ipc.sock` |

权限：仅当前用户。MCP / CLI / TUI / GUI 都连这个口，不各自实现一套。

---

## 4. 技术栈

无头 Linux 必须是「一个二进制」，不能要求先装 Node 才能被控。MCP 又必须进 npm。

**拍板：**

| 部分 | 选型 | 理由 |
| --- | --- | --- |
| Runtime + Relay + CLI + TUI | **Go** | 交叉编译、单文件、systemd 友好 |
| MCP 头 | **TypeScript**（`xallor-remote-mcp`） | Cursor 生态、ensure 时拉起 Go Runtime |
| GUI v0.1 | **Tauri 2** | 小体积、调本机 Runtime；不做 Electron 远控壳 |

TUI 做进同一 Go 二进制（`xallor-remote tui`），不另发一套 Node TUI。

---

## 5. Workspace

拒绝启动会挡住「先上线、再 issue」；没有根目录又会让 exec 落在盘符根上。

**拍板：永远有一个显式默认沙箱，启动时打印；不许静默使用 `/` 或 `C:\`。**

- 首次启动若未指定 `--workspace`，创建并使用默认目录（见 §1）。
- 每次 `start` / `status` 打印当前 workspace。
- `--workspace` 或 `config.json` 可改；改完只影响后续 invoke。
- 目录被删：重建默认或报 `workspace_missing`，不要退回用户主目录根。
- 路径穿越仍 `policy_deny`。
