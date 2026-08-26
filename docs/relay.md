# Relay

SSOT：中转职责、连接表、落盘、运维。消息与限额数字见 [protocol.md](protocol.md)。策略不在这里执行。

---

## 1. 做 / 不做

做：登记 device 连接；校验 client 的 grant；按 device_id 转发 invoke/事件；online/offline；grant 轮换与吊销；审计元数据。

不做：解释命令、读目标盘、存 stdout/文件正文、P2P、远程桌面、策略判定（只认 Runtime 的 nack）。

控制面与数据面 v0 可同进程，逻辑上分开。

---

## 2. 连接

同一 `device_id`：**一条** device 连接（新的踢旧的）。**多条** client 连接。回流只回发起该 `exec_id` 的那条 client。

无 heartbeat → offline。invoke 不排队。

---

## 3. 路由（内存）

```text
devices[id] = { device_conn, last_seen, meta, secret_hash, inbound }
grants[id]  = { grant_hash }
inflight[exec_id] = { device_id, client_conn }
```

invoke：grant 匹配 → online → inbound 开 → 未超并发 → 转发。否则 nack。控制端断开则对设备 cancel。设备断开则 inflight 全 `device_offline`。Relay 重启：inflight 失败，设备重连即可。

---

## 4. 落盘

必须持久：

| 集合 | 内容 |
| --- | --- |
| `devices` | id, secret_hash, os, arch, hostname, version, workspace, caps, last_seen, status, owner_account_id?（v0 空） |
| `grants` | device_id, grant_hash, 时间戳 |
| `audit` | ts, device_id, tool, args_digest, exec_id, decision, code, duration, client_meta |

禁止持久：grant/secret 明文、完整 stdout、write 正文、屏幕。

audit 建议 ≥7 天。备份数据目录 ≠ 备份用户文件。

---

## 5. 运维

公网只开 WSS。限制消息体。每 IP / 每 device 连接上限。管理口 v0 用本机 loopback CLI，不要无鉴权的 0.0.0.0 HTTP。

```text
xallor-remote relay --listen 0.0.0.0:8443 --data ./var/xallor-remote
```
