// MonitorAlertRuleRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 告警规则是评估器的运行时真相（指标阈值 → 显式渠道告警，不经路由表），测试盯三件事：
//   - 枚举往返：metric / op / channel 三个可空指针枚举列经 converter 双向命中
//     （channel 与通知域共用 notificationV1.Channel，两侧字面量都必须对得上；读视图钉住即可）；
//   - 掩码更新与评估回写互不覆盖：Update 走掩码语义、永不触碰 last_firing/last_value/
//     last_alerted_at；MarkFiring 只写这三列、不走掩码——管理员的编辑与系统的扫描各写各的；
//   - 值型默认列（cooldown_minutes=30 / is_enabled=true）显式给 0/false 必须如实落库。
package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	monitorAlertV1 "go-wind-admin/api/gen/go/monitor_alert/service/v1"
	notificationV1 "go-wind-admin/api/gen/go/notification/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent/monitoralertrule"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newAlertRule 构造一行合法告警规则载荷（GO_GOROUTINES ≥ 阈值 → EMAIL），字段可按需覆盖。
func newAlertRule() *monitorAlertV1.MonitorAlertRule {
	return &monitorAlertV1.MonitorAlertRule{
		Name:            trans.Ptr("goroutine 超限"),
		Metric:          monitorAlertV1.MonitorMetric_GO_GOROUTINES.Enum(),
		Op:              monitorAlertV1.AlertOp_GE.Enum(),
		Threshold:       trans.Ptr(500.0),
		CooldownMinutes: trans.Ptr(uint32(30)),
		Channel:         notificationV1.Channel_EMAIL.Enum(),
		Target:          trans.Ptr("ops@example.com"),
		IsEnabled:       trans.Ptr(true),
	}
}

