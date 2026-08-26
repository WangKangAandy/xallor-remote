# Runtime：被控与本地枢纽

SSOT：本机执行、策略、workspace、本地数据。CLI 命令表见 [clients.md](clients.md)。OS 差异见 [os.md](os.md)。进程关系见 [model.md](model.md)。

---

## 1. 职责

长驻、用户态（v0 不做 SYSTEM/root）。ensure / CLI / GUI 装起，幂等。

- `hello_device` + heartbeat
- 身份；`grant issue` 前入站关
- 收 invoke → 策略 → 执行 → 回推事件
- cancel；断线杀无主进程
- 对本机头提供：identity、peers、审批、代发 `hello_client`

不做：MCP stdio 本身、屏幕采集、经 MCP 签发 grant。

---

## 2. 数据目录

```text
Windows: %APPDATA%\XallorRemote\
Linux:   ~/.config/xallor-remote/
```

IPC 路径见 [decisions.md](decisions.md)。

| 文件 | 内容 |
| --- | --- |
| `identity.json` | device_id |
| `device_secret` | 仅当前用户可读 |
| `grant` | 仅 issue 之后；明文只为再显示 |
| `peers.json` | 对方 id + grant（控制名单） |
| `config.json` | relay、workspace、shell |
| `policy.json` | allow / deny / approval |

---

## 3. Workspace

未指定则使用默认沙箱并在首次创建（见 [decisions.md](decisions.md)）。启动 / status 必须打印当前路径。所有 read/write 与默认 cwd 在此树内。穿越 → `policy_deny`。目录丢失 → `workspace_missing`，不要退回盘符根。

---

## 4. 循环

```text
hello_device (inbound 按是否已 issue)
loop heartbeat
on invoke: 策略 → exec|read|write|processes|info|cancel
on disconnect: 退避重连；不恢复旧子进程
```

---

## 5. exec

禁止 `run()` 完再整包发送。必须 Popen，按行/块推 `stdout`/`stderr`，最后 `exit`。

- Windows：Job Object 杀树。Linux：process group。
- 字节转 UTF-8 再上送，见 [os.md](os.md)。
- 继承用户环境；不接受任意远程 env。
- v0 无交互 stdin。
- 并发见 [protocol.md](protocol.md)。

---

## 6. 文件 / 进程 / info

read：head/tail 或区间；v0 可拒二进制。write：覆盖。processes：当前用户可见即可。info：os/arch/hostname/version/workspace；可选探测 docker / nvidia-smi / git（false 即可，不要整次失败）。

---

## 7. Capability 与策略

v0 名字：

| Capability | 对应 op |
| --- | --- |
| `device.inventory` | info |
| `shell` | exec / cancel |
| `filesystem.workspace` | read / write |
| `process.list` | processes |
| `system.info` | info 内 |

未知 capability 默认拒绝。高危与凭据路径 denylist 见 [os.md](os.md)。sudo / 提权默认 deny 或 approval。审批在 **被控机** 本地；超时拒绝；无 TTY 且无 UI 则 approval 直接 deny。

Relay 已鉴权不能替代本步。

---

## 8. 卸载

断开；建议 `revoke`；删数据目录。只删二进制会留下 identity，重装抢同一 ID。
