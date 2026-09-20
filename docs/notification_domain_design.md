# 通知域（Notification Domain）设计文档

> **状态：P0 + P1 已落地（后端内聚 + 三端投递台账页 + 邮件文案 i18n + SSE 事件类型注册表，2026-09-19），
> P2 第一块已落地（站内信注册为 INTERNAL 渠道 + 定向发送改走缝 + 台账 `related_id`，2026-09-20，见 §4 P2）；
> P2 剩余（规则表 / WEBHOOK 渠道 / 异步投递）与 P3 仍是设计提案。**
> 第 2 节「现状盘点」是 P1 之前的基线（核对至 commit `29d700b9`），其中被改动的事实在就地标注；
> 第 3~4 节的实施状态以 §4 的分期标记为准，落地验收进度在 §7。实施进度更新时改本文状态标记，不要另开文档。
>
> 参照物：兄弟项目 `go-wind-im` 的 `backend/app/notification/service`（独立通知微服务 + Kafka 事件驱动）。
> 本文借用它的**领域词汇**，明确不借它的**部署拓扑**。

## 1. 概述与目标

当前仓库的"通知"能力是**两个互不相认的半个域**：站内信（`internal_message`）成熟但只能站内投递，
通知渠道（`notification_channel`）能发站外但没有调度层。本文提出的 Notification 域回答三个问题：

1. 业务代码想"通知某个人"时，**唯一的入口**是什么；
2. 一条通知走了哪些渠道、成功没有、失败为什么——**台账**在哪；
3. 站内信与站外发送如何**收敛成同一套渠道抽象**，而不必重写已有的收件箱。

**明确不做什么**：

| 不做 | 原因 |
| --- | --- |
| 拆独立 notification 微服务 | admin 是单体（`backend/AGENTS.md`「当前 admin 服务为单体架构」），无服务发现与 gRPC 互联；`gow extract` 是后话 |
| 引入 Kafka 事件总线 | 仓内无 Kafka。异步已有 asynq（`docs/task_system.md`），`pkg/eventbus` 声明了 `EventEmailSent` 等事件但全库零发布零订阅，属死脚手架，不在其上盖楼 |
| 一期做规则引擎 / 模板管理 / 用户偏好 | 今天完全没有 → 无兼容包袱，但工作量最大。推到 P2/P3，一期用 Go 侧静态路由表 |
| 新增 `identity.service.v1.Module` 枚举值 | `Module` 直接挂套餐模块白名单门禁（见 `plan_billing.md`），新增值要联动已部署租户的套餐数据。通知相关服务归 `Module_SYSTEM`，站内信保持 `Module_INTERNAL_MESSAGE` |

## 2. 现状盘点

### 2.1 两个半个域

| | 站内信 | 通知渠道 |
| --- | --- | --- |
| proto 包 | `internal_message.service.v1` | `notification_channel.service.v1` |
| 表 | `internal_messages` / `internal_message_recipients` / `internal_message_categories` | `sys_notification_channels` |
| 租户 | 租户级（`mixin.TenantID` + 5 个租户前缀索引 + `TenantMutationGuardPolicy`） | **平台全局**（无 tenant mixin，`name` 全局唯一索引） |
| 存什么 | 消息本体 + 每人一行收件记录 | **只有 SMTP 账号凭证** |
| 能发什么 | SSE `notification` 事件 | 一封测试邮件、一封找回密码码、一封绑定邮箱码 |
| 互相引用 | 无 | 无（仅 `ent/schema/notification_channel.go:13-15` 一句"供站内信外发使用"的注释，未实现） |

> **P2 之后这一张表只剩基线价值**：两个半个域已经由 `Notifier` 缝 + `InternalMessageSender` 连起来
> （站内信成为 INTERNAL 渠道），"供站内信外发使用"那句注释不再是空头支票。落地形态见 §3.1 与 §4 P2。

### 2.2 站外发送的全部三处，且无抽象

> **本节是 P1 之前的盘点，行号冻结在当时的形状上**（这三处已在 P1 迁完，见 §4）。
> 留着是因为"三处各自内联同一套策略"就是这个域存在的理由，而且表里的行号是当时实测的产物。

`grep -rn "func .*\.Notify(" app/ pkg/` → **零命中**。不存在发送抽象。全仓出站调用点穷举如下，
每处各自内联"挑渠道"+ 字符串拼正文：

| # | 位置 | 渠道选择 | 正文 |
| --- | --- | --- | --- |
| 1 | `service/notification_channel_service.go:133`（`SendTestEmail`） | `GetDecryptedSmtpAccount(id)`，按显式 ID | `:128-131` 拼接，烘焙操作人 ID |
| 2 | `service/authentication_forgot_password.go:56` | `GetFirstEnabledEmailChannel()` | `:54-55` 硬编码中文 |
| 3 | `service/user_profile_contact.go:32` | `GetFirstEnabledEmailChannel()` | `:30-31` 硬编码中文 |

底层能力只有 `pkg/mailer/smtp.go:27` 的 `SendMail(cfg, to, subject, body) error`——
`net/smtp` + PLAIN，同步、无 ctx、纯文本（`buildMessage` 在 `smtp.go:117` 写死 `Content-Type: text/plain`）、
无队列无重试。三处调用全部**同步跑在请求路径上**（`ForgotPassword` 会阻塞在 SMTP 上）。

后果：无重试、无出站审计、无按收件人去重、无限流、无 HTML、无正文 i18n、主渠道坏了无回退，
且"取 ID 最小的启用 EMAIL 渠道"这条策略（`notification_channel_repo.go:331-339`）在两个 caller 重复实现。

### 2.3 站内那半边已经是能用的

`InternalMessageService.SendMessage`（`internal_message_service.go:316`）→ 父消息落库 →
全员广播走 asynq 任务 `broadcast_message`（`pkg/task/broadcast_message.go`，handler 注册在
`internal/server/asynq_server.go:91`）→ `CreateBulk` 的 `ON CONFLICT DO NOTHING` 保证重试幂等 →
落库后 `TryPublish` 推 SSE（`streamID == userId`，见 `sse_architecture.md`）。

**这套东西是统一域要 wrap 的实现，不是要重写的对象。** 去重唯一约束、幂等批量插入、租户隔离、
SSE 扇出与 streamID 归属校验（`HandleAuthorize` 不匹配即 403）都已建成并验证过。

### 2.4 P0 实测复核（2026-09-19，已修）

