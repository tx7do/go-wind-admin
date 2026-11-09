package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// MonitorAlertRule 监控告警规则：指标阈值 → 触发通知（显式渠道 + 目标，不经路由表）。
// 平台全局（监控是平台域），与渠道/模板同域。
type MonitorAlertRule struct {
	ent.Schema
}

func (MonitorAlertRule) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "monitor_alert_rules",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("监控告警规则表"),
	}
}

// Fields of the MonitorAlertRule.
func (MonitorAlertRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("规则名称").
			NotEmpty().
			Optional().
			Nillable(),

		field.Enum("metric").
			Comment("监控指标").
			NamedValues(
				"GoGoroutines", "GO_GOROUTINES",
				"GoMemAllocMb", "GO_MEM_ALLOC_MB",
				"DbOpenConnections", "DB_OPEN_CONNECTIONS",
				"DbPingFail", "DB_PING_FAIL",
				"RedisDbSize", "REDIS_DB_SIZE",
			).
			Optional().
			Nillable(),

		field.Enum("op").
			Comment("比较运算（DB_PING_FAIL 忽略本列）").
			NamedValues(
				"Ge", "GE",
				"Le", "LE",
			).
			Optional().
			Nillable(),

		field.Float("threshold").
			Comment("阈值（DB_PING_FAIL 忽略本列）").
			Optional().
			Nillable(),

		field.Uint32("cooldown_minutes").
			Comment("重复告警冷却（分钟）：持续越限时按此间隔重发").
			Default(30),

		field.Enum("channel").
			Comment("告警渠道（显式指定，不经路由表）").
			NamedValues(
				"Email", "EMAIL",
				"Webhook", "WEBHOOK",
			).
			Optional().
			Nillable(),

		field.String("target").
			Comment("投递目标：EMAIL 为收件地址，WEBHOOK 为回调 URL（显式指定）").
			NotEmpty().
			Optional().
			Nillable(),

		field.Bool("is_enabled").
			Comment("是否启用").
			Default(true),

		field.Bool("last_firing").
			Comment("上次扫描是否越限（用于区分「首次触发」与「恢复」）").
			Default(false),

		field.Float("last_value").
			Comment("上次扫描的指标值").
			Optional().
			Nillable(),

		field.Time("last_alerted_at").
			Comment("上次告警时间（冷却判断锚）").
			Optional().
			Nillable(),
	}
}

// Mixin of the MonitorAlertRule.
func (MonitorAlertRule) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID64{},
		mixin.Remark{},
	}
}

// Indexes of the MonitorAlertRule.
func (MonitorAlertRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("is_enabled").StorageKey("idx_monitor_alert_enabled"),
	}
}
