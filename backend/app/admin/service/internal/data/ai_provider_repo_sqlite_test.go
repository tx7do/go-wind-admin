// AiProviderRepo 的 SQLite 内存库测试（白盒，包内测试）。
//
// 这张表是 AI 功能的钥匙柜，测试盯三件"读错一次就全线断供或泄密"的事：
//   - 密文不出 repo：DTO 的 api_key 在 Get/List 里必须恒为 nil（写入走 service 层
//     加密后另行传递），脱敏展示只靠 api_key_hint——这条一旦失守，任何列表页都会
//     把密文摆到前端；
//   - 枚举往返：model_type 是实体侧可空指针枚举列，必须经 converter 双向精确命中
//     （读视图落成 UNSPECIFIED 会让 chat 链路分不清云端/本地部署形态）；
//   - GetEnabledDefault 的三级语义：默认+启用优先、任一启用兜底、全禁用报错——
//     chat 服务靠它挑提供商，三级语义错了要么用错模型要么全线 404。
package data

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	aiV1 "go-wind-admin/api/gen/go/ai/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent/aiprovider"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newProviderPayload 构造一份完整提供商载荷（云端、启用、默认），可按需覆盖。
// 注意 provider 的 Create 没有 operatorID 参数，created_by 由载荷 CreatedBy 传递。
func newProviderPayload() *aiV1.AiProvider {
	return &aiV1.AiProvider{
		Name:           trans.Ptr("云上通义"),
		ModelType:      aiV1.AiProvider_CLOUD.Enum(),
		ModelName:      trans.Ptr("qwen-max"),
		BaseUrl:        trans.Ptr("https://dashscope.example.com/compatible-mode/v1"),
		ApiKey:         trans.Ptr("enc:sealed-secret"),
		ApiKeyHint:     trans.Ptr("sk-***abcd"),
		TimeoutSeconds: trans.Ptr(int32(60)),
		IsDefault:      trans.Ptr(true),
		IsEnabled:      trans.Ptr(true),
		Remark:         trans.Ptr("主力云端提供商"),
		CreatedBy:      trans.Ptr(uint64(1001)),
	}
}

// TestAiProviderRepoSqlite_Create 验证字段级落库与枚举双向映射：
// model_type 按 proto 枚举名落为实体枚举字面量、operatorID 落 created_by、
// created_at 显式写入；读视图上 model_type 精确回映射、api_key 恒为 nil。
func TestAiProviderRepoSqlite_Create(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiProviderRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	payload := newProviderPayload()
	payload.LocalPort = trans.Ptr(int32(8123))
	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: payload}), "完整载荷的 Create 应成功")

	row, err := entClient.Client().AiProvider.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "云上通义", *row.Name, "name 应按载荷落库")
	require.NotNil(t, row.ModelType)
	require.Equal(t, aiprovider.ModelTypeCLOUD, *row.ModelType,
		"proto 枚举名应经 converter 落为实体枚举字面量")
	require.Equal(t, "qwen-max", *row.ModelName)
	require.Equal(t, "https://dashscope.example.com/compatible-mode/v1", *row.BaseURL)
	require.Equal(t, "enc:sealed-secret", *row.APIKey, "api_key 密文应落库（解密在 service 层）")
	require.Equal(t, "sk-***abcd", *row.APIKeyHint)
	require.NotNil(t, row.LocalPort)
	require.Equal(t, 8123, *row.LocalPort, "proto int32 应经 int32PtrToIntPtr 转为 ent int 落库")
	require.Equal(t, 60, *row.TimeoutSeconds, "timeout_seconds 的 int32→int 转换同样要保值")
	require.NotNil(t, row.IsDefault)
	require.True(t, *row.IsDefault)
	require.NotNil(t, row.IsEnabled)
	require.True(t, *row.IsEnabled, "is_enabled 应按载荷落库")
	require.NotNil(t, row.CreatedBy)
	require.Equal(t, uint64(1001), *row.CreatedBy, "operatorID 应落 created_by")
	require.NotNil(t, row.CreatedAt, "created_at 必须显式写入：留给 NULL 列表页会出一排空行")

	// 读视图：枚举回映射 + 密文脱敏
	dto, err := repo.Get(ctx, &aiV1.GetAiProviderRequest{
		QueryBy: &aiV1.GetAiProviderRequest_Id{Id: row.ID},
	})
	require.NoError(t, err)
	require.Equal(t, row.ID, dto.GetId())
	require.Equal(t, aiV1.AiProvider_CLOUD, dto.GetModelType(),
		"实体枚举应经 converter 回映射为 proto 枚举，不能落 UNSPECIFIED 零值")
	require.Nil(t, dto.ApiKey, "密文不出 repo：Get 回读的 api_key 必须恒为空")
	require.Equal(t, "sk-***abcd", dto.GetApiKeyHint(), "脱敏提示是唯一的对外展示，必须保留")
	require.Equal(t, "qwen-max", dto.GetModelName())

	// 本地部署形态（LOCAL）：int 端口字段不能因为转换链路丢值，枚举也要走对字面量
	localPayload := newProviderPayload()
	localPayload.ModelType = aiV1.AiProvider_LOCAL.Enum()
	localPayload.BaseUrl = nil
	localPayload.ApiKey = nil
	localPayload.TimeoutSeconds = nil
	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: localPayload}), "本地形态（无 api_key）的 Create 应成功")

	row2, err := entClient.Client().AiProvider.Query().Where(aiprovider.ModelTypeEQ(aiprovider.ModelTypeLOCAL)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, aiprovider.ModelTypeLOCAL, *row2.ModelType)
	require.Nil(t, row2.APIKey, "载荷未带 api_key 就不该落值")

	// 入参守卫
	require.Error(t, repo.Create(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{}), "nil 载荷应返回 BadRequest")
}

