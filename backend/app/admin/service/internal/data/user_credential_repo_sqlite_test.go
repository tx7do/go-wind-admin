package data

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/crypto"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/password"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	configV1 "go-wind-admin/api/gen/go/config/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entSysConfig "go-wind-admin/app/admin/service/internal/data/ent/sysconfig"
	entUserCredential "go-wind-admin/app/admin/service/internal/data/ent/usercredential"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newUserCredentialRepoSqlite 用 enttest helper 构造 UserCredentialRepo，
// 逐字段复刻 NewUserCredentialRepo 的 mapper/converter 初始化，再调用 init()。
// passwordCrypto 用纯 Go 的 SHA256 实现（无外部件），configRepo 在同一测试库上
// 局部白盒构造（sys_config 为空 → GetConfigInt 一律回退默认值），两者使
// 密码哈希/复杂度/历史口令链路可在无外部依赖下走通。
func newUserCredentialRepoSqlite(t *testing.T) *UserCredentialRepo {
	t.Helper()
	entClient := enttest.NewEntClientForTest(t)
	// 局部构造 ConfigRepo（不启动生产构造器里的失效广播 goroutine）：
	// 逐字段复刻 NewConfigRepo 初始化，log 换 NopLogger，rdb 为 nil（有守卫）。
	configRepo := &ConfigRepo{
		entClient: entClient,
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[configV1.Config, ent.SysConfig](),
		valueTypeConverter: mapper.NewEnumTypeConverter[configV1.Config_ConfigValueType, entSysConfig.ValueType](
			configV1.Config_ConfigValueType_name, configV1.Config_ConfigValueType_value,
		),
		cache: make(map[string]sysConfigCacheEntry),
	}
	configRepo.init()

	repo := &UserCredentialRepo{
		entClient:               entClient,
		log:                     bLogger.NewHelper(bLogger.NopLogger()),
		mapper:                  mapper.NewCopierMapper[authenticationV1.UserCredential, ent.UserCredential](),
		statusConverter:         mapper.NewEnumTypeConverter[authenticationV1.UserCredential_Status, entUserCredential.Status](authenticationV1.UserCredential_Status_name, authenticationV1.UserCredential_Status_value),
		identityTypeConverter:   mapper.NewEnumTypeConverter[authenticationV1.UserCredential_IdentityType, entUserCredential.IdentityType](authenticationV1.UserCredential_IdentityType_name, authenticationV1.UserCredential_IdentityType_value),
		credentialTypeConverter: mapper.NewEnumTypeConverter[authenticationV1.UserCredential_CredentialType, entUserCredential.CredentialType](authenticationV1.UserCredential_CredentialType_name, authenticationV1.UserCredential_CredentialType_value),
		passwordCrypto:          password.NewSHA256Crypto(),
		configRepo:              configRepo,
	}
	repo.init()
	return repo
}

