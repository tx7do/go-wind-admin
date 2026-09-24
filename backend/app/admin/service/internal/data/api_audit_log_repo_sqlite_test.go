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
	entApiAuditLog "go-wind-admin/app/admin/service/internal/data/ent/apiauditlog"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newApiAuditLogRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的 ApiAuditLogRepo，
// 逐字段复刻 NewApiAuditLogRepo 的 mapper/converter 初始化，再调用 init()。
func newApiAuditLogRepoSqlite(t *testing.T) *ApiAuditLogRepo {
	t.Helper()
	repo := &ApiAuditLogRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[auditV1.ApiAuditLog, ent.ApiAuditLog](),
	}
	repo.init()
	return repo
}

// TestApiAuditLogRepoSqlite_Create 通过 repo.Create 写入一条含全部标量字段的
// API 审计日志，ent client 直查断言各字段按请求落库。
func TestApiAuditLogRepoSqlite_Create(t *testing.T) {
	repo := newApiAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &auditV1.CreateApiAuditLogRequest{
		Data: &auditV1.ApiAuditLog{
			Username:       trans.Ptr("sqlite_user_alpha"),
			IpAddress:      trans.Ptr("10.0.0.1"),
			Referer:        trans.Ptr("https://referer.example/sqlite"),
			AppVersion:     trans.Ptr("1.0.0-sqlite"),
			HttpMethod:     trans.Ptr("GET"),
			Path:           trans.Ptr("/sqlite/api-audit/create"),
			RequestUri:     trans.Ptr("/sqlite/api-audit/create?q=1"),
			ApiModule:      trans.Ptr("sqlite-module"),
			ApiOperation:   trans.Ptr("sqlite-op"),
			ApiDescription: trans.Ptr("sqlite 描述"),
			RequestId:      trans.Ptr("req-sqlite-create-1"),
			TraceId:        trans.Ptr("trace-sqlite-create-1"),
			SpanId:         trans.Ptr("span-sqlite-create-1"),
			LatencyMs:      trans.Ptr(uint32(42)),
			Success:        trans.Ptr(true),
			StatusCode:     trans.Ptr(uint32(200)),
			Reason:         trans.Ptr("ok"),
			RequestHeader:  trans.Ptr(`{"X-Sqlite":"1"}`),
			RequestBody:    trans.Ptr(`{"k":"v"}`),
			Response:       trans.Ptr(`{"r":"v"}`),
			LogHash:        trans.Ptr("hash-sqlite-create-1"),
			Signature:      []byte{0x01, 0x02, 0x03},
		},
	})
	require.NoError(t, err, "repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().ApiAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "sys_api_audit_logs 应有 1 条记录")
	row := rows[0]
	require.Equal(t, "sqlite_user_alpha", *row.Username, "username 应按请求落库")
	require.Equal(t, "10.0.0.1", *row.IPAddress, "ip_address 应按请求落库")
	require.Equal(t, "https://referer.example/sqlite", *row.Referer, "referer 应按请求落库")
	require.Equal(t, "1.0.0-sqlite", *row.AppVersion, "app_version 应按请求落库")
	require.Equal(t, "GET", *row.HTTPMethod, "http_method 应按请求落库")
	require.Equal(t, "/sqlite/api-audit/create", *row.Path, "path 应按请求落库")
	require.Equal(t, "/sqlite/api-audit/create?q=1", *row.RequestURI, "request_uri 应按请求落库")
	require.Equal(t, "sqlite-module", *row.APIModule, "api_module 应按请求落库")
	require.Equal(t, "sqlite-op", *row.APIOperation, "api_operation 应按请求落库")
	require.Equal(t, "sqlite 描述", *row.APIDescription, "api_description 应按请求落库")
	require.Equal(t, "req-sqlite-create-1", *row.RequestID, "request_id 应按请求落库")
	require.Equal(t, "trace-sqlite-create-1", *row.TraceID, "trace_id 应按请求落库")
	require.Equal(t, "span-sqlite-create-1", *row.SpanID, "span_id 应按请求落库")
	require.Equal(t, uint32(42), *row.LatencyMs, "latency_ms 应按请求落库")
	require.Equal(t, true, *row.Success, "success 应按请求落库")
	require.Equal(t, uint32(200), *row.StatusCode, "status_code 应按请求落库")
	require.Equal(t, "ok", *row.Reason, "reason 应按请求落库")
	require.Equal(t, `{"X-Sqlite":"1"}`, *row.RequestHeader, "request_header 应按请求落库")
	require.Equal(t, `{"k":"v"}`, *row.RequestBody, "request_body 应按请求落库")
	require.Equal(t, `{"r":"v"}`, *row.Response, "response 应按请求落库")
	require.Equal(t, "hash-sqlite-create-1", *row.LogHash, "log_hash 应按请求落库")
	require.NotNil(t, row.Signature, "signature 应按请求落库")
	require.Equal(t, []byte{0x01, 0x02, 0x03}, *row.Signature, "signature 应按请求落库")
	require.False(t, row.CreatedAt.IsZero(), "created_at 应由 repo 写入")
}