本小节初稿是**静态阅读得出的**，三条里只有一条真是缺陷、且成因和我写的不一样。
以下按"跑起来的实测"重写（依据：go-crud v0.0.55 `rule/tenant.go` + `mixin/tenant_id.go` 源码、
现网库 `docker exec citus-server-standalone psql` 查询、两个 sqlite 集成测试）。

1. **`title`/`content` 不是"两种形状"，而是**正文只归父消息**的单一真相源。**
   收件行 ent schema 上确实没有这两列，repo 的 `Create`（`internal_message_recipient_repo.go:143`）与
   `CreateBulk`（`:177`）也不写它们；但 DTO 上带这两字段是**有意的瞬时载荷**：push 路径从内存 DTO
   marshal 给 SSE，pull 路径由 `ListUserInbox`（`internal_message_recipient_service.go:54-66`）
   一次 `ListByIds` 批量回填父消息正文。两侧最终形状一致。
   → 真正要守的是不变式：**任何新发送路径必须先落父消息再落收件行**，否则 pull 侧空标题、push 侧有标题。
   已在 repo `Create` 上补文档注释钉住它。统一域的 `InternalMessageSender` 沿用同一条。
2. **收件行租户归属确实有 bug，但不是"从未赋值 / 落 NULL"。** 机制实测为：repo 会调
   `SetNillableTenantID`（值来自调用方），列上 `Default(0)` 所以缺值是 **0 而非 NULL**；
   租户上下文下 go-crud `TenantPrivacy` 还会强制覆盖为 viewer 租户。
   真正的破口是 `AsyncBroadcastMessage` 无条件用 `SystemViewer` 重建 ctx：
   平台上下文放行 → ① `userRepo.List` 不受租户约束，**租户管理员的"全员广播"实际向全平台用户扇出**；
   ② 收件行落成 `tenant_id=0`，而收件箱读取被强制过滤为本租户 → **谁都读不到**。
   修复：任务载荷带 `TenantId`，handler 据此构造 `UserViewer` ctx；`executeBroadcast` 的租户
   **从 ctx 的 viewer 反取**而非另传一份，使"受众范围"与"行打标"不可能分叉。回退 goroutine 路径
   本来就带着请求 viewer，无需改。
   （P2-2 之后这一句只对**受众范围**成立：viewer 租户决定扇出到哪，行打标改由**每个收件用户自己的**租户逐行决定，
   因为平台上下文的扇出覆盖全平台，用它打标会把租户用户的行写进租户 0。）
   回归测试：`TestInternalMessageRecipientTenantSqlite`（钉住 go-crud 三种上下文行为）+
   `TestInternalMessageServiceSqlite_AsyncBroadcastTenantScoping`（钉住扇出只覆盖本租户、
   行可被本租户读者读到、他租户读者读到 0 行）。
   遗留：平台管理员（tenant 0）广播的收件行仍是 `tenant_id=0`，租户用户读不到 —— **已由 P2-2 修掉**
   （打标改逐行跟随受众，见 §4 P2-2 与 §6 决策点 4）。
3. **前端订阅了后端从不发布的事件** —— 成立。vue-element `components/NoticeDropdown/useNotice.ts`
   订阅 `"notification-revoke"`，全仓 `backend/` 零命中该字符串，约 20 行撤销逻辑不可达。
   已删除：撤回在库侧直接删收件行，下次拉取自然消失，不需要增量同步事件。

### 2.5 名称冲突陷阱

`InternalMessageService.sendNotification`（`:436`）与 `publishNotification`（`:412`）听起来像统一入口，
实际是**私有方法、只处理站内、与渠道域零依赖关系**。新域落地后这两个命名必须避开或改名。

> P2 处置（2026-09-20）：没有改名，而是**把语义兑现成名字**——这两个方法现在是 INTERNAL 渠道的
> 投递内核，由 `InternalMessageSender` 经缝调用，"看起来像统一入口"的地方（谁该发一条站内信）
> 真的收敛到了 `Notifier`。改名解决不了当初担心的事（有人绕过缝直接调），边界写在 §4 P2 与
> `sse_architecture.md` 第 4 节，并由接缝测试守着。

## 3. 设计

### 3.1 分层与依赖方向

```
业务 service（authentication / user_profile / notification_channel / 未来的告警、审批…）
        │ 只依赖 Notifier 接口（internal/service/notifier.go）
        ▼
NotificationService.SendDirect（internal/service/notification_service.go）
   ├─ 事件类型 → 渠道（一期 Go 静态表 eventChannels；请求可显式覆盖；二期 notification_rules 表）
   ├─ sys_notification_deliveries 台账：先落 SENDING，再发
   └─ 同步 channel.Registry.Sender(ch).Send()  ← P2 才换成 asynq 投递任务
                │
   ┌────────────┴──────────────────┐
   ▼                               ▼
EmailSender                  InternalMessageSender（P2 已落地，见下）
internal/data/channel/        internal/service/（成环，位置不可照抄 IM）
```

**为什么 `InternalMessageSender` 不在 `data/channel/`**：IM 那边它是通过 gRPC client 回打
internal_message 服务，所以在 data 层；admin 是单体，站内信的实现住在 `internal/service`，
而 `data` 包 import `service` 包会成环。故该适配器定义在 `internal/service` 内，由
`wiring_ent.go` 在装配期注册进 `data/channel` 的 Registry。这是单体**唯一不能照抄 IM 的地方**。

P2 落地时补两条首稿没写的约束：

- 它持有的是 `*InternalMessageService`，**不是**收件记录仓储。差别在于 SSE 实时推送：
  仓储是构造期就定死的依赖，而 publisher 是 `NewSseServer` 之后才 `RegisterInternalMessagePublisher`
  注入的——直连仓储会把"启动早期那次投递静默不推送"变成常态；
- 不成环的前提是装配顺序：`NotificationService` 只持有 `*Registry` 指针，
  两条注册（`channelRegistry.Register(sender)` 与 `internalMessageService.RegisterNotifier(svc)`）
  都在两个 service 构造完之后执行，所以是"运行期互相引用"，不是"构造期循环"。
  反过来，`InternalMessageService` 拿不到 `Notifier` 时**必须报错而不是 noop**
  （`unwiredNotifier`：宁可 HTTP 500 也不让一条站内信静默消失）。

**为什么用接口不用包**：`Notifier` 接口声明在**使用方**所在处（`internal/service/notifier.go`，
Go 惯例），实现是 `*NotificationService`，wiring 传具体指针。不成环——`NotificationService`
只依赖 deliveryRepo + Registry。

契约里容易想当然的一条：`SendDirect` 失败时**同时**返回非 nil 响应（带 delivery_id 与
SKIPPED/FAILED）和非 nil error。只回 error，调用方拿不到台账行号；只回响应，业务分支会以为发成功了。