// TestAiProviderRepoSqlite_GetEnabledDefault 验证三级挑选语义：
// 默认+启用优先、无默认时任一启用兜底、全禁用时 NotFound（chat 链路据此报
// "没有可用提供商"而不是静默用错行）。
func TestAiProviderRepoSqlite_GetEnabledDefault(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiProviderRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	// 空库：没有启用提供商就是 NotFound，不是 InternalServerError
	_, err := repo.GetEnabledDefault(ctx)
	require.Error(t, err, "无任何启用提供商应返回 NotFound")
	require.Contains(t, err.Error(), "no enabled ai provider")

	// 只有一个非默认的启用提供商：兜底语义，直接用它
	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: func() *aiV1.AiProvider {
		p := newProviderPayload()
		p.Name = trans.Ptr("备用提供商")
		p.IsDefault = trans.Ptr(false)
		return p
	}()}))
	fallbackRow, err := entClient.Client().AiProvider.Query().Where(aiprovider.NameEQ("备用提供商")).Only(ctx)
	require.NoError(t, err)
	idFallback := fallbackRow.ID

	got, err := repo.GetEnabledDefault(ctx)
	require.NoError(t, err)
	require.Equal(t, idFallback, got.ID, "无默认时应兜底取任一启用的提供商")

	// 再来一个默认+启用的：优先级应压过兜底行
	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: newProviderPayload()}))
	defaultRow, err := entClient.Client().AiProvider.Query().Where(aiprovider.NameEQ("云上通义")).Only(ctx)
	require.NoError(t, err)
	idDefault := defaultRow.ID

	got, err = repo.GetEnabledDefault(ctx)
	require.NoError(t, err)
	require.Equal(t, idDefault, got.ID, "默认且启用的提供商应优先于普通启用行")
	require.NotEqual(t, idFallback, got.ID)

	// 默认行被禁用：不能挑一个禁用的默认行，应回落到普通启用行
	_, err = entClient.Client().AiProvider.UpdateOneID(idDefault).
		SetIsEnabled(false).Save(ctx)
	require.NoError(t, err)

	got, err = repo.GetEnabledDefault(ctx)
	require.NoError(t, err)
	require.Equal(t, idFallback, got.ID, "禁用的默认行不参与挑选，应回落到启用的兜底行")
}

// TestAiProviderRepoSqlite_GetApiKeyEncrypted 验证密文按 ID 单独可取（service 层解密
// 前的取数通道）：有密文回密文、没配过回空串、行不存在回 NotFound。
func TestAiProviderRepoSqlite_GetApiKeyEncrypted(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiProviderRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: newProviderPayload()}))
	withKeyRow, err := entClient.Client().AiProvider.Query().Where(aiprovider.NameEQ("云上通义")).Only(ctx)
	require.NoError(t, err)
	withKey := withKeyRow.ID

	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: func() *aiV1.AiProvider {
		p := newProviderPayload()
		p.Name = trans.Ptr("本地无钥")
		p.ApiKey = nil
		return p
	}()}))
	withoutKeyRow, err := entClient.Client().AiProvider.Query().Where(aiprovider.NameEQ("本地无钥")).Only(ctx)
	require.NoError(t, err)
	withoutKey := withoutKeyRow.ID

	key, err := repo.GetApiKeyEncrypted(ctx, withKey)
	require.NoError(t, err)
	require.Equal(t, "enc:sealed-secret", key, "取出的应是落库的原始密文（解密是 service 层的事）")

	key, err = repo.GetApiKeyEncrypted(ctx, withoutKey)
	require.NoError(t, err, "没配 api_key 不是错误")
	require.Equal(t, "", key, "api_key 为 NULL 应回空串而不是报错")

	_, err = repo.GetApiKeyEncrypted(ctx, 9999999)
	require.Error(t, err, "不存在的提供商应返回 NotFound")
	require.Contains(t, err.Error(), "ai provider not found")
}