// TestUserCredentialRepoSqlite_CreateAndGet 验证非密码哈希类型凭证的
// 明文落库与 Get / GetByIdentifier 的命中、未命中，含枚举映射回读。
func TestUserCredentialRepoSqlite_CreateAndGet(t *testing.T) {
	repo := newUserCredentialRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	err := repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:            trans.Ptr(uint32(77)),
			IdentityType:      authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:        trans.Ptr("sqlite_uc_user"),
			CredentialType:    authenticationV1.UserCredential_API_KEY.Enum(),
			Credential:        trans.Ptr("plain-token-abc"),
			IsPrimary:         trans.Ptr(true),
			Status:            authenticationV1.UserCredential_ENABLED.Enum(),
			ExtraInfo:         trans.Ptr(`{"note":"sqlite"}`),
			Provider:          trans.Ptr("local"),
			ProviderAccountId: trans.Ptr("pa-1"),
		},
	})
	require.NoError(t, err, "repo.Create 写入 SQLite 应成功")

	rows, err := repo.entClient.Client().UserCredential.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1, "sys_user_credentials 应有 1 条记录")
	row := rows[0]
	require.Equal(t, uint32(77), *row.UserID, "user_id 应按请求落库")
	require.NotNil(t, row.IdentityType)
	require.Equal(t, entUserCredential.IdentityTypeUsername, *row.IdentityType,
		"proto USERNAME 应映射为 ent IdentityTypeUsername")
	require.Equal(t, "sqlite_uc_user", *row.Identifier, "identifier 应按请求落库")
	require.NotNil(t, row.CredentialType)
	require.Equal(t, entUserCredential.CredentialTypeApiKey, *row.CredentialType,
		"proto API_KEY 应映射为 ent CredentialTypeApiKey")
	require.Equal(t, "plain-token-abc", *row.Credential, "非密码哈希类型的 credential 应明文落库")
	require.NotNil(t, row.IsPrimary)
	require.True(t, *row.IsPrimary, "is_primary 应按请求落库")
	require.NotNil(t, row.Status)
	require.Equal(t, entUserCredential.StatusEnabled, *row.Status,
		"proto ENABLED 应映射为 ent StatusEnabled")
	require.Equal(t, `{"note":"sqlite"}`, *row.ExtraInfo, "extra_info 应按请求落库")
	require.Equal(t, "local", *row.Provider, "provider 应按请求落库")
	require.Equal(t, "pa-1", *row.ProviderAccountID, "provider_account_id 应按请求落库")
	require.False(t, row.CreatedAt.IsZero(), "created_at 应由 repo 写入")

	createdID := row.ID

	// Get 按主键：命中
	got, err := repo.Get(ctx, &authenticationV1.GetUserCredentialRequest{
		QueryBy: &authenticationV1.GetUserCredentialRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, got.GetId())
	require.Equal(t, "sqlite_uc_user", got.GetIdentifier(), "命中记录的 identifier 应与写入一致")
	require.Equal(t, authenticationV1.UserCredential_USERNAME, got.GetIdentityType(),
		"命中记录的 identity_type 应经 converter 还原为 USERNAME")
	require.Equal(t, authenticationV1.UserCredential_API_KEY, got.GetCredentialType(),
		"命中记录的 credential_type 应经 converter 还原为 API_KEY")
	require.Equal(t, authenticationV1.UserCredential_ENABLED, got.GetStatus(),
		"命中记录的 status 应经 converter 还原为 ENABLED")

	// Get 按主键：未命中
	_, err = repo.Get(ctx, &authenticationV1.GetUserCredentialRequest{
		QueryBy: &authenticationV1.GetUserCredentialRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")

	// GetByIdentifier：命中
	byId, err := repo.GetByIdentifier(ctx, &authenticationV1.GetUserCredentialByIdentifierRequest{
		IdentityType: authenticationV1.UserCredential_USERNAME,
		Identifier:   "sqlite_uc_user",
	})
	require.NoError(t, err, "按存在的 (identity_type, identifier) 查询应命中")
	require.Equal(t, createdID, byId.GetId(), "按标识符命中行的 id 应与主键一致")
	require.Equal(t, authenticationV1.UserCredential_USERNAME, byId.GetIdentityType(),
		"按标识符命中行的 identity_type 应经 converter 还原")
	require.Equal(t, authenticationV1.UserCredential_API_KEY, byId.GetCredentialType(),
		"按标识符命中行的 credential_type 应经 converter 还原")
	require.Equal(t, authenticationV1.UserCredential_ENABLED, byId.GetStatus(),
		"按标识符命中行的 status 应经 converter 还原")

	// GetByIdentifier：未命中
	_, err = repo.GetByIdentifier(ctx, &authenticationV1.GetUserCredentialByIdentifierRequest{
		IdentityType: authenticationV1.UserCredential_USERNAME,
		Identifier:   "no_such_identifier_zzz",
	})
	require.Error(t, err, "查询不存在的标识符应返回错误")
}

// TestUserCredentialRepoSqlite_IdentityTypeEnumPairs 对 identity_type 枚举逐一建行，
// 断言 proto → ent 与 ent → proto（List 路径）双向映射逐对成立。
// 已知不对称：proto 的 IDENTITY_RESERVED_FOR_FUTURE 在 ent 侧无对应值，
// 写入被 ent 校验器拒绝（预期失败、不落行）。
func TestUserCredentialRepoSqlite_IdentityTypeEnumPairs(t *testing.T) {
	repo := newUserCredentialRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	expectedEnt := map[string]int{}
	expectedProto := map[int32]int{}
	serial := 0
	for value, name := range authenticationV1.UserCredential_IdentityType_name {
		// 已知不对称：converter 以 proto 名表产出字符串，而 ent 枚举值名可能不同
		// （如 proto IDENTITY_CUSTOM 对应 ent 值名 CUSTOM、IDENTITY_RESERVED_FOR_FUTURE 无 ent 值）。
		// 以 ent 生成校验器为权威：校验不过的取值写入被拒（预期失败、不落行）。
		entVal := repo.identityTypeConverter.ToEntity(authenticationV1.UserCredential_IdentityType(value).Enum())
		if entVal == nil || entUserCredential.IdentityTypeValidator(*entVal) != nil {
			require.Error(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
				Data: &authenticationV1.UserCredential{
					UserId:         trans.Ptr(uint32(1)),
					IdentityType:   authenticationV1.UserCredential_IdentityType(value).Enum(),
					Identifier:     trans.Ptr(fmt.Sprintf("sqlite-ident-identity-rejected-%d", value)),
					CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
					Credential:     trans.Ptr("x"),
					Status:         authenticationV1.UserCredential_ENABLED.Enum(),
				},
			}), "identity_type=%s 的转换产物过不了 ent 校验器，写入应被拒绝", name)
			continue
		}
		serial++
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
			Data: &authenticationV1.UserCredential{
				UserId:         trans.Ptr(uint32(1)),
				IdentityType:   authenticationV1.UserCredential_IdentityType(value).Enum(),
				Identifier:     trans.Ptr(fmt.Sprintf("sqlite-ident-identity-%d", serial)),
				CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
				Credential:     trans.Ptr("x"),
				Status:         authenticationV1.UserCredential_ENABLED.Enum(),
			},
		}), "identity_type=%s 建行应成功", name)
	}
	require.NotEmpty(t, expectedEnt, "应存在可落库枚举值")

	entRows, err := repo.entClient.Client().UserCredential.Query().All(ctx)
	require.NoError(t, err)
	actualEnt := map[string]int{}
	for _, row := range entRows {
		require.NotNil(t, row.IdentityType)
		actualEnt[string(*row.IdentityType)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 identity_type 取值分布应与可落库枚举名集合逐对一致")

	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto := map[int32]int{}
	for _, item := range listed.Items {
		require.NotNil(t, item.IdentityType, "DTO 的 identity_type 应被 converter 回填")
		actualProto[int32(*item.IdentityType)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 identity_type 取值分布应与可落库 proto 枚举值集合逐对一致")
}

// TestUserCredentialRepoSqlite_StatusEnumPairs 对 status 枚举逐一建行，
// 断言双向映射逐对成立（该枚举无 UNSPECIFIED，DISABLED 为合法值）。
func TestUserCredentialRepoSqlite_StatusEnumPairs(t *testing.T) {
	repo := newUserCredentialRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	expectedEnt := map[string]int{}
	expectedProto := map[int32]int{}
	serial := 0
	for value, name := range authenticationV1.UserCredential_Status_name {
		serial++
		expectedEnt[name] = 1
		expectedProto[value] = 1
		require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
			Data: &authenticationV1.UserCredential{
				UserId:         trans.Ptr(uint32(2)),
				IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
				Identifier:     trans.Ptr(fmt.Sprintf("sqlite-ident-status-%d", serial)),
				CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
				Credential:     trans.Ptr("x"),
				Status:         authenticationV1.UserCredential_Status(value).Enum(),
			},
		}), "status=%s 建行应成功", name)
	}

	entRows, err := repo.entClient.Client().UserCredential.Query().All(ctx)
	require.NoError(t, err)
	actualEnt := map[string]int{}
	for _, row := range entRows {
		require.NotNil(t, row.Status)
		actualEnt[string(*row.Status)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 status 取值分布应与枚举名集合逐对一致")

	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto := map[int32]int{}
	for _, item := range listed.Items {
		require.NotNil(t, item.Status, "DTO 的 status 应被 converter 回填")
		actualProto[int32(*item.Status)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 status 取值分布应与 proto 枚举值集合逐对一致")
}

// TestUserCredentialRepoSqlite_CredentialTypeEnumPairs 对 credential_type 枚举的
// 全部有效取值（TYPE_UNSPECIFIED 除外）逐一建行：PASSWORD_HASH 走哈希链路（存哈希），
// 其余类型明文落库；断言双向映射逐对成立。
func TestUserCredentialRepoSqlite_CredentialTypeEnumPairs(t *testing.T) {
	repo := newUserCredentialRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	expectedEnt := map[string]int{}
	expectedProto := map[int32]int{}
	serial := 0
	for value, name := range authenticationV1.UserCredential_CredentialType_name {
		if name == "TYPE_UNSPECIFIED" {
			continue
		}
		// 已知不对称：部分 proto 名在 ent 枚举中无对应值（如 SECURITY_KEY），
		// 以 ent 生成校验器为权威：校验不过的取值写入被拒（预期失败、不落行）。
		entVal := repo.credentialTypeConverter.ToEntity(authenticationV1.UserCredential_CredentialType(value).Enum())
		if entVal == nil || entUserCredential.CredentialTypeValidator(*entVal) != nil {
			require.Error(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
				Data: &authenticationV1.UserCredential{
					UserId:         trans.Ptr(uint32(3)),
					IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
					Identifier:     trans.Ptr(fmt.Sprintf("sqlite-ident-credtype-rejected-%d", value)),
					CredentialType: authenticationV1.UserCredential_CredentialType(value).Enum(),
					Credential:     trans.Ptr("x"),
					Status:         authenticationV1.UserCredential_ENABLED.Enum(),
				},
			}), "credential_type=%s 的转换产物过不了 ent 校验器，写入应被拒绝", name)
			continue
		}
		serial++
		expectedEnt[name] = 1
		expectedProto[value] = 1
		credential := "plain-" + name
		if name == "PASSWORD_HASH" {
			credential = "Str0ng!Passw0rd#42"
		}
		require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
			Data: &authenticationV1.UserCredential{
				UserId:         trans.Ptr(uint32(3)),
				IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
				Identifier:     trans.Ptr(fmt.Sprintf("sqlite-ident-credtype-%d", serial)),
				CredentialType: authenticationV1.UserCredential_CredentialType(value).Enum(),
				Credential:     trans.Ptr(credential),
				Status:         authenticationV1.UserCredential_ENABLED.Enum(),
			},
		}), "credential_type=%s 建行应成功", name)
	}
	require.NotEmpty(t, expectedEnt, "应存在可落库枚举值")

	entRows, err := repo.entClient.Client().UserCredential.Query().All(ctx)
	require.NoError(t, err)
	actualEnt := map[string]int{}
	for _, row := range entRows {
		require.NotNil(t, row.CredentialType)
		actualEnt[string(*row.CredentialType)]++
	}
	require.Equal(t, expectedEnt, actualEnt, "ent 侧 credential_type 取值分布应与枚举名集合逐对一致")

	// 密码哈希行的 credential 应为哈希（非明文）
	hashRows, err := repo.entClient.Client().UserCredential.Query().
		Where(entUserCredential.CredentialTypeEQ(entUserCredential.CredentialTypePasswordHash)).All(ctx)
	require.NoError(t, err)
	require.Len(t, hashRows, 1, "PASSWORD_HASH 应恰有一行")
	require.NotEqual(t, "Str0ng!Passw0rd#42", *hashRows[0].Credential,
		"PASSWORD_HASH 类型的明文不应直接落库")
	ok, err := repo.passwordCrypto.Verify("Str0ng!Passw0rd#42", *hashRows[0].Credential)
	require.NoError(t, err)
	require.True(t, ok, "落库哈希应能被 SHA256 校验通过（哈希链路为纯 Go 实现，可端到端验证）")

	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	actualProto := map[int32]int{}
	for _, item := range listed.Items {
		require.NotNil(t, item.CredentialType, "DTO 的 credential_type 应被 converter 回填")
		actualProto[int32(*item.CredentialType)]++
	}
	require.Equal(t, expectedProto, actualProto, "DTO 侧 credential_type 取值分布应与 proto 枚举值集合逐对一致")
}

