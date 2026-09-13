# SSE 推送架构（Server-Sent Events）参考文档

> **定位**：本仓 SSE 推送链路的唯一权威说明——服务端配置与生命周期、流鉴权与 streamID 语义、
> 事件生产方、三端消费模块、部署拓扑与排障。改推送链路、加新事件类型、排"收不到通知"前先读它。
> 部署侧（nginx 反代网关）见 [backend_deploy.md](./backend_deploy.md)「SSE 反向代理网关」节；
> 站内信子系统的业务语义见其管理页与 `internal_message` 服务。

## 1. 组件地图

```
生产方    InternalMessageService（当前唯一）
          └─ publishNotification：收件记录落库后 TryPublish（非阻塞、best-effort）
                     │ 事件 { id: GUIDv4, event: "notification", data: <收件记录 JSON> }
                     ▼
SSE 服务  :7789 /events（configs/server.yaml sse 块；独立于 REST :7788 的 transport）
          ├─ HandleAuthorize：连接鉴权（token 校验 + stream 归属强制）
          ├─ 流注册：streamID = userId，auto_stream 自动建流
          └─ fan-out：同一 stream 的全部 subscriber（该用户多设备/多标签页）
消费方    三端 SSEClient（fetch-event-source 封装，自动重连）
          └─ URL = `${VITE_SSE_URL}?stream=${自身 userId}`（VITE_SSE_URL：dev=后端直连，
             生产=SSE 网关域名；见各端 .env.*）
```

## 2. 服务端配置与生命周期

`configs/server.yaml`：

```yaml
sse:
  addr: ":7789"      # 独立监听端口，不与 REST 共用
  codec: "json"      # 事件 data 序列化格式
  path: "/events"    # 事件端点路径
  auto_stream: true  # 订阅时自动建流（无需服务端预 CreateStream）
  auto_reply: false
```

装配（`internal/server/sse_server.go` `NewSseServer`）：配置缺失 → 返回 nil，
站内信服务持有 **noop 发布者**（推送静默降级，落库投递不受影响）；
配置存在 → 注入 `WithSubscriberFunction`/`WithAuthorizeFunc`（均指向 InternalMessageService）
并 `RegisterInternalMessagePublisher` 注入真实发布者。

## 3. 流鉴权与 streamID 语义（`HandleAuthorize`）

SSE 连接不走 REST 的 auth 中间件链——transport 自带鉴权钩子，逐连接执行：

1. `authenticator.Authenticate`（完整校验：JWT 验签 + 过期 + Redis 缓存吊销核对 + `IsBlocked`
   封禁位）——失败即拒绝连接；
2. **流归属强制**：URL `?stream=<uid>` 解析出的用户 ID 必须与 token payload 的 userId 一致，
   否则 403 "stream user mismatch"——**该校验是强制的、不可移除**：
   streamID 已从 access token 字符串（不可伪造，无需校验）改为 userId（可被任意构造），
   没有归属校验则用户 A 订阅 `?stream=<userB_id>` 即可收走 B 的全部站内信通知。

订阅回调 `HandleSubscribe` 仅记日志。流按 streamID（userId）注册，
**同一用户的多设备/多标签页订阅同一条流**——库的 fan-out 把事件投递给该流全部 subscriber，
生产方只需单次 publish（这是 streamID 取 userId 而非 token 的动机：token 每设备不同，
以 token 为流 ID 则同用户多设备互相收不到）。

## 4. 事件生产（当前唯一生产方：站内信收件通知）

`InternalMessageService.publishNotification`（投递链上每个收件人调用）：

- **落库优先**：收件记录先经 `internalMessageRecipientRepo.Create` 落库，
  推送是 best-effort 增强——`TryPublish` 非阻塞（流不存在=用户离线、或缓冲已满，立即跳过，
  只记 debug 日志），失败不影响投递；离线用户重连后从**收件箱接口**补取（不依赖推送）；
