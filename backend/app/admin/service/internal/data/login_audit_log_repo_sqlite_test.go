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
	entLoginAuditLog "go-wind-admin/app/admin/service/internal/data/ent/loginauditlog"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newLoginAuditLogRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的
// LoginAuditLogRepo，逐字段复刻 NewLoginAuditLogRepo 的 mapper/converter 初始化，
// 再调用 init()。
func newLoginAuditLogRepoSqlite(t *testing.T) *LoginAuditLogRepo {
	t.Helper()
	repo := &LoginAuditLogRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[auditV1.LoginAuditLog, ent.LoginAuditLog](),
		statusConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_Status, entLoginAuditLog.Status](
			auditV1.LoginAuditLog_Status_name, auditV1.LoginAuditLog_Status_value,
		),
		actionTypeConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_ActionType, entLoginAuditLog.ActionType](
			auditV1.LoginAuditLog_ActionType_name, auditV1.LoginAuditLog_ActionType_value,
		),
		riskLevelConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_RiskLevel, entLoginAuditLog.RiskLevel](
			auditV1.LoginAuditLog_RiskLevel_name, auditV1.LoginAuditLog_RiskLevel_value,
		),
		loginMethodConverter: mapper.NewEnumTypeConverter[auditV1.LoginAuditLog_LoginMethod, entLoginAuditLog.LoginMethod](
			auditV1.LoginAuditLog_LoginMethod_name, auditV1.LoginAuditLog_LoginMethod_value,
		),
	}
	repo.init()
	return repo
}

// TestLoginAuditLogRepoSqlite_Create 通过 repo.Create 写入一条含全部标量字段的
// 登录审计日志，ent client 直查断言各字段按请求落库。
func TestLoginAuditLogRepoSqlite_Create(t *testing.T) {
	repo := newLoginAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{
		Data: &auditV1.LoginAuditLog{
			UserId:        trans.Ptr(uint32(7)),
			Username:      trans.Ptr("sqlite_login_user"),
			IpAddress:     trans.Ptr("10.0.0.3"),
			SessionId:     trans.Ptr("sess-sqlite-1"),
			RequestId:     trans.Ptr("req-sqlite-login-create-1"),
			TraceId:       trans.Ptr("trace-sqlite-login-create-1"),
			ActionType:    auditV1.LoginAuditLog_LOGIN.Enum(),
			Status:        auditV1.LoginAuditLog_FAILED.Enum(),
			LoginMethod:   auditV1.LoginAuditLog_PASSWORD.Enum(),
			FailureReason: trans.Ptr("密码错误"),
			MfaStatus:     trans.Ptr("NOT_REQUIRED"),
			RiskScore:     trans.Ptr(uint32(35)),
			RiskLevel:     auditV1.LoginAuditLog_MEDIUM.Enum(),
			RiskFactors:   []string{"weak_password", "new_device"},
			LogHash:       trans.Ptr("hash-sqlite-login-create-1"),
			Signature:     []byte{0x0c},
		},
	})
	require.NoError(t, err, "repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().LoginAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "sys_login_audit_logs 应有 1 条记录")
	row := rows[0]
	require.Equal(t, uint32(7), *row.UserID, "user_id 应按请求落库")
	require.Equal(t, "sqlite_login_user", *row.Username, "username 应按请求落库")
	require.Equal(t, "10.0.0.3", *row.IPAddress, "ip_address 应按请求落库")
	require.Equal(t, "sess-sqlite-1", *row.SessionID, "session_id 应按请求落库")
	require.Equal(t, "req-sqlite-login-create-1", *row.RequestID, "request_id 应按请求落库")
	require.Equal(t, "trace-sqlite-login-create-1", *row.TraceID, "trace_id 应按请求落库")
	require.NotNil(t, row.ActionType, "action_type 枚举应经 converter 落库")
	require.Equal(t, entLoginAuditLog.ActionTypeLogin, *row.ActionType,
		"proto ACTION_TYPE_LOGIN 应映射为 ent ActionTypeLogin")
	require.NotNil(t, row.Status, "status 枚举应经 converter 落库")
	require.Equal(t, entLoginAuditLog.StatusFailed, *row.Status,
		"proto LOGIN_STATUS_FAILED 应映射为 ent StatusFailed")
	require.NotNil(t, row.LoginMethod, "login_method 枚举应经 converter 落库")
	require.Equal(t, entLoginAuditLog.LoginMethodPassword, *row.LoginMethod,
		"proto LOGIN_METHOD_PASSWORD 应映射为 ent LoginMethodPassword")
	require.Equal(t, "密码错误", *row.FailureReason, "failure_reason 应按请求落库")
	require.Equal(t, "NOT_REQUIRED", *row.MfaStatus, "mfa_status 应按请求落库")
	require.Equal(t, uint32(35), *row.RiskScore, "risk_score 应按请求落库")
	require.NotNil(t, row.RiskLevel, "risk_level 枚举应经 converter 落库")
	require.Equal(t, entLoginAuditLog.RiskLevelMedium, *row.RiskLevel,
		"proto RISK_LEVEL_MEDIUM 应映射为 ent RiskLevelMedium")
	require.Equal(t, []string{"weak_password", "new_device"}, row.RiskFactors, "risk_factors 应按请求落库")
	require.Equal(t, "hash-sqlite-login-create-1", *row.LogHash, "log_hash 应按请求落库")
	require.NotNil(t, row.Signature, "signature 应按请求落库")
	require.Equal(t, []byte{0x0c}, *row.Signature, "signature 应按请求落库")
}

