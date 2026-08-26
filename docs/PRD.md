# PRD v0.5 — XallorRemote

| 字段 | 内容 |
| --- | --- |
| 状态 | 产品范围。读法见 [README.md](README.md)；名字与选型见 [decisions.md](decisions.md) |
| 产品名 | **XallorRemote** |
| 日期 | 2026-08-26 |

---

## 1. 一句话

任意一台授权电脑成为 Agent（以及人）可调用的执行节点：凭 **对方设备 ID + 授权码** 把任务送过去，**实时回流**。无需公网 IP、SSH、开放端口。

同一套安装既可被控，也可反过来控对方（两份授权）。不是远程桌面，不是 MCP 版 SSH。

---

## 2. 为什么做

Agent 能调 API，不能安全地用用户自己的电脑。SSH MCP 要端口和可达性；向日葵为人看屏幕；`host-execution-mcp` 管企业跳板 / k8s，不管家庭 NAT 后的个人设备。

用户要说的是：「去我那台 Windows 跑这个」，并在对话里看着输出往下走。

---

## 3. 原则

1. 主路径：ID + 授权码 → 任务 → 流式结果。
2. 一个 Node，多个头；禁止控制端专用包。
3. 安装幂等；身份直到 `xallor-remote reset` / `revoke` 才死。
4. 双向 = 两份 grant；入站默认关；首次启动不签发授权码。
5. 控制头经本机 Runtime 再出站；Cursor 关 MCP 不影响入站。
6. 流式默认（含 GUI 会话屏）。
7. 设备只出站；Relay 转发载荷、只存元数据。
8. 签发授权 / 开入站只走人或 CLI，不走 MCP tool。
9. 账号与订阅是后一层，见 [platform.md](platform.md)。
10. OS 差异只写在 [os.md](os.md)。
11. 默认走官方 hosted Relay，自托管随时可换，见 [decisions.md](decisions.md)。

---

## 4. 范围

### v0 Must

- 通用 MCP 包 `xallor-remote-mcp` + 幂等 ensure Go Runtime
- Windows + Linux Runtime：身份、出站、入站默认关、`grant issue`、流式 exec、默认 workspace、进程、本地策略
- CLI 覆盖人侧全部操作
- Relay：官方 hosted 默认 URL + `xallor-remote relay` 自托管
- 两台机器互换 grant 后都能 exec（可经官方 Relay，无需彼此公网 IP）

### v0.1

- Linux TUI、Tauri GUI、macOS Runtime、`xallor-remote mcp merge-config`

### v1

- hosted 账号、名下设备、云侧吊销、订阅骨架；mcp.json 仍允许纯 grant

### 不做（v0–v1 默认）

远程桌面 / 键鼠 / WebRTC / P2P / 目录同步 / K8s / Computer Use / 支付与组织 RBAC / MCP 签发 grant / 浏览器当第三条执行通道 / 短配对码 / Electron 壳。

---

## 5. 成功标准

双 OS 各跑一遍才算过。细节在 [os.md](os.md)。

1. 只装 MCP：Runtime 被 ensure 出来，**没有**授权码、入站仍关；默认 workspace 已创建并打印。
2. 再装一次 / 再开 Cursor：同一 `device_id`，仍一个 Runtime。
3. `xallor-remote grant issue` 后，对方只配 ID+授权码（默认官方 Relay）即可流式 exec。
4. 反向第二份 grant 同样成立。
5. 高危 `policy_deny` 或审批超时拒绝；rotate / revoke 立即失效；断网 `device_offline`。
6. 无 DISPLAY 的 Linux：v0 用 CLI 完成 peer / 流式 / cancel。
7. 改 `XALLOR_REMOTE_RELAY_URL` 指向自托管 Relay，同样能跑通。

v0.1 另加：TUI 或 GUI 在命令结束前能看到行。

---

## 6. 路线图

| 阶段 | 内容 |
| --- | --- |
| v0 | 执行闭环 + CLI + 官方 Relay 默认 + 自托管 |
| v0.1 | TUI / Tauri GUI / macOS |
| v1 | 账号与设备清单 |
| v1.x | 计量与套餐 |
| v2 | 组织、fleet、P2P 旁路 |
| 以后 | Desktop capability、A2A |

---

## 7. 和 host-execution-mcp

互补，不是替代。那边是 SSH / 跳板 / k8s；这边是个人设备 + 出站中转。Capability / 审计语义应对齐，Transport 不同。不要在 v0 重做异步大文件 copy。