### 3.2 系统上下文，不是操作人上下文

`SendDirect()` **不得**读 `auth.FromContext(ctx)`：租户到期扫描、定时任务、审计归档等系统发起的通知没有操作人，
找回密码更是免鉴权入口。操作人由调用方显式传 `operator_user_id`，只用于台账归类；
写入走调用方带进来的 ctx（系统任务侧即 `viewer` 系统视角，与 `docs/task_system.md` 的常驻任务口径一致）。

### 3.3 数据模型：`sys_notification_deliveries`（P1 已按此落地）

一条通知 × 一个渠道 × 一个收件人 = 一行。列全通用，不带 SMTP 形状。表名带 `sys_` 前缀，
与"渠道花名册 / 投递台账 = 平台级配置与运行痕迹"这条既有约定同形。

| 列 | 说明 | 状态 |
| --- | --- | --- |
| `event_type` | 枚举，业务事件标识 | 已建（`Optional().Nillable()`） |
| `channel` | 枚举，渠道 | 已建（同上） |
| `channel_id` | 可空，指向实际选中的 `sys_notification_channels` 行（策略结果落档，便于排障） | 已建 |
| `recipient_user_id` | 可空（直发模式无 userId） | 已建 |
| `target` | 脱敏后的投递目标（邮箱留首字符与域名，见 `maskTarget`）。**INTERNAL 例外：原样存收件用户 ID** —— 那是本平台内部主键、台账本就只对平台管理员开放，掩成 `****1024` 只会把台账里唯一可读的字段变成噪音 | 已建 |
| `status` | `SENDING` / `SENT` / `FAILED` / `SKIPPED` | 已建 |
| `last_error` / `sent_at` | 结果与完成时间（FAILED/SKIPPED 不留 `sent_at`） | 已建 |
| `related_id` | 可空，**按 `event_type` 解释**的业务对象 ID（`INTERNAL_MESSAGE` → `internal_messages.id`）。台账不存正文快照，这一列是"这条投递发的是什么"的唯一回跳入口 | P2 已建（`Optional().Nillable()` + 索引 `(event_type, related_id)`，启动期 ent 自动迁移加列加索引，实测已在 `gwa` 库出现） |
| `event_type/channel/status` 三枚枚举 | 必须 `Optional().Nillable()`：值型枚举列 + 可选指针 DTO 会踩 copier 的"指针↔指点对"失配，读回恒为零值（同款坑见 `notification_channel_repo.go` 的 `queryTypeByIDs` 注释） | 已按此建 |
| `request_id` | 唯一索引，幂等锚 | **推迟 P2**：P1 无异步重试，重复调用收敛暂无消费方 |
| `attempts` | 重试计数 | **推迟 P2**，同上 |
| `title` / `content` | 正文快照 | **不做**：一期三个事件里两个的正文就是验证码本身，快照进永久台账等于建了一张 OTP 明文表。P2 引入模板后按"模板 ID + 渲染参数"存，不存渲染结果 |
| mixin | `AutoIncrementId` / `TimeAt` / `OperatorID` | 已按此建；**不挂 `TenantID`**（见 §6 决策点 3 结论），也不挂 `SwitchStatus`（台账没有"停用"语义） |

台账**不提供 Update/Delete 业务方法**：改一条已发生的投递等于伪造事实；只有 `MarkResult`
按主键回写结果字段。

### 3.4 文件清单（P1 已落地，与首稿的差异已就地标注）

```
api/protos/notification/service/v1/notification.proto          源域：SendDirect / ListDelivery / GetDelivery + 三个枚举
api/protos/admin/service/v1/i_notification.proto               BFF：只出台账 List/Get 两个读路由（无发送路由，见下）
app/admin/service/internal/data/ent/schema/notification_delivery.go
app/admin/service/internal/data/notification_delivery_repo.go
app/admin/service/internal/data/channel/sender.go              Sender / SendRequest / SendReceipt / Registry + ErrChannelNotConfigured
app/admin/service/internal/data/channel/email_sender.go        包装 pkg/mailer，持有渠道选择策略（不持 logger，见下）
app/admin/service/internal/service/notification_service.go     SendDirect 实现 + 事件路由表 + maskTarget
app/admin/service/internal/service/notifier.go                 Notifier 接口
app/admin/service/internal/service/internal_message_sender.go  P2：INTERNAL 渠道适配器（成环分析见 §3.1）
--- 以下为 P2 剩余项，P1 未建 ---
app/admin/service/internal/data/channel/webhook_sender.go      等 §6 决策点 2 定了表结构再写
pkg/task/notification_dispatch.go                              见下"为什么 P1 没有异步投递"
```

两处刻意偏离首稿：

1. **BFF 不开放"发一条通知"的 HTTP 路由**。`SendDirect` 只由进程内业务 service 经 `Notifier` 调用；
   开成端点等于给任意已登录操作员一个"向任意邮箱发信"的入口，而唯一的站外手动触发口
   （渠道测试邮件）已在 `notification-channels` 路由上存在。
2. **P1 没有 asynq 投递任务**：`SendTestEmail` 的产物就是"SMTP 报错原文"，必须同步返回；
   验证码邮件同样要立刻知道"渠道没配"以便回不同文案。异步化在 P2 随 `request_id`/`attempts`
   一起引入（届时的形态：入队 delivery_id，正文由 handler 重新渲染而非从台账取）。

`sys_notification_channels` 与 `NotificationChannelService` 在 P1 **不改一行业务代码**——
它继续作为"渠道花名册"的 CRUD 存在，只是渠道选择策略从两个 caller 收进 `email_sender.go` 一处；
`SendTestEmail` 改为转调 `Notifier`，自身不再碰 SMTP。

### 3.5 一期路由：Go 静态表，不建规则表

`event_type → 渠道`，写在 Go 里（`notification_service.go` 的 `eventChannels`）；请求可显式传
`channel` 覆盖（覆盖优先，用于显式指定走哪一条已实现渠道的调试场景）。P1 三个事件
（密码重置码、联系人绑定码、渠道测试邮件）全部 → EMAIL，P2 加 `INTERNAL_MESSAGE → INTERNAL`。
路由表指向了没有注册实现的渠道 = 代码 bug，记 **FAILED** 而非 SKIPPED，并且照样留台账行。
理由：一期事件种类个位数，DB 规则表带来的"不发版改路由"价值，在这个规模下抵不上多一张表 + 一套 BFF + 三端页面的成本。
P2 建 `notification_rules` 时，这张 Go 表整体平移进 DB，调用方签名不变——所以不是白做的过渡态。

