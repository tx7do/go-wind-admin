package data

import (
	"context"
	"testing"

	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entLoginPolicy "go-wind-admin/app/admin/service/internal/data/ent/loginpolicy"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newLoginPolicyRepoSqlite 白盒构造 LoginPolicyRepo：
// 逐字段复刻 NewLoginPolicyRepo 的 mapper/converter 初始化并调用 init()，
// 仅将 log 换为 NopLogger、entClient 换为 SQLite 内存库测试 client。
func newLoginPolicyRepoSqlite(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *LoginPolicyRepo {
	t.Helper()
	repo := &LoginPolicyRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[authenticationV1.LoginPolicy, ent.LoginPolicy](),
		typeConverter: mapper.NewEnumTypeConverter[authenticationV1.LoginPolicy_Type, entLoginPolicy.Type](
			authenticationV1.LoginPolicy_Type_name,
			authenticationV1.LoginPolicy_Type_value,
		),
		methodConverter: mapper.NewEnumTypeConverter[authenticationV1.LoginPolicy_Method, entLoginPolicy.Method](
			authenticationV1.LoginPolicy_Method_name,
			authenticationV1.LoginPolicy_Method_value,
		),
	}
	repo.init()
	return repo
}

// TestLoginPolicyRepoSqlite_CreateAndGet 验证 Create 的字段级落库
// （枚举经 converter 映射、created_by 落 operator 载荷）与
// Get 按 ID 的回读映射 / 未命中错误；以及参数校验与
// UNSPECIFIED 枚举被 ent 枚举校验器拒绝的错误路径。
func TestLoginPolicyRepoSqlite_CreateAndGet(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newLoginPolicyRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId:  trans.Ptr(int64(7)),
			TargetId:  trans.Ptr(int64(42)),
			Type:      authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:    authenticationV1.LoginPolicy_IP.Enum(),
			Value:     trans.Ptr("10.0.0.0/8"),
			Reason:    trans.Ptr("初始原因"),
			CreatedBy: trans.Ptr(int64(1001)),
		},
	}))

	rows, err := entClient.Client().LoginPolicy.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SQLite 中应有 1 条登录策略记录")
	row := rows[0]
	require.NotNil(t, row.TenantID)
	require.Equal(t, int64(7), *row.TenantID, "tenant_id 应按载荷落库")
	require.NotNil(t, row.TargetID)
	require.Equal(t, int64(42), *row.TargetID, "target_id 应按载荷落库")
	require.NotNil(t, row.Type)
	require.Equal(t, entLoginPolicy.TypeBlacklist, *row.Type, "proto BLACKLIST 应经 converter 映射为 ent TypeBlacklist")
	require.NotNil(t, row.Method)
	require.Equal(t, entLoginPolicy.MethodIp, *row.Method, "proto IP 应经 converter 映射为 ent MethodIp")
	require.NotNil(t, row.Value)
	require.Equal(t, "10.0.0.0/8", *row.Value, "value 应按载荷落库")
	require.NotNil(t, row.Reason)
	require.Equal(t, "初始原因", *row.Reason, "reason 应按载荷落库")
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, int64(1001), *row.CreatedBy, "created_by 应按载荷落库")
	require.NotNil(t, row.CreatedAt)
	createdID := row.ID

	// Get 命中：字段与枚举回映射
	dto, err := repo.Get(ctx, &authenticationV1.GetLoginPolicyRequest{
		QueryBy: &authenticationV1.GetLoginPolicyRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在 ID 查询应命中")
	require.Equal(t, createdID, dto.GetId(), "DTO 应带回正确 ID")
	require.Equal(t, int64(42), dto.GetTargetId(), "DTO 应回读 target_id")
	require.Equal(t, authenticationV1.LoginPolicy_BLACKLIST, dto.GetType(), "ent TypeBlacklist 应回映射为 proto BLACKLIST")
	require.Equal(t, authenticationV1.LoginPolicy_IP, dto.GetMethod(), "ent MethodIp 应回映射为 proto IP")
	require.Equal(t, "10.0.0.0/8", dto.GetValue(), "DTO 应回读 value")
	require.Equal(t, "初始原因", dto.GetReason(), "DTO 应回读 reason")

	// Get 未命中
	_, err = repo.Get(ctx, &authenticationV1.GetLoginPolicyRequest{
		QueryBy: &authenticationV1.GetLoginPolicyRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "不存在的 ID 查询应返回错误")

	// Get 参数校验
	_, err = repo.Get(ctx, nil)
	require.Error(t, err, "nil 请求应返回 BadRequest")

	// Create 参数校验
	require.Error(t, repo.Create(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{}), "nil Data 应返回 BadRequest")

	// UNSPECIFIED 枚举：ent 枚举校验器拒绝
	require.Error(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(7)),
			Type:     authenticationV1.LoginPolicy_LOGIN_RESTRICTION_TYPE_UNSPECIFIED.Enum(),
			Method:   authenticationV1.LoginPolicy_IP.Enum(),
			Value:    trans.Ptr("172.16.0.1"),
		},
	}), "UNSPECIFIED 类型应被 ent 枚举校验器拒绝")
	require.Error(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(7)),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_LOGIN_RESTRICTION_METHOD_UNSPECIFIED.Enum(),
			Value:    trans.Ptr("172.16.0.1"),
		},
	}), "UNSPECIFIED 方式应被 ent 枚举校验器拒绝")
	cnt, err := entClient.Client().LoginPolicy.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "失败创建不应增加行数")
}