// TestLoginAuditLogRepoSqlite_EnumPairs 对 status / action_type / login_method /
// risk_level 四个枚举的全部有效非零取值逐一建行，断言 proto → ent 与
// ent → proto（List 路径）的双向映射逐对成立。
//
// 已知不对称：proto LoginMethod 含 FIDO2，ent 枚举未声明该值，
// 该取值在写入时被 ent 校验器拒绝（写入报错、不落行），属预期行为。
func TestLoginAuditLogRepoSqlite_EnumPairs(t *testing.T) {
	repo := newLoginAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	seeded := 0
	nextMarker := func() string {
		seeded++
		return fmt.Sprintf("login-enum-%d", seeded)
	}

	// —— status 枚举对 ——
	expectedEnt := map[string]int{}
	expectedProto := map[int32]int{}
	for value, name := range auditV1.LoginAuditLog_Status_name {
		if value == 0 {
			continue
		}
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{
			Data: &auditV1.LoginAuditLog{
				FailureReason: trans.Ptr(nextMarker()),
				Status:        auditV1.LoginAuditLog_Status(value).Enum(),
			},
		}), "status=%s 建行应成功", name)
	}
	entRows, err := repo.entClient.Client().LoginAuditLog.Query().All(ctx)
	require.NoError(t, err)
	actualEnt := map[string]int{}
	for _, row := range entRows {
		if row.Status == nil {
			continue
		}
		actualEnt[string(*row.Status)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 status 分布应与枚举名集合逐对一致")
	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto := map[int32]int{}
	for _, item := range listed.Items {
		if item.Status == nil {
			continue
		}
		actualProto[int32(*item.Status)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 status 分布应与 proto 枚举值集合逐对一致")

	// —— action_type 枚举对 ——
	expectedEnt = map[string]int{}
	expectedProto = map[int32]int{}
	for value, name := range auditV1.LoginAuditLog_ActionType_name {
		if value == 0 {
			continue
		}
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{
			Data: &auditV1.LoginAuditLog{
				FailureReason: trans.Ptr(nextMarker()),
				ActionType:    auditV1.LoginAuditLog_ActionType(value).Enum(),
			},
		}), "action_type=%s 建行应成功", name)
	}
	entRows, err = repo.entClient.Client().LoginAuditLog.Query().All(ctx)
	require.NoError(t, err)
	actualEnt = map[string]int{}
	for _, row := range entRows {
		if row.ActionType == nil {
			continue
		}
		actualEnt[string(*row.ActionType)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 action_type 分布应与枚举名集合逐对一致")
	listed, err = repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto = map[int32]int{}
	for _, item := range listed.Items {
		if item.ActionType == nil {
			continue
		}
		actualProto[int32(*item.ActionType)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 action_type 分布应与 proto 枚举值集合逐对一致")

	// —— login_method 枚举对 ——
	expectedEnt = map[string]int{}
	expectedProto = map[int32]int{}
	for value, name := range auditV1.LoginAuditLog_LoginMethod_name {
		if value == 0 {
			continue
		}
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{
			Data: &auditV1.LoginAuditLog{
				FailureReason: trans.Ptr(nextMarker()),
				LoginMethod:   auditV1.LoginAuditLog_LoginMethod(value).Enum(),
			},
		}), "login_method=%s 建行应成功", name)
	}
	entRows, err = repo.entClient.Client().LoginAuditLog.Query().All(ctx)
	require.NoError(t, err)
	actualEnt = map[string]int{}
	for _, row := range entRows {
		if row.LoginMethod == nil {
			continue
		}
		actualEnt[string(*row.LoginMethod)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 login_method 分布应与可落库枚举名集合逐对一致")
	listed, err = repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto = map[int32]int{}
	for _, item := range listed.Items {
		if item.LoginMethod == nil {
			continue
		}
		actualProto[int32(*item.LoginMethod)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 login_method 分布应与可落库 proto 枚举值集合逐对一致")

	// —— risk_level 枚举对 ——
	expectedEnt = map[string]int{}
	expectedProto = map[int32]int{}
	for value, name := range auditV1.LoginAuditLog_RiskLevel_name {
		if value == 0 {
			continue
		}
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{
			Data: &auditV1.LoginAuditLog{
				FailureReason: trans.Ptr(nextMarker()),
				RiskLevel:     auditV1.LoginAuditLog_RiskLevel(value).Enum(),
			},
		}), "risk_level=%s 建行应成功", name)
	}
	entRows, err = repo.entClient.Client().LoginAuditLog.Query().All(ctx)
	require.NoError(t, err)
	actualEnt = map[string]int{}
	for _, row := range entRows {
		if row.RiskLevel == nil {
			continue
		}
		actualEnt[string(*row.RiskLevel)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 risk_level 分布应与枚举名集合逐对一致")
	listed, err = repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto = map[int32]int{}
	for _, item := range listed.Items {
		if item.RiskLevel == nil {
			continue
		}
		actualProto[int32(*item.RiskLevel)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 risk_level 分布应与 proto 枚举值集合逐对一致")
}

// TestLoginAuditLogRepoSqlite_ListFilterAndPaging 验证 List 的无过滤全量、
// username 列 contains 模糊搜索、id 列等值过滤与分页语义。
func TestLoginAuditLogRepoSqlite_ListFilterAndPaging(t *testing.T) {
	repo := newLoginAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for i, marker := range []string{"MARKEREPSILON", "MARKERZETA"} {
		require.NoError(t, repo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{
			Data: &auditV1.LoginAuditLog{
				Username:      trans.Ptr(marker + " 用户"),
				RequestId:     trans.Ptr(fmt.Sprintf("req-sqlite-login-list-%d", i)),
				FailureReason: trans.Ptr(fmt.Sprintf("list-login-%d", i)),
			},
		}), "写入第 %d 行应成功", i)
	}

	rows, err := repo.entClient.Client().LoginAuditLog.Query().All(ctx)
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
						Field:      "username",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKEREPSILON"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetUsername(), "MARKEREPSILON", "命中行应为含标记的那条")

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
	require.Equal(t, firstID, byID.Items[0].GetId())

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

// TestLoginAuditLogRepoSqlite_Get 验证 Get 按主键的命中与未命中。
func TestLoginAuditLogRepoSqlite_Get(t *testing.T) {
	repo := newLoginAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{
		Data: &auditV1.LoginAuditLog{
			Username:  trans.Ptr("sqlite_login_get_user"),
			RequestId: trans.Ptr("req-sqlite-login-get-1"),
		},
	}))

	rows, err := repo.entClient.Client().LoginAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	got, err := repo.Get(ctx, &auditV1.GetLoginAuditLogRequest{
		QueryBy: &auditV1.GetLoginAuditLogRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, got.GetId())
	require.Equal(t, "sqlite_login_get_user", got.GetUsername(), "命中记录的 username 应与写入一致")

	_, err = repo.Get(ctx, &auditV1.GetLoginAuditLogRequest{
		QueryBy: &auditV1.GetLoginAuditLogRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")
}

// TestLoginAuditLogRepoSqlite_CountAndIsExist 验证 Count 的带谓词/无谓词语义
// 与 IsExist 的命中/未命中。
func TestLoginAuditLogRepoSqlite_CountAndIsExist(t *testing.T) {
	repo := newLoginAuditLogRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{
		Data: &auditV1.LoginAuditLog{
			Username:  trans.Ptr("sqlite_login_count_a"),
			RequestId: trans.Ptr("req-sqlite-login-count-1"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &auditV1.CreateLoginAuditLogRequest{
		Data: &auditV1.LoginAuditLog{
			Username:  trans.Ptr("sqlite_login_count_b"),
			RequestId: trans.Ptr("req-sqlite-login-count-2"),
		},
	}))

	rows, err := repo.entClient.Client().LoginAuditLog.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	createdID := rows[0].ID

	total, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total, "无谓词 Count 应为全量 2")

	byID, err := repo.Count(ctx, []func(s *sql.Selector){entLoginAuditLog.IDEQ(createdID)})
	require.NoError(t, err)
	require.Equal(t, 1, byID, "主键等值谓词应只命中 1 行")

	byUsername, err := repo.Count(ctx, []func(s *sql.Selector){entLoginAuditLog.UsernameEQ("sqlite_login_count_a")})
	require.NoError(t, err)
	require.Equal(t, 1, byUsername, "username 等值谓词应只命中 1 行")

	exist, err := repo.IsExist(ctx, createdID)
	require.NoError(t, err)
	require.True(t, exist, "存在的主键 IsExist 应为 true")
	exist, err = repo.IsExist(ctx, 99999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的主键 IsExist 应为 false")
}
