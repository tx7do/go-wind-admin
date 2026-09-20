# 通知域（Notification Domain）设计文档

> **状态：P0 + P1 已落地（后端内聚 + 三端投递台账页 + 邮件文案 i18n + SSE 事件类型注册表，2026-09-19），
> P2 已落地五块（见 §4：P2-1 站内信注册为 INTERNAL 渠道 + 定向发送改走缝 + 台账 `related_id`；
> P2-2 收件行租户打标跟着受众走 + 修掉收件箱一处越权读；P2-3 异步投递 = 入队前同步预检 + asynq 派发 +
> 台账 `request_id`/`attempts`；P2-4 台账 `SENDING` 超时清扫 = 系统级常驻 cron；P2-5 台账结论回写失败按出口上抛，
> 均 2026-09-20）；P2 剩余（规则表 / WEBHOOK 渠道）与 P3 仍是设计提案，
> 另有一处已定位、尚未修的写侧姊妹缺陷记在 §7「收件箱读侧不钉归属」一节末尾。**
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
   ├─ sys_notification_deliveries 台账：先落 SENDING（带 request_id 幂等锚），再发
   └─ 两条出口，按事件类型分流（asyncDispatchEvents，见 §4 P2-3）：
        ├─ 异步：sender.(Prechecker).Precheck() 通过 → asynq 入队 → 立刻回 SENDING
        │         （预检不过 → 当场结台账 SKIPPED/FAILED，回话与同步路径逐字相同）
        └─ 同步：deliver() = sendOnce() + 回写结论（渠道测试邮件、站内信、入队失败兜底）
                │  两条出口共用 sendOnce 这一处分类，结论语义不会分叉
   ┌────────────┴──────────────────┐
   ▼                               ▼
EmailSender                  InternalMessageSender（P2 已落地，见下）
internal/data/channel/        internal/service/（成环，位置不可照抄 IM）
（同时实现 Sender 与 Prechecker）
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
| `status` | `SENDING` / `SENT` / `FAILED` / `SKIPPED`（`SENDING` 不是终态、也不许是永久态：超期未结算由 P2-4 的常驻清扫定案成 `FAILED` 并写明 `swept by …` 原因） | 已建 |
| `last_error` / `sent_at` | 结果与完成时间（FAILED/SKIPPED 不留 `sent_at`） | 已建 |
| `related_id` | 可空，**按 `event_type` 解释**的业务对象 ID（`INTERNAL_MESSAGE` → `internal_messages.id`）。台账不存正文快照，这一列是"这条投递发的是什么"的唯一回跳入口 | P2 已建（`Optional().Nillable()` + 索引 `(event_type, related_id)`，启动期 ent 自动迁移加列加索引，实测已在 `gwa` 库出现） |
| `event_type/channel/status` 三枚枚举 | 必须 `Optional().Nillable()`：值型枚举列 + 可选指针 DTO 会踩 copier 的"指针↔指点对"失配，读回恒为零值（同款坑见 `notification_channel_repo.go` 的 `queryTypeByIDs` 注释） | 已按此建 |
| `request_id` | 幂等锚：**一次业务调用 → 若干条投递**的唯一串线抓手。调用方不传则服务端生成（GUIDv4 无连字符，32 字符）。唯一索引是 `(request_id, channel)` 复合，不是一列唯一 —— 一次调用同时发邮件 + 站内信是这条台账的**预期用法**，单列唯一会把这种调用变成插入冲突 | P2-3 已建（`Optional().Nillable()` + `uidx_sys_notification_delivery_request_channel`，实测已在 `gwa` 库出现；三个生产点今天都不传，全靠服务端生成） |
| `attempts` | 已尝试投递的次数。异步路径由 handler 每次开拨前 `MarkAttempted` +1，配合 `status=SENDING` 就是"还在重试"；同步路径把 `attempts=1` 写在结论那一条更新里，"当场投一次就是一次"。**P2-4 之后这条判据有了时限**：超过清扫阈值仍挂在 `SENDING` 的行不是"还在重试"，而是"没人结算"（阈值下限就是按重试预算算出来的，见 §6 决策点 8）。**P2-5 之后 `attempts=0` 有了第二种成因**：结论回写失败时异步侧照样在涨（先记账再拨号），同步侧那一格却是 0 —— 信可能真的发出去了，只是这一列和结论写在同一条被挡下的更新里（实测见 §4 P2-5 表倒数第二行；不改成的理由见 §4 P2-5 第 4 条） | P2-3 已建（`Uint32().Default(0)`；预检失败/配置类 SKIPPED 的行留 0 = 一次都没真试过） |
| `title` / `content` | 正文快照 | **不做**：一期三个事件里两个的正文就是验证码本身，快照进永久台账等于建了一张 OTP 明文表。P2 引入模板后按"模板 ID + 渲染参数"存，不存渲染结果 |
| mixin | `AutoIncrementId` / `TimeAt` / `OperatorID` | 已按此建；**不挂 `TenantID`**（见 §6 决策点 3 结论），也不挂 `SwitchStatus`（台账没有"停用"语义） |

