package channel

// 扩展渠道（EXT_*）机制：schema 声明 + settings 构造 + 通用 Sender。
//
// 设计见 docs/notify_channels_expansion.md。要点：
//   - 配置整包存渠道行 settings 列（JSON），schema 声明字段（label/类型/必填/敏感），
//     前端按 ListChannelSchemas 动态渲染表单——新渠道接入 = 一份声明 + 一份 builder；
//   - 敏感字段加密在 repo 层（存储职责），这里只管"解密后的 map → notify.Service"；
//   - 发送统一桥到 go-utils/notify 的 Service 接口（错误哨兵/SSRF 防护库内自管）。

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	notificationChannelV1 "go-wind-admin/api/gen/go/notification_channel/service/v1"

	"github.com/tx7do/go-utils/notify"

	"go-wind-admin/app/admin/service/internal/data"
)

// settingsHTTPClient 扩展渠道共享的出站 HTTP 客户端。
// P2 修复：此前是裸 http.Client——39 种 EXT 渠道（钉钉/飞书/Telegram/Matrix/
// Mattermost/……）的 webhookUrl/serverUrl 无任何 SSRF 防线，而 WEBHOOK 渠道反而
// 有 GuardedClient（拨号级内网拦截）。对齐 WEBHOOK 口径换 GuardedClient；
// 自建内网渠道（自托管 Matrix/Mattermost 等）经 EnvAllowPrivateWebhook=1 显式放行
// （与 WEBHOOK 渠道同一开关、同一风险声明）。
var settingsHTTPClient = notify.NewGuardedClient(allowPrivateWebhook())

// SettingsField 字段声明（构造 proto ChannelField 的原料）。
type SettingsField struct {
	Name        string
	Label       string
	Type        string // text | textarea | number | password | select
	Required    bool
	Sensitive   bool
	Placeholder string
	Hint        string
	Options     []string
}

// SettingsSchema 渠道声明（构造 proto ChannelSchema 的原料）。
type SettingsSchema struct {
	Type        string // proto Type 枚举名（与 ent 列值逐字相同）
	Group       string // IM | PUSH | MOBILE | SMS | MAIL | AUTO | OPS
	Label       string
	TargetLabel string // 运行目标含义；空 = 该渠道无独立目标
	NeedsTarget bool
	Fields      []SettingsField
}

// toProto schema → proto ChannelSchema（service 层经此下发前端）。
func (s *SettingsSchema) toProto() *notificationChannelV1.ChannelSchema {
	fields := make([]*notificationChannelV1.ChannelField, 0, len(s.Fields))
	for _, f := range s.Fields {
		fields = append(fields, &notificationChannelV1.ChannelField{
			Name:        f.Name,
			Label:       f.Label,
			Type:        f.Type,
			Required:    f.Required,
			Sensitive:   f.Sensitive,
			Placeholder: f.Placeholder,
			Hint:        f.Hint,
			Options:     f.Options,
		})
	}
	return &notificationChannelV1.ChannelSchema{
		Type:        s.Type,
		Group:       s.Group,
		Label:       s.Label,
		TargetLabel: s.TargetLabel,
		NeedsTarget: s.NeedsTarget,
		Fields:      fields,
	}
}

// settingsSchemas 全部扩展渠道声明（39 个；注册表与 builder 表同一清单来源）。
var settingsSchemas []*SettingsSchema

// settingsBuilders 类型名 → 构造器（settings map → notify.Service）。
var settingsBuilders = map[string]func(m settingsMap) notify.Service{}

// registerSettingsChannel 声明 + builder 一次注册。
func registerSettingsChannel(schema SettingsSchema, build func(m settingsMap) notify.Service) {
	settingsSchemas = append(settingsSchemas, &schema)
	settingsBuilders[schema.Type] = build
}

// SettingsTypeNames 全部扩展渠道的类型名（repo 自选查询按它过滤）。
func SettingsTypeNames() []string {
	out := make([]string, 0, len(settingsSchemas))
	for _, s := range settingsSchemas {
		out = append(out, s.Type)
	}
	return out
}

// ListSettingsSchemas 全量 schema（service 层的 ListChannelSchemas 数据源）。
func ListSettingsSchemas() []*notificationChannelV1.ChannelSchema {
	out := make([]*notificationChannelV1.ChannelSchema, 0, len(settingsSchemas))
	for _, s := range settingsSchemas {
		out = append(out, s.toProto())
	}
	return out
}

// SettingsSchemaByType 按类型名取 schema；找不到（老三样/未知）返回 nil。
func SettingsSchemaByType(typeName string) *SettingsSchema {
	for _, s := range settingsSchemas {
		if s.Type == typeName {
			return s
		}
	}
	return nil
}

// SettingsAccount 一条扩展渠道的解密后配置（repo 提供，仅发送链路内部使用）。
type SettingsAccount struct {
	ID       uint32
	TypeName string // proto Type 枚举名（与 settingsBuilders 键一致）
	Settings map[string]any
}

// BuildSettingsService 按 type 构造 notify.Service；未知类型报错。
func BuildSettingsService(typeName string, settings map[string]any) (notify.Service, error) {
	build, ok := settingsBuilders[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown settings channel type %q", typeName)
	}
	if settings == nil {
		settings = settingsMap{}
	}
	return build(settingsMap(settings)), nil
}

// ---- settings map 取值辅助（JSON 反序列化数字为 float64，统一在此消化） ----

type settingsMap map[string]any

func (m settingsMap) str(key string) string {
	if m == nil {
		return ""
	}
	switch v := m[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func (m settingsMap) int(key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		return 0
	}
}

func (m settingsMap) int64(key string) int64 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	default:
		return 0
	}
}