一条 P2 定的调用方纪律：**业务侧不显式传 `channel`**。站内信生产点只声明事件类型，渠道由这张表决定——
否则"路由表"和"调用方自报渠道"变成两个真相源，`notification_rules` 平移进 DB 那天会有一处是摆设。

### 3.6 SSE 事件类型注册表（P1 已落地 2026-09-19）

`sse_architecture.md:72` 明确记着当前**没有事件类型注册表**，生产方与消费方靠手工对齐裸字符串，
全仓只有一个 `"notification"`。新域应 own 一个注册表（Go 常量 + 三端同步的枚举），
而不是往这个无政府命名空间里加第四个裸串。§2.4-3 的 `"notification-revoke"` 已按"删前端"处置。

落地形态：后端 `pkg/sseevent`（`const Notification = "notification"`，包注释钉住"已发布事件的常量值不可改——
线上前端按字面值订阅，改名等于静默断流"），三端各一个 `transport/sse/event.ts`（`SSE_EVENT.Notification`），
消费点由裸串改为引用常量：react `HeaderContent.tsx`、ele `useNotice.ts`（顺带删掉本地重复常量
`NOTICE_EVENT`）、vben `basic.vue`。新增事件类型时四处一起加，任一端漏改由契约测试/编译期挡。

**注册表这件事顺带暴露了一个真 bug**：给事件类型建表时才发现 `internal_message_service.go` 的推送载荷
用的是 `encoding/json`——它对生成的 protobuf 结构体按 **struct tag** 出蛇形键（`message_id`/`created_at`），
而三端读的是驼峰。后果分两种，且都不报错：ele 端 `data.messageId` 恒 undefined，实时通知从来不亮；
vben 端乐观插入拿到 `messageId:0` + Invalid Date，其去重逻辑随后把真正的推送也吞了。
改为 `protojson.Marshal`（输出驼峰 + 枚举名），`pkg/sseevent.Notification` 与流 ID 一起由
`internal_message_sse_payload_test.go` 钉住载荷契约。**结论：载荷编码不是可以"顺手换一个"的细节，
它是三端的解析契约**——`sse_architecture.md` 若补写载荷格式，应写死 protojson。

## 4. 实施阶段

### P0 前置修复（已完成，见 §2.4）

1. 定 `title`/`content` 归属：正文只归 `internal_messages`，收件行 DTO 上两字段作为瞬时载荷，
   repo `Create` 补注释钉住"先落父消息"不变式；
2. 收件行 `tenant_id` 按 viewer 打标，广播扇出收敛到本租户（附两个 sqlite 回归测试）；
3. `"notification-revoke"` 删除前端订阅。

### P1 核心内聚（已完成 2026-09-19）

已完成并验证（`go test ./app/... ./pkg/...` 全绿；台账状态机 7 例 + 渠道页发送分支 1 例）：

1. §3.4 清单里的后端文件全部落地（含 proto 生成、ent 生成、手写 wiring 装配、rest 注册）；
2. §2.2 的三处裸发送点迁完：找回密码 / 联系人绑定码 / 渠道测试邮件；
3. `mailer.SendMail` 加 ctx（拨号与 STARTTLS 都吃 ctx，登录页不再可能被 OS 级连接超时挂住）；
4. `ErrChannelNotConfigured` 哨兵把"渠道没配/没启用/类型不对"（SKIPPED）与"SMTP 报错"（FAILED）分开——
   找回密码据此回两句不同文案，配置页据此决定是去启用渠道还是去查 SMTP。

**验收信号（已达成）**：`grep -rn "mailer.SendMail" app/` 只命中 `internal/data/channel/email_sender.go` 一处。

前端「通知投递记录」只读页已按 react 先行 → 移植的顺序做完（三端 typecheck 全绿，vben 端 `--force` 复跑确认）：
三端形状差异与移植时踩到的框架落差见 §7 末尾的"移植记录"。

首稿列为"P1 剩余"的两项也已在同日做完，落地形态与最初设想不同，记下来免得按原设想去验收：

- **邮件正文 i18n**：仓里没有服务端 i18n 设施（无 localizer、无 .prom、x/text 只是间接依赖），
  因此不是"改成 i18n key 交给前端翻译"——邮件正文的读者是邮箱客户端，前端根本不在链路上。
  实际做法是新增 `pkg/mailtext`：单一文案出口 + 按请求 `Accept-Language` 选中/英文案表
  （`netutil.HeaderFromContext` 读头，与全仓其余读头方式一致），三个生产点（找回密码 / 绑定码 / 测试邮件）
  只调它拿 `title/content`。zh 文案与迁移前逐字节相同，en 为新增。
  **未做**：用户偏好级语言（用户表无 locale 字段），找回密码这类免鉴权入口只能按请求头判。
- **SSE 事件类型注册表**：见 §3.6，落地时顺带修掉一个静默失效（同节）。

### P2 路由可配 + 渠道解耦 + 异步投递

**P2-1（站内信接入缝，已完成 2026-09-20）** —— 契约、表、渠道、缝四件事一起落：

1. `EventType_INTERNAL_MESSAGE` + `NotificationDelivery.related_id`（含索引）+ `SendDirect` 入参
   `related_id`；proto → `buf generate` → `make ts` → `make openapi` → `ent generate` 全链走完，
   三端 TS 的 `notificationservicev1_EventType` 联合类型确认含新值；
2. `InternalMessageSender` 注册进 Registry（位置与依赖见 §3.1），`InternalMessageService` 经
   `Notifier` 接口反向拿缝（`RegisterNotifier` + 未装配时 loud-fail 的 `unwiredNotifier`）；
3. **定向发送（`recipientUserId` 与 `targetUserIds` 两条分支）改走缝**：`deliverViaNotifier` 落台账、
   缝调 INTERNAL 渠道落收件行并推 SSE。收件箱/未读数仍以 `internal_message_recipients` 为真相源，
   台账只回答"发过没发过、走哪个渠道、成没成"；
4. **全员广播不入台账**（本条是刻意边界，不是遗漏）：广播是"一条消息 × N 个收件人"的批量扇出，
   逐行进台账会让一次点击产出与用户数同阶的行数，而它给出的信息（每个收件人一行 INTERNAL/SENT）
   与收件记录表**完全重复**。台账的价值在于"平台对外发了什么"（SMTP 抖动、渠道没配、走了哪条配置），
   站内投递本来就由内容表自己记账。为此 `NotificationDeliveryRepo.CreateBulk` 写了又删——
   广播不写台账，它就是死代码。**遗留决策点**：若 P2 的用量计量真要求"每个收件人一行"，
   再按批量补写（一次 `INSERT ... SELECT`，不是逐行），并重新评估索引 `(event_type, related_id)` 的选择性。