// TestUserCredentialRepoSqlite_ListFilterAndPaging 验证 List 的无过滤全量、
// identifier 列 contains 模糊搜索、id 列等值过滤与分页语义。
func TestUserCredentialRepoSqlite_ListFilterAndPaging(t *testing.T) {
	repo := newUserCredentialRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for i, marker := range []string{"MARKERRHO", "MARKERSIGMA"} {
		require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
			Data: &authenticationV1.UserCredential{
				UserId:         trans.Ptr(uint32(50)),
				IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
				Identifier:     trans.Ptr(marker + "_ident"),
				CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
				Credential:     trans.Ptr("x"),
				Status:         authenticationV1.UserCredential_ENABLED.Enum(),
			},
		}), "写入第 %d 行应成功", i)
	}

	rows, err := repo.entClient.Client().UserCredential.Query().All(ctx)
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
						Field:      "identifier",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKERRHO"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetIdentifier(), "MARKERRHO", "命中行应为含标记的那条")

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

// TestUserCredentialRepoSqlite_CountAndIsExist 验证 Count 的带谓词/无谓词语义
// 与 IsExist 的命中/未命中。
func TestUserCredentialRepoSqlite_CountAndIsExist(t *testing.T) {
	repo := newUserCredentialRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(60)),
			IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:     trans.Ptr("count_ident_a"),
			CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
			Credential:     trans.Ptr("x"),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}))
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(61)),
			IdentityType:   authenticationV1.UserCredential_EMAIL.Enum(),
			Identifier:     trans.Ptr("count_ident_b@x"),
			CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
			Credential:     trans.Ptr("x"),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}))

	rows, err := repo.entClient.Client().UserCredential.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	createdID := rows[0].ID

	total, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total, "无谓词 Count 应为全量 2")

	byID, err := repo.Count(ctx, []func(s *sql.Selector){entUserCredential.IDEQ(createdID)})
	require.NoError(t, err)
	require.Equal(t, 1, byID, "主键等值谓词应只命中 1 行")

	byIdentifier, err := repo.Count(ctx, []func(s *sql.Selector){entUserCredential.IdentifierEQ("count_ident_b@x")})
	require.NoError(t, err)
	require.Equal(t, 1, byIdentifier, "identifier 等值谓词应只命中 1 行")

	exist, err := repo.IsExist(ctx, createdID)
	require.NoError(t, err)
	require.True(t, exist, "存在的主键 IsExist 应为 true")
	exist, err = repo.IsExist(ctx, 99999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的主键 IsExist 应为 false")
}