// TestAiProviderRepoSqlite_GetEntityByID 验证实体取数通道：命中回实体、不存在回 NotFound
// （chat 链路据此做提供商存在性校验）。
func TestAiProviderRepoSqlite_GetEntityByID(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiProviderRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: newProviderPayload()}))
	row, err := entClient.Client().AiProvider.Query().Where(aiprovider.NameEQ("云上通义")).Only(ctx)
	require.NoError(t, err)
	id := row.ID

	entity, err := repo.GetEntityByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, entity.ID)
	require.Equal(t, "云上通义", *entity.Name)

	_, err = repo.GetEntityByID(ctx, 424242)
	require.Error(t, err, "不存在的 ID 应返回 NotFound")
	require.Contains(t, err.Error(), "ai provider not found")
}

// TestAiProviderRepoSqlite_List 验证分页 total 与 name 的 contains 过滤；
// 列表同样不得漏密文；nil 请求返回 BadRequest。
func TestAiProviderRepoSqlite_List(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiProviderRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: func() *aiV1.AiProvider {
		p := newProviderPayload()
		p.Name = trans.Ptr("marker-云端")
		return p
	}()}))
	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: func() *aiV1.AiProvider {
		p := newProviderPayload()
		p.Name = trans.Ptr("本地ollama")
		p.ModelType = aiV1.AiProvider_LOCAL.Enum()
		p.IsDefault = trans.Ptr(false)
		p.ApiKey = nil
		return p
	}()}))

	all, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), all.GetTotal())
	require.Len(t, all.GetItems(), 2)
	for _, it := range all.GetItems() {
		require.Nil(t, it.ApiKey, "列表读视图同样不得带出密文（这条失守就是批量泄密）")
	}

	filtered, err := repo.List(ctx, &paginationV1.PagingRequest{
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: `{"name__contains":"marker"}`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.GetTotal(), "contains 过滤应只统计命中行")
	require.Len(t, filtered.GetItems(), 1)
	require.Equal(t, "marker-云端", filtered.GetItems()[0].GetName())

	_, err = repo.List(ctx, nil)
	require.Error(t, err, "nil 分页请求应返回 BadRequest")

	// Count 与 List 的 total 口径一致
	cnt, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(2), cnt.GetCount(), "Count 应与 List 的 total 同口径")
}