// TestLoginPolicyRepoSqlite_ListForLoginTenantScope 验证 ListForLogin
// 仅返回请求租户的策略行，且字段映射到 EffectivePolicy；
// 不存在策略的租户返回空切片；全 nil 字段行回读为零值。
func TestLoginPolicyRepoSqlite_ListForLoginTenantScope(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newLoginPolicyRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	// 租户 7：一条全局（target/value/reason 均省略）+ 一条定向
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(7)),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_TIME.Enum(),
		},
	}))
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(7)),
			TargetId: trans.Ptr(int64(42)),
			Type:     authenticationV1.LoginPolicy_WHITELIST.Enum(),
			Method:   authenticationV1.LoginPolicy_DEVICE.Enum(),
			Value:    trans.Ptr("device-xyz"),
			Reason:   trans.Ptr("白名单设备"),
		},
	}))
	// 租户 9：一条
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(9)),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_IP.Enum(),
			Value:    trans.Ptr("198.51.100.7"),
		},
	}))

	// 租户 7：两条，跨租户隔离
	policies, err := repo.ListForLogin(ctx, 7)
	require.NoError(t, err)
	require.Len(t, policies, 2, "租户 7 应只见到自己的 2 条策略")
	byTarget := map[int64]EffectivePolicy{}
	for _, p := range policies {
		byTarget[p.TargetID] = p
	}
	require.Contains(t, byTarget, int64(42), "应存在定向策略")
	require.Contains(t, byTarget, int64(0), "应存在全局策略（target_id 回读为零值）")
	directed := byTarget[42]
	require.Equal(t, "WHITELIST", directed.Type, "type 应回读字符串值")
	require.Equal(t, "DEVICE", directed.Method, "method 应回读字符串值")
	require.Equal(t, "device-xyz", directed.Value, "value 应回读")
	require.Equal(t, "白名单设备", directed.Reason, "reason 应回读")
	global := byTarget[0]
	require.Equal(t, "BLACKLIST", global.Type)
	require.Equal(t, "TIME", global.Method)
	require.Equal(t, "", global.Value, "未落库的 value 应回读为空串")
	require.Equal(t, "", global.Reason, "未落库的 reason 应回读为空串")

	// 租户 9：一条
	policies, err = repo.ListForLogin(ctx, 9)
	require.NoError(t, err)
	require.Len(t, policies, 1, "租户 9 应只见到自己的 1 条策略")
	require.Equal(t, "198.51.100.7", policies[0].Value)

	// 无策略租户：空切片
	policies, err = repo.ListForLogin(ctx, 12345)
	require.NoError(t, err)
	require.Empty(t, policies, "无策略租户应返回空切片")
}

// TestLoginPolicyRepoSqlite_List 验证 List 的分页返回与
// value 的 contains 过滤（跨租户计数）；nil 请求返回 BadRequest。
func TestLoginPolicyRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newLoginPolicyRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(7)),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_IP.Enum(),
			Value:    trans.Ptr("10.0.0.0/8"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(9)),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_IP.Enum(),
			Value:    trans.Ptr("192.168.5.5"),
		},
	}))

	// 无过滤：2 行
	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(2), all.Total, "无过滤时应统计全部 2 条")
	require.Len(t, all.Items, 2, "无过滤时应返回 2 行")
	for _, item := range all.Items {
		// 列表读视图：type/method 经回填如实呈现写入值。
		require.Equal(t, authenticationV1.LoginPolicy_BLACKLIST, item.GetType(), "列表读视图应回填 type")
		require.Equal(t, authenticationV1.LoginPolicy_IP, item.GetMethod(), "列表读视图应回填 method")
	}

	// contains 过滤：仅命中 value 含 10.0. 的那一行
	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"value__contains":"10.0."}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Equal(t, "10.0.0.0/8", filtered.Items[0].GetValue(), "命中行应为 value=10.0.0.0/8 的行")

	// nil 请求
	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")
}