// TestMonitorAlertRuleRepoSqlite_Create 验证字段级落库与三枚举双向映射、
// 值型默认列的显式 0/false 不被吞、operatorID 落 created_by、评估状态列初始为空。
func TestMonitorAlertRuleRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewMonitorAlertRuleRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.Create(ctx, &monitorAlertV1.MonitorAlertRule{
		Name:            trans.Ptr("goroutine 超限"),
		Metric:          monitorAlertV1.MonitorMetric_GO_GOROUTINES.Enum(),
		Op:              monitorAlertV1.AlertOp_GE.Enum(),
		Threshold:       trans.Ptr(500.0),
		CooldownMinutes: trans.Ptr(uint32(15)),
		Channel:         notificationV1.Channel_EMAIL.Enum(),
		Target:          trans.Ptr("ops@example.com"),
		IsEnabled:       trans.Ptr(true),
		Remark:          trans.Ptr("持续越限按冷却重发"),
	}, 1001)
	require.NoError(t, err, "完整载荷的 Create 应成功")
	require.NotZero(t, id)

	row, err := entClient.Client().MonitorAlertRule.Get(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, row.Name)
	require.Equal(t, "goroutine 超限", *row.Name)
	require.NotNil(t, row.Metric)
	require.Equal(t, monitoralertrule.MetricGoGoroutines, *row.Metric,
		"proto 枚举名应经 converter 落为实体枚举字面量")
	require.NotNil(t, row.Op)
	require.Equal(t, monitoralertrule.OpGe, *row.Op)
	require.NotNil(t, row.Channel)
	require.Equal(t, monitoralertrule.ChannelEmail, *row.Channel)
	require.NotNil(t, row.Threshold)
	require.Equal(t, 500.0, *row.Threshold)
	require.NotNil(t, row.CooldownMinutes)
	require.Equal(t, uint32(15), row.CooldownMinutes)
	require.Equal(t, "ops@example.com", *row.Target)
	require.True(t, row.IsEnabled)
	require.Equal(t, "持续越限按冷却重发", *row.Remark)
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, uint64(1001), *row.CreatedBy, "operatorID 应落 created_by")
	require.NotNil(t, row.CreatedAt, "created_at 必须显式写入：mixin 列没有 DB 默认值")
	// 评估状态列初始为空：还没扫过第一轮
	require.False(t, row.LastFiring, "last_firing 应落列默认 false")
	require.Nil(t, row.LastValue)
	require.Nil(t, row.LastAlertedAt)

	// 读视图：三枚举回映射为 proto 值
	dto, err := repo.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, dto.GetId())
	require.Equal(t, monitorAlertV1.MonitorMetric_GO_GOROUTINES, dto.GetMetric(),
		"实体枚举应经 converter 回映射为 proto 枚举，不能落 UNSPECIFIED 零值")
	require.Equal(t, monitorAlertV1.AlertOp_GE, dto.GetOp())
	require.Equal(t, notificationV1.Channel_EMAIL, dto.GetChannel())
	require.Equal(t, 500.0, dto.GetThreshold())

	// 另一组枚举取值（DB_PING_FAIL / LE / WEBHOOK）：ping 失败忽略 op/threshold，
	// 但列本身仍要如实落库
	id2, err := repo.Create(ctx, &monitorAlertV1.MonitorAlertRule{
		Name:      trans.Ptr("DB ping 失败"),
		Metric:    monitorAlertV1.MonitorMetric_DB_PING_FAIL.Enum(),
		Op:        monitorAlertV1.AlertOp_LE.Enum(),
		Threshold: trans.Ptr(0.0),
		Channel:   notificationV1.Channel_WEBHOOK.Enum(),
		Target:    trans.Ptr("https://hooks.example.com/alert"),
	}, 1)
	require.NoError(t, err)
	row2, err := entClient.Client().MonitorAlertRule.Get(ctx, id2)
	require.NoError(t, err)
	require.Equal(t, monitoralertrule.MetricDbPingFail, *row2.Metric)
	require.Equal(t, monitoralertrule.OpLe, *row2.Op)
	require.Equal(t, monitoralertrule.ChannelWebhook, *row2.Channel)

	// 值型默认列的显式零值不得被默认吞：
	// cooldown=0 表示"每次扫描都重发"、is_enabled=false 表示规则不上岗
	id3, err := repo.Create(ctx, &monitorAlertV1.MonitorAlertRule{
		Name:            trans.Ptr("零值规则"),
		CooldownMinutes: trans.Ptr(uint32(0)),
		IsEnabled:       trans.Ptr(false),
	}, 1)
	require.NoError(t, err)
	row3, err := entClient.Client().MonitorAlertRule.Get(ctx, id3)
	require.NoError(t, err)
	require.NotNil(t, row3.CooldownMinutes)
	require.Equal(t, uint32(0), row3.CooldownMinutes, "cooldown=0 是合法配置，不得落成默认 30")
	require.NotNil(t, row3.IsEnabled)
	require.False(t, row3.IsEnabled, "is_enabled=false 不得落成默认 true")

	// 未指定的默认列：cooldown 落默认 30、is_enabled 落默认 true；operatorID=0 不写 created_by
	id4, err := repo.Create(ctx, &monitorAlertV1.MonitorAlertRule{
		Name:   trans.Ptr("缺省规则"),
		Target: trans.Ptr("nobody@example.com"),
	}, 0)
	require.NoError(t, err)
	row4, err := entClient.Client().MonitorAlertRule.Get(ctx, id4)
	require.NoError(t, err)
	require.Equal(t, uint32(30), row4.CooldownMinutes, "未指定 cooldown 应落列默认 30")
	require.True(t, row4.IsEnabled, "未指定 is_enabled 应落列默认 true")
	require.Nil(t, row4.CreatedBy, "播种（operatorID=0）不该写出一个不存在的操作人")

	// 入参守卫
	_, err = repo.Create(ctx, nil, 1)
	require.Error(t, err, "nil 载荷应返回 BadRequest")
}

// TestMonitorAlertRuleRepoSqlite_Get 验证命中回 DTO、id=0 的 BadRequest、
// 不存在 ID 回 NotFound。
func TestMonitorAlertRuleRepoSqlite_Get(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewMonitorAlertRuleRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.Create(ctx, newAlertRule(), 1)
	require.NoError(t, err)

	dto, err := repo.Get(ctx, id)
	require.NoError(t, err, "存在的 ID 应命中")
	require.Equal(t, id, dto.GetId())
	require.Equal(t, monitorAlertV1.MonitorMetric_GO_GOROUTINES, dto.GetMetric())
	require.Equal(t, "ops@example.com", dto.GetTarget())

	_, err = repo.Get(ctx, 0)
	require.Error(t, err, "id=0 应返回 BadRequest")

	_, err = repo.Get(ctx, 9999999)
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
	require.Contains(t, err.Error(), "not found")
}