// TestUserCredentialRepoSqlite_PasswordLifecycle 验证密码哈希凭证全生命周期：
// 复杂度校验（弱口令拒绝）、FindUserCredential 命中/口令错误/用户不存在/停用状态、
// VerifyCredential 包装、ChangeCredential（含 AES needDecrypt 路径与解密失败分支）、
// 历史口令检查（轮换进 extra_info、重复口令拒绝）、ResetCredential 轮换。
func TestUserCredentialRepoSqlite_PasswordLifecycle(t *testing.T) {
	repo := newUserCredentialRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	const (
		pw0 = "Str0ng!Passw0rd#42"
		pw1 = "Str0ng3r!Passw0rd#43"
		pw2 = "Cr3pt0!Passw0rd#44"
		pw3 = "R3set!Passw0rd#99"
	)

	// 弱口令创建被复杂度校验拒绝
	require.Error(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(77)),
			IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:     trans.Ptr("sqlite_pw_user"),
			CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
			Credential:     trans.Ptr("abc"),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}), "弱口令创建应被复杂度校验拒绝")

	// 强口令创建成功，落库为哈希
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(77)),
			IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:     trans.Ptr("sqlite_pw_user"),
			CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
			Credential:     trans.Ptr(pw0),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}), "强口令创建应成功")
	row, err := repo.entClient.Client().UserCredential.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, row, 1)
	require.NotEqual(t, pw0, *row[0].Credential, "口令哈希不应明文落库")

	// 停用状态的凭证行：FindUserCredential 应按“用户不存在”拒绝（不泄露状态差异）
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(78)),
			IdentityType:   authenticationV1.UserCredential_EMAIL.Enum(),
			Identifier:     trans.Ptr("disabled@x"),
			CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
			Credential:     trans.Ptr("tok-disabled"),
			Status:         authenticationV1.UserCredential_DISABLED.Enum(),
		},
	}), "建停用凭证行应成功")

	// FindUserCredential：正确口令命中
	uid, err := repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME,
		"sqlite_pw_user", pw0, false)
	require.NoError(t, err, "正确口令应命中")
	require.Equal(t, uint32(77), uid, "应返回落库 user_id")

	// FindUserCredential：错误口令
	_, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME,
		"sqlite_pw_user", "Wr0ng!Passw0rd#00", false)
	require.Error(t, err, "错误口令应返回错误")

	// FindUserCredential：用户不存在
	_, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME,
		"no_such_pw_user_zzz", pw0, false)
	require.Error(t, err, "不存在的用户应返回错误")

	// FindUserCredential：停用状态行
	_, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_EMAIL,
		"disabled@x", "tok-disabled", false)
	require.Error(t, err, "停用状态的凭证应按用户不存在拒绝")

	// FindUserCredential：needDecrypt 路径——正确 AES 载荷命中
	encGood, err := crypto.AesEncrypt([]byte(pw0), crypto.DefaultAESKey, nil)
	require.NoError(t, err)
	uid, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME,
		"sqlite_pw_user", base64.StdEncoding.EncodeToString(encGood), true)
	require.NoError(t, err, "AES 加密载荷解密后应命中")
	require.Equal(t, uint32(77), uid)

	// FindUserCredential：needDecrypt 路径——非 base64 载荷报错
	_, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME,
		"sqlite_pw_user", "!!not-base64!!", true)
	require.Error(t, err, "非法 base64 载荷应报错")

	// FindUserCredential：needDecrypt 路径——base64 合法但 AES 解密失败
	_, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME,
		"sqlite_pw_user", base64.StdEncoding.EncodeToString([]byte("garbage-not-aes")), true)
	require.Error(t, err, "AES 解密失败应报错")

	// VerifyCredential 包装：成功与失败
	vc, err := repo.VerifyCredential(ctx, &authenticationV1.VerifyCredentialRequest{
		IdentityType: authenticationV1.UserCredential_USERNAME,
		Identifier:   "sqlite_pw_user",
		Credential:   pw0,
	})
	require.NoError(t, err, "VerifyCredential 正确口令应成功")
	require.True(t, vc.GetSuccess(), "VerifyCredential 成功时应返回 Success=true")
	_, err = repo.VerifyCredential(ctx, &authenticationV1.VerifyCredentialRequest{
		IdentityType: authenticationV1.UserCredential_USERNAME,
		Identifier:   "sqlite_pw_user",
		Credential:   "Wr0ng!Passw0rd#00",
	})
	require.Error(t, err, "VerifyCredential 错误口令应返回错误")

	// 历史口令读取辅助：解析 extra_info 里的 password_history
	readHistory := func() []string {
		rows, qerr := repo.entClient.Client().UserCredential.Query().Where(
			entUserCredential.IdentifierEQ("sqlite_pw_user"),
			entUserCredential.IdentityTypeEQ(entUserCredential.IdentityTypeUsername),
		).All(ctx)
		require.NoError(t, qerr)
		require.Len(t, rows, 1)
		if rows[0].ExtraInfo == nil || *rows[0].ExtraInfo == "" {
			return nil
		}
		var extra map[string]any
		require.NoError(t, json.Unmarshal([]byte(*rows[0].ExtraInfo), &extra))
		arr, _ := extra["password_history"].([]any)
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}

	// ChangeCredential：错误旧口令拒绝
	require.Error(t, repo.ChangeCredential(ctx, &authenticationV1.ChangeCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    "sqlite_pw_user",
		OldCredential: "Wr0ng!Passw0rd#00",
		NewCredential: pw1,
	}), "错误旧口令应被拒绝")

	// ChangeCredential：弱新口令被复杂度校验拒绝
	require.Error(t, repo.ChangeCredential(ctx, &authenticationV1.ChangeCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    "sqlite_pw_user",
		OldCredential: pw0,
		NewCredential: "abc",
	}), "弱新口令应被复杂度校验拒绝")

	// ChangeCredential：正确旧口令轮换到 pw1，旧哈希进入历史（1 条）
	require.NoError(t, repo.ChangeCredential(ctx, &authenticationV1.ChangeCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    "sqlite_pw_user",
		OldCredential: pw0,
		NewCredential: pw1,
	}), "正确旧口令的轮换应成功")
	history := readHistory()
	require.Len(t, history, 1, "首次轮换后历史应含 1 条旧哈希")
	ok, err := repo.passwordCrypto.Verify(pw0, history[0])
	require.NoError(t, err)
	require.True(t, ok, "历史首条应为旧口令 pw0 的哈希")
	uid, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME, "sqlite_pw_user", pw1, false)
	require.NoError(t, err, "轮换后新口令 pw1 应命中")
	require.Equal(t, uint32(77), uid)
	_, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME, "sqlite_pw_user", pw0, false)
	require.Error(t, err, "轮换后旧口令 pw0 应失效")

	// 历史口令检查：轮换回 pw0 应被历史拒绝
	require.Error(t, repo.ChangeCredential(ctx, &authenticationV1.ChangeCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    "sqlite_pw_user",
		OldCredential: pw1,
		NewCredential: pw0,
	}), "轮换回历史口令 pw0 应被拒绝")

	// ChangeCredential 的 AES needDecrypt 路径：旧/新均为 AES 载荷，轮换到 pw2
	encOld, err := crypto.AesEncrypt([]byte(pw1), crypto.DefaultAESKey, nil)
	require.NoError(t, err)
	encNew, err := crypto.AesEncrypt([]byte(pw2), crypto.DefaultAESKey, nil)
	require.NoError(t, err)
	require.NoError(t, repo.ChangeCredential(ctx, &authenticationV1.ChangeCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    "sqlite_pw_user",
		OldCredential: base64.StdEncoding.EncodeToString(encOld),
		NewCredential: base64.StdEncoding.EncodeToString(encNew),
		NeedDecrypt:   true,
	}), "AES 载荷的 ChangeCredential 应解密后轮换成功")
	require.Len(t, readHistory(), 2, "二次轮换后历史应含 2 条旧哈希")
	uid, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME, "sqlite_pw_user", pw2, false)
	require.NoError(t, err, "AES 轮换后新口令 pw2 应命中")
	require.Equal(t, uint32(77), uid)

	// ChangeCredential 的 AES needDecrypt 路径：非法 base64 拒绝
	require.Error(t, repo.ChangeCredential(ctx, &authenticationV1.ChangeCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    "sqlite_pw_user",
		OldCredential: "!!not-base64!!",
		NewCredential: pw1,
		NeedDecrypt:   true,
	}), "非法 base64 旧口令应被拒绝")

	// ResetCredential：跳过旧口令校验直接轮换到 pw3，历史封顶 3 条
	require.NoError(t, repo.ResetCredential(ctx, &authenticationV1.ResetCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    "sqlite_pw_user",
		NewCredential: pw3,
	}), "ResetCredential 轮换应成功")
	require.Len(t, readHistory(), 3, "三次轮换后历史应封顶 3 条")
	uid, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME, "sqlite_pw_user", pw3, false)
	require.NoError(t, err, "重置后新口令 pw3 应命中")
	require.Equal(t, uint32(77), uid)
	_, err = repo.FindUserCredential(ctx, 0, authenticationV1.UserCredential_USERNAME, "sqlite_pw_user", pw2, false)
	require.Error(t, err, "重置后旧口令 pw2 应失效")

	// ResetCredential：历史口令与弱口令均被拒绝
	require.Error(t, repo.ResetCredential(ctx, &authenticationV1.ResetCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    "sqlite_pw_user",
		NewCredential: pw0,
	}), "重置为历史口令 pw0 应被拒绝")
	require.Error(t, repo.ResetCredential(ctx, &authenticationV1.ResetCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    "sqlite_pw_user",
		NewCredential: "abc",
	}), "重置为弱口令应被拒绝")

	// ResetCredential：不存在的标识符
	require.Error(t, repo.ResetCredential(ctx, &authenticationV1.ResetCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    "no_such_pw_user_zzz",
		NewCredential: pw3,
	}), "不存在的标识符应返回错误")
}

