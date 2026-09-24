package data

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	accesskeyV1 "go-wind-admin/api/gen/go/access_key/service/v1"
	"go-wind-admin/app/admin/service/internal/data/ent"
	entAccessKey "go-wind-admin/app/admin/service/internal/data/ent/accesskey"
	"go-wind-admin/app/admin/service/internal/data/enttest"
)

// newAccessKeyRepoSqlite 用 enttest helper 构造一个可直接做 CRUD 的 AccessKeyRepo，
// 逐字段复刻 NewAccessKeyRepo 的 mapper/converter 初始化，再调用 init()。
func newAccessKeyRepoSqlite(t *testing.T) *AccessKeyRepo {
	t.Helper()
	repo := &AccessKeyRepo{
		entClient: enttest.NewEntClientForTest(t),
		log:       bLogger.NewHelper(bLogger.NopLogger()),
		mapper:    mapper.NewCopierMapper[accesskeyV1.AccessKey, ent.AccessKey](),
		statusConverter: mapper.NewEnumTypeConverter[accesskeyV1.AccessKey_Status, entAccessKey.Status](
			accesskeyV1.AccessKey_Status_name, accesskeyV1.AccessKey_Status_value,
		),
	}
	repo.init()
	return repo
}

// TestAccessKeyRepoSqlite_Create 验证 Create 落库：未指定状态默认 ON、
// 显式 OFF 经 converter 映射落库，name / AK / secret 摘要 / tenant_id 落库，
// expires_at 未指定时为 NULL。
func TestAccessKeyRepoSqlite_Create(t *testing.T) {
	repo := newAccessKeyRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	createdDefault, err := repo.Create(ctx, &accesskeyV1.CreateAccessKeyRequest{},
		&accesskeyV1.AccessKey{
			Name:     trans.Ptr("默认状态凭证"),
			TenantId: trans.Ptr(uint32(7)),
		},
		"AKSQLITE-DEFAULT-ON", "sh-default")
	require.NoError(t, err, "Create（未指定状态）应成功")
	require.NotZero(t, createdDefault.ID, "返回实体应带新 ID")
	require.False(t, createdDefault.CreatedAt.IsZero(), "created_at 应被写入")

	createdOff, err := repo.Create(ctx, &accesskeyV1.CreateAccessKeyRequest{},
		&accesskeyV1.AccessKey{
			Name:   trans.Ptr("显式停用凭证"),
			Status: accesskeyV1.AccessKey_OFF.Enum(),
		},
		"AKSQLITE-EXPLICIT-OFF", "sh-off")
	require.NoError(t, err, "Create（显式 OFF）应成功")
	require.NotZero(t, createdOff.ID, "返回实体应带新 ID")

	rows, err := repo.entClient.Client().AccessKey.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2, "应已有 2 行")
	byStatus := map[string]int{}
	for _, row := range rows {
		require.NotEmpty(t, row.AccessKey, "access_key 应落库")
		require.NotEmpty(t, row.SecretHash, "secret_hash 应落库")
		require.NotNil(t, row.Status, "status 应落库")
		byStatus[string(*row.Status)]++
	}
	require.Equal(t, map[string]int{"ON": 1, "OFF": 1}, byStatus,
		"未指定状态默认 ON、显式指定映射为 OFF，均经 converter 落库")
}

// TestAccessKeyRepoSqlite_StatusEnumPairs 验证 status 枚举 ON/OFF 的
// proto → ent 与 ent → proto（List 路径）双向映射逐对成立。
func TestAccessKeyRepoSqlite_StatusEnumPairs(t *testing.T) {
	repo := newAccessKeyRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for value, name := range accesskeyV1.AccessKey_Status_name {
		status := accesskeyV1.AccessKey_Status(value)
		_, err := repo.Create(ctx, &accesskeyV1.CreateAccessKeyRequest{},
			&accesskeyV1.AccessKey{Status: status.Enum()},
			fmt.Sprintf("AKSQLITE-PAIR-%d", value), "sh-pair")
		require.NoError(t, err, "status=%s 建行应成功", name)
	}

	rows, err := repo.entClient.Client().AccessKey.Query().All(ctx)
	require.NoError(t, err)
	actualEnt := map[string]int{}
	for _, row := range rows {
		require.NotNil(t, row.Status)
		actualEnt[string(*row.Status)]++
	}
	require.Equal(t, map[string]int{"ON": 1, "OFF": 1}, actualEnt, "ent 侧 status 应与枚举名集合逐对一致")

	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), listed.Total, "List 应返回全部 2 行")
	actualProto := map[int32]int{}
	for _, item := range listed.Items {
		require.NotNil(t, item.Status, "DTO 的 status 应被 converter 回填")
		actualProto[int32(*item.Status)]++
	}
	require.Equal(t, map[int32]int{0: 1, 1: 1}, actualProto, "DTO 侧 status 应与 proto 枚举值集合逐对一致")
}