// TestMonitorAlertRuleRepoSqlite_ListAndListEnabled 验证分页 total 与
// ListEnabled 只回启用规则（评估器每轮只跑启用的，禁用规则混进去就是误报）；
// nil 请求返回 BadRequest。
func TestMonitorAlertRuleRepoSqlite_ListAndListEnabled(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewMonitorAlertRuleRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	idOn1, err := repo.Create(ctx, newAlertRule(), 1)
	require.NoError(t, err)
	idOn2, err := repo.Create(ctx, newAlertRule(), 1)
	require.NoError(t, err)
	idOff, err := repo.Create(ctx, &monitorAlertV1.MonitorAlertRule{
		Name:      trans.Ptr("停用规则"),
		Metric:    monitorAlertV1.MonitorMetric_REDIS_DB_SIZE.Enum(),
		Op:        monitorAlertV1.AlertOp_GE.Enum(),
		Threshold: trans.Ptr(1000.0),
		Channel:   notificationV1.Channel_WEBHOOK.Enum(),
		Target:    trans.Ptr("https://hooks.example.com/x"),
		IsEnabled: trans.Ptr(false),
	}, 1)
	require.NoError(t, err)

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(3), all.GetTotal())
	require.Len(t, all.GetItems(), 3)

	enabled, err := repo.ListEnabled(ctx)
	require.NoError(t, err)
	require.Len(t, enabled, 2, "ListEnabled 只应回启用规则")
	enabledIDs := map[uint64]bool{}
	for _, it := range enabled {
		enabledIDs[it.GetId()] = true
	}
	require.True(t, enabledIDs[idOn1])
	require.True(t, enabledIDs[idOn2])
	require.False(t, enabledIDs[idOff], "禁用规则不得进评估器的每轮清单")

	// 把停用规则启用后，ListEnabled 应把它捞进来（掩码更新 is_enabled 生效的旁证）
	require.NoError(t, repo.Update(ctx, &monitorAlertV1.UpdateMonitorAlertRuleRequest{
		Id:         idOff,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"is_enabled"}},
		Data:       &monitorAlertV1.MonitorAlertRule{IsEnabled: trans.Ptr(true)},
	}, 2))
	enabled, err = repo.ListEnabled(ctx)
	require.NoError(t, err)
	require.Len(t, enabled, 3, "启用后规则应进入评估清单")

	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestMonitorAlertRuleRepoSqlite_Update 验证掩码语义：掩码内更新（含枚举经 converter
// 落库）、掩码外保持原值、updated_by 盖操作人；评估状态三列是掩码之外的专属写路径，
// 管理员编辑（含全字段载荷）永不该碰它们；不存在 ID 静默无效果。
func TestMonitorAlertRuleRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewMonitorAlertRuleRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.Create(ctx, newAlertRule(), 1001)
	require.NoError(t, err)

	// 先由评估器回写一轮状态，验证随后的掩码更新不碰它
	alertedAt := time.Now().Add(-time.Hour)
	require.NoError(t, repo.MarkFiring(ctx, id, true, trans.Ptr(620.0), &alertedAt))

	// 管理员把阈值与渠道改掉（掩码内），metric/op/target 等掩码外保持原值；
	// Data 里塞了 last_firing/last_value 也必须被无视
	require.NoError(t, repo.Update(ctx, &monitorAlertV1.UpdateMonitorAlertRuleRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"threshold", "channel"}},
		Data: &monitorAlertV1.MonitorAlertRule{
			Threshold:  trans.Ptr(300.0),
			Channel:    notificationV1.Channel_WEBHOOK.Enum(),
			Metric:     monitorAlertV1.MonitorMetric_REDIS_DB_SIZE.Enum(), // 掩码外
			Target:     trans.Ptr("hacked@example.com"),                   // 掩码外
			LastFiring: trans.Ptr(false),                                  // 掩码外（专属写路径）
			LastValue:  trans.Ptr(0.0),                                    // 掩码外（专属写路径）
		},
	}, 2001))

	row, err := entClient.Client().MonitorAlertRule.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 300.0, *row.Threshold, "掩码内的阈值应更新")
	require.Equal(t, monitoralertrule.ChannelWebhook, *row.Channel, "掩码内的渠道应经 converter 更新")
	require.Equal(t, monitoralertrule.MetricGoGoroutines, *row.Metric, "掩码外的指标不得被改走")
	require.Equal(t, "ops@example.com", *row.Target, "掩码外的投递目标应保持原值")
	require.True(t, row.LastFiring, "管理员掩码更新不得触碰评估回写的 last_firing")
	require.Equal(t, 620.0, *row.LastValue, "管理员掩码更新不得触碰评估回写的 last_value")
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, uint64(2001), *row.UpdatedBy, "operatorID 应落 updated_by")
	require.NotNil(t, row.UpdatedAt, "更新应写入 updated_at")

	// 不存在 ID：无效果、不报错
	require.NoError(t, repo.Update(ctx, &monitorAlertV1.UpdateMonitorAlertRuleRequest{
		Id:         987654,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"threshold"}},
		Data:       &monitorAlertV1.MonitorAlertRule{Threshold: trans.Ptr(1.0)},
	}, 1))
	cnt, err := entClient.Client().MonitorAlertRule.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "对不存在 ID 的更新不应新增行")

	// 入参守卫
	require.Error(t, repo.Update(ctx, nil, 1), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &monitorAlertV1.UpdateMonitorAlertRuleRequest{Id: id}, 1),
		"nil Data 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &monitorAlertV1.UpdateMonitorAlertRuleRequest{
		Id:   0,
		Data: &monitorAlertV1.MonitorAlertRule{},
	}, 1), "id=0 应返回 BadRequest")
}

