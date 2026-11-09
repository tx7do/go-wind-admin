// AiProviderService 的 SQLite 内存库测试（白盒，包内测试）。
//
// 覆盖目标：
//   - api_key 的落库纪律：Create 时明文经全局加密器（测试进程未初始化=明文直通，
//     生产配 GOWIND_CRYPTO_KEY 时为 enc: 密文）+ hint 只存脱敏形态；
//   - Update 的 key 语义：带新 key → 加密落库并刷新 hint（掩码追加 api_key_hint）；
//     不带 key → 本次不改 key（DTO 摘空 + 掩码摘除 api_key/api_key_hint，防
//     空值经 FilterByFieldMask 被当成"显式清空"把已存的 key 抹掉）；
//   - 密文不出 repo：List / Get 的 DTO 里 api_key 恒为空；
//   - 纯函数：maskApiKey 的脱敏规则、removeMaskPaths 的掩码摘除；
//   - 入参守卫：nil 请求体 / nil Data / id=0。
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	entCrud "github.com/tx7do/go-crud/entgo"
	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/ent/aiprovider"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/middleware/auth"
)

// newAiProviderServiceForTest 白盒复刻 NewAiProviderService 的字段初始化。
func newAiProviderServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *AiProviderService {
	t.Helper()
	return &AiProviderService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: data.NewAiProviderRepoForTest(entClient),
	}
}

// aiProviderOperator 构造指定用户的操作者上下文。
// Provider 管理已是平台管理员闸门，测试统一以平台管理员身份执行。
func aiProviderOperator(ctx context.Context, userID uint64) context.Context {
	return auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: userID, IsPlatformAdmin: trans.Ptr(true)})
}

// aiProviderCreateViaService 经服务层创建一个提供商，返回其落库 ID。
func aiProviderCreateViaService(t *testing.T, svc *AiProviderService, entClient *entCrud.EntClient[*ent.Client], ctx context.Context, name, apiKey string) uint64 {
	t.Helper()
	_, err := svc.Create(ctx, &aiV1.CreateAiProviderRequest{
		Data: &aiV1.AiProvider{
			Name:      trans.Ptr(name),
			ModelType: aiV1.AiProvider_CLOUD.Enum(),
			ModelName: trans.Ptr("qwen-max"),
			ApiKey:    trans.Ptr(apiKey),
			IsEnabled: trans.Ptr(true),
		},
	})
	require.NoError(t, err, "经服务层创建提供商应成功")
	rows, err := entClient.Client().AiProvider.Query().All(ctx)
	require.NoError(t, err)
	for _, r := range rows {
		if r.Name != nil && *r.Name == name {
			return r.ID
		}
	}
	t.Fatalf("创建后应能按名字找到提供商 %s", name)
	return 0
}

// TestAiProviderService_MaskApiKey 验证脱敏规则：保留前 3 后 4、中间 *** 代替、
// 过短整体打码、首尾空白先裁剪。
func TestAiProviderService_MaskApiKey(t *testing.T) {
	require.Equal(t, "sk-***cdef", maskApiKey("sk-abcdefcdef"), "常规长度保留前 3 后 4")
	require.Equal(t, "***", maskApiKey("short"), "过短（<=7）整体打码")
	require.Equal(t, "***", maskApiKey("1234567"), "恰好 7 位也算过短")
	require.Equal(t, "***", maskApiKey(""), "空串打码")
	require.Equal(t, "ab1***cd34", maskApiKey("  ab12cd34  "), "脱敏前应裁剪首尾空白")
}

// TestAiProviderService_RemoveMaskPaths 验证掩码摘除：指定路径被剔除、其余保序保留、
// 无命中时原样返回。
func TestAiProviderService_RemoveMaskPaths(t *testing.T) {
	got := removeMaskPaths([]string{"name", "api_key", "api_key_hint"}, "api_key", "api_key_hint")
	require.Equal(t, []string{"name"}, got, "指定路径应被摘除")

	got = removeMaskPaths([]string{"name", "remark"}, "api_key")
	require.Equal(t, []string{"name", "remark"}, got, "无命中时应原样保留")

	got = removeMaskPaths(nil, "api_key")
	require.Empty(t, got, "nil 掩码摘除后为空切片而非 nil（防上层 append 语义漂移）")

	got = removeMaskPaths([]string{"api_key", "name", "api_key"}, "api_key")
	require.Equal(t, []string{"name"}, got, "重复出现的路径应全部摘除")
}

