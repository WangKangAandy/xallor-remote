# 账号、设备资产、订阅

SSOT：以后 hosted 的对象关系。v0 不实现登录和计费。执行协议见 [protocol.md](protocol.md)。OS 差异在 [os.md](os.md)，不要和本文件搞混。

对象关系现在定，避免以后推翻 grant。Relay 表预留可空 `owner_account_id`。

---

## 1. 先看全图

```text
Account（用户）
  ├── Subscription（订阅 / 权益）
  ├── Membership → Organization（以后团队）
  └── Device[]（名下机器）
         ├── identity（device_id + secret 哈希）
         ├── inbound 开关
         └── Grant[]（签发的「谁可以控我」）
                └── 被某个 Controller 持有
                     Controller = 另一台 Device 上的 MCP/TUI
                                  或（以后）某个 Account 的云侧会话
```

v0 坍缩：

| 以后 | v0 实际 |
| --- | --- |
| Account | 不存在；官方或自托管 Relay 都靠 grant |
| Subscription | 官方 Relay 可有连接/流量硬顶；无账单 |
| Device | `devices` 表 |
| Grant | `grants` 表，持有者 = 出示明文的人 |
| Organization | 无 |

Relay 表建议 **预留** `owner_account_id` 可空，避免 v1 迁移改主键。

---

## 2. 用户（Account）

### 2.1 是什么

能登录 hosted 控制面的人。不是 Device ID。一台电脑不属于「用户名」，属于某个 Account（或 v0 不属于任何人）。

职责：

- 认证（email / SSO 以后）
- 作为订阅主体
- 看名下全部设备、审计、账单
- 以后：把「授权」绑到账号而不是单台控制端机器（换笔记本不必重贴 grant）

### 2.2 不做的

- v0 不强制账号。自托管可以永远无账号。
- Device 执行权 **不能** 只靠「知道邮箱」。执行仍要 grant 或以后的短时 session。
- Agent（Cursor 里的模型）不是 Account。审计里 `actor` 要能区分：`user:…` / `device:…` / `mcp-agent`（能区分就区分，不能就记 client_meta）。

### 2.3 角色（团队以后）

| 角色 | 能力 |
| --- | --- |
| Owner | 账单、删组织、踢人 |
| Admin | 管设备、吊销 grant、看审计 |
| Member | 控被授权的设备、不能改订阅 |
| Device-only | 无登录，仅 Runtime（服务器） |

v0 只有隐式 Owner = 拿着自托管磁盘的人。

---

## 3. 设备（Device）

一台装了 Node 的电脑。主键 `device_id`。

属性（逻辑）：

- 所属 `owner_account_id?`
- os / arch / hostname / agent_version
- inbound_enabled
- workspace、capabilities 摘要
- online / last_seen
- 签发中的 grants

**设备管理** 产品能力（按阶段）：

| 能力 | 阶段 | 说明 |
| --- | --- | --- |
| 本机看自己 ID/状态 | v0 | CLI/GUI/TUI |
| Peer 列表（我能控谁） | v0 | 存在控制端本地 + 每次问 Relay 在线 |
| 名下所有设备一览 | v1 hosted / 自托管控制面 | 含办公室那台没开 GUI 的 |
| 远程吊销、改名、禁用入站 | v1 | 丢电脑时从网页踢掉 |
| 强制升级 Runtime | 以后 | 和幂等 ensure 对齐 |
| 库存标签、分组、fleet 调度 | v2 | 「找一台有 GPU 的」 |

设备管理 **不是** MDM（不管锁屏、刷机）。管的是：身份、入站、grant、在线、审计。

一台设备同时：

- 作为 **Target**：Runtime 在 Relay 上挂着
- 作为 **Controller**：其 MCP/TUI 持有别人的 grant

同一 `device_id` 两种角色，不要拆成两张「控制端表 / 被控端表」。

---

## 4. 授权与「好友关系」

v0：Grant 绑 `device_id`（被控）+ 哈希。控制端是匿名持有者。

以后建议演进（现在只占位，不实现）：