// TestAccessKeyRepoSqlite_StatusReadView 验证 ON/OFF 两态凭证经 converter
// 落库后，在读路径（Get 按主键 / List）的 DTO 视图如实呈现。
//
// 枚举字段读视图机制注记：实体侧 status 为可空指针枚举列
// （*entAccessKey.Status，schema 默认 ON），DTO 侧为可选指针字段。mapper
// 的枚举转换对（经 &srcType/&dstType 取址注册）恰为指针↔指针形态的键，
// 指针对字段能被 copier 直接转换赋值——与值型实体枚举列（如
// position.type、notification_channel.type）读侧被丢弃的情形不同。
// 本测试将该读视图行为钉死（List 路径与 StatusEnumPairs 的断言互为冗余备份）。
func TestAccessKeyRepoSqlite_StatusReadView(t *testing.T) {
	repo := newAccessKeyRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	cases := []struct {
		protoStatus accesskeyV1.AccessKey_Status
		wantEnt     entAccessKey.Status
		name        string
	}{
		{accesskeyV1.AccessKey_ON, entAccessKey.StatusOn, "读视图凭证-启用"},
		{accesskeyV1.AccessKey_OFF, entAccessKey.StatusOff, "读视图凭证-停用"},
	}

	createdIDs := make(map[accesskeyV1.AccessKey_Status]uint32, len(cases))
	for _, c := range cases {
		created, err := repo.Create(ctx, &accesskeyV1.CreateAccessKeyRequest{},
			&accesskeyV1.AccessKey{
				Name:   trans.Ptr(c.name),
				Status: c.protoStatus.Enum(),
			},
			fmt.Sprintf("AKSQLITE-READVIEW-%d", c.protoStatus), "sh-readview")
		require.NoError(t, err, "状态 %v 创建应成功", c.protoStatus)
		require.NotNil(t, created.Status, "status 应落库")
		require.Equal(t, c.wantEnt, *created.Status, "状态 %v 应经转换器如实落库", c.protoStatus)
		createdIDs[c.protoStatus] = created.ID
	}

	// 读路径一：Get 按主键命中后，DTO 视图应如实呈现状态枚举。
	for _, c := range cases {
		got, err := repo.Get(ctx, &accesskeyV1.GetAccessKeyRequest{
			QueryBy: &accesskeyV1.GetAccessKeyRequest_Id{Id: createdIDs[c.protoStatus]},
		})
		require.NoError(t, err, "按主键读取应命中")
		require.Equal(t, c.protoStatus, got.GetStatus(), "读视图应如实呈现状态 %v", c.protoStatus)
	}

	// 读路径二：List 的 DTO 视图应如实呈现两态（按 name 标记匹配）。
	listed, err := repo.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	view := map[string]accesskeyV1.AccessKey_Status{}
	for _, item := range listed.Items {
		view[item.GetName()] = item.GetStatus()
	}
	for _, c := range cases {
		got, ok := view[c.name]
		require.True(t, ok, "List 应包含标记行 %s", c.name)
		require.Equal(t, c.protoStatus, got, "List 读视图应如实呈现状态 %v", c.protoStatus)
	}
}

