package data

import (
	"context"
	"fmt"
	"testing"

	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	auditV1 "go-wind-admin/api/gen/go/audit/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entDataAccessAuditLog "go-wind-admin/app/admin/service/internal/data/ent/dataaccessauditlog"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newDataAccessAuditLogRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的
// DataAccessAuditLogRepo，逐字段复刻 NewDataAccessAuditLogRepo 的
// mapper/converter 初始化，再调用 init()。
func newDataAccessAuditLogRepoSqlite(t *testing.T) *DataAccessAuditLogRepo {
	t.Helper()
	repo := &DataAccessAuditLogRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[auditV1.DataAccessAuditLog, ent.DataAccessAuditLog](),
		accessTypeConverter: mapper.NewEnumTypeConverter[auditV1.DataAccessAuditLog_AccessType, entDataAccessAuditLog.AccessType](
			auditV1.DataAccessAuditLog_AccessType_name, auditV1.DataAccessAuditLog_AccessType_value,
		),
		sensitiveLevelConverter: mapper.NewEnumTypeConverter[auditV1.SensitiveLevel, entDataAccessAuditLog.SensitiveLevel](
			auditV1.SensitiveLevel_name, auditV1.SensitiveLevel_value,
		),
	}
	repo.init()
	return repo
}

// TestDataAccessAuditLogRepoSqlite_Create 通过 repo.Create 写入一条含全部标量字段的
// 数据访问审计日志，ent client 直查断言各字段按请求落库。
func TestDataAccessAuditLogRepoSqlite_Create(t *testing.T) {
	repo := newDataAccessAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{
		Data: &auditV1.DataAccessAuditLog{
			Username:        trans.Ptr("sqlite_dal_user"),
			IpAddress:       trans.Ptr("10.0.0.2"),
			RequestId:       trans.Ptr("req-sqlite-dal-create-1"),
			DataSource:      trans.Ptr("mysql"),
			TableName:       trans.Ptr("sys_tenants"),
			DataId:          trans.Ptr("42"),
			AccessType:      auditV1.DataAccessAuditLog_SELECT.Enum(),
			SqlDigest:       trans.Ptr("digest-abc"),
			SqlText:         trans.Ptr("SELECT 1"),
			AffectedRows:    trans.Ptr(uint32(3)),
			LatencyMs:       trans.Ptr(uint32(11)),
			Success:         trans.Ptr(true),
			SensitiveLevel:  auditV1.SensitiveLevel_INTERNAL.Enum(),
			DataMasked:      trans.Ptr(false),
			MaskingRules:    trans.Ptr("rule-1"),
			BusinessPurpose: trans.Ptr("测试用途"),
			DataCategory:    trans.Ptr("category-1"),
			DbUser:          trans.Ptr("svc_user"),
			LogHash:         trans.Ptr("hash-sqlite-dal-create-1"),
			Signature:       []byte{0x0a, 0x0b},
		},
	})
	require.NoError(t, err, "repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().DataAccessAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "sys_data_access_audit_logs 应有 1 条记录")
	row := rows[0]
	require.Equal(t, "sqlite_dal_user", *row.Username, "username 应按请求落库")
	require.Equal(t, "10.0.0.2", *row.IPAddress, "ip_address 应按请求落库")
	require.Equal(t, "req-sqlite-dal-create-1", *row.RequestID, "request_id 应按请求落库")
	require.Equal(t, "mysql", *row.DataSource, "data_source 应按请求落库")
	require.Equal(t, "sys_tenants", *row.TableName, "table_name 应按请求落库")
	require.Equal(t, "42", *row.DataID, "data_id 应按请求落库")
	require.NotNil(t, row.AccessType, "access_type 枚举应经 converter 落库")
	require.Equal(t, entDataAccessAuditLog.AccessTypeSelect, *row.AccessType,
		"proto ACCESS_TYPE_SELECT 应映射为 ent AccessTypeSelect")
	require.Equal(t, "digest-abc", *row.SQLDigest, "sql_digest 应按请求落库")
	require.Equal(t, "SELECT 1", *row.SQLText, "sql_text 应按请求落库")
	require.Equal(t, uint32(3), *row.AffectedRows, "affected_rows 应按请求落库")
	require.Equal(t, uint32(11), *row.LatencyMs, "latency_ms 应按请求落库")
	require.Equal(t, true, *row.Success, "success 应按请求落库")
	require.NotNil(t, row.SensitiveLevel, "sensitive_level 枚举应经 converter 落库")
	require.Equal(t, entDataAccessAuditLog.SensitiveLevelInternal, *row.SensitiveLevel,
		"proto SENSITIVE_LEVEL_INTERNAL 应映射为 ent SensitiveLevelInternal")
	require.Equal(t, false, *row.DataMasked, "data_masked 应按请求落库")
	require.Equal(t, "rule-1", *row.MaskingRules, "masking_rules 应按请求落库")
	require.Equal(t, "测试用途", *row.BusinessPurpose, "business_purpose 应按请求落库")
	require.Equal(t, "category-1", *row.DataCategory, "data_category 应按请求落库")
	require.Equal(t, "svc_user", *row.DbUser, "db_user 应按请求落库")
	require.Equal(t, "hash-sqlite-dal-create-1", *row.LogHash, "log_hash 应按请求落库")
	require.NotNil(t, row.Signature, "signature 应按请求落库")
	require.Equal(t, []byte{0x0a, 0x0b}, *row.Signature, "signature 应按请求落库")
}

