package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"

	scriptV1 "go-wind-admin/api/gen/go/script/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/predicate"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newScriptLogRepoSqlite 在给定 enttest client 上白盒构造 ScriptLogRepo，
// 逐字段复刻 NewScriptLogRepo 的 mapper/repository/converter 内联装配
// （该构造器无 init()，repository 直接内联构造）。
func newScriptLogRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *ScriptLogRepo {
	t.Helper()
	repo := &ScriptLogRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[scriptV1.ScriptLog, ent.ScriptLog](),
	}
	repo.repository = entCrud.NewRepository[
		ent.ScriptLogQuery, ent.ScriptLogSelect,
		ent.ScriptLogCreate, ent.ScriptLogCreateBulk,
		ent.ScriptLogUpdate, ent.ScriptLogUpdateOne,
		ent.ScriptLogDelete,
		predicate.ScriptLog,
		scriptV1.ScriptLog, ent.ScriptLog,
	](repo.mapper)

	repo.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	repo.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	return repo
}

// TestScriptLogRepoSqlite_RecordAndList 验证 Record 落库、CountLog 计数、
// List 全量返回与 contains 过滤语义。
func TestScriptLogRepoSqlite_RecordAndList(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newScriptLogRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	repo.Record(ctx, ScriptLogRecord{
		ScriptID:   0,
		ScriptName: "markerqaz_script_a",
		Language:   "LUA",
		Trigger:    "manual",
		HookPoint:  "",
		Version:    1,
		Success:    true,
		DurationMS: 5,
	})
	repo.Record(ctx, ScriptLogRecord{
		ScriptID:   0,
		ScriptName: "unrelated_script_b",
		Language:   "LUA",
		Trigger:    "manual",
		HookPoint:  "",
		Version:    1,
		Success:    false,
		DurationMS: 7,
	})

	// CountLog：全部行数
	countResp, err := repo.CountLog(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), countResp.Count, "Record 两行后 CountLog 应为 2")

	// 直查确认两行均带请求字段
	rows, err := entClient.Client().ScriptLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	for _, row := range rows {
		require.Equal(t, "LUA", *row.Language, "language 应按记录落库")
		require.Equal(t, "manual", *row.TriggerType, "trigger_type 应按记录落库")
	}

	// List 全量
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤 List 应返回全部 2 行")
	require.Len(t, all.Items, 2)

	// List contains 过滤：只命中携带标记的行
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "script_name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "markerqaz"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤后 Total 应为 1")
	require.Len(t, filtered.Items, 1, "contains 过滤后只应返回 1 行")

	// contains 无命中
	none, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "script_name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "no-such-marker-zzz"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), none.Total, "无命中 contains 应返回 Total=0")
	require.Empty(t, none.Items)
}

// TestScriptLogRepoSqlite_Purge 验证滚动清理：
// 早于阈值时间的行被删、晚于阈值时间的行保留。
func TestScriptLogRepoSqlite_Purge(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newScriptLogRepoSqlite(t, entClient)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	repo.Record(ctx, ScriptLogRecord{
		ScriptName: "purge_old_row",
		Language:   "LUA",
		Trigger:    "manual",
		Success:    true,
	})
	repo.Record(ctx, ScriptLogRecord{
		ScriptName: "purge_recent_row",
		Language:   "LUA",
		Trigger:    "manual",
		Success:    true,
	})

	// 阈值取过去时间：不删任何行
	deleted, err := repo.Purge(ctx, time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	require.Zero(t, deleted, "过去时间阈值不应删除任何行")
	remain, err := repo.CountLog(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), remain.Count, "过去时间阈值后仍应有 2 行")

	// 阈值取未来时间：全部删除
	deleted, err = repo.Purge(ctx, time.Now().Add(24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, uint64(2), deleted, "未来时间阈值应删除全部 2 行")
	after, err := repo.CountLog(ctx)
	require.NoError(t, err)
	require.Zero(t, after.Count, "清理后 CountLog 应为 0")
}