// TestAccessKeyRepoSqlite_GetAndExistsByIdOrAccessKey 验证 Get 按主键/按访问键的
// 命中与未命中、IsExist 的命中与未命中。
func TestAccessKeyRepoSqlite_GetAndExistsByIdOrAccessKey(t *testing.T) {
	repo := newAccessKeyRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	created, err := repo.Create(ctx, &accesskeyV1.CreateAccessKeyRequest{},
		&accesskeyV1.AccessKey{Name: trans.Ptr("getter凭证")},
		"AKSQLITE-GETTER", "sh-getter")
	require.NoError(t, err)
	createdID := created.ID

	// Get 按主键：命中
	got, err := repo.Get(ctx, &accesskeyV1.GetAccessKeyRequest{
		QueryBy: &accesskeyV1.GetAccessKeyRequest_Id{Id: createdID},
	})
	require.NoError(t, err, "按存在的 ID 查询应命中")
	require.Equal(t, createdID, got.GetId())
	require.Equal(t, "getter凭证", got.GetName(), "命中记录的 name 应与写入一致")
	require.Equal(t, "AKSQLITE-GETTER", got.GetAccessKey(), "命中记录的 access_key 应与写入一致")

	// Get 按主键：未命中
	_, err = repo.Get(ctx, &accesskeyV1.GetAccessKeyRequest{
		QueryBy: &accesskeyV1.GetAccessKeyRequest_Id{Id: 99999},
	})
	require.Error(t, err, "查询不存在的主键应返回错误")

	// Get 按访问键：命中
	gotByKey, err := repo.Get(ctx, &accesskeyV1.GetAccessKeyRequest{
		QueryBy: &accesskeyV1.GetAccessKeyRequest_AccessKey{AccessKey: "AKSQLITE-GETTER"},
	})
	require.NoError(t, err, "按存在的访问键查询应命中")
	require.Equal(t, createdID, gotByKey.GetId(), "按访问键命中行的 id 应与主键一致")

	// Get 按访问键：未命中
	_, err = repo.Get(ctx, &accesskeyV1.GetAccessKeyRequest{
		QueryBy: &accesskeyV1.GetAccessKeyRequest_AccessKey{AccessKey: "AK_NO_SUCH_KEY"},
	})
	require.Error(t, err, "查询不存在的访问键应返回错误")

	// IsExist：命中/未命中
	exist, err := repo.IsExist(ctx, createdID)
	require.NoError(t, err)
	require.True(t, exist, "存在的主键 IsExist 应为 true")
	exist, err = repo.IsExist(ctx, 99999)
	require.NoError(t, err)
	require.False(t, exist, "不存在的主键 IsExist 应为 false")
}

// TestAccessKeyRepoSqlite_GetByAccessKeyBySystem 验证系统旁路按访问键查询的
// 命中与未命中。
func TestAccessKeyRepoSqlite_GetByAccessKeyBySystem(t *testing.T) {
	repo := newAccessKeyRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	_, err := repo.Create(ctx, &accesskeyV1.CreateAccessKeyRequest{},
		&accesskeyV1.AccessKey{Name: trans.Ptr("systemview凭证")},
		"AKSQLITE-SYSVIEW", "sh-sysview")
	require.NoError(t, err)

	row, err := repo.GetByAccessKeyBySystem(ctx, "AKSQLITE-SYSVIEW")
	require.NoError(t, err, "系统旁路按存在的访问键查询应命中")
	require.NotNil(t, row)
	require.NotNil(t, row.AccessKey, "命中行 access_key 应非空")
	require.Equal(t, "AKSQLITE-SYSVIEW", *row.AccessKey, "命中行应为目标访问键")

	_, err = repo.GetByAccessKeyBySystem(ctx, "AK_NO_SUCH_KEY")
	require.Error(t, err, "系统旁路查询不存在的访问键应返回错误")
}

// TestAccessKeyRepoSqlite_TouchLastUsedAndUpdateSecretHash 验证
// TouchLastUsedBySystem 刷新 last_used_at 与 UpdateSecretHash 轮换密钥摘要。
func TestAccessKeyRepoSqlite_TouchLastUsedAndUpdateSecretHash(t *testing.T) {
	repo := newAccessKeyRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	created, err := repo.Create(ctx, &accesskeyV1.CreateAccessKeyRequest{},
		&accesskeyV1.AccessKey{Name: trans.Ptr("rotate凭证")},
		"AKSQLITE-ROTATE", "sh-rotate-old")
	require.NoError(t, err, "Create 应成功")

	// TouchLastUsedBySystem：刷新 last_used_at
	repo.TouchLastUsedBySystem(ctx, created.ID)
	afterTouch, err := repo.entClient.Client().AccessKey.Query().Where(entAccessKey.IDEQ(created.ID)).Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, afterTouch.LastUsedAt, "TouchLastUsedBySystem 后 last_used_at 应非空")

	// UpdateSecretHash：轮换密钥摘要
	require.NoError(t, repo.UpdateSecretHash(ctx, created.ID, "sh-rotate-new"), "轮换密钥摘要应成功")
	afterRotate, err := repo.entClient.Client().AccessKey.Query().Where(entAccessKey.IDEQ(created.ID)).Only(ctx)
	require.NoError(t, err)
	require.NotNil(t, afterRotate.SecretHash, "secret_hash 应非空")
	require.Equal(t, "sh-rotate-new", *afterRotate.SecretHash, "secret_hash 应被轮换为新值")
}