// TestApiAuditLogRepoSqlite_ListFilterAndPaging 验证 List 的无过滤全量、
// username 列 contains 模糊搜索（仓规：文本列一律 contains）、
// id 列等值过滤与 page/page_size 分页语义。
func TestApiAuditLogRepoSqlite_ListFilterAndPaging(t *testing.T) {
	repo := newApiAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for i, marker := range []string{"MARKERALPHA", "MARKERBETA"} {
		require.NoError(t, repo.Create(ctx, &auditV1.CreateApiAuditLogRequest{
			Data: &auditV1.ApiAuditLog{
				Username:  trans.Ptr(marker + " 用户"),
				RequestId: trans.Ptr(fmt.Sprintf("req-sqlite-list-%d", i)),
				LogHash:   trans.Ptr(fmt.Sprintf("hash-sqlite-list-%d", i)),
			},
		}), "写入第 %d 行应成功", i)
	}

	rows, err := repo.entClient.Client().ApiAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2, "直查应有 2 行")
	firstID := rows[0].ID
	secondID := rows[1].ID
	require.NotEqual(t, firstID, secondID, "两行应有不同主键")

	// 无过滤：全部 2 行
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 条")

	// contains 过滤：仅命中含 MARKERALPHA 的行
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "username",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKERALPHA"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetUsername(), "MARKERALPHA", "命中行应为含标记的那条")

	// contains 无命中
	none, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "username",
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

	// ID 列等值过滤（仓规：ID 列不进模糊搜索，只做等值）
	byID, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "id",
						Op:         paginationV1.Operator_EQ,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: fmt.Sprintf("%d", firstID)},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), byID.Total, "id 等值过滤应只统计目标行")
	require.Len(t, byID.Items, 1, "id 等值过滤应只返回目标行")
	require.Equal(t, firstID, byID.Items[0].GetId(), "id 等值过滤命中行的 id 应与过滤值一致")

	// id 等值过滤：不存在的 ID 无命中
	byMissingID, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_FilterExpr{
			FilterExpr: &paginationV1.FilterExpr{
				Type: paginationV1.ExprType_AND,
				Conditions: []*paginationV1.FilterCondition{
					{
						Field:      "id",
						Op:         paginationV1.Operator_EQ,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "999999"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), byMissingID.Total, "不存在的 id 等值过滤应 Total=0")
	require.Empty(t, byMissingID.Items)

	// 分页语义：page=1/pageSize=1 与 page=2/pageSize=1 各取一行且互不重叠，
	// Total 恒为全量计数（分页不影响 Total）。
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
	require.NotEqual(t, page1.Items[0].GetId(), page2.Items[0].GetId(), "两页返回的行应不同")
	require.ElementsMatch(t, []uint32{firstID, secondID},
		[]uint32{page1.Items[0].GetId(), page2.Items[0].GetId()}, "两页合并应覆盖全部行")
}

// TestApiAuditLogRepoSqlite_Get 验证 Get 按主键的命中与未命中。
func TestApiAuditLogRepoSqlite_Get(t *testing.T) {
	repo := newApiAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &auditV1.CreateApiAuditLogRequest{
		Data: &auditV1.ApiAuditLog{
			Username:  trans.Ptr("sqlite_user_get"),
			Path:      trans.Ptr("/sqlite/api-audit/get"),
			RequestId: trans.Ptr("req-sqlite-get-1"),
			LogHash:   trans.Ptr("hash-sqlite-get-1"),
		},
	}))

	rows, err := repo.entClient.Client().ApiAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	// 命中：按主键
	got, err := repo.Get(ctx, &auditV1.GetApiAuditLogRequest{
		QueryBy: &auditV1.GetApiAuditLogRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, got.GetId(), "命中记录的 id 应与主键一致")
	require.Equal(t, "sqlite_user_get", got.GetUsername(), "命中记录的 username 应与写入一致")
	require.Equal(t, "/sqlite/api-audit/get", got.GetPath(), "命中记录的 path 应与写入一致")

	// 未命中：不存在的主键
	_, err = repo.Get(ctx, &auditV1.GetApiAuditLogRequest{
		QueryBy: &auditV1.GetApiAuditLogRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")
}

// TestApiAuditLogRepoSqlite_CountAndIsExist 验证 Count 的带谓词/无谓词语义
// 与 IsExist 的命中/未命中。
func TestApiAuditLogRepoSqlite_CountAndIsExist(t *testing.T) {
	repo := newApiAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &auditV1.CreateApiAuditLogRequest{
		Data: &auditV1.ApiAuditLog{
			Username:  trans.Ptr("sqlite_user_count"),
			RequestId: trans.Ptr("req-sqlite-count-1"),
			LogHash:   trans.Ptr("hash-sqlite-count-1"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &auditV1.CreateApiAuditLogRequest{
		Data: &auditV1.ApiAuditLog{
			Username:  trans.Ptr("sqlite_user_count_other"),
			RequestId: trans.Ptr("req-sqlite-count-2"),
			LogHash:   trans.Ptr("hash-sqlite-count-2"),
		},
	}))

	rows, err := repo.entClient.Client().ApiAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	createdID := rows[0].ID

	// 无谓词：全量计数
	total, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total, "无谓词 Count 应为全量 2")

	// 主键等值谓词：只命中目标行
	byID, err := repo.Count(ctx, []func(s *sql.Selector){entApiAuditLog.IDEQ(createdID)})
	require.NoError(t, err)
	require.Equal(t, 1, byID, "主键等值谓词应只命中 1 行")

	byMissingID, err := repo.Count(ctx, []func(s *sql.Selector){entApiAuditLog.IDEQ(99999)})
	require.NoError(t, err)
	require.Zero(t, byMissingID, "不存在的主键谓词应命中 0 行")

	// 文本列等值谓词
	byUsername, err := repo.Count(ctx, []func(s *sql.Selector){entApiAuditLog.UsernameEQ("sqlite_user_count")})
	require.NoError(t, err)
	require.Equal(t, 1, byUsername, "username 等值谓词应只命中 1 行")

	// IsExist 命中/未命中
	exist, err := repo.IsExist(ctx, createdID)
	require.NoError(t, err)
	require.True(t, exist, "存在的主键 IsExist 应为 true")
	exist, err = repo.IsExist(ctx, 99999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的主键 IsExist 应为 false")
}