- 事件结构：`{ ID: GUIDv4, Event: "notification", Data: <收件记录 JSON> }`；
  `Event` 字段即前端的事件名（三端 `on('notification', …)`）；
- 广播投递的落库侧（`broadcast_message` 任务、幂等约束）见
  [task_system.md](./task_system.md) 第 5.4 节。

**新增事件类型**的生产端落点：持有 publisher 的 service 内调用
`TryPublish(StreamID(userId), &sse.Event{Event: []byte("<类型名>"), …})`；
消费端在页面 `globalSSEClient.on('<类型名>', handler)` 注册。事件类型当前无注册表/枚举约束，
生产与消费两端字符串需人工对齐。

## 5. 前端消费（三端）

| | react | vue-element | vue-vben |
|---|---|---|---|
| 模块 | `src/core/transport/sse/`（`sse_client.ts` + `index.ts` 单例 `globalSSEClient`） | `src/core/transport/sse/`（同构） | `apps/admin/src/transport/sse/`（路径不同，同构） |
| 传输 | `@microsoft/fetch-event-source`（支持自定义 headers 携带凭证；原生 EventSource 不支持） | 同左 | 同左 |
| URL 构造 | `${VITE_SSE_URL}?stream=${userInfo.id}`（`useTokenRefresh.ts`；id 取自登录用户信息） | 同构（`VITE_APP_SSE_URL`） | 同构（`VITE_GLOB_SSE_URL`） |
| 重连 | 内置，`reconnectDelay` 5000ms | 同左 | 同左 |
| 现有消费 | `HeaderContent.tsx`：`on('notification')` 刷新顶栏铃铛未读数（卸载时 `off`） | 同构（顶栏通知组件） | 同构 |

订阅生命周期：仅登录会话内；登出/会话吊销后连接鉴权失效（下次重连被拒）。
收件数据补取一律走收件箱查询接口（`internal_message` 域），
**前端不得假设推送可靠**（见第 4 节 best-effort 语义）。

## 6. 部署拓扑

生产拓扑中 SSE 由**独立 nginx 反代网关**承载（透传 `/events` 至 7789），
不随 docker-compose 启动，需单独构建运行（8013）；网关配置已关闭 gzip 与全部代理缓冲
（SSE 实时性的必要条件，勿回退）；必须与 admin-service 同处 `app-tier` 网络。
完整命令与配置见 [backend_deploy.md](./backend_deploy.md)「SSE 反向代理网关」节。
TLS 由外层负载均衡终止（与静态前端一致），网关仅监听 HTTP。

## 7. 运维与排障

| 症状 | 核对 |
|---|---|
| 连接 401/403 | token 失效/被吊销/被封禁（`HandleAuthorize` 日志）；`?stream=` 与登录用户不一致（越权校验拒绝） |
| 连接 404/502 | SSE 网关未起/路径不对（`/events`）/网关与后端不同网络；dev 直连时后端 7789 未起 |
| 收不到通知但收件箱有 | 正常路径之一：推送 best-effort（离线/缓冲满跳过），刷新收件箱补取 |
| 通知延迟/断续 | 网关侧 buffering/gzip 是否被回退；中间层是否有响应缓冲 |
| 事件 JSON 解析失败 | codec 固定 json；生产端 `Data` 序列化格式与消费端解析约定须一致 |

## 8. 边界与已知问题

| 项 | 现状 |
|---|---|
| 事件类型 | 仅 `notification`（站内信）；无类型注册表，新增类型靠两端字符串人工对齐 |
| 推送可靠性 | 设计即 best-effort：无重放、无 ack、离线不积压（补取靠收件箱） |
| 慢消费者 | `TryPublish` 缓冲满即丢（debug 日志），不阻塞生产方；无背压 |
| 观测 | 连接建立/断开（HandleSubscribe/日志）与跳过（debug 级）可查，无投递指标面板 |
| 与 REST 的中间件差异 | SSE transport 不走 REST 中间件链（审计中间件不覆盖 /events 连接与事件），鉴权由 HandleAuthorize 独立承担 |