```text
Grant
  target_device_id
  scope: shell, files, …
  expires_at?
  consumer_type: opaque_secret | account | device
  consumer_id?
```

`opaque_secret` = 今天的授权码（可贴进 mcp.json）。  
`account` = 登录用户下的所有控制端都能控，换电脑不用换码。  
双向「互加好友」= 两个 Grant + 可选的一键互签（产品动作，协议仍两条）。

解绑：

- 只撤一个方向
- 或组织动作「解除配对」两边一起 revoke

---

## 5. 订阅（Subscription）

订阅卖的是 **权益与限额**，不卖协议。Free 与 Pro 走同一套 WSS/invoke。超限时错误码稳定，例如 `quota_exceeded`，MCP/GUI 都能展示。

### 5.1 建议计量维度（hosted）

| 维度 | 为什么存在 |
| --- | --- |
| 名下 Device 数 | 防止一张账号挂无数被控机 |
| 同时在线 Device | 成本在 Relay 连接 |
| 并发 exec / 账户 | CPU 与带宽 |
| 月中转流量或 stdout 字节 | 流式是贵的 |
| 审计保留天数 | 存储 |
| 座位（Team） | 多人 |

自托管：计划上可以是 `plan=self_hosted`，限额关或由运营方自己配。**不要把计费逻辑打进 Device Agent。** Agent 只听 Relay 的 nack。

### 5.2 套餐怎么切（示例，可改数字）

仅用于把「以后要有这一层」想清楚，不是报价单。

| | 自托管 / 无账号 | Hosted Free | Pro | Team |
| --- | --- | --- | --- | --- |
| 账号 | 无 | 有 | 有 | 有 + 组织 |
| 设备数 | 不限 | 少 | 多 | 更多 |
| 双向互控 | 有 | 有 | 有 | 有 |
| 审计保留 | 本地盘 | 短 | 长 | 长 + 导出 |
| GUI/TUI | 全功能执行 | 全功能 | 全功能 | 角色权限 |
| SSO / RBAC | 无 | 无 | 无 | 有 |

v0 只实现左列。Relay 对未知账户字段忽略。

### 5.3 订阅生命周期

`trialing → active → past_due → cancelled → revoked_features`

取消订阅 **不应** 删除 device_id（机器还在）。应：限额降级、hosted Relay 可拒绝超量连接、用户可导出审计、身份可迁回自托管。

支付提供商、发票、税：本文件不锁。控制面「订阅」页只需要：当前计划、用量、升级入口。

---

## 6. 控制面放哪

| | v0 | 以后 hosted |
| --- | --- | --- |
| 用户管理 | 无 | Web |
| 设备管理 | 本机客户端 | Web + 客户端 |
| 订阅管理 | 无 | Web（Stripe 等） |
| 执行与流式 | 不经 Web | 仍不经 Web；Web 不是 ToDesk |

Web 控制面 **不要** 做成第三种远程执行通道（浏览器里直接 shell），除非单独 PRD。默认：Web 管账号/设备/票，执行仍走 Node + MCP/TUI。

---

## 7. 和 MCP 配置的关系（演进时别把用户坑了）

| 阶段 | mcp.json 里是什么 |
| --- | --- |
| v0 | `RELAY_URL` + `DEVICE_ID` + `GRANT` |
| v1 有账号后仍允许 | 同上（CI、无头、不登录） |
| v1 额外允许 | `RELAY_URL` + `ACCOUNT_TOKEN`，peer 从云端拉列表 |

两种控制端认证长期并存。不要宣布「以后必须登录才能 MCP」。无 GUI 服务器永远要能只靠 grant。

---

## 8. 合规与信任（影响产品叙述）

- 用户管理要支持注销账号、导出审计、吊销所有 grant。
- 设备管理要支持「这台电脑卖掉了」：revoke + 新机新 ID。
- 订阅数据与 exec 载荷分离：账单系统拿不到 stdout。
- 区域：Relay 存审计在哪，以后再写；v0 自托管则数据在用户磁盘。

---

阶段与 [PRD.md](PRD.md) 路线图一致。任何账号/订阅需求塞进 v0 执行闭环都算膨胀。