// TestAiProviderService_CreateSealsApiKey 验证创建路径：明文 key 经全局加密器落库
// （测试进程加密器未初始化 → 明文直通；生产为 enc: 密文，语义由 crypto 包钉死）、
// hint 只存脱敏形态、Created/Updated 归属操作人；密文经 List/Get 不外泄。
func TestAiProviderService_CreateSealsApiKey(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiProviderServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := aiProviderOperator(ctx, 81)

	id := aiProviderCreateViaService(t, svc, entClient, opCtx, "创建校验提供商", "sk-plaintext-abcdef")

	row, err := entClient.Client().AiProvider.Query().
		Where(aiprovider.IDEQ(id)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "sk-plaintext-abcdef", *row.APIKey,
		"未配置 GOWIND_CRYPTO_KEY 时明文直通落库（配了则为 enc: 密文，读路径解密对称）")
	require.Equal(t, "sk-***cdef", *row.APIKeyHint, "hint 只存脱敏形态")
	require.NotNil(t, row.CreatedBy, "CreatedBy 应盖章为操作人")
	require.Equal(t, uint64(81), *row.CreatedBy)
	// 仓内约定：Create 构造器只落 created_by，updated_by 首次更新才落库
	//（access_key 等全线服务一致），故这里不断言 updated_by。
	require.True(t, *row.IsEnabled, "启用位应经转换器落库")

	// 密文不出 repo：List / Get 的 DTO 里 api_key 恒为空
	listResp, err := svc.List(opCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.GetItems(), 1)
	require.Nil(t, listResp.GetItems()[0].ApiKey, "List 的 DTO 不得携带 api_key")

	getResp, err := svc.Get(opCtx, &aiV1.GetAiProviderRequest{
		QueryBy: &aiV1.GetAiProviderRequest_Id{Id: id},
	})
	require.NoError(t, err)
	require.Nil(t, getResp.ApiKey, "Get 的 DTO 不得携带 api_key")
	require.Equal(t, "sk-***cdef", getResp.GetApiKeyHint(), "hint 正常回显")

	// 无 key 的创建：hint 不落脏值
	_, err = svc.Create(opCtx, &aiV1.CreateAiProviderRequest{
		Data: &aiV1.AiProvider{Name: trans.Ptr("无密钥提供商")},
	})
	require.NoError(t, err, "无 key 的创建应成功")
	rows, err := entClient.Client().AiProvider.Query().All(ctx)
	require.NoError(t, err)
	for _, r := range rows {
		if r.Name != nil && *r.Name == "无密钥提供商" {
			require.Nil(t, r.APIKey, "无 key 创建不得落 key")
			require.Nil(t, r.APIKeyHint, "无 key 创建不得落 hint")
		}
	}

	// 入参守卫：nil 请求体 / nil Data / 无身份
	_, err = svc.Create(opCtx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")
	require.Contains(t, err.Error(), "invalid parameter")
	_, err = svc.Create(opCtx, &aiV1.CreateAiProviderRequest{})
	require.Error(t, err, "nil Data 应被拒绝")
	_, err = svc.Create(ctx, &aiV1.CreateAiProviderRequest{Data: &aiV1.AiProvider{Name: trans.Ptr("x")}})
	require.Error(t, err, "缺操作人声明的创建应被拒绝")
}

// TestAiProviderService_UpdateApiKeySemantics 验证更新路径的 key 语义两分支：
// 带新 key → 加密落库 + 刷新 hint + 掩码追加 api_key_hint；不带 key → 本次不改 key
// （DTO 摘空 + 掩码摘除），已存 key/hint 原样保留。
func TestAiProviderService_UpdateApiKeySemantics(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiProviderServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := aiProviderOperator(ctx, 82)
	id := aiProviderCreateViaService(t, svc, entClient, opCtx, "更新校验提供商", "sk-oldkey12345")

	// 分支一：带新 key（掩码里甚至没写 api_key_hint）→ key/hint 都落库，掩码自动补 api_key_hint
	req := &aiV1.UpdateAiProviderRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"api_key", "name"}},
		Data: &aiV1.AiProvider{
			Name:   trans.Ptr("更新后名字"),
			ApiKey: trans.Ptr("sk-newkey98765"),
		},
	}
	_, err := svc.Update(opCtx, req)
	require.NoError(t, err, "带新 key 的更新应成功")
	require.Contains(t, req.UpdateMask.Paths, "api_key_hint", "带新 key 时掩码应自动追加 api_key_hint")

	row, err := entClient.Client().AiProvider.Query().
		Where(aiprovider.IDEQ(id)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "sk-newkey98765", *row.APIKey, "新 key 应加密（明文直通）落库")
	require.Equal(t, "sk-***8765", *row.APIKeyHint, "hint 应随新 key 刷新")
	require.Equal(t, "更新后名字", *row.Name)
	require.Equal(t, uint64(82), *row.UpdatedBy, "updated_by 应盖章为操作人")

	// 分支二：不带 key（DTO 带空串也算不带）→ 已存 key/hint 原样保留，掩码被摘除
	req2 := &aiV1.UpdateAiProviderRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "api_key", "api_key_hint"}},
		Data: &aiV1.AiProvider{
			Name:   trans.Ptr("换名不改密钥"),
			ApiKey: trans.Ptr(""), // 空串 = 本次不改 key
		},
	}
	_, err = svc.Update(opCtx, req2)
	require.NoError(t, err, "不带 key 的更新应成功")
	require.NotContains(t, req2.UpdateMask.Paths, "api_key", "掩码应摘除 api_key（防空值显式清写已存密钥）")
	require.NotContains(t, req2.UpdateMask.Paths, "api_key_hint", "掩码应摘除 api_key_hint")
	require.Contains(t, req2.UpdateMask.Paths, "updated_by", "服务层追加的盖章路径应保留")
	require.Nil(t, req2.Data.ApiKey, "DTO 的 api_key 应被摘空，防止空值经掩码写库")

	row, err = entClient.Client().AiProvider.Query().
		Where(aiprovider.IDEQ(id)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "sk-newkey98765", *row.APIKey, "不带 key 的更新不得把已存 key 抹掉")
	require.Equal(t, "sk-***8765", *row.APIKeyHint, "不带 key 的更新不得把已存 hint 抹掉")
	require.Equal(t, "换名不改密钥", *row.Name)

	// 入参守卫：nil 请求体 / nil Data / id=0 / 无身份
	_, err = svc.Update(opCtx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")
	require.Contains(t, err.Error(), "invalid parameter")
	_, err = svc.Update(opCtx, &aiV1.UpdateAiProviderRequest{})
	require.Error(t, err, "nil Data 应被拒绝")
	_, err = svc.Update(opCtx, &aiV1.UpdateAiProviderRequest{Id: 0, Data: &aiV1.AiProvider{}})
	require.Error(t, err, "id=0 应被拒绝")
	require.Contains(t, err.Error(), "id is required")
	_, err = svc.Update(ctx, &aiV1.UpdateAiProviderRequest{Id: id, Data: &aiV1.AiProvider{}})
	require.Error(t, err, "缺操作人声明的更新应被拒绝")
}

// TestAiProviderService_Delete 验证删除：删除后行数归零、再查报错、nil 请求被拒。
func TestAiProviderService_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newAiProviderServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := aiProviderOperator(ctx, 83)
	id := aiProviderCreateViaService(t, svc, entClient, opCtx, "待删除提供商", "sk-del12345678")

	_, err := svc.Delete(opCtx, &aiV1.DeleteAiProviderRequest{
		QueryBy: &aiV1.DeleteAiProviderRequest_Id{Id: id},
	})
	require.NoError(t, err, "删除已存在提供商应成功")

	cnt, err := entClient.Client().AiProvider.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, cnt, "删除后提供商表计数应归零")

	_, err = svc.Get(opCtx, &aiV1.GetAiProviderRequest{
		QueryBy: &aiV1.GetAiProviderRequest_Id{Id: id},
	})
	require.Error(t, err, "删除后的提供商应查询不到")

	_, err = svc.Delete(opCtx, nil)
	require.Error(t, err, "nil 请求体应被拒绝")
	require.Contains(t, err.Error(), "invalid parameter")
}