// TestMonitorAlertRuleRepoSqlite_MarkFiring 验证评估器回写：firing=true 时状态三列落位；
// firing=false（恢复）时 last_value/last_alerted_at 保持上次值——SetNillable 对 nil 是
// no-op，恢复不等于清场，"上次什么时候告的警"是冷却判断的锚，评估器要清场得显式传值。
func TestMonitorAlertRuleRepoSqlite_MarkFiring(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewMonitorAlertRuleRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	id, err := repo.Create(ctx, newAlertRule(), 1)
	require.NoError(t, err)

	alertedAt := time.Now()
	require.NoError(t, repo.MarkFiring(ctx, id, true, trans.Ptr(92.5), &alertedAt), "首次越限回写应成功")
	row, err := entClient.Client().MonitorAlertRule.Get(ctx, id)
	require.NoError(t, err)
	require.True(t, row.LastFiring)
	require.NotNil(t, row.LastValue)
	require.Equal(t, 92.5, *row.LastValue, "本次指标值应落库")
	require.NotNil(t, row.LastAlertedAt)
	require.WithinDuration(t, alertedAt, *row.LastAlertedAt, 2*time.Second, "告警时间（冷却锚）应落库")

	// 恢复：firing=false、值与时间传 nil → 只翻 last_firing，锚列保持上次值
	require.NoError(t, repo.MarkFiring(ctx, id, false, nil, nil))
	row, err = entClient.Client().MonitorAlertRule.Get(ctx, id)
	require.NoError(t, err)
	require.False(t, row.LastFiring, "恢复态应如实翻转")
	require.Equal(t, 92.5, *row.LastValue, "nil last_value 不得清掉上次指标值")
	require.NotNil(t, row.LastAlertedAt, "nil last_alerted_at 不得清掉冷却锚")

	// 再次越限：新值与新时间覆盖旧锚（冷却间隔从新告警起算）
	nextAlert := alertedAt.Add(10 * time.Minute)
	require.NoError(t, repo.MarkFiring(ctx, id, true, trans.Ptr(95.0), &nextAlert))
	row, err = entClient.Client().MonitorAlertRule.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 95.0, *row.LastValue)
	require.WithinDuration(t, nextAlert, *row.LastAlertedAt, 2*time.Second)

	// 不存在 ID：回写应报错而非静默（评估器据此知道规则已被删除）
	err = repo.MarkFiring(ctx, 9999999, true, trans.Ptr(1.0), &alertedAt)
	require.Error(t, err)
	require.Contains(t, err.Error(), "update monitor alert rule failed")
}

// TestMonitorAlertRuleRepoSqlite_Delete 验证删除指定行、id=0 的 BadRequest、
// 不存在 ID 返回错误；删除后评估器下一轮自然少跑一条（Get 回 NotFound）。
func TestMonitorAlertRuleRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewMonitorAlertRuleRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	idA, err := repo.Create(ctx, newAlertRule(), 1)
	require.NoError(t, err)
	idB, err := repo.Create(ctx, newAlertRule(), 1)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, idA), "按存在 ID 删除应成功")

	_, err = repo.Get(ctx, idA)
	require.Error(t, err, "删除后 Get 应回 NotFound")

	cnt, err := entClient.Client().MonitorAlertRule.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "只该删掉一行")

	_, err = repo.Get(ctx, idB)
	require.NoError(t, err, "未删除的行应仍在")

	require.Error(t, repo.Delete(ctx, 0), "id=0 应返回 BadRequest")
	require.Error(t, repo.Delete(ctx, 9999999), "删除不存在的 ID 应返回错误")
	cnt, err = entClient.Client().MonitorAlertRule.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "失败删除不应影响行数")
}