**运行期实测（本机实例 + 现网 `gwa` 库，不是替身测试）**：

| 观测点 | 结果 |
| --- | --- |
| `POST /admin/v1/internal-message/send`（定向 user 2） | 200，`messageId=9` |
| `sys_notification_deliveries` 新行 | `event_type=INTERNAL_MESSAGE`、`channel=INTERNAL`、`status=SENT`、`target="2"`（未脱敏的用户 ID，见 §3.3 例外）、`related_id=9`、`channel_id` **为空**（站内信没有渠道配置行，与 EMAIL 的"自选 SMTP 必须回填"不同形）、`created_by=1`（操作人） |
| `internal_message_recipients` 新行 | `message_id=9`、`recipient_user_id=2`、`tenant_id=0`（当时取的是**操作人** viewer 租户 —— 该缺陷已由下面的 P2-2 修掉，现在这列是收件人自己的租户）、`status=RECEIVED` |
| 台账读回 | `GET /admin/v1/notification-deliveries?query={"eventType__contains":"INTERNAL_MESSAGE"}` → 200、total=1、枚举与 `relatedId` 如实呈现 |
| 自动迁移 | 重启后 `pg_indexes` 多出 `idx_sys_notification_delivery_event_related`，`related_id` 列可用（无需手写 DDL） |
| 广播 SSE 帧（订阅 `:7789/events?token=…&stream=1`） | `{"id":10,"messageId":11,…}` —— **`id` 非零**，见下 |

顺带修掉的第二个静默失效（同一条链路，只有真订一次 SSE 才看得见）：`executeBroadcast` 的收件行是
批量构造的，落库后没有回读主键，于是载荷里 `id` 恒缺（protojson 省略零值 optional）→
vue-element 的 `if (!data.id || !data.messageId) return` 把**每一条广播通知**丢掉，桌面通知与未读数
不触发也不报错。修法：`IdsByMessageAndRecipients` 按 `(message_id, recipient_user_id)` 批量回读主键再推。
回归测试 `internal_message_notify_seam_sqlite_test.go` 用变异验证过（删掉回填那一行即红）。

**P2-2（收件行的租户打标跟着受众走，已完成 2026-09-20）** —— 落 §6 决策点 4 的第一条出路，三处一起改：

1. **写侧定向路径**：`sendNotification` 的收件行租户改为查收件用户（`recipientTenantID` → `userRepo.Get`），
   不再取操作人 viewer；查不到时回退 viewer 租户并留 error（回退成 0 会把行藏进"平台"这个谁都读不到的地方）。
2. **写侧广播路径**：`executeBroadcast` 逐行取受众 DTO 自带的 `tenant_id`，不再整批取 viewer 租户。
   平台管理员的广播在 SystemViewer 下跑，viewer 租户恒为 0，按它打标等于把全平台的收件行写进租户 0。
   受众没带租户时（`tenant_id=0` 而广播方是租户）按广播方租户兜底并留 error —— 这是 go-crud DTO 映射退化的唯一可察觉窗口。
3. **读侧**：`ListUserInbox` 回填父消息改走 SystemViewer。平台公告的父消息行落在租户 0，而收件行按读者租户过滤，
   用读者的 viewer 读父消息 → 收件箱有行、标题正文为空（推送侧从内存 DTO 取正文，反而是全的，两条路径就此分叉）。
   `messageIds` 全部来自已按读者租户过滤过的收件行，"能读到这条收件行"就是授权凭据，不构成跨租户读取口。

**运行期实测（本机实例重启到新代码 + 现网 `gwa` 库）**：

| 观测点 | 结果 |
| --- | --- |
| admin（平台，租户 0）广播 `target_all` | 改动前的同类广播（message 10/11）两行收件全是 `tenant_id=0`；改动后 message 12：user 2 → **`tenant_id=1`**、user 1（平台用户）→ `tenant_id=0`，逐行随受众 |
| admin → tenant_admin(租户 1) 定向（message 13） | 收件行 `tenant_id=1`，父消息 `tenant_id=0` —— 打标跟的是收件人，不是操作人/viewer |
| admin → 探针租户(租户 2) 用户定向（message 17） | 收件行 `tenant_id=2` |
| **租户用户读自己的收件箱**（`probe_admin` @ 租户 2，`GET /admin/v1/internal-message/inbox`） | total=1、`tenant=2`、**`title`/`content` 回填到位**（父消息在租户 0）—— 这一行是"平台公告租户读不到"缺陷的反面证据 |

租户态 token 是本次现场造出来的：`POST /admin/v1/tenants:with-admin`（租户 2 + 管理员，绑定企业版套餐 3，
否则闸门回 `no subscription plan`）→ 登录取 token → 探针结束后按各资源的 DELETE 路由清掉。
**残留两处没清**：`tenant:manager`（role id=4，租户 2 的模板副本）被"protected role cannot be deleted"挡住，
删租户时也没带走它；`sys_user_credentials` 里探针用户 3/4 的两行在用户删除后仍在且 `deleted_at` 为空 ——
**用户删除不级联凭证**是本次顺手发现的一个既有缺口（不在本次范围内，登录侧因用户已不存在而 fail-closed）。

回归测试：`TestInternalMessageServiceSqlite_PlatformBroadcastIsReadableByTenantUser`（平台广播 → 两个租户的读者各读到自己的行 + 标题正文）、
`TestNotifySeamDirectedSend`（收件行落在收件人租户而非 SystemViewer 的租户 0）。

**环境发现（不是代码缺陷，但会让广播看起来"没发"**）：本机 `backend` 与兄弟项目 `go-wind-quant` 的 asynq
共用同一个 Redis 的 **DB 1 `default` 队列**，谁先抢到谁处理，没有对应 handler 的一方报
`handler not found for task "…"` 进退避重试。实测 `asynq:{default}:retry` 里躺着 `tenant_expiry_scan`（本仓任务类型），
而探针的两次广播任务延迟 58 秒才被处理、后续两次干脆没被本实例处理。上表里的运行期证据因此走的是**同步的定向路径**。