// strSlice 逗号分隔字符串或 JSON 数组 → []string（tags 类字段两写法都收）。
func (m settingsMap) strSlice(key string) []string {
	if m == nil {
		return nil
	}
	switch v := m[key].(type) {
	case string:
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// ---- SettingsSender：EXT 渠道的通用 Sender ----

// SettingsSender 扩展渠道通用发送器：settings → notify.Service → Send。
// 与 EmailSender/WebhookSender/SmsSender 共用同一套渠道选择策略
// （显式 channel_id 优先，否则取第一个启用的同类型行）。
type SettingsSender struct {
	channel *data.NotificationChannelRepo
}

var _ Prechecker = (*SettingsSender)(nil)

func NewSettingsSender(channelRepo *data.NotificationChannelRepo) *SettingsSender {
	return &SettingsSender{channel: channelRepo}
}

// Channel 渠道标识（扩展渠道统一记 EXT）。
func (s *SettingsSender) Channel() notificationV1.Channel {
	return notificationV1.Channel_EXT
}

// Precheck 解析配置并按 schema 校验必填，不拨号。见 channel.Prechecker。
func (s *SettingsSender) Precheck(ctx context.Context, channelID uint32) error {
	account, err := s.pickAccount(ctx, channelID)
	if err != nil {
		return err
	}
	return validateSettingsRequired(account)
}

// Send 投递一条消息。error 语义与其它 Sender 一致：配置类包 ErrChannelNotConfigured。
func (s *SettingsSender) Send(ctx context.Context, req *SendRequest) (*SendReceipt, error) {
	if req == nil {
		return nil, fmt.Errorf("send request is empty")
	}

	account, err := s.pickAccount(ctx, req.ChannelID)
	if err != nil {
		return nil, err
	}
	if req.Target == "" && account.needsTarget() {
		return nil, fmt.Errorf("%w: channel [%d] (%s) requires a target (%s)",
			ErrChannelNotConfigured, account.ID, account.TypeName, account.targetLabel())
	}

	service, err := BuildSettingsService(account.TypeName, account.Settings)
	if err != nil {
		return nil, err
	}

	receipt, sendErr := service.Send(ctx, &notify.Message{
		Target:  req.Target,
		Title:   req.Title,
		Content: req.Content,
	})
	if sendErr != nil {
		return &SendReceipt{ChannelID: account.ID}, fmt.Errorf("send via settings channel [%d] (%s) failed: %w", account.ID, account.TypeName, sendErr)
	}
	// 服务商侧消息 ID（阿里云 RequestId 等）带回落台账，对账用
	providerID := ""
	if receipt != nil {
		providerID = receipt.ProviderID
	}
	return &SendReceipt{ChannelID: account.ID, ProviderID: providerID}, nil
}

// pickAccount 显式 ID 优先，否则取第一个启用的扩展渠道行——
// EXT 是一个 Channel 大类，下含 39 种 type：自选策略按"第一个启用的 settings 行"落。
func (s *SettingsSender) pickAccount(ctx context.Context, channelID uint32) (*SettingsAccount, error) {
	if channelID == 0 {
		typeName, storeJSON, err := s.channel.GetFirstEnabledSettingsRaw(ctx, SettingsTypeNames())
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrChannelNotConfigured, err)
		}
		settings, err := DecryptSettings(typeName, storeJSON)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrChannelNotConfigured, err)
		}
		return &SettingsAccount{ID: 0, TypeName: typeName, Settings: settings}, nil
	}

	typeName, storeJSON, enabled, err := s.channel.GetSettingsRaw(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("%w: channel [%d] not found: %v", ErrChannelNotConfigured, channelID, err)
	}
	if !isSettingsType(typeName) {
		return nil, fmt.Errorf("%w: channel [%d] (%s) is not an EXT settings channel", ErrChannelNotConfigured, channelID, typeName)
	}
	if !enabled {
		return nil, fmt.Errorf("%w: channel [%d] is disabled", ErrChannelNotConfigured, channelID)
	}

	settings, err := DecryptSettings(typeName, storeJSON)
	if err != nil {
		return nil, fmt.Errorf("%w: channel [%d] credentials: %v", ErrChannelNotConfigured, channelID, err)
	}
	return &SettingsAccount{ID: channelID, TypeName: typeName, Settings: settings}, nil
}

// isSettingsType 类型是否属于扩展渠道（settings 表构建器注册了就有 Sender）。
func isSettingsType(typeName string) bool {
	_, ok := settingsBuilders[typeName]
	return ok
}

// needsTarget / targetLabel 运行目标语义（schema 声明）。
func (a *SettingsAccount) needsTarget() bool {
	if s := SettingsSchemaByType(a.TypeName); s != nil {
		return s.NeedsTarget
	}
	return false
}

func (a *SettingsAccount) targetLabel() string {
	if s := SettingsSchemaByType(a.TypeName); s != nil && s.TargetLabel != "" {
		return s.TargetLabel
	}
	return "target"
}

// validateSettingsRequired 按 schema 校验必填字段（Precheck 用）。
func validateSettingsRequired(account *SettingsAccount) error {
	schema := SettingsSchemaByType(account.TypeName)
	if schema == nil {
		return fmt.Errorf("unknown settings channel type %q", account.TypeName)
	}
	for _, f := range schema.Fields {
		if !f.Required {
			continue
		}
		if strings.TrimSpace(settingsMap(account.Settings).str(f.Name)) == "" {
			return fmt.Errorf("%s is required", f.Label)
		}
	}
	return nil
}