// TestAiProviderRepoSqlite_Update 验证掩码语义：掩码内字段更新（枚举经 converter 落库）、
// 掩码外字段保持原值、updated_by 盖操作人、updated_at 写入；allow_missing 时对不存在 ID
// 转创建；nil/缺 ID 返回 BadRequest。
func TestAiProviderRepoSqlite_Update(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiProviderRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: newProviderPayload()}))
	createdRow, err := entClient.Client().AiProvider.Query().Where(aiprovider.NameEQ("云上通义")).Only(ctx)
	require.NoError(t, err)
	id := createdRow.ID

	// 把主力从云端切到本地：这正是提供商编辑页存在的理由。
	// 掩码内显式给 nil 的字段（base_url/api_key）走置 NULL 语义，掩码外字段保持原值。
	require.NoError(t, repo.Update(ctx, &aiV1.UpdateAiProviderRequest{
		Id:         id,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"model_type", "model_name", "base_url", "api_key"}},
		Data: &aiV1.AiProvider{
			ModelType: aiV1.AiProvider_LOCAL.Enum(),
			ModelName: trans.Ptr("qwen2.5-72b"),
			BaseUrl:   nil,
			ApiKey:    nil,
			Remark:    trans.Ptr("masked-out-remark"),
		},
	}))

	row, err := entClient.Client().AiProvider.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, aiprovider.ModelTypeLOCAL, *row.ModelType, "掩码内的 model_type 应经 converter 更新")
	require.Equal(t, "qwen2.5-72b", *row.ModelName, "掩码内的 model_name 应更新")
	require.Nil(t, row.BaseURL, "掩码内显式 nil 的 base_url 应被置 NULL（本地部署没有 API 地址）")
	require.Nil(t, row.APIKey, "掩码内显式 nil 的 api_key 应被置 NULL")
	require.Equal(t, "主力云端提供商", *row.Remark, "掩码外的 remark 应保持原值（Data 里带了个 masked-out-remark 也不许写进去）")
	require.Nil(t, row.UpdatedBy, "掩码未点名 updated_by 且载荷未带时不得凭空造出操作人（创建路径本就不写 updated_by）")
	require.NotNil(t, row.UpdatedAt, "更新应写入 updated_at")

	// 不带掩码的全量更新：Data.UpdatedBy 直写 updated_by（操作人由载荷传递）
	require.NoError(t, repo.Update(ctx, &aiV1.UpdateAiProviderRequest{
		Id: id,
		Data: &aiV1.AiProvider{
			Remark:    trans.Ptr("已切换本地部署"),
			UpdatedBy: trans.Ptr(uint64(2001)),
		},
	}))
	row, err = entClient.Client().AiProvider.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "已切换本地部署", *row.Remark)
	require.Equal(t, "qwen2.5-72b", *row.ModelName, "无掩码时未带字段是 SetNillable no-op，不应清掉原值")
	require.NotNil(t, row.UpdatedBy)
	require.Equal(t, uint64(2001), *row.UpdatedBy, "updated_by 应按载荷盖成新操作人")
	require.NotNil(t, row.UpdatedAt)

	// allow_missing：对不存在的 ID 转创建（upsert 语义）。目标 ID 由载荷 Data.Id 承载
	// （service 层 Update 前固定填 req.Data.Id = req.GetId()，repo 的创建路径据此 SetID）
	require.NoError(t, repo.Update(ctx, &aiV1.UpdateAiProviderRequest{
		Id:           777001,
		AllowMissing: trans.Ptr(true),
		Data: func() *aiV1.AiProvider {
			p := newProviderPayload()
			p.Id = trans.Ptr(uint64(777001))
			return p
		}(),
	}))
	created, err := entClient.Client().AiProvider.Get(ctx, 777001)
	require.NoError(t, err, "allow_missing 应把不存在的 ID 转为创建（ID 取自载荷 Data.Id）")
	require.Equal(t, "云上通义", *created.Name)
	require.NotNil(t, created.CreatedAt, "转创建路径也要写 created_at")

	// 不存在 ID 且未开 allow_missing：无效果、不新增行
	require.NoError(t, repo.Update(ctx, &aiV1.UpdateAiProviderRequest{
		Id:   987654,
		Data: &aiV1.AiProvider{Name: trans.Ptr("幽灵提供商")},
	}))
	_, err = entClient.Client().AiProvider.Get(ctx, 987654)
	require.Error(t, err, "对不存在 ID 的普通更新不应创建行")
	cnt, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(2), cnt.GetCount(), "失败更新不应影响行数")

	// 入参守卫
	require.Error(t, repo.Update(ctx, nil), "nil 请求应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &aiV1.UpdateAiProviderRequest{Id: id}), "nil Data 应返回 BadRequest")
	require.Error(t, repo.Update(ctx, &aiV1.UpdateAiProviderRequest{
		Data: &aiV1.AiProvider{},
	}), "id=0 应返回 BadRequest")
}

// TestAiProviderRepoSqlite_Delete 验证按 ID 删除行、nil 请求 BadRequest、
// 不存在 ID 无错误且行数不变。
func TestAiProviderRepoSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	repo := NewAiProviderRepoForTest(entClient)
	ctx := enttest.NewSystemContext(context.Background())

	require.NoError(t, repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: newProviderPayload()}))
	idARow, err := entClient.Client().AiProvider.Query().Where(aiprovider.NameEQ("云上通义")).Only(ctx)
	require.NoError(t, err)
	idA := idARow.ID
	err = repo.Create(ctx, &aiV1.CreateAiProviderRequest{Data: func() *aiV1.AiProvider {
		p := newProviderPayload()
		p.Name = trans.Ptr("留任提供商")
		p.IsDefault = trans.Ptr(false)
		return p
	}()})
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, &aiV1.DeleteAiProviderRequest{
		QueryBy: &aiV1.DeleteAiProviderRequest_Id{Id: idA},
	}), "按存在 ID 删除应成功")

	_, err = repo.GetEntityByID(ctx, idA)
	require.Error(t, err, "删除后该行应查不到")

	cnt, err := repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), cnt.GetCount(), "只该删掉一行")

	require.Error(t, repo.Delete(ctx, nil), "nil 请求应返回 BadRequest")
	require.NoError(t, repo.Delete(ctx, &aiV1.DeleteAiProviderRequest{
		QueryBy: &aiV1.DeleteAiProviderRequest_Id{Id: 9999999},
	}), "删除不存在的 ID 不应报错（ent 删除 0 行不视为失败）")
	cnt, err = repo.Count(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), cnt.GetCount(), "失败删除不应影响行数")
}