P2 剩余：`notification_rules` 表 + 管理页，替换 §3.5 的 Go 表；`sys_notification_channels` 解掉"WEBHOOK 类型
无处存 URL"的问题（方案见 §6 决策点 2）；`webhook_sender.go` 落地。
**异步投递随本阶段一起引入**：`request_id`（唯一索引，幂等锚）+ `attempts` + `pkg/task/notification_dispatch.go`
（载荷只带 delivery_id 列表，正文由 handler 按事件重新渲染，见 §3.3 不做正文快照的理由）。
在此之前，一次 SMTP 抖动就是一次永久 FAILED，且业务侧（找回密码）会把这句话回给用户。

### P3 偏好与模板

用户通知偏好 / 分类退订 / 静音时段 + 模板管理与渲染。今天这三样全部不存在
（`pkg/constants/default_data.go:964` 只种了 3 条密码策略配置，无通知相关；
提交 `9f10f789` 曾删掉 ele+vben 个人中心一个假的"消息通知" tab，理由正是"无用户通知偏好能力"）。
这是 IM 那套里工作量最大的部分，单独排期。

## 5. 与 go-wind-im 的可抄性对照

| IM 的做法 | 结论 |
| --- | --- |
| `NotificationType` / `Priority` / `Status` / `EventType` 四枚举形状 | **抄形状**。内容重写：IM 的 `EventType` 全是 IM 业务事件（`IM_MESSAGE`、`CRM_OPPORTUNITY_STALLED`、`APP_VERSION_RELEASED`），照抄会带入一堆用不上的常量 |
| `channel_sender.go` 的 `Sender` / `TypedSender` / `SendResult` 接口 | **抄**，去掉 `templateID` 参数（P1 无模板） |
| `internal_message_sender.go` 在 data 层经 gRPC client 回打 | **不抄**，见 §3.1 成环分析 |
| `notification_config.Settings` 的 11 种渠道多态信封（Email/Sms/Telegram/Wechat/APNs/FCM/华为/小米/OPPO/vivo/荣耀） | **P2 起部分采用**。admin 无移动客户端，五种厂商推送整体不适用；SMS 属 P3 |
| 独立微服务 + `kafka_server.go` 消费 `im.event.notification.send` | **不采用**，见 §1「不做什么」 |
| `push_device` 表与设备注册 | **不适用**，admin 无 App 端 |

## 6. 待决策点

1. **渠道配置的租户作用域**。`sys_notification_channels` 平台全局（`name` 全局唯一索引），
   `internal_messages` 租户级。P2 引入规则表时，规则要引用渠道——**规则表会骑在两个隔离域上**。
   倾向：P1/P2 保持平台全局，"租户 A 用哪个 SMTP" 若成为真需求再引入租户覆盖行，不做提前设计。
2. **`sys_notification_channels` 怎么容下非 SMTP 渠道**。今天除 `name`/`type` 外每一列都是 SMTP 形状，
   `WEBHOOK` 行没有地方存 URL 与密钥（且三端 UI 都把 type 列硬编码渲染成「邮件 (SMTP)」标签，
   API 建出的 WEBHOOK 行会显示成 EMAIL）。两个选项：
   - **A（倾向）**：为 WEBHOOK 加可空 `webhook_url` / `webhook_secret` 列。改动最小，但每加一种新渠道都要迁一次表；
   - **B**：加一个 `settings` JSON 列（加密存储），新渠道一律走它。一次解决未来所有类型，但 SMTP 会变成两种真相源，
     且 ent 自动迁移只加列不删列，`smtp_*` 会长期挂着。
3. **`sys_notification_deliveries` 是否挂租户谓词** —— **已定：不挂**（P1 建表时按此落地）。
   台账与渠道花名册同域：一行记的是"平台用哪条 SMTP 发给了某个地址"，收件人未必是租户用户
   （找回密码的标识符可以是任意注册邮箱）。真要按租户看用量，走 `recipient_user_id` 关联用户表即可，
   不需要在写入路径上多一个可能填错的列。若 P2 的用量计量证明需要，再补列 + 回填，比现在就背
   "系统上下文里 viewer 租户为 0 → 全部落在租户 0"的坑便宜。
4. **平台公告是否要让租户用户读到**（§2.4-2 修复后暴露）—— **已定并落地：第一条出路，P2-2（2026-09-20）**。
   平台管理员广播的收件行落 `tenant_id=0`，而收件箱读取被 go-crud 强制过滤为 viewer 租户 → 租户用户读不到平台公告，
   平台侧只能靠 `internal_messages` 列表页看。三条出路：收件行写入时按受众租户扇出（平台上下文 `userRepo.List`
   已覆盖全平台，只需把每行的 tenant 换成**收件用户自己的** `tenant_id`）、收件箱读侧对 `tenant_id=0`
   开口（要改隔离层，风险大）、或明确"平台公告不进站内信、只走站内公告栏"。
   选了第一条：改动局限在站内信的写侧与读侧，不碰隔离层。**落地比原设想多一处**——收件行按受众打标之后，
   父消息（平台公告本体）仍在租户 0，收件箱回填必须换 SystemViewer 才读得到，否则"有行没标题"（见 §4 P2-2 第 3 条）。
   **P2 补充事实（2026-09-20 实测）**：定向路径的收件行 `tenant_id` 取的是**操作人** viewer 的租户
   （改缝前后同形，实测 admin→tenant_admin 一次投递落 `tenant_id=0`），所以"平台公告租户读不到"
   这个缺陷在定向路径上同样存在，不止广播。已按同一规则一起覆盖两个入口。

5. **站内信投递在台账里 `channel_id` 留空是否可接受** —— **P2 已定：可接受**。
   `channel_id` 的语义是"实际选中的 `sys_notification_channels` 行"，站内信压根没有配置行，
   硬塞 0 会让"渠道自选失败"与"该渠道不需要配置"两种事实混成一行。台账里判"走的哪条 SMTP"
   仍按 EMAIL 读这一列，其余渠道读 `channel`。
6. **`related_id` 要不要做成带类型的多态外键** —— **P2 已定：不做**，只有这一列 + 按 `event_type` 解释的约定。
   多态外键的代价（无外键约束、跨表 JOIN 要按类型分支）在"事件种类个位数"的规模下换不来任何东西；
   代价是**前端列必须自带解释**（三端台账页的"关联对象"列 tooltip/注释都写死了"站内信 = 消息ID"）。

## 7. 落地验收清单

后端生成链（按根 `AGENTS.md`，gow 优先）。P1 实测有两处要走不通，记下来免得下次重踩：

