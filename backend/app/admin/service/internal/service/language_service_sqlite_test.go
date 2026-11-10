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

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	dictV1 "go-wind-admin/api/gen/go/dict/service/v1"
	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/ent"
	"go-wind-admin/app/admin/service/internal/data/enttest"
	"go-wind-admin/pkg/constants"
	"go-wind-admin/pkg/middleware/auth"
)

// newLanguageServiceForTest 白盒构造 LanguageService，逐字段对齐 NewLanguageService
// 的装配（log 用 NopLogger helper）；init()（默认语言播种）由测试按需显式调用。
func newLanguageServiceForTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client]) *LanguageService {
	t.Helper()
	return &LanguageService{
		log:          bLogger.NewHelper(bLogger.NopLogger()),
		languageRepo: data.NewLanguageRepoForTest(entClient),
	}
}

// TestLanguageServiceSqlite_InitSeedsDefaultsOnEmptyTable 空表上调用 init() 应播种
// constants.DefaultLanguages 全部默认语言（count==0 守卫的正分支），List 应全部返回；
// 其中 zh-CN 应是唯一默认语言。
func TestLanguageServiceSqlite_InitSeedsDefaultsOnEmptyTable(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newLanguageServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	svc.init()

	cnt, err := entClient.Client().Language.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, len(constants.DefaultLanguages), cnt, "init() 后应播种全部默认语言")

	listResp, err := svc.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(len(constants.DefaultLanguages)), listResp.Total, "List 应统计全部已播种语言")
	require.Len(t, listResp.Items, len(constants.DefaultLanguages), "List 应返回全部已播种语言")

	zhFound := false
	for _, item := range listResp.Items {
		if item.GetLanguageCode() == "zh-CN" {
			zhFound = true
			require.True(t, item.GetIsDefault(), "zh-CN 应是默认语言")
			require.True(t, item.GetIsEnabled(), "zh-CN 应处于启用状态")
		} else {
			require.False(t, item.GetIsDefault(), "除 zh-CN 外不应有其他默认语言")
		}
	}
	require.True(t, zhFound, "播种结果中应存在 zh-CN")
}

// TestLanguageServiceSqlite_InitGuardSkipsReseedWhenNonEmpty 表非空时再次调用 init()
// 不应重复播种（count==0 守卫的负分支）。同时覆盖服务层 Create：操作人 ID 盖入 created_by。
func TestLanguageServiceSqlite_InitGuardSkipsReseedWhenNonEmpty(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newLanguageServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	svc.init()
	seeded := len(constants.DefaultLanguages)

	_, err := svc.Create(opCtx, &dictV1.CreateLanguageRequest{
		Data: &dictV1.Language{
			LanguageName: trans.Ptr("服务层新增语言"),
			LanguageCode: trans.Ptr("tst-SVCX"),
			NativeName:   trans.Ptr("native-svcx"),
		},
	})
	require.NoError(t, err, "服务层 Create 应成功")

	rows, err := entClient.Client().Language.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, seeded+1, "播种后再加一条自定义语言")
	var customFound bool
	for _, r := range rows {
		if r.LanguageCode != nil && *r.LanguageCode == "tst-SVCX" {
			customFound = true
			require.NotNil(t, r.CreatedBy, "created_by 应被服务层盖入操作人 ID")
			require.Equal(t, int64(7), *r.CreatedBy, "created_by 应等于令牌声明中的操作人 ID")
		}
	}
	require.True(t, customFound, "自定义语言行应存在")

	// 表非空：再次 init() 不应重新播种
	svc.init()
	cnt, err := entClient.Client().Language.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, seeded+1, cnt, "非空表上重复 init() 不应追加或重置语言行")
}