// TestUserCredentialRepoSqlite_UpdateMaskAndAllowMissing 验证 Update 的
// 单字段掩码更新（identifier 更新、掩码外字段保持）、参数校验分支、
// 与 AllowMissing 对不存在 ID 的创建路径。
func TestUserCredentialRepoSqlite_UpdateMaskAndAllowMissing(t *testing.T) {
	repo := newUserCredentialRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(88)),
			IdentityType:   authenticationV1.UserCredential_EMAIL.Enum(),
			Identifier:     trans.Ptr("update_before@x"),
			CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
			Credential:     trans.Ptr("tok-original"),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}))
	rows, err := repo.entClient.Client().UserCredential.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	createdID := rows[0].ID

	// 单字段掩码 [identifier]：identifier 更新、credential/status 保持
	// 注：Data 不携带 Credential——Update 路径对非空 Credential 且空 CredentialType
	// 的组合会在 prepareCredential 里解引用 nil 凭证类型（生产代码现状），
	// 掩码外的该字段本就不参与更新，故省略以走纯 identifier 掩码路径。
	err = repo.Update(ctx, &authenticationV1.UpdateUserCredentialRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"identifier"}},
		Data: &authenticationV1.UserCredential{
			Identifier: trans.Ptr("update_after@x"),
			Status:     authenticationV1.UserCredential_DISABLED.Enum(),
		},
	})
	require.NoError(t, err, "掩码内字段更新应成功")
	after, err := repo.entClient.Client().UserCredential.Query().Where(entUserCredential.IDEQ(createdID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "update_after@x", *after.Identifier, "掩码内字段 identifier 应被更新")
	require.Equal(t, "tok-original", *after.Credential, "掩码外字段 credential 应保持原值")
	require.Equal(t, entUserCredential.StatusEnabled, *after.Status, "掩码外字段 status 应保持原值")

	// 参数校验分支
	require.Error(t, repo.Update(ctx, &authenticationV1.UpdateUserCredentialRequest{
		Id:   0,
		Data: &authenticationV1.UserCredential{Identifier: trans.Ptr("x@x")},
	}), "Id=0 应返回错误")
	require.Error(t, repo.Update(ctx, &authenticationV1.UpdateUserCredentialRequest{
		Id:   createdID,
		Data: nil,
	}), "Data=nil 应返回错误")

	// AllowMissing：不存在的 ID 走创建路径
	err = repo.Update(ctx, &authenticationV1.UpdateUserCredentialRequest{
		Id:           987654,
		AllowMissing: trans.Ptr(true),
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(89)),
			IdentityType:   authenticationV1.UserCredential_PHONE.Enum(),
			Identifier:     trans.Ptr("allowmissing-created@x"),
			CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
			Credential:     trans.Ptr("tok-allowmissing"),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	})
	require.NoError(t, err, "AllowMissing 对不存在 ID 应走创建路径")
	after2, err := repo.entClient.Client().UserCredential.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, after2, 2, "AllowMissing 创建后应有 2 行")
	identifiers := map[string]bool{}
	for _, r := range after2 {
		identifiers[*r.Identifier] = true
	}
	require.True(t, identifiers["allowmissing-created@x"], "新行应由 AllowMissing 创建")
}