```bash
gow api      # ✗ 本机当前会 abort：它先跑 go mod tidy，而未跟踪的 backend/cmd/server/main_ent.go
             #   import 了 kratos v2.9.2 里不存在的 middleware/auth。等价替代：cd api && buf generate
gow ent      # ✓（Windows 下偶发 "user-mapped section open" 中断并留下半截生成树：
             #   确认未跟踪文件只有自己的新 schema 后 git checkout -- .../data/ent/ 重跑即可）
make register ENTITY=notification_delivery   # ✗ 不适用：只覆盖 New<entity>Repo(ctx, entClient) /
                                             #   New<entity>Service(ctx, repo) 的标准形状，
                                             #   本模块 service 还多吃一个 channel.Registry，
                                             #   故 wiring_ent.go / rest_server.go 手写加行
make ts      # 三端 TS（gow 未覆盖，退回 Makefile，在 backend/ 根执行）—— 前端台账页前必做
gow run admin
```

其余必做项：

- [x] `mailer.SendMail` 全仓只命中 `data/channel/email_sender.go` 一处；
- [x] `pkg/constants/default_data.go` 菜单种子：Id 72 / path `notification-deliveries` /
      `component: app/system/notification_delivery/index.vue` / `Authority: ["sys:platform_admin"]`（只读页，无按钮权限码）；
- [x] 三端门禁全绿：react `npm run typecheck` / vue-element `npx vue-tsc --noEmit` / vue-vben `pnpm run check:type`
      （turbo 会 cache-hit，复跑要带 `--force`）；
- [x] 通知渠道三处 UI 的 `type` 列改为按枚举渲染（react / ele / vben 均已接 record，色表按 `CHANNEL_TYPE_*`
      枚举查，未知值回落原文 + 默认色；新增 WEBHOOK 文案 key 三端 zh/en 各一份）。
      首稿记录的三处硬编码位置（react `notification-channel/index.tsx:127`、ele `index.vue:12`、
      vben `index.vue:261`）现已全部改掉；台账页从一开始就是按枚举渲染的，没重复这个错。
      另：react 端路由 meta 的 `permission` 是**开发阶段整体注释掉的**（`router/modules/system.tsx` 每条 system
      路由都如此，ele/vben 则带 `authority`），本页跟随同端惯例不单独启用；菜单可见性另有后端
      `Menu.meta.authority` 这条线。首稿把它记成"react 漏配 authority"是静态阅读误判，此处更正。
- [x] 搜索条件一律 contains、ID 类字段不进模糊搜索（根 `AGENTS.md` 铁律 3）；CRUD 请求体包 `{ data: {...} }`；
      —— 本页只有 `target` 一个文本搜索项，`recipient_user_id` / `channel_id` 不进搜索。
      **实测（本机 `gwa` 库 + 真实 HTTP）**：`query={"status__contains":"SKIPPED"}` 在枚举列上既不过滤器报错、
      也确实筛掉了 FAILED 行；`orderBy=["-created_at"]` 返回倒序。**查询参数名是 `query`，不是 `where`**——
      手写 curl 用 `where=` 会被静默忽略（返回全量、total 不变），首稿那条"不报错"的结论差点被这个假阴性读成"过滤失效"。
- [x] 已部署实例的**新端点须在管理页「接口同步」手动触发全量重建**进 Api 表（根 `AGENTS.md` 铁律）。
      实测：本机 `gwa` 的 `sys_apis` 从 203 → 205 行，两条 `notification-deliveries` 路由（GET 列表 / GET 详情）
      都在重建结果里；以 platform_admin 身份同步前后均 200。
      **坑（比铁律本身更值得记）：`SyncApis` 是 `Truncate` + 从**内嵌资源** `cmd/server/assets/openapi.yaml`
      重建**（`api_service.go:145-162`）。所以「接口同步」只能同步到打进二进制的那份 OpenAPI——
      加了新 proto 却没重跑 `make openapi`，同步会"成功"而新端点照样不在表里，且现象与"没点同步"完全一样。
      本次即如此：asset 落后一个功能，先 `make openapi`（+182 行，纯增量只多 `/admin/v1/notification-deliveries`）
      再重启进程，才轮得到点按钮。**顺序：改 proto → `buf generate` → `make ts` → `make openapi` → 重启 → 接口同步。**
      未实测的一半：租户闸门 fail-closed 403 → 同步后 200。本机只有 platform 侧 `admin`（tenant 0）与
      `tenant_admin`（tenant 1），后者的密码三次尝试均被拒，而重置它的密码属于改用户库里数据、不是本次任务的授权范围，
      故**没有**做出"租户用户点同步前 403 / 点后 200"的对照。这一条留给有租户口令的人补测。
- [x] 已有实例的新菜单：Go 侧种子是 `count==0` 守卫（`menu_service.go:42-46`），**老库不会自动多出这一行**，
      必须走各端管理页的「菜单同步」(SyncMenus, MERGE)。只在新建库上验证过本页菜单的人容易漏掉这一步。
- [x] `docs/sse_architecture.md` 已随 P2 更新（P1 那条"不改"的判断当时是对的：P1 只做了 EMAIL 渠道，
      站内 SSE 的生产方没变）。P2 之后成立的两件事写进了它：`publishNotification` 现在是**被通知域调用**的
      INTERNAL 投递内核（新增站内信一律走 `Notifier`），以及载荷格式即 protojson 驼峰 + `id` 必须非零。

P2 新增事件类型时的落点清单（一枚 `INTERNAL_MESSAGE` 要逐个点到的地方，漏任一处都是静默不一致）：

- 后端：proto 的 `EventType` → ent schema 的 `NamedValues` → `eventChannels` 路由表 → `make openapi` / `make ts`；
- react：`notification-delivery/index.tsx` 的 `eventTypeOptions` + 该模块 `locales/{zh-CN,en-US}` 的 `eventTypeMap`；
- vue-element：composable 的 `EVENT_TYPE_KEYS` + 全局 `locales/{zh-CN,en-US}/enum.json` 的 `notificationDelivery.eventType`；
- vue-vben：composable 的 `notificationDeliveryEventTypeList` + `locales/langs/{zh-CN,en-US}/enum.json`。

两条本次实测踩到的：

- **自定义 RPC 的请求体是扁平的，不包 `{ data: {...} }`**：根 `AGENTS.md` 铁律 3 那条只适用于 CRUD 路由。
  `POST /admin/v1/internal-message/send` 是 `body: "*"` 的自定义 RPC，包了 `data` 会被 protojson 当未知字段丢弃 →
  接口照样 200、消息照样落库（`type`/`status` 来自服务硬编码与列默认值），但**收件人与标题全为空**，
  于是一封"发出去了但没人收到"的幽灵消息进了库。本次实测第一次就踩中，产生了 id=8 这条孤儿行（已用
  `DELETE /admin/v1/internal-message/messages/8` 清掉）。验收"发信有没有生效"不许只看 HTTP 200。
