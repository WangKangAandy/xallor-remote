# 计划书怎么读

本目录是 **XallorRemote** 的产品定义。从这份文件进；每件事只在一个文件里写全。

**版本：** v0.5（2026-08-26）。品牌定为 XallorRemote；其余未决已按产品方向拍板，见 [decisions.md](decisions.md)。

---

## 1. 阅读顺序

| 顺序 | 文件 | 只回答 |
| --- | --- | --- |
| 1 | [PRD.md](PRD.md) | 做什么、不做什么、何时算做成 |
| 2 | [decisions.md](decisions.md) | 品牌、Relay、IPC、技术栈、workspace |
| 3 | [model.md](model.md) | 一台机器是什么、怎么装、怎么双向 |
| 4 | [credentials.md](credentials.md) | ID / 授权码 / 设备密钥 |
| 5 | [architecture.md](architecture.md) | 一次任务怎么走 |
| 6 | [protocol.md](protocol.md) | 消息与错误码 |
| 7 | [runtime.md](runtime.md) | 被控侧怎么执行 |
| 8 | [mcp.md](mcp.md) | Agent 侧 MCP |
| 9 | [relay.md](relay.md) | 中转与落盘 |
| 10 | [clients.md](clients.md) | GUI / TUI / CLI |
| 11 | [os.md](os.md) | Windows / Linux 差异 |
| 12 | [platform.md](platform.md) | 以后的用户 / 设备 / 订阅 |

---

## 2. 术语

| 说法 | 含义 | 不要再说 |
| --- | --- | --- |
| **XallorRemote** | 产品名 | Remote Device Runtime、rdr |
| **Node** | 一台装了本产品的电脑 | 控制端包 / 被控端包 |
| **Runtime** | 本机长驻 Go 进程 | Device Agent、rdr-agent |
| **头** | MCP / GUI / TUI / CLI | 四个独立产品 |
| **Relay** | 中转；默认官方 hosted，可自托管 | 向日葵服务器 |
| **设备 ID** | 路由键 | 当密码用 |
| **授权码 grant** | 控制端凭证 | 验证码 |
| **设备密钥 secret** | 仅 Runtime 上线用 | 配进 mcp.json |
| **Peer** | 对方 ID + 对方授权码 | 好友隧道 |
| **入站** | 允许别人对本机 invoke | 远控 |

CLI：`xallor-remote`。路径与前缀见 [decisions.md](decisions.md)。`platform.md` = 账号订阅；`os.md` = 操作系统。

---

## 3. SSOT

| 主题 | 只在这里写全 |
| --- | --- |
| 范围、成功标准 | [PRD.md](PRD.md) |
| 品牌与选型 | [decisions.md](decisions.md) |
| 一个包、幂等、双向 | [model.md](model.md) |
| 三套秘密 | [credentials.md](credentials.md) |
| 时序 | [architecture.md](architecture.md) |
| 消息与错误码 | [protocol.md](protocol.md) |
| 执行、策略、workspace | [runtime.md](runtime.md) |
| mcp.json 与 tools | [mcp.md](mcp.md) |
| 落盘 | [relay.md](relay.md) |
| CLI/GUI/TUI | [clients.md](clients.md) |
| Windows/Linux | [os.md](os.md) |
| Account / 订阅 | [platform.md](platform.md) |

---

## 4. 已拍板（摘要）

架构（v0.4）：头经 Runtime 出站；peers 只在 Runtime；首次不签发 grant；v0 人机靠 CLI；策略只在 Runtime；无短配对码。

产品（v0.5）：

1. 品牌 **XallorRemote**，CLI `xallor-remote`。
2. 默认官方 hosted Relay；自托管随时可换。
3. Windows named pipe / Unix socket，不对 localhost 随机开端口。
4. Runtime/Relay/CLI/TUI = Go；MCP = TypeScript；GUI = Tauri 2。
5. 默认 workspace 为用户目录下 `XallorRemote/workspace`，启动打印，禁止静默用盘符根。

细节只在 [decisions.md](decisions.md)。