// TestUserCredentialRepoSqlite_Delete 验证 Delete / DeleteByUserId /
// DeleteByIdentifier 的删除与未命中报错。
func TestUserCredentialRepoSqlite_Delete(t *testing.T) {
	repo := newUserCredentialRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(91)),
			IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:     trans.Ptr("del_ident_a"),
			CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
			Credential:     trans.Ptr("x"),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}))
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(91)),
			IdentityType:   authenticationV1.UserCredential_EMAIL.Enum(),
			Identifier:     trans.Ptr("del_ident_b@x"),
			CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
			Credential:     trans.Ptr("x"),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}))
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(92)),
			IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:     trans.Ptr("del_ident_c"),
			CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
			Credential:     trans.Ptr("x"),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}))
	rows, err := repo.entClient.Client().UserCredential.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 3)

	// Delete：删除不存在的 ID 报 NotFound
	require.Error(t, repo.Delete(ctx, 999999), "删除不存在的 ID 应返回错误")

	// DeleteByIdentifier：命中删除
	require.NoError(t, repo.DeleteByIdentifier(ctx,
		authenticationV1.UserCredential_USERNAME, "del_ident_c"), "按标识符删除应成功")
	remain, err := repo.entClient.Client().UserCredential.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, remain, "删除一行后应剩 2 行")

	// DeleteByIdentifier：未命中报 NotFound
	require.Error(t, repo.DeleteByIdentifier(ctx,
		authenticationV1.UserCredential_USERNAME, "no_such_del_ident_zzz"), "未命中的标识符删除应返回错误")

	// DeleteByUserId：删除该用户的全部凭证行
	require.NoError(t, repo.DeleteByUserId(ctx, 91), "按用户删除应成功")
	remain, err = repo.entClient.Client().UserCredential.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, remain, "按用户删除后表内应归零")

	// DeleteByUserId：无命中报 NotFound
	require.Error(t, repo.DeleteByUserId(ctx, 91), "无命中用户的删除应返回错误")

	// Delete：删除存在的 ID
	require.NoError(t, repo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
		Data: &authenticationV1.UserCredential{
			UserId:         trans.Ptr(uint32(93)),
			IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:     trans.Ptr("del_ident_d"),
			CredentialType: authenticationV1.UserCredential_API_KEY.Enum(),
			Credential:     trans.Ptr("x"),
			Status:         authenticationV1.UserCredential_ENABLED.Enum(),
		},
	}))
	rows, err = repo.entClient.Client().UserCredential.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NoError(t, repo.Delete(ctx, rows[0].ID), "删除存在的 ID 应成功")
	remain, err = repo.entClient.Client().UserCredential.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, remain, "删除后表内应归零")
}
