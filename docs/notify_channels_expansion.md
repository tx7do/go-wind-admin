# 通知渠道扩展：接入 go-utils/notify 全量渠道

> **状态：已立项（2026-10-15 用户拍板"都搞"），M1 实施**——settings 列 + schema 驱动动态表单 +
> 通用 SettingsSender + 39 个新渠道全量接入（react 先行）。
>
> 立项背景：通知发送层已下沉 `go-utils/notify v0.0.6`（40+ 渠道包，统一
> `Service{Name(); Send(ctx, *Message)}` 接口），admin 渠道管理页却只支持
> EMAIL / WEBHOOK / SMS 三类——库与配置面之间的缺口就是本档要填的。

## 1. 范围与渠道清单

**不接**（3+1 个）：
- `email` / `webhook` / `sms`：admin 已有同类且更强（WEBHOOK 带出站风格与载荷模板），不重复接；
- `push`（APNs/FCM）：需要 .p8 证书/FCM 凭据且 admin 无移动客户端——沿通知域既有决策不接。

**接入 39 个**，按 schema 分组：

| 组 | 渠道（proto Type 枚举名） |
| --- | --- |
| IM（15） | WECOM_APP（企业微信应用）、DINGTALK_APP（钉钉工作通知）、FEISHU_APP（飞书应用消息）、WECHAT（公众号模板消息）、TELEGRAM、SLACK、DISCORD、TEAMS、GCHAT、WEBEX、LINE、MATRIX、MATTERMOST、ROCKETCHAT、ZULIP |
| 个人推送（8） | BARK、NTFY、GOTIFY、SERVERCHAN（Server酱）、WXPUSHER、PUSHOVER、PUSHBULLET、PUSHDEER |
| 移动推送服务（2） | JPUSH（极光）、GETUI（个推） |
| 短信/国际（4） | HUAWEISMS、YUNPIAN、TWILIO、VONAGE、WHATSAPP（5 个，归"短信/消息"组） |
| 邮件 API（3） | MAILGUN、SENDGRID、POSTMARK |
| 效率自动化（3） | IFTTT、JOIN、HOMEASSISTANT |
| 运维告警（3） | DATADOG、OPSGENIE、PAGERDUTY |

## 2. 架构

### 2.1 存储：settings JSONB（决策点 2 的修订）

`sys_notification_channels` 加一列 `settings`（TEXT，JSON 对象）。**新渠道配置整包存 settings；
老三样（EMAIL/WEBHOOK/SMS）维持原列不动**——各自只有一种真相源，§6 决策点 2 否决 settings 的
"双真相源"顾虑在新旧分治下不成立。这与 go-wind-im 的多态信封、Dify 的渠道配置同构
（当时不抄是因为只有两种渠道，情境已变）。

**敏感字段加密**：schema 声明 `sensitive: true` 的字段，写入前逐字段 `EncryptIfNeeded`，
运行时解密给 sender；读回显（DTO 的 settings）敏感字段值替换为空串——前端约定
"敏感字段留空 = 不修改"，更新时后端按 schema 把空值合并回旧值。

### 2.2 schema 驱动

每渠道在 Go 里声明一份 `ChannelSchema`（分组/标签/Target 语义/字段列表），经
`ListChannelSchemas` RPC 下发；前端**一个动态表单组件**按 schema 渲染——新渠道接入成本
= 一份 Go 声明 + 一份 factory（各 5~15 行），三端零逐渠道表单。

```go
type ChannelSchema struct {
    Type        string        // proto Type 枚举名（与 ent 列值逐字相同）
    Group       string        // IM / PUSH / MOBILE / SMS / MAIL / AUTO / OPS
    Label       string
    TargetLabel string        // 运行目标字段名（chat_id / 用户ID / topic…）；无目标渠道留空
    NeedsTarget bool          // 测试发送与投递是否要求 target
    Fields      []FieldSchema // name/label/type(text|textarea|number|password|select)/required/sensitive/options/hint
}
```

### 2.3 发送：通用 SettingsSender

一个 `SettingsSender` 实现 `channel.Sender`（`Channel() = Channel_EXT`）：
解密 settings → 按 type 调对应 factory 构造 `notify.Service` → `Send(Message{Target,Title,Content})`。
factory 表 `map[类型]func(settings) notify.Service`，39 份各 5~15 行；HTTP 客户端共享一个
（10s 超时）；SSRF 防护由库内 `ssrf.go` 自管。

### 2.4 台账与路由

- `notification.proto` 的 `Channel` 枚举加 `EXT = 5`：扩展渠道的台账行统一记 EXT，
  具体是哪种渠道由 `channel_id` → 渠道行 type 回答（台账粒度=渠道大类，与既有口径一致）；
- 规则页渠道下拉加 EXT（告警/通知可路由到任一扩展渠道，Title/Content 直发无需验证码参数）；
- 台账页筛选项按"有 Sender 即进下拉"的规矩加 EXT。

## 3. 关键语义

- **Target**：运行目标按渠道解释（telegram=chat_id、钉钉=userid_list、ntfy=topic、
  bark=deviceKey、wecom=touser…），schema 的 `targetLabel` 把这事说给用户听；
  无目标渠道（ServerChan/PushDeer/群 webhook 类，凭据即设备）`needsTarget=false`。
- **测试发送**：`SendTest` 对扩展渠道 = 构造 service 发一条测试消息（needsTarget 的要求
  填 recipient）；新事件类型 `CHANNEL_TEST_EXT = 8`（台账 event_type 枚举同步）。
- **校验**：保存时按 schema 校验 required；敏感字段空值合并旧值；错误码与既有渠道同风格。

## 4. 分期

- **M1（本期）**：基建（settings/schema/sender/proto/ent）+ 39 渠道全量声明 + react 动态表单
  + e2e（本地 ntfy mock 全链路：建渠道 → 测试发送 → mock 收到 POST）。
- **M2（已落地，fd708496 三端收口 + 后续 ele 抽屉交互修复）**：ele/vben 渠道页移植
  （动态表单组件同构，vben 937 行 / ele 抽屉 580 行）；剩余尾巴仅各渠道 Target 提示打磨。