// TestLoginPolicyRepoSqlite_CountAndIsExist 验证 Count 的
// 无条件计数与带 tenant 谓词的条件计数，及 IsExist 的命中/未命中。
func TestLoginPolicyRepoSqlite_CountAndIsExist(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newLoginPolicyRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(7)),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_IP.Enum(),
			Value:    trans.Ptr("10.1.1.1"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(9)),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_IP.Enum(),
			Value:    trans.Ptr("10.9.9.9"),
		},
	}))

	// 无条件：全部行
	total, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total, "无条件计数应为全部 2 行")

	// 带 tenant 谓词：仅统计租户 7
	scoped, err := repo.Count(ctx, []func(s *sql.Selector){entLoginPolicy.TenantIDEQ(7)})
	require.NoError(t, err)
	require.Equal(t, 1, scoped, "带租户谓词的计数应只统计租户 7 的 1 行")

	scoped, err = repo.Count(ctx, []func(s *sql.Selector){entLoginPolicy.TenantIDEQ(99)})
	require.NoError(t, err)
	require.Zero(t, scoped, "不存在策略的租户计数应为 0")

	// IsExist
	rows, err := entClient.Client().LoginPolicy.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	existID := rows[0].ID
	exist, err := repo.IsExist(ctx, existID)
	require.NoError(t, err)
	require.True(t, exist, "已写入的策略应判定为存在")
	exist, err = repo.IsExist(ctx, 9999999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的策略应判定为不存在")
}

// TestLoginPolicyRepoSqlite_Update 验证 Update 的掩码语义：
// 掩码内字段更新（含枚举与 target_id）、掩码外字段保持原值、
// 掩码内且载荷为 nil 的字段被置 NULL；不存在的 ID 无效果不报错；
// AllowMissing 对存在 ID 走更新、对不存在 ID 走创建（created_by 取自 updated_by 载荷）。
func TestLoginPolicyRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newLoginPolicyRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(7)),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_IP.Enum(),
			Value:    trans.Ptr("10.0.0.1"),
			Reason:   trans.Ptr("初始原因"),
		},
	}))
	rows, err := entClient.Client().LoginPolicy.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	rowID := rows[0].ID

	// 掩码 [value]：仅 value 更新，reason/type/method/target_id 保持
	require.NoError(t, repo.Update(ctx, &authenticationV1.UpdateLoginPolicyRequest{
		Id:         rowID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value"}},
		Data: &authenticationV1.LoginPolicy{
			Value:    trans.Ptr("192.168.0.0/16"),
			Reason:   trans.Ptr("掩码外原因"),
			Type:     authenticationV1.LoginPolicy_WHITELIST.Enum(),
			Method:   authenticationV1.LoginPolicy_DEVICE.Enum(),
			TargetId: trans.Ptr(int64(66)),
		},
	}))
	row, err := entClient.Client().LoginPolicy.Get(ctx, rowID)
	require.NoError(t, err)
	require.Equal(t, "192.168.0.0/16", *row.Value, "掩码内的 value 应更新")
	require.Equal(t, "初始原因", *row.Reason, "掩码外的 reason 应保持原值")
	require.Equal(t, entLoginPolicy.TypeBlacklist, *row.Type, "掩码外的 type 应保持原值")
	require.Equal(t, entLoginPolicy.MethodIp, *row.Method, "掩码外的 method 应保持原值")
	require.Nil(t, row.TargetID, "掩码外的 target_id 应保持原值")

	// 掩码 [type, method, reason]：枚举更新、reason 置 NULL（载荷为 nil）
	require.NoError(t, repo.Update(ctx, &authenticationV1.UpdateLoginPolicyRequest{
		Id:         rowID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"type", "method", "reason"}},
		Data: &authenticationV1.LoginPolicy{
			Type:   authenticationV1.LoginPolicy_WHITELIST.Enum(),
			Method: authenticationV1.LoginPolicy_TIME.Enum(),
			Reason: nil,
			Value:  trans.Ptr("0.0.0.0/0"),
		},
	}))
	row, err = entClient.Client().LoginPolicy.Get(ctx, rowID)
	require.NoError(t, err)
	require.Equal(t, entLoginPolicy.TypeWhitelist, *row.Type, "掩码内的 type 应更新为 WHITELIST")
	require.Equal(t, entLoginPolicy.MethodTime, *row.Method, "掩码内的 method 应更新为 TIME")
	require.Nil(t, row.Reason, "掩码内且载荷为 nil 的 reason 应被置 NULL")
	require.Equal(t, "192.168.0.0/16", *row.Value, "掩码外的 value 应保持上次值")

	// 不存在 ID：无效果、不报错
	require.NoError(t, repo.Update(ctx, &authenticationV1.UpdateLoginPolicyRequest{
		Id:         987654,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value"}},
		Data:       &authenticationV1.LoginPolicy{Value: trans.Ptr("255.255.255.255")},
	}), "对不存在 ID 的更新应静默无效果")
	cnt, err := entClient.Client().LoginPolicy.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "对不存在 ID 的更新不应新增行")

	// AllowMissing=true 且行存在：仍走更新路径
	require.NoError(t, repo.Update(ctx, &authenticationV1.UpdateLoginPolicyRequest{
		Id:           rowID,
		AllowMissing: trans.Ptr(true),
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"reason"}},
		Data:         &authenticationV1.LoginPolicy{Reason: trans.Ptr("补写原因")},
	}))
	row, err = entClient.Client().LoginPolicy.Get(ctx, rowID)
	require.NoError(t, err)
	require.Equal(t, "补写原因", *row.Reason, "AllowMissing=true 且行存在时应按掩码更新")

	// AllowMissing=true 且行不存在：走创建路径，created_by 取自 updated_by 载荷
	require.NoError(t, repo.Update(ctx, &authenticationV1.UpdateLoginPolicyRequest{
		Id:           888888,
		AllowMissing: trans.Ptr(true),
		Data: &authenticationV1.LoginPolicy{
			TenantId:  trans.Ptr(int64(9)),
			Type:      authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:    authenticationV1.LoginPolicy_MAC.Enum(),
			Value:     trans.Ptr("aa:bb:cc:dd:ee:ff"),
			UpdatedBy: trans.Ptr(int64(55)),
		},
	}), "AllowMissing=true 且行不存在时应转为创建")
	rows, err = entClient.Client().LoginPolicy.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2, "补建后应有 2 行")
	var created *ent.LoginPolicy
	for _, r := range rows {
		if r.ID != rowID {
			created = r
		}
	}
	require.NotNil(t, created, "补建的行应可查回")
	require.Equal(t, "aa:bb:cc:dd:ee:ff", *created.Value, "补建行的 value 应按载荷落库")
	require.Equal(t, entLoginPolicy.MethodMac, *created.Method, "补建行的 method 应按载荷映射落库")
	require.NotNil(t, created.TenantID)
	require.Equal(t, int64(9), *created.TenantID, "补建行的 tenant_id 应按载荷落库")
	require.NotNil(t, created.CreatedBy)
	require.Equal(t, int64(55), *created.CreatedBy, "补建行的 created_by 应取自 updated_by 载荷")
	require.Nil(t, created.UpdatedBy, "补建行的 updated_by 应保持空")

	// 参数校验
	require.Error(t, repo.Update(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &authenticationV1.UpdateLoginPolicyRequest{Id: rowID}), "nil Data 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &authenticationV1.UpdateLoginPolicyRequest{
		Id:   0,
		Data: &authenticationV1.LoginPolicy{},
	}), "id=0 应返回 BadRequest")
}