- **台账页的渠道筛选项 = 已注册实现的渠道**（`EMAIL` + `INTERNAL`），SMS/WEBHOOK 只留展示文案：
  路由表还没把它们分给任何事件，放进下拉只会让"筛出空表"被误读成条件写错。三端同形
  （react `channelOptions` / ele `notificationDeliveryChannelFilterList` / vben 同名导出）。

### 运行期实测发现并修掉的两个缺陷（2026-09-19，只有真发一次信才看得见）

静态审读时两处都"看着对"，跑起来才暴露：

1. **自选渠道丢掉了渠道 ID**。`SmtpAccount` 没有 ID 字段，`EmailSender.pickAccount` 的自选分支只能
   `return account, 0, nil`，于是事件路由（找回密码 / 绑定码）发出的信在台账上 `channel_id` 恒空、
   错误文本恒为 `channel [0]`——"这条走的哪个 SMTP 账号"恰恰是这张台账被造出来要回答的问题。
   修法：`SmtpAccount` 带主键（两个 getter 都填），`SendReceipt.ChannelID` 直接取 `account.ID`。
2. **枚举外的 `smtp_tls` 被记成 FAILED**。库里那列是 varchar（PG 上 ent 枚举列并非原生 enum），
   演示数据留了一条 `SSL_TLS`（proto 只有 NONE/START_TLS/SSL），`mailer` 到拨号那一步才报
   `unsupported tls mode` → 归成"投递失败"。但它按定义就是配置问题，从没拨过号。
   修法：`mailer.IsSupportedTlsMode`（与拨号 switch 同一份判定）+ `EmailSender.checkUsable` 在投递前拦，
   包成 `ErrChannelNotConfigured` → SKIPPED。演示 SQL 里那个值同时改成 `SSL`。

修完的运行期对照（同一台实例、同一份库、同一个入口 `POST /admin/v1/forgot-password`）：

| 台账行 | 状态 | `last_error` |
| --- | --- | --- |
| 修复前 id=1 | FAILED | `send mail via channel [0] failed: unsupported tls mode: SSL_TLS` |
| 修复后 id=2 | SKIPPED | `no enabled notification channel configured: channel [1] smtp_tls "SSL_TLS" is not one of NONE/START_TLS/SSL` |

`POST /admin/v1/notification-channels/{id}/send-test-email` 也照同样两条路回话：id=1 →
`... smtp_tls "SSL_TLS" is not one of ...`，id=3（停用）→ `channel [3] is disabled`；配置页拿得到原始原因，
而找回密码对最终用户只回一句 `email channel is not configured`（细节不该泄给未鉴权调用方）。
回归测试：`data/channel/email_sender_sqlite_test.go`（自选带回真实 ID / 无可用渠道归 SKIPPED / 枚举外 tls 归不可用）、
`pkg/mailer` 的 `TestIsSupportedTlsMode`、repo sqlite 测试补 `SmtpAccount.ID` 断言。
`EmailSender` 与 mailer 之间"成功投递补 channel_id"这一段仍只有服务层替身测试覆盖——本机没有可控的 SMTP 服务端，
真发一封才算闭环。

### 移植记录（react → ele → vben，2026-09-19）

同一张台账在三端落地时被迫换形的地方，记下来给 P2 的规则/模板页省时间：

| 差异 | 说明 |
| --- | --- |
| 枚举文案的 i18n 落点 | react 用页面自己的命名空间嵌套 map（`statusMap.*`）；ele / vben 按各自约定放进全局 `enum.json`（`enum.notificationDelivery.status.*`）。移植时**不能**照搬 key 路径 |
| 列头 tooltip | 三端只有 react 的 ProTable 能挂列头 tooltip；ele 挪进搜索项 `tips`，vben 挪进 VbenForm schema 的 `help` |
| 服务端排序 | ele 的 `ProPage` 不绑 vxe 的 `sort-change`（全仓没有任何页面用列排序），本页固定 `-created_at`；vben 需同时给 `sortable` 与 `sortConfig.remote: true`（vxe 默认 remote=false，否则只对当页排序） |
| orderBy 字段名 | 必须是**数据库列名**。react 侧 ProTable 的 sorter key 是 camelCase 的 dataIndex，本页显式蛇形化；vben 侧同理用局部映射表 |
| Tag 色系 | Element Plus 没有 purple/cyan/processing 这一档，ele 把 SENDING/EMAIL 落 primary、WEBHOOK 落 info；antd 系（react / vben）保留 blue/green/purple/cyan |
| 导出 | ele 端每个只读页都带 `createPagedExportAction`，react 台账页没有导出——移植时按目标端惯例补齐，不算行为差异 |

实测过、写下来免得下次再猜的两条（sqlite 内存库 + 现网 `gwa` 库）：

- **枚举列可以直接 `__contains` 过滤**：`query={"status__contains":"SKIPPED"}` 打到真实 HTTP 端点，
  既不被过滤器拒绝、也确实把 FAILED 行筛掉了（首稿只验证到"不报错"，那半边是假阴性风险——见 §7 那条实测）；
  且现网 `gwa` 库里 ent 枚举列（`internal_messages.status` / `.type` 等）实测是 `character varying` 而非 PG 原生 enum，
  所以"LIKE 作用在 enum 上会找不到操作符"这个担心在本仓不成立。
- **orderBy 传非法列不报错**：go-crud 的标识符正则只挡 SQL 元字符，列白名单 `CheckColumn` 只登记了它自带的
  `user` / `menu` 两张表，对本仓所有表 fail-open → 拼错的列名静默失效，不会 500。列表页排序"点了没反应"时先查列名。

## 8. 相关文档

- `sse_architecture.md` — 站内推送传输层，本域的 `INTERNAL_MESSAGE` 渠道依赖它
- `task_system.md` — asynq 投递任务的调度语义与常驻任务口径
- `tenant_isolation.md` / `plan_billing.md` — 决策点 1、3 的约束来源
- `list_query_rule.md` — 台账列表的查询协议
- `script_system.md` — `InternalMessage` 与 `NotificationChannel` 两个实体**已经注册了**
  `after_create` 等钩子点（`service/script_entity_hooks.go:34-35`），这是不改 Go 代码就能挂通知规则的
  现成扩展点，P2 设计规则引擎时应一并评估（DB 规则表 vs 脚本钩子的取舍）。