// TestDataAccessAuditLogRepoSqlite_AccessTypeEnumPairs 对 access_type 枚举的全部
// 有效非零取值逐一建行，断言 proto → ent 与 ent → proto（List 路径）的双向映射逐对成立。
func TestDataAccessAuditLogRepoSqlite_AccessTypeEnumPairs(t *testing.T) {
	repo := newDataAccessAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	expectedEnt := map[string]int{}
	expectedProto := map[int32]int{}
	serial := 0
	for value, name := range auditV1.DataAccessAuditLog_AccessType_name {
		if value == 0 {
			continue
		}
		serial++
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{
			Data: &auditV1.DataAccessAuditLog{
				DataId:     trans.Ptr(fmt.Sprintf("enum-pair-dal-%d", serial)),
				AccessType: auditV1.DataAccessAuditLog_AccessType(value).Enum(),
			},
		}), "access_type=%s 建行应成功", name)
	}
	require.NotEmpty(t, expectedEnt, "应存在有效非零枚举值")

	// ent 侧：每个映射名应恰好出现一次
	entRows, err := repo.entClient.Client().DataAccessAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, entRows, len(expectedEnt), "应每个枚举取值各落库一行")
	actualEnt := map[string]int{}
	for _, row := range entRows {
		require.NotNil(t, row.AccessType, "access_type 不应为空")
		actualEnt[string(*row.AccessType)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 access_type 取值分布应与枚举名集合逐对一致")

	// DTO 侧（List 经 mapper/converter 回填）：proto 枚举值应逐一还原
	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(len(expectedProto)), listed.Total, "List 应返回全部枚举行")
	require.Len(t, listed.Items, len(expectedProto))
	actualProto := map[int32]int{}
	for _, item := range listed.Items {
		require.NotNil(t, item.AccessType, "DTO 的 access_type 应被 converter 回填")
		actualProto[int32(*item.AccessType)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 access_type 取值分布应与 proto 枚举值集合逐对一致")
}

// TestDataAccessAuditLogRepoSqlite_SensitiveLevelEnumPairs 对 sensitive_level 枚举的
// 全部有效非零取值逐一建行，断言双向映射逐对成立。
func TestDataAccessAuditLogRepoSqlite_SensitiveLevelEnumPairs(t *testing.T) {
	repo := newDataAccessAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	expectedEnt := map[string]int{}
	expectedProto := map[int32]int{}
	serial := 0
	for value, name := range auditV1.SensitiveLevel_name {
		if value == 0 {
			continue
		}
		serial++
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{
			Data: &auditV1.DataAccessAuditLog{
				DataId:         trans.Ptr(fmt.Sprintf("sensitive-pair-dal-%d", serial)),
				SensitiveLevel: auditV1.SensitiveLevel(value).Enum(),
			},
		}), "sensitive_level=%s 建行应成功", name)
	}
	require.NotEmpty(t, expectedEnt, "应存在有效非零枚举值")

	entRows, err := repo.entClient.Client().DataAccessAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, entRows, len(expectedEnt), "应每个枚举取值各落库一行")
	actualEnt := map[string]int{}
	for _, row := range entRows {
		require.NotNil(t, row.SensitiveLevel, "sensitive_level 不应为空")
		actualEnt[string(*row.SensitiveLevel)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 sensitive_level 取值分布应与枚举名集合逐对一致")

	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(len(expectedProto)), listed.Total, "List 应返回全部枚举行")
	require.Len(t, listed.Items, len(expectedProto))
	actualProto := map[int32]int{}
	for _, item := range listed.Items {
		require.NotNil(t, item.SensitiveLevel, "DTO 的 sensitive_level 应被 converter 回填")
		actualProto[int32(*item.SensitiveLevel)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 sensitive_level 取值分布应与 proto 枚举值集合逐对一致")
}

// TestDataAccessAuditLogRepoSqlite_ListFilterAndPaging 验证 List 的无过滤全量、
// table_name 列 contains 模糊搜索、id 列等值过滤与分页语义。
func TestDataAccessAuditLogRepoSqlite_ListFilterAndPaging(t *testing.T) {
	repo := newDataAccessAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for i, marker := range []string{"MARKERGAMMA", "MARKERDELTA"} {
		require.NoError(t, repo.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{
			Data: &auditV1.DataAccessAuditLog{
				TableName: trans.Ptr(marker + "_table"),
				DataId:    trans.Ptr(fmt.Sprintf("list-dal-%d", i)),
			},
		}), "写入第 %d 行应成功", i)
	}

	rows, err := repo.entClient.Client().DataAccessAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	firstID := rows[0].ID
	secondID := rows[1].ID

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2)

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "table_name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKERGAMMA"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetTableName(), "MARKERGAMMA", "命中行应为含标记的那条")

	byID, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "id",
						Op:         paginationV1.Operator_EQ,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: fmt.Sprintf("%d", secondID)},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), byID.Total, "id 等值过滤应只统计目标行")
	require.Len(t, byID.Items, 1, "id 等值过滤应只返回目标行")
	require.Equal(t, secondID, byID.Items[0].GetId())

	page1, err := repo.List(ctx, &paginationV1.PagingRequest{
		Page:     trans.Ptr(uint32(1)),
		PageSize: trans.Ptr(uint32(1)),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), page1.Total, "分页时 Total 应仍为全量 2")
	require.Len(t, page1.Items, 1, "pageSize=1 第一页应只含 1 行")

	page2, err := repo.List(ctx, &paginationV1.PagingRequest{
		Page:     trans.Ptr(uint32(2)),
		PageSize: trans.Ptr(uint32(1)),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(2), page2.Total, "分页时 Total 应仍为全量 2")
	require.Len(t, page2.Items, 1, "pageSize=1 第二页应只含 1 行")
	require.ElementsMatch(t, []uint32{firstID, secondID},
		[]uint32{page1.Items[0].GetId(), page2.Items[0].GetId()}, "两页合并应覆盖全部行")
}