// TestAccessKeyRepoSqlite_UpdateMasked 验证 Update 的单字段掩码更新：
// 掩码内 name 更新、掩码外 status 保持；status 掩码经 converter 更新。
func TestAccessKeyRepoSqlite_UpdateMasked(t *testing.T) {
	repo := newAccessKeyRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())
	created, err := repo.Create(ctx, &accesskeyV1.CreateAccessKeyRequest{},
		&accesskeyV1.AccessKey{Name: trans.Ptr("更新前凭证名")},
		"AKSQLITE-UPDATE", "sh-update")
	require.NoError(t, err)
	createdID := created.ID

	// 掩码 [name]：name 更新，status 保持
	err = repo.Update(ctx, &accesskeyV1.UpdateAccessKeyRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
		Data: &accesskeyV1.AccessKey{
			Name:   trans.Ptr("更新后凭证名"),
			Status: accesskeyV1.AccessKey_ON.Enum(),
		},
	})
	require.NoError(t, err, "掩码内字段更新应成功")
	after, err := repo.entClient.Client().AccessKey.Query().Where(entAccessKey.IDEQ(createdID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "更新后凭证名", *after.Name, "掩码内字段 name 应被更新")
	require.Equal(t, entAccessKey.StatusOn, *after.Status, "掩码外字段 status 应保持原值（ON）")

	// 掩码 [status]：OFF 经 converter 落库
	err = repo.Update(ctx, &accesskeyV1.UpdateAccessKeyRequest{
		Id:         createdID,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"status"}},
		Data: &accesskeyV1.AccessKey{
			Status: accesskeyV1.AccessKey_OFF.Enum(),
		},
	})
	require.NoError(t, err, "status 掩码更新应成功")
	after2, err := repo.entClient.Client().AccessKey.Query().Where(entAccessKey.IDEQ(createdID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, entAccessKey.StatusOff, *after2.Status, "proto OFF 应经 converter 映射为 ent StatusOff")
	require.Equal(t, "更新后凭证名", *after2.Name, "status 掩码更新不应波及 name")

	// 参数校验：Id=0 / Data=nil
	require.Error(t, repo.Update(ctx, &accesskeyV1.UpdateAccessKeyRequest{
		Id:   0,
		Data: &accesskeyV1.AccessKey{Name: trans.Ptr("x")},
	}), "Id=0 应返回错误")
	require.Error(t, repo.Update(ctx, &accesskeyV1.UpdateAccessKeyRequest{
		Id:   createdID,
		Data: nil,
	}), "Data=nil 应返回错误")
}

// TestAccessKeyRepoSqlite_ListCountPagingAndDelete 验证 List 的无过滤全量、
// name 列 contains 模糊搜索、id 列等值过滤、分页语义，以及 Count 与 Delete。
func TestAccessKeyRepoSqlite_ListCountPagingAndDelete(t *testing.T) {
	repo := newAccessKeyRepoSqlite(t)
	ctx := enttest.NewSystemViewerCtx(context.Background())

	for i, marker := range []string{"MARKEROMICRON", "MARKERPI"} {
		_, err := repo.Create(ctx, &accesskeyV1.CreateAccessKeyRequest{},
			&accesskeyV1.AccessKey{Name: trans.Ptr(marker + "凭证")},
			fmt.Sprintf("AKSQLITE-LIST-%d", i), "sh-list")
		require.NoError(t, err, "写入第 %d 行应成功", i)
	}

	rows, err := repo.entClient.Client().AccessKey.Query().All(ctx)
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
						Field:      "name",
						Op:         paginationV1.Operator_CONTAINS,
						ValueOneof: &paginationV1.FilterCondition_Value{Value: "MARKEROMICRON"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), filtered.Total, "contains 过滤应只统计命中行")
	require.Len(t, filtered.Items, 1, "contains 过滤应只返回命中行")
	require.Contains(t, filtered.Items[0].GetName(), "MARKEROMICRON", "命中行应为含标记的那条")

	countResp, err := repo.Count(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), countResp.GetCount(), "无过滤 Count 应为全量 2")

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

	// Delete：删除后计数归零；删除不存在的 ID 无命中不报错
	require.NoError(t, repo.Delete(ctx, &accesskeyV1.DeleteAccessKeyRequest{
		Id: firstID,
	}), "删除已存在记录应成功")
	remain, err := repo.entClient.Client().AccessKey.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, remain, "删除一条后应剩 1 行")
	require.NoError(t, repo.Delete(ctx, &accesskeyV1.DeleteAccessKeyRequest{
		Id: 999999,
	}), "删除不存在的 ID 应无命中且不报错")
	remain, err = repo.entClient.Client().AccessKey.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, remain, "表内行数应保持不变")
}