// TestLoginPolicyRepoSqlite_Delete 验证 Delete 按指定 ID 删行、
// 不存在 ID 无效果不报错、nil 请求返回 BadRequest。
func TestLoginPolicyRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := newLoginPolicyRepoSqlite(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(7)),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_IP.Enum(),
			Value:    trans.Ptr("10.0.0.2"),
		},
	}))
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateLoginPolicyRequest{
		Data: &authenticationV1.LoginPolicy{
			TenantId: trans.Ptr(int64(7)),
			Type:     authenticationV1.LoginPolicy_BLACKLIST.Enum(),
			Method:   authenticationV1.LoginPolicy_IP.Enum(),
			Value:    trans.Ptr("10.0.0.3"),
		},
	}))
	rows, err := entClient.Client().LoginPolicy.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	idA, idB := rows[0].ID, rows[1].ID

	// 删除 A：仅剩 B
	require.NoError(t, repo.Delete(ctx, &authenticationV1.DeleteLoginPolicyRequest{
		QueryBy: &authenticationV1.DeleteLoginPolicyRequest_Id{Id: idA},
	}))
	rows, err = entClient.Client().LoginPolicy.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "删除后应仅剩 1 行")
	require.Equal(t, idB, rows[0].ID, "剩余行应为未删除的 B")

	// 不存在的 ID：无效果不报错
	require.NoError(t, repo.Delete(ctx, &authenticationV1.DeleteLoginPolicyRequest{
		QueryBy: &authenticationV1.DeleteLoginPolicyRequest_Id{Id: 9999999},
	}), "删除不存在的 ID 应无效果且不报错")
	cnt, err := entClient.Client().LoginPolicy.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, cnt, "对不存在 ID 的删除不应影响行数")

	// nil 请求
	require.Error(t, repo.Delete(ctx, nil), "nil 请求应返回 BadRequest")
}