台账**不提供 Update/Delete 业务方法**：改一条已发生的投递等于伪造事实；只有 `MarkResult`
按主键回写结果字段。P2-4 多加的 `SweepStaleSending` 不算例外 —— 它只把 `SENDING` 定案成 `FAILED`
（带 `status=SENDING` 谓词的批量 UPDATE），从不改写任何已有结论，也不碰 `attempts`。

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
--- 以下为 P2-3 异步投递新增 ---
pkg/task/notification_dispatch.go                              任务类型常量 + 载荷结构（delivery_id + 正文三件）
app/admin/service/internal/data/channel/sender.go              新增 Prechecker 接口（配置的可用性自检，不拨号）
app/admin/service/internal/server/asynq_server.go              订阅 notification_dispatch + 把 TaskService 注成 TaskEnqueuer
--- 以下为 P2-4 台账 SENDING 清扫新增/改动（无契约变更：proto / ent schema 一列未动）---
pkg/task/notification_delivery_sweep.go                        清扫任务类型 + cron 常量（载荷为空）
app/admin/service/internal/data/notification_delivery_repo.go  + SweepStaleSending：批量 + `status=SENDING` 谓词的定案
app/admin/service/internal/service/notification_service.go     + AsyncDeliverySweep handler + deliveryStaleAfter 阈值解析
app/admin/service/internal/server/asynq_server.go              + 订阅 notification_delivery_sweep
app/admin/service/internal/service/task_service.go             + startAllTask 尾部重注册这条 cron（见 §4 P2-4 第 1 条）
--- 以下为 P2-5 结论回写上抛（同样零契约变更：只动 service 层的错误去处）---
app/admin/service/internal/service/notification_service.go     markResult 改返回 error + 八处调用点按出口定去处
app/admin/service/internal/service/notification_record_sqlite_test.go  回写失败注入的五条回归（一次性 ent hook）
--- 以下为 P2 剩余项，尚未建 ---
app/admin/service/internal/data/channel/webhook_sender.go      等 §6 决策点 2 定了表结构再写
```

两处刻意偏离首稿：

1. **BFF 不开放"发一条通知"的 HTTP 路由**。`SendDirect` 只由进程内业务 service 经 `Notifier` 调用；
   开成端点等于给任意已登录操作员一个"向任意邮箱发信"的入口，而唯一的站外手动触发口
   （渠道测试邮件）已在 `notification-channels` 路由上存在。
2. **P1 没有 asynq 投递任务**（P2-3 已引入，见 §4）：`SendTestEmail` 的产物就是"SMTP 报错原文"，必须同步返回；
   验证码邮件同样要立刻知道"渠道没配"以便回不同文案。首稿为异步形态写的"载荷只带 delivery_id，正文由
   handler 按事件重新渲染"那条**没有采纳**，原因记在 §6 决策点 7：重渲染要 handler 重新拿一遍调用方的
   输入（验证码值在调用那一刻就只存在于内存），找回密码的码根本渲染不出来。

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

同一次改动顺带钉掉一个越权读：收件箱 `List` 的查询条件整个来自调用方的 `query` 字符串，服务端此前不注入
`recipient_user_id`，同租户任意登录用户改一个 ID 就能翻别人的收件记录（标题正文随父消息一并跟走）。
现在 `InternalMessageRecipientRepo.List` 在非平台/非系统上下文下强制 `recipient_user_id = viewer.UserID()`，
平台侧仍可按用户筛（用户详情页要看指定用户的收件箱）。详见 §7 的"第三个缺陷"。

**运行期实测（本机实例重启到新代码 + 现网 `gwa` 库）**：

| 观测点 | 结果 |
| --- | --- |
| admin（平台，租户 0）广播 `target_all` | 改动前的同类广播（message 10/11）两行收件全是 `tenant_id=0`；改动后 message 12：user 2 → **`tenant_id=1`**、user 1（平台用户）→ `tenant_id=0`，逐行随受众 |
| admin → tenant_admin(租户 1) 定向（message 13） | 收件行 `tenant_id=1`，父消息 `tenant_id=0` —— 打标跟的是收件人，不是操作人/viewer |
| admin → 探针租户(租户 2) 用户定向（message 17） | 收件行 `tenant_id=2` |
| **租户用户读自己的收件箱**（`probe_admin` @ 租户 2，`GET /admin/v1/internal-message/inbox`） | total=1、`tenant=2`、**`title`/`content` 回填到位**（父消息在租户 0）—— 这一行是"平台公告租户读不到"缺陷的反面证据 |
| 同一 token 发 `?query={"recipientUserId__eq":1}` | total=0（改动前：读到别人的收件行含标题正文） |
| admin（平台）发同一条件按用户筛 | total=1、`tenant=2` —— 平台豁免仍成立，用户详情页不受影响 |

租户态 token 是本次现场造出来的：`POST /admin/v1/tenants:with-admin`（租户 2 + 管理员，绑定企业版套餐 3，
否则闸门回 `no subscription plan`）→ 登录取 token → 探针结束后按各资源的 DELETE 路由清掉。
**残留两处没清**：`tenant:manager`（role id=4，租户 2 的模板副本）被"protected role cannot be deleted"挡住，
删租户时也没带走它；`sys_user_credentials` 里探针用户 3/4 的两行在用户删除后仍在且 `deleted_at` 为空 ——
**用户删除不级联凭证**是本次顺手发现的一个既有缺口（不在本次范围内，登录侧因用户已不存在而 fail-closed）。

回归测试：`TestInternalMessageServiceSqlite_PlatformBroadcastIsReadableByTenantUser`（平台广播 → 两个租户的读者各读到自己的行 + 标题正文）、
`TestNotifySeamDirectedSend`（收件行落在收件人租户而非 SystemViewer 的租户 0）、
`TestInternalMessageRecipientTenantSqlite` 的 `listAs(t, uid)`（同租户换一个收件人就读不到）。

**环境发现（不是代码缺陷，但会让广播看起来"没发"**）：本机 `backend` 与同机其他项目的 asynq 共用同一个
Redis DB 的**同名队列**，谁先抢到谁处理，没有对应 handler 的一方报 `handler not found for task "…"`
进退避重试。实测 `asynq:{default}:retry` 里躺着 `tenant_expiry_scan`（本仓任务类型），而探针的两次广播任务
延迟 58 秒才被处理、后续两次干脆没被本实例处理。上表里的运行期证据因此走的是**同步的定向路径**。
队列键形如 `asynq:{<queue>}:…`、不带应用前缀这件事的完整后果与本次的处置，见下面 P2-3 末尾那条。

**P2-3（异步投递：同步预检 + asynq 派发，已完成 2026-09-20）** —— 落 §3.1 图里那条异步出口，
把"一次 SMTP 抖动 = 一次永久 FAILED，且这句话还被回给最终用户"断掉：

1. **哪些事件异步，判据是"调用方需不需要这次投递的结论"**（`asyncDispatchEvents` 白名单）：
   `PASSWORD_RESET_CODE`、`CONTACT_BIND_CODE` 异步；`CHANNEL_TEST_EMAIL`（产物就是 SMTP 报错原文）、
   `INTERNAL_MESSAGE`（收件行 + SSE 本身就是投递，异步化只会让收件箱晚一点亮）保持同步。
2. **入队前先做一次同步预检**（新接口 `channel.Prechecker`，实现是 `EmailSender.Precheck` → `pickAccount`，
   只解析并自检配置、不拨号）：配置不可用就当场结台账 SKIPPED/FAILED 并把原始 error 回给调用方 ——
   找回密码那句 `email channel is not configured` 的回话因此与异步化之前逐字相同，不会退化成
   "永远 200 + 用户等一封不会来的信"。没有 `Prechecker` 实现的 sender（站内信）直接入队。
3. **载荷带正文**（`pkg/task.NotificationDispatchTaskData{DeliveryId, Target, Title, Content}`）：
   台账既不存正文、`target` 又是脱敏串，handler 无法从库里读回可投递的三元组（取舍见 §6 决策点 7）。
4. **幂等门是台账状态，不是 asynq 的 task id**：`AsyncNotificationDispatch` 读回该行，`status != SENDING` 即
   返回 nil。由此推出重试的写法——**还有额度的失败必须留在 SENDING**（只写 `last_error` + `attempts`），
   否则第一次瞬态错误落了终态就把重试额度静默作废了。`asynq.MaxRetry(3)` ⇒ 最多 4 次尝试
   （`notificationDispatchMaxAttempts`），第 4 次才落 FAILED；配置类错误（`ErrChannelNotConfigured`）与
   未注册渠道一律 `asynq.SkipRetry`，不让一个 OTP 载荷在归档队列里过夜。
5. **入队失败不等于丢通知**：`NewTask` 报错时退回当场投递（结论照旧同步给出），只多一条 error 日志；
   `TaskEnqueuer` 未注入（没配 asynq 的部署）同理整体退回同步路径 —— 异步是加速，不是新依赖。

**运行期实测（本机实例重启到新代码 + 现网 `gwa` 库 + mailpit；探针造成的变更与复旧列在最后）**：

| 观测点 | 结果 |
| --- | --- |
| 自动迁移 | 重启后 `request_id`(varchar, nullable) + `attempts`(bigint default 0) + `uidx_sys_notification_delivery_request_channel` 三项在 `information_schema` / `pg_indexes` 里出现，无需手写 DDL |
| **预检失败路径**（自选命中演示渠道 1 的 `SSL_TLS` 坏配置） | `POST /admin/v1/forgot-password` → **71ms** 内 500 `email channel is not configured`（与异步化之前逐字相同）；台账 id=9 **当场即 `SKIPPED`**、`attempts=0`、`last_error` 带原始原因；该 Redis DB 里 `pending`/`active`/`retry`/`archived`/`processed` 一个键都不存在 ⇒ **一条任务都没入队** |
| **异步成功路径**（自选命中指向 mailpit 的探针渠道 id=7） | `forgot-password` → **200 / 61ms**，请求全程没有拨号；台账 id=11 `SENT`、`channel_id=7`、`attempts=1`、`request_id=81b0607a…`、`target=t***@company.com`（脱敏）；mailpit 里唯一一封 `subject "GoWind Admin 密码重置验证码"`，正文验证码 **455900** 与 Redis `gowind:vcode:reset_password:tenant@company.com` 逐字一致 |
| **重试额度真的走完**（探针渠道 id=6 = `127.0.0.1:1099`，无监听） | `forgot-password` → 200 / 64ms，台账 id=10 先 `SENDING/attempts=1`（`asynq:{default}:retry` 的 score 显示退避 +2s），+25s 读到 `SENDING/2`，终态 **`FAILED/attempts=4`**（`asynq` 侧单次运行计数亦为 4）；额度用尽后任务落入 `asynq:{default}:archived`=1、`retry` 清空。中间每一次只写 `last_error` + `attempts` 而**不改状态** —— 这正是幂等门放行重试的原因 |
| **同步事件不入队** | `POST /notification-channels/7/send-test-email` → 200 用时 **1246ms**（SMTP 握手发生在请求内），台账 id=12 `CHANNEL_TEST_EMAIL` / `SENT` / `attempts=1`；调用前后队列键数不变（仍只有上面那条归档任务）。同一台机器上"当场投"与"入队投"的响应时间差就是 1246ms vs 61ms |
| `attempts` 的两义 | `0` = 一次都没真投过（预检拦截 / 配置类 SKIPPED），`≥1` = 拨过号 |

**队列没有命名空间（部署边界，不是代码缺陷）**：asynq 的键形如 `asynq:{<queue>}:…`，**不带任何应用前缀**，
所以"同一个 Redis DB + 同一个队列名"就是同一个队列。DB 选择只有 `server.asynq.uri` 一处，队列名相同
时两个项目无法区分归属 —— 对方 worker 抢走本仓任务就 `handler not found` 反复重试到归档，反过来一样。
本机实测：DB 1 的存活消费者除本实例（`asynq:servers:{MSI:<pid>:…}` 心跳）外还有
`D:\GoProject\loft\backend\bin\task.exe`，其 `app/task/service/configs/server.yaml` 同样是 DB 1 +
`critical:10 / default:5 / low:1`（`asynq:{default}:t:order_timeout:…` 就是它家的任务）。
上表的异步证据因此是**临时把本机实例的 asynq 指到一个空闲 DB** 取得的，测完已复旧：
`configs/server.yaml` 无 diff、该 DB 的 11 个 asynq 键逐个 `DEL` 清空（该 Redis 实例禁用了 `FLUSHDB`）。

探针造成的变更（全部如实记账）：新建渠道 `b4-deadport`(id=6) / `b4-mailpit`(id=7)，测完都经
`DELETE /notification-channels/{id}` 删除；停用又启用渠道 1、2；台账新增 id=9…12 四行**保留**作上表证据
（id=10、id=11 的 `channel_id` 因此是悬空引用 —— 该列无外键，删渠道才过得去，读它时按此理解）；
mailpit 容器随测随删。**没有做运行期验证的一段**：归档后手工 `RetryTask` 重跑会被台账幂等门挡成 no-op，
sqlite 测试 `AsyncDispatchHandlerSettlesLedger` 覆盖了这道门，但本机没装 asynqmon，这一键没真点过。

回归测试：`notification_dispatch_sqlite_test.go` 九条（入队不投递、handler 结台账 + 幂等门、重试额度到 4 落 FAILED
且额度用尽后不再拨号、配置类错误 `SkipRetry`、预检失败不入队、入队失败退回当场投、同步事件不入队、
`request_id` 调用方优先 + `(request_id, channel)` 冲突、未知 delivery 交给重试）。

**P2-4（台账 `SENDING` 超时清扫，已完成 2026-09-20）** —— 补上台账生命周期缺的最后一步。此前四个 `SENDING`
写入点全是「开始」、没有一处「收尸」：进程死在拨号中途、Redis 里那条任务随进程一起没了，或者部署方压根没配
asynq 消费者 —— 三种情况都会留下一行永远 `SENDING` 的台账，而台账页对"为什么还在发"这个问题给不出任何答案。

1. **落点形状 = 系统级常驻任务**（这条口径的权威说明在 `docs/task_system.md` §5.6）：类型与 cron 常量在
   `pkg/task/notification_delivery_sweep.go`（`notification_delivery_sweep` / `*/5 * * * *`），handler 是
   `NotificationService.AsyncDeliverySweep`，订阅在 `NewAsynqServer`，cron 重注册在 `TaskService.startAllTask`
   末尾 —— 因此它不进 `sys_tasks`，任务管理页看不见也停不掉，跑在 SystemViewer 上下文下。
2. **年龄锚点用 `created_at`，不是 `updated_at`**：本仓没有任何一处会写 `updated_at`（mixin 列
   `Optional().Nillable()` 且无默认值），下表里被扫过的那两行至今 `updated_at` 为空 —— 这就是证据。
   顺带复用已有的 `(status, created_at)` 索引。代价是"一行被合法地反复推进"这种场景扫不出来，目前没有这种场景。
3. **阈值缺省 15 分钟、下限 5 分钟**（`NOTIFICATION_DELIVERY_STALE_MINUTES` 覆盖；坏值按缺省并 WARN，不静默采纳）：
   下限由异步派发的重试预算推出 —— 算术上界 ≈ 221s（4 次尝试 × `asynq.Timeout(30s)` + 退避 2s/17s/82s；
   实测的快失败路径 ~101s，见 §6 决策点 8）。低于上界的阈值会把还在正常重试的投递扫死。
   15 分钟则给"SMTP 恢复得慢一点"留余量。
4. **写法是 DB 侧带谓词的批量更新**：先按 `status=SENDING AND created_at < cutoff` 扫出至多 500 个 id，
   再一条 `UPDATE … WHERE id IN (…) AND status=SENDING` 落 `FAILED` + `last_error`，`attempts` 一律不动。
   重查状态而不是逐行 `MarkResult`，是为了不跟仍在跑的 handler 抢同一行 —— 结论已经写回来的行必须原样留下。
   满一批就再来一批，直到某轮不足一批。
5. **代价要说清楚：这是「清扫」不是「补投」**。幂等门读的就是台账状态（P2-3 第 4 条），一行被扫成 `FAILED`
   之后晚到的 handler 读到 `status != SENDING` 直接返回 nil —— 那封信**永远不会再发出去**。所以阈值宁松勿紧，
   且 reason 写死 `swept by <task_type>` 前缀，排障时一眼分得开"投递失败"与"没人结算"。
   这条代价由专门的回归测试 `SweptRowIsNeverDispatched` 钉住，不是注释里的口头承诺。取舍见 §6 决策点 8。

**运行期实测（本机实例 + 现网 `gwa` 库 + asynq 独占空闲 DB 12 + `NOTIFICATION_DELIVERY_STALE_MINUTES=5`；
探针账目列在最后）**：

| 观测点 | 结果 |
| --- | --- |
| 注册 | 13:33:19 启动日志 `通知台账清扫定时任务已注册（cron=*/5 * * * *）` + asynq DEBUG `registered an entry`（类型 `notification_delivery_sweep`）；第一个刻度 13:35:00 即开工 |
| **超期行落定**（直插 `SENDING`，`created_at = now() - interval '2 hours'`、`attempts=2`、`request_id=sweep-live-1`） | 13:35:00.172 WARN `AsyncDeliverySweep: 1 stale deliveries settled as FAILED (staleAfter=5m0s)`；台账 id=13 现为 **`FAILED`**、`attempts=2` **保留**、`sent_at` 为空、`updated_at` 为空、`last_error = "swept by notification_delivery_sweep: still SENDING 5m0s after creation, no delivery conclusion written back"` |
| **未超期行不误伤**（直插 `SENDING`，`created_at = now()` = 13:36:02、`attempts=1`、`request_id=sweep-live-2`） | 13:40:00 刻度时年龄 3m58s < 5m ⇒ 13:41:52 读回**仍是 `SENDING`**；该轮**没有** WARN（扫到 0 行本就不记日志） |
| **到点即扫** | 13:45:00.220 同一条 WARN，id=14 → `FAILED`、`attempts=1` 保留、`sent_at` 与 `updated_at` 均为空 |
| 前端零改动 | 本次没动 proto ⇒ 无 `make ts` / `make openapi` / 接口同步；清扫原因经 `last_error` 出镜，而该列三端台账页本来就在渲染（react `pages/app/system/notification-delivery/index.tsx`、ele 与 vben 的 `notification_delivery/index.vue`） |

**顺手挖出一条本仓通用的时间坑（不止清扫会踩）**：`SweepStaleSending` 的第一版在 sqlite 回归测试里把"一分钟前
才落的行"扫掉了。根因是两侧时间呈现方式不同 —— `created_at` 经 `timestamppb` 往返后按 UTC 渲染，而谓词参数
`time.Now()` 带 `+0800 CST`；modernc.org/sqlite 把 `time.Time` 存成 `Time.String()` 文本，于是这个比较是
**字典序**的，一行 1 分钟前的行读起来像 8 小时前的。Postgres 的 `timestamptz` 不受影响（实测 `gwa` 库该列类型
为 `timestamp with time zone`、会话 `TimeZone = Etc/UTC`），所以它只在测试里炸、不在现网炸。修法是在比较侧统一
`.UTC()`（repo 里一行 + 注释），完整的边界表记在 `docs/task_system.md` §10。

探针造成的变更（全部如实记账）：台账新增 id=13、14 两行**保留**作上表证据（与 id=9…12 同一处理，两行的
`request_id` 就是 `sweep-live-1/2`，直插 SQL 造的，不经业务链路）；`configs/server.yaml` 的 `server.asynq.uri`
临时从 DB 1 改指 DB 12（理由同 P2-3 那条「队列没有命名空间」），测完已复旧、该 DB 的 asynq 键逐个 `DEL` 清空。

回归测试：`notification_sweep_sqlite_test.go` 六条（只扫超期 SENDING / 不覆盖并发写回的结论 / 被扫行不再投递 /
批量逐轮推进 / 阈值解析含坏值与下限 / handler 崩溃后端到端闭环），另把 `TestTaskService_RestartAllTask` 的常驻
注册计数从 3 项改 4 项并加断言。

**P2-5（台账结论回写失败上抛，已完成 2026-09-20）** —— 把 `markResult` 从"只记日志"改成"把事实交给还能补救它的那一方"。
此前八处回写点（四种出口形状、三种去处）一律丢弃错误，于是两件坏事同时成立：进程内的调用方以为一切正常（同步侧本来就只能这样，见下表），
而**异步侧连 asynq 的重投机会都被丢掉** —— handler 返回 nil 等于告诉队列"这条做完了"，一行本该重试的回写失败
从此只活在日志里，最后由清扫写成 `FAILED`（账面上是"没送到"，实际上信早就出去了）。

1. **策略按出口分，不是一条"上抛"了之**（口径全在 `notification_service.go` 的逐条注释里）：

| 出口 | 回写失败去处 | 为什么是这条 |
| --- | --- | --- |
| 同步 `deliver`：投递成功 | **不回错**给业务调用方，返回值照旧 `SENT` | 找回密码的调用方看到 error 就再生成一封验证码 ⇒ 拿"用户收到两封信"去换一个"缺格的账"，是更贵的交换 |
| 同步 `deliver`：投递失败 | `errors.Join(sendErr, recordErr)` | 这条调用本来就在报错，多一个事实不改变任何行为 |
| 同步 `SendDirect`：渠道未注册（代码 bug） | `errors.Join(errors.New(reason), recordErr)` | 同上 |
| 异步 `dispatchAsync`：入队前预检失败 | `errors.Join(err, recordErr)` | 预检失败的回话本来就要给业务侧（"渠道没配"与"已发送"是不同文案） |
| 异步 handler：投递成功 | `return s.markResult(...)` ⇒ 交给 asynq 重投 | 唯一的"上抛即补救"出口：结论还能落地，比留给清扫诚实 |
| 异步 handler：失败且还有额度 | `errors.Join(sendErr, recordErr)` | 这一格写的是"上一次为什么失败"，丢了不致命（下一次带着 `sendErr` 再来） |
| 异步 handler：额度用尽 | `errors.Join(sendErr, markResult(FAILED))` | 报错给队列 ⇒ 同时进归档 |
| 异步 handler：两支 `SkipRetry`（渠道未注册 / 配置类 SKIPPED） | `errors.Join(带 SkipRetry 的错, recordErr)` | `errors.Is` 穿得过 `Join` ⇒ SkipRetry 仍然优先：从没拨过号的行不值得为一次没落库的结论，把明文验证码在队列里多留一轮 |

2. **刻意不做进程内重试**：异步侧的重试机制就是 asynq，再造一个只会把"重投递"与"重写台账"混进同一个计数器；
   同步侧无论重试几次都不改变"不能回错给调用方"这个结论。
3. **`errors.Join` 与 `asynq.SkipRetry` 的兼容性是这张表成立的前提**（`errors.Is` 会遍历 Join 的每个分支），
   所以它由回归测试钉住而不是靠注释承诺。
4. **同步侧 `attempts` 的不对称由本次实测暴露**（下表倒数第二行）：异步在拨号前先 `MarkAttempted`，
   所以回写失败时 `attempts` 照样涨；同步把 `attempts=1` 写在同一条结论更新里，回写失败 ⇒ 这一格停在 **0**，
   哪怕信真的发出去了。**没有**改成"同步也先记账"：那会破坏 §3.3 的 `SKIPPED ⇒ attempts=0` 判据
   （"没拨过号"与"拨了没记账"就分不开了）。读台账时的口径：`attempts=0` 配 `FAILED` 有两种成因。

**运行期实测（本机实例 + 现网 `gwa` 库 + mailpit 假 SMTP + asynq 独占 DB 12 + 一次性 Postgres 触发器 +
`NOTIFICATION_DELIVERY_STALE_MINUTES=5`；探针账目列在最后）**：

| 观测点 | 结果 |
| --- | --- |
| 故障注入方式 | `BEFORE UPDATE` 触发器：`NEW.id >= 15 AND NEW.status IN ('SENT','FAILED')` ⇒ `RAISE EXCEPTION 'gwa probe: …'`。只咬结论回写，放过 `Create` 与 `MarkAttempted`（后者只写 `attempts`），所以看到的错误与"生产库写不进去"同形（`code = 500 … mark notification delivery result failed`） |
| 自选渠道命中探针 | 演示通道 1/2 临时 `OFF`，新建探针通道 8 = `127.0.0.1:1025` / `smtp_tls=NONE` / 用户名留空（空用户名即不认证，`pkg/mailer/smtp.go:93`），密码列留 NULL 以绕开解密路径 |
| 异步：结论写不进 | 14:38:10.867 台账 id=15 落 `SENDING`；14:38:13.164 出现新日志 `write back delivery [15] result [SENT] failed, the row keeps its previous status (notification_delivery_sweep will settle it): … mark notification delivery result failed` |
| **异步：asynq 因此重投（P2-5 的新行为）** | 同一行该 ERROR 共 4 条：14:38:13.164 / 14:38:49.429 / 14:39:34.320 / 14:41:09.720 ⇒ `attempts` 1→4，退避间隔 36.3s / 44.9s / 95.4s，全程 178.9s |
| **代价：收件人真收到 4 封** | mailpit 4 封 `tenant@company.com`，正文验证码是**同一个 `170300`** ⇒ 重投是"重复投递同一内容"，不是重新生成一个码（载荷带正文的直接后果，§6 决策点 7） |
| 归档里留下了什么 | `asynq:{default}:archived` 1 条，其 `msg` 字段明文含 `"target":"tenant@company.com"` 与 `验证码是：170300`，`error` 字段正是那条回写失败 ⇒ 回写失败把它推进归档，归档顺手把明文验证码留在 Redis（测完连 DB 12 一起清） |
| 同步：不回错给业务侧 | 重挂触发器（`id >= 16`）后 `POST /admin/v1/notification-channels/8/send-test-email` ⇒ **HTTP 200 `{}`**，同时 14:46:58.290 落了同一条 ERROR 日志，mailpit 真收到 `sync-probe@company.com` 一封 |
| 同步：`attempts` 停在 0 | id=16 定案前为 `SENDING` / `attempts=0` / `sent_at` 空，而 `channel_id=8` 有值（显式指定渠道时它属于投递意图，`Create` 阶段就写了）⇒ 台账写着"一次都没试"，账外信已送达，这就是第 4 点那条不对称 |
| 清扫兜底（顺带被同一故障咬住） | 14:45:00.565 触发器还挂着时，清扫任务自己报错 `pq: gwa probe: ledger conclusion write blocked` → `AsyncDeliverySweep: sweep failed` → asynq 重试该任务；14:45:19 摘掉触发器后 14:45:24.873 WARN `1 stale deliveries settled as FAILED (staleAfter=5m0s)` ⇒ id=15 定案 `FAILED`、`attempts=4` **原样保留**、`last_error = swept by …`、`sent_at` 仍空 |

探针造成的变更（全部如实记账）：`sys_notification_deliveries` 上的触发器与函数 `gwa_probe_break_ledger_result`
建两次、每次测完即 `DROP`（库内 `pg_trigger` 复查为空）；通道 1/2 `ON→OFF→ON` 已复原，探针通道 8 建后已删
（表回到 1/2/3 三条、状态 `ON/ON/OFF`，与动手前逐列一致）；台账 id=15、16 两行**保留**作上表证据（与 id=9…14
同一处理，两行现在都已是终态：id=15 于 14:45:24.873、id=16 于 14:55:00.321 各被清扫定案，
`attempts` 分别保留 4 与 0）；`configs/server.yaml` 的 `server.asynq.uri` 临时 DB 1 → DB 12，测完复旧并对
DB 12 的 asynq 键逐个 `DEL`；mailpit 容器 `gwa-mailpit` 为本轮新起（镜像已在本机）。

回归测试：`notification_record_sqlite_test.go` 五条（异步成功→交给 asynq / 同步成功→不回错且 `attempts=0` /
同步失败→两个事实并呈 / `SkipRetry` 活过 `Join` / 重试窗口那一格仍可重试）。故障注入用一次性 ent mutation hook
（`client.Use`，只在 `NotificationDelivery` 的 `UpdateOne` 且字段含 `status` 时命中）而不是删行或关库 —— 那样
`Get`/`MarkAttempted` 还能照常跑，且 hook 命中与否由测试自己断言（`fired()`），注入落空不会假装通过。
配套把 `notificationSvcEnv` 加了一个 `client *ent.Client` 字段（只给测试注入故障用，生产代码不从这里走）。

P2 剩余：`notification_rules` 表 + 管理页，替换 §3.5 的 Go 表；`sys_notification_channels` 解掉"WEBHOOK 类型
无处存 URL"的问题（方案见 §6 决策点 2）；`webhook_sender.go` 落地。

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
7. **异步载荷带不带正文** —— **P2-3 已定：带**（`{delivery_id, target, title, content}`），首稿的
   "只带 delivery_id，handler 重新渲染"那条不采纳。三条理由：
   - **重渲染做不到**：找回密码的验证码在调用那一刻才生成，只活在 Redis 与这次调用的内存里，handler 拿不到；
     要让 handler 拿得到，等于把码写进台账或另建一张明文表 —— 比"载荷在 Redis 里躺一会儿"更糟。
   - **台账故意没有正文列**（§3.3 的 `title`/`content` 不做）：永久表 + 正文快照 = 一张 OTP 明文表，
     异步路径不该反过来破坏这条边界。
   - **代价有边界**：明文验证码进的是 Redis（asynq 载荷）而不是 Postgres，存活期由
     `MaxRetry(3)` + 4 次尝试封顶（实测最坏 ~100s 后落归档），且入队前的同步预检已经把"配置不可用"
     这类注定失败的载荷挡在队列外，配置类错误还额外 `SkipRetry`。接受的是"一次崩溃窗口内可能重发一封"
     （至少一次投递），换掉的是"业务侧被 SMTP 抖动绑住响应时间"。
   若将来引入模板（P3），这一条应重估：载荷换成 `{delivery_id, template_id, render_params}` 才是既无明文
   又能重渲染的形态 —— 届时正文仍不落台账，与 §3.3 同构。
8. **清扫阈值取多少、要不要给"迟但会到"留活路** —— **P2-4 已定：缺省 15 分钟、下限 5 分钟，宁可漏扫不误扫，
   且清扫只定案不补投**。
   下限不是拍脑袋：异步派发的重试预算上界约 221s —— 算术值（4 次尝试 × `asynq.Timeout(30s)` + 默认退避
   2s/17s/82s，口径同 `docs/task_system.md` §5.6），实测的快失败路径（连不上端口、每次立即报错）则是
   ~101s 后落归档（§4 P2-3 表）。5 分钟在算术值上留约 1.35 倍余量，15 分钟留约 4 倍；
   代码里 `deliveryStaleAfter` 会把更低的配置值抬到下限并 WARN，
   因为一个手滑填 `1` 的运维不该看见"还在重试的信被判定失败"。
   被扫的行列不出第二种结局（幂等门读的就是状态），所以"SMTP 抖 6 分钟"确实会丢一封验证码 —— 换来的是一行
   不会永远挂在 `SENDING` 的台账。真需要"迟到也要送到"时，正确的改法**不是**继续拉长阈值，
   而是让清扫读 `attempts` 后**重新入队**，那要先有"下次可投递时间"这类列，属 P3+ 的量级，本次不做（也不预埋）。
   另一个否掉的方案是用 `updated_at` 当年龄锚点：本仓没有任何代码写过这一列（§4 P2-4 表里两行 `updated_at`
   至今为空），拿它比较等于"永远按创建时间算"，却要让人误以为有滑动窗口 —— 不如把事实写在列名上。
9. **回写失败该回给谁** —— **P2-5 已定：回给"还能补救它的那一方"，而不是无差别上抛**。
   异步侧回给 asynq（重试是唯一还能把结论落地的机制），同步侧在投递成功时**绝不**回给业务调用方
   （找回密码的调用方看到 error 会再生成一封验证码，等于拿"用户收到两封信"换一个"缺格的账"），
   两支 `SkipRetry` 不为回写失败破例（从没拨过号的行不值得把明文验证码在队列里多留一轮）。
   逐条去处见 §4 P2-5 的策略表。两个被否掉的方案：
   - **一律上抛**：形状最干净，代价是同步侧把可观测性缺陷换成用户可见的重复信件 —— 而调用方手里没有幂等键，
     它唯一的反应就是重发。
   - **`markResult` 内部进程内重试三次**：看着比"交给 asynq"更主动，实际是把"重投递"和"重写台账"
     混进同一个计数器（重试到第几次到底在重试哪件事，日志答不出来），且同步侧无论如何重试都改变不了
     "不能回错给调用方"这个结论 —— 于是它只解决了异步侧，而异步侧本来就有 asynq。
   仍然没有解决的那一半要说清楚：**"信已发出、结论写不进"这段窗口现在靠重投缩小、靠清扫兜底，
   但消除不了**（要消除得把投递与写结论拆成两态，见 §7 P2-5 那条）。

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
- [x] 台账加列（`request_id` / `attempts`）的三端落点清单，与 P2 新事件类型那张同形，漏任一处都是静默不一致：
      ent schema → `gow ent admin` → proto → `cd api && buf generate` → `make ts`（三端 `index.ts` 里
      确认两个新字段都已生成，再动页面）→ 三端页面列 + 各自 locales（react `notification-delivery` 命名空间 /
      ele `pages/notification_delivery.json` / vben `page.json` 的 `notificationDelivery.*`）。
      **本次留下的一处不对称（已认定可接受，不是漏做）**：`attempts` / `request_id` 的表头解释只有 react 挂了
      `Tooltip`；ele 的 `ProPage` 与 vben 的 vxe 适配都没有列头 tooltip 机制（P2 移植记录里那条老结论，本次复核
      仍成立），ele 侧把说明写进列定义的代码注释、vben 侧同。真要在页面上给最终用户解释这两列，得先给两端
      的表格适配层加列头提示能力。
- [x] P2-4 台账 `SENDING` 清扫：**契约与三端零改动**（没动 proto，故无 `buf generate` / `make ts` /
      `make openapi` / 接口同步；清扫原因走既有 `last_error` 列，三端台账页本来就在渲染它）。
      这条清单短是因为改动全在写侧：验收看三样 —— 启动日志一行
      `通知台账清扫定时任务已注册（cron=*/5 * * * *）`、一行超期 `SENDING` 在下一个 5 分钟刻度变 `FAILED`
      且 `attempts` 保留 / `sent_at` 为空 / `last_error` 以 `swept by ` 开头、以及一行**未超期**的 `SENDING`
      在同一刻度不动（本次实测把阈值降到 5 分钟复现：13:36 直插的行 13:40 刻度不动、13:45 刻度才落定）。
      **给任何"按时间比较"的 repo 方法提个醒**：谓词参数必须先 `.UTC()`，否则 sqlite 回归测试里的比较是
      字典序的（Postgres 上不复现）—— 详见 §4 P2-4 那条时间坑与 `docs/task_system.md` §10。
- [x] P2-5 台账结论回写失败上抛：**契约、三端、ent schema 一列未动**（改动全在 service 层的错误去处，
      加一个新回归文件），故无 `buf generate` / `make ts` / `make openapi` / 接口同步。验收看四样：
      ① 异步侧注入回写失败 ⇒ handler 返回 error、asynq 重投、`attempts` 涨到 4 后落归档；
      ② 同步侧同样注入 ⇒ `send-test-email` 仍返回 **200**（业务调用方拿不到这个错误），
      而日志有 `write back delivery [n] result [SENT] failed, …`；
      ③ 两支 `SkipRetry` 在 `errors.Join` 之后仍被 `errors.Is` 认出（由测试钉住）；
      ④ 兜底仍由清扫闭环：`swept by …` + `attempts` 原样保留 + `sent_at` 为空。
      **副作用必须知道**：这一条把"回写失败"从静默变成了 asynq 重投 ⇒ 收件人可能收到重复信件
      （实测 4 封、同一个验证码 `170300`）。若部署方不接受这个交换，正确的改法**不是**回退本块，
      而是把"投递"与"写结论"拆开（拨号前先落一条中间态，成功回写只更新那一格）—— 那要先有能表达
      "已发出待定案"的列语义，属 P3+ 的量级，本次不做也不预埋。

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
~~`EmailSender` 与 mailer 之间"成功投递补 channel_id"这一段仍只有服务层替身测试覆盖~~ —— 已由下面的
§7「本机跑真 SMTP 的办法」补上运行期证据（2026-09-20）。

### 本机跑真 SMTP 的办法（2026-09-20，补上"成功投递"这一格的运行期证据）

替身测试能证明"代码把 ID 传给了台账"，证明不了"真发出去的信长什么样"。本机不需要装 SMTP 服务，
一条 docker 命令即可（镜像已留在本地）：

```bash
docker run -d --name gwa-mailpit -p 127.0.0.1:1025:1025 -p 127.0.0.1:8025:8025 axllent/mailpit:latest
curl -s http://127.0.0.1:8025/api/v1/messages            # 收了哪些信
curl -s http://127.0.0.1:8025/api/v1/message/<id>        # 正文（注意是路径参数，不是 ?id=）
```

Mailpit 在 1025 上明文接收、默认不要求 AUTH，所以渠道配置要 `smtp_tls=NONE` 且**用户名留空**
（`pkg/mailer/smtp.go` 仅在 `Username != ""` 时才 `client.Auth`，填了就会对一个不 advertise AUTH 的服务端报错）。

探针步骤与账务对照（入口仍是免鉴权的 `POST /admin/v1/forgot-password`，identifier=`tenant@company.com`）：

1. 建一条指向 `127.0.0.1:1025` 的启用 EMAIL 渠道（实测拿到 id=5）。
2. **把渠道 1、2 停用** —— 自选走 `GetFirstEnabledEmailChannel`（`type=EMAIL` + `status=ON` + `ORDER BY id ASC LIMIT 1`），
   不停用就永远先命中 id 最小的那条，而 1 号恰好是 `SSL_TLS` 坏配置。这一步是"为什么探针要动演示数据"的答案。
3. 触发一次找回密码 → **台账 id=8：`status=SENT`、`channel_id=5`、`last_error` 空**；
   Mailpit 同期收到 `from d1-probe@local.test / to tenant@company.com / subject "GoWind Admin 密码重置验证码"`，
   正文 `您的密码重置验证码是：333000 …`，与 Redis 里 `gowind:vcode:reset_password:tenant@company.com` 的值逐字一致
   （即邮件没被编码/模板链路换掉内容）。
4. 复旧：渠道 1、2 重新启用、删掉探针渠道。删完台账 id=8 的 `channel_id=5` 就成了悬空引用
   （该列无外键，删除才能通过），保留这一行作为证据，读它时按此理解。

| 台账行 | 路径 | `status` | `channel_id` | 说明 |
| --- | --- | --- | --- | --- |
| id=1 | 自选（修 `SmtpAccount.ID` 前） | FAILED | 空 | 错误文本 `channel [0]` —— 就是被补上的那一格 |
| id=2 | 自选（修 ID 后、拨号前被拦） | SKIPPED | 空 | 配置不可用不该伪装成"发过" |
| id=3 / id=4 | 显式指定渠道 | SKIPPED | 1 / 3 | 显式路径失败也带 ID —— 那是 `Create` 时从 `req.ChannelId` 直接落的（`notification_service.go:99`），不是回执；自选路径 `Create` 时无从得知，只能发完补（`:154` 的 `pickedChannelId`） |
| **id=8** | **自选 + 真发成功** | **SENT** | **5** | 本次闭环：`SendReceipt.ChannelID = account.ID` 在真实投递下成立 |

本节取证时仍未覆盖的一段：`SendDirect` 之后的**异步**补台账 —— 已由 §4 P2-3（2026-09-20）连运行期证据一起补上；
SMS / WEBHOOK 两个渠道到今天**依然没有**实现（连 Registry 都没接进去，不是"缺测试"而是"缺代码"，见 §6 决策点 2）。

### 收件箱读侧不钉归属：一处越权读（2026-09-20，运行期实测发现并修掉）

`GET /admin/v1/internal-message/inbox` 的过滤条件整个来自调用方的 `query` 字符串，服务端只让 `TenantPrivacy`
注入租户谓词，**从不注入收件人谓词**——三端页面各自在前端塞 `recipientUserId`，于是"只看自己的收件箱"这件事
从来只是客户端约定。实测：同租户的两个用户，A 用自己的 token 发 `?query={"recipientUserId__eq":<B 的 id>}`
就读到了 B 的收件行，`title`/`content` 由父消息回填一并带出。租户隔离把"跨租户"堵住了，没堵"同租户跨用户"。

修法：`InternalMessageRecipientRepo.List` 在非平台、非系统上下文下强制 `recipient_user_id = viewer.UserID()`
（`viewer` 由 `pkg/middleware/ent/ent.go:29` 从令牌注入，uid 是真用户 ID，不是 0）。
平台/系统上下文豁免：用户详情页要看指定用户的收件箱（`ListUserInbox` 是这条 `List` 在生产里唯一的调用方），
异步任务路径需要全量读。
回归测试：`TestInternalMessageRecipientTenantSqlite` 新增"同租户、换一个收件人就读不到"的断言。

**没一起修的姊妹问题（写侧，已定位未修）**：`MarkNotificationAsRead`（`:329`）与
`DeleteNotificationFromInbox`（`:510`）都以 `Where(RecipientUserIDEQ(req.GetUserId()))` 定作用域，
而 `user_id` 取自**请求体**、不是 viewer —— 同租户用户可以传别人的 `user_id` 去标记已读/删除别人的收件行
（跨租户仍被 `TenantMutationGuardPolicy` 拦住）。这与上面修掉的是同一个假设（"调用方报的归属可信"），
只是落在写侧。修法同形（非平台上下文以 viewer 覆盖 `req.UserId`），但要一并确认三端是否有
"平台管理员代客操作"的调用，本次未动，留作 P2 收尾项。

同一批实测里另有一处**不是缺陷但值得记**：每次创建实体都回一条
`script entity hook <table>.after_create failed: ... no scripts mounted on hook point` 的 ERROR 日志
（站内信/用户/角色/租户都中招）。钩子点上没挂脚本是常态，不该按错误记账；属脚本系统（`docs/script_system.md`）
的日志分级问题，与通知域无关，未在本次改动内。

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