// TestDataAccessAuditLogRepoSqlite_Get 验证 Get 按主键的命中与未命中。
func TestDataAccessAuditLogRepoSqlite_Get(t *testing.T) {
	repo := newDataAccessAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{
		Data: &auditV1.DataAccessAuditLog{
			DataSource: trans.Ptr("redis"),
			TableName:  trans.Ptr("cache:session"),
			DataId:     trans.Ptr("get-dal-1"),
			AccessType: auditV1.DataAccessAuditLog_VIEW.Enum(),
		},
	}))

	rows, err := repo.entClient.Client().DataAccessAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	got, err := repo.Get(ctx, &auditV1.GetDataAccessAuditLogRequest{
		QueryBy: &auditV1.GetDataAccessAuditLogRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, got.GetId())
	require.Equal(t, "redis", got.GetDataSource(), "命中记录的 data_source 应与写入一致")
	require.Equal(t, auditV1.DataAccessAuditLog_VIEW, got.GetAccessType(),
		"命中记录的 access_type 应经 converter 还原为 VIEW")

	_, err = repo.Get(ctx, &auditV1.GetDataAccessAuditLogRequest{
		QueryBy: &auditV1.GetDataAccessAuditLogRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")
}

// TestDataAccessAuditLogRepoSqlite_CountAndIsExist 验证 Count 的带谓词/无谓词语义
// 与 IsExist 的命中/未命中。
func TestDataAccessAuditLogRepoSqlite_CountAndIsExist(t *testing.T) {
	repo := newDataAccessAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{
		Data: &auditV1.DataAccessAuditLog{
			DataSource: trans.Ptr("mongodb"),
			DataId:     trans.Ptr("count-dal-1"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &auditV1.CreateDataAccessAuditLogRequest{
		Data: &auditV1.DataAccessAuditLog{
			DataSource: trans.Ptr("elasticsearch"),
			DataId:     trans.Ptr("count-dal-2"),
		},
	}))

	rows, err := repo.entClient.Client().DataAccessAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	createdID := rows[0].ID

	total, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total, "无谓词 Count 应为全量 2")

	byID, err := repo.Count(ctx, []func(s *sql.Selector){entDataAccessAuditLog.IDEQ(createdID)})
	require.NoError(t, err)
	require.Equal(t, 1, byID, "主键等值谓词应只命中 1 行")

	byDataSource, err := repo.Count(ctx, []func(s *sql.Selector){entDataAccessAuditLog.DataSourceEQ("mongodb")})
	require.NoError(t, err)
	require.Equal(t, 1, byDataSource, "data_source 等值谓词应只命中 1 行")

	exist, err := repo.IsExist(ctx, createdID)
	require.NoError(t, err)
	require.True(t, exist, "存在的主键 IsExist 应为 true")
	exist, err = repo.IsExist(ctx, 99999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的主键 IsExist 应为 false")
}