// TestLanguageServiceSqlite_Get_ByIdAndByCode 验证服务层 Get 按主键与按语言代码
// 查询的命中与未命中。
func TestLanguageServiceSqlite_Get_ByIdAndByCode(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newLanguageServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	svc.init()

	rows, err := entClient.Client().Language.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(constants.DefaultLanguages))
	var zhID int64
	for _, r := range rows {
		if r.LanguageCode != nil && *r.LanguageCode == "zh-CN" {
			zhID = int64(r.ID)
		}
	}
	require.NotZero(t, zhID, "播种结果中应找到 zh-CN 的主键")

	gotByID, err := svc.Get(ctx, &dictV1.GetLanguageRequest{
		QueryBy: &dictV1.GetLanguageRequest_Id{Id: zhID},
	})
	require.NoError(t, err, "按已存在主键查询应命中")
	require.Equal(t, "zh-CN", gotByID.GetLanguageCode(), "命中记录的 language_code 应为 zh-CN")
	require.True(t, gotByID.GetIsDefault(), "zh-CN 应标记为默认语言")

	gotByCode, err := svc.Get(ctx, &dictV1.GetLanguageRequest{
		QueryBy: &dictV1.GetLanguageRequest_Code{Code: "en-US"},
	})
	require.NoError(t, err, "按已存在语言代码查询应命中")
	require.Equal(t, "en-US", gotByCode.GetLanguageCode(), "按代码查询应命中 en-US")
	require.False(t, gotByCode.GetIsDefault(), "en-US 不应是默认语言")

	_, err = svc.Get(ctx, &dictV1.GetLanguageRequest{
		QueryBy: &dictV1.GetLanguageRequest_Id{Id: 9999999},
	})
	require.Error(t, err, "按不存在主键查询应返回错误")

	_, err = svc.Get(ctx, &dictV1.GetLanguageRequest{
		QueryBy: &dictV1.GetLanguageRequest_Code{Code: "no-such-lang-XX"},
	})
	require.Error(t, err, "按不存在的语言代码查询应返回错误")
}

// TestLanguageServiceSqlite_Update_OnlyMaskedFields 验证服务层 Update 在掩码内字段
// 更新、掩码外字段（language_code）保持原值；注意 language_name/native_name 必须同时
// 进掩码（更新路径对二者做非空校验）。并断言 updated_by 被服务层盖入操作人 ID。
func TestLanguageServiceSqlite_Update_OnlyMaskedFields(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newLanguageServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())
	opCtx := auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)})

	svc.init()

	rows, err := entClient.Client().Language.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(constants.DefaultLanguages))
	var koID int64
	for _, r := range rows {
		if r.LanguageCode != nil && *r.LanguageCode == "ko-KR" {
			koID = int64(r.ID)
		}
	}
	require.NotZero(t, koID, "播种结果中应找到 ko-KR 的主键")

	_, err = svc.Update(opCtx, &dictV1.UpdateLanguageRequest{
		Id:         koID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"language_name", "native_name"}},
		Data: &dictV1.Language{
			LanguageName: trans.Ptr("更新后的韩语名称"),
			NativeName:   trans.Ptr("업데이트된 이름"),
		},
	})
	require.NoError(t, err, "掩码内字段更新应成功")

	after, err := entClient.Client().Language.Query().All(ctx)
	require.NoError(t, err)
	for _, r := range after {
		if r.LanguageCode != nil && *r.LanguageCode == "ko-KR" {
			require.Equal(t, "更新后的韩语名称", *r.LanguageName, "掩码内字段 language_name 应被更新")
			require.Equal(t, "업데이트된 이름", *r.NativeName, "掩码内字段 native_name 应被更新")
			require.NotNil(t, r.UpdatedBy, "服务层应把操作人 ID 盖入 updated_by")
			require.Equal(t, int64(7), *r.UpdatedBy, "updated_by 应等于令牌声明中的操作人 ID")
		}
	}
}

// TestLanguageServiceSqlite_Delete 验证服务层 Delete 删除一条已播种语言后计数减一。
func TestLanguageServiceSqlite_Delete(t *testing.T) {
	entClient := enttest.NewEntClientForTest(t)
	svc := newLanguageServiceForTest(t, entClient)
	ctx := enttest.NewSystemContext(context.Background())

	svc.init()

	rows, err := entClient.Client().Language.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(constants.DefaultLanguages))
	delID := int64(rows[0].ID)

	_, err = svc.Delete(auth.NewContext(ctx, &authenticationV1.UserTokenPayload{UserId: 7, IsPlatformAdmin: trans.Ptr(true)}), &dictV1.DeleteLanguageRequest{
		QueryBy: &dictV1.DeleteLanguageRequest_Id{Id: delID},
	})
	require.NoError(t, err, "删除已存在记录应成功")

	cnt, err := entClient.Client().Language.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, len(constants.DefaultLanguages)-1, cnt, "删除一条后计数应减一")
}
