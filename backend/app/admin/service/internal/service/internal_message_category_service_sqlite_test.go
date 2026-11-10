// InternalMessageCategoryService 的 SQLite 内存库测试。
//
// 站内信分类是收件箱筛选的字典表，测试盯两层：
//   - 守卫：Create/Update 缺 Data（或 nil 请求）报 BadRequest——此服务的守卫
//     曾漏 req 本体判空（nil 请求直接 panic），修复后以测试钉住不回退；
//   - 操作人盖章：Create 落 CreatedBy、Update 落 UpdatedBy 并追加进掩码；
//     分类落库/读回走 repo（repo 侧语义由 data 包测试覆盖）。
package service

import (
	"context"
	"testing"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"go-wind-admin/app/admin/service/internal/data"
	"go-wind-admin/app/admin/service/internal/data/enttest"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	internalMessageV1 "go-wind-admin/api/gen/go/internal_message/service/v1"
	"go-wind-admin/pkg/middleware/auth"
)

func newInternalMessageCategoryServiceForTest(t *testing.T) (*InternalMessageCategoryService, context.Context) {
	t.Helper()
	return &InternalMessageCategoryService{
		log:  bLogger.NewHelper(bLogger.NopLogger()),
		repo: data.NewInternalMessageCategoryRepoForTest(enttest.NewEntClientForTest(t)),
	}, auth.NewContext(enttest.NewSystemContext(context.Background()),
		&authenticationV1.UserTokenPayload{UserId: 81})
}

func TestInternalMessageCategoryService_CreateAndUpdateStamps(t *testing.T) {
	svc, opCtx := newInternalMessageCategoryServiceForTest(t)
	sysCtx := enttest.NewSystemContext(context.Background())

	_, err := svc.Create(opCtx, &internalMessageV1.CreateInternalMessageCategoryRequest{
		Data: &internalMessageV1.InternalMessageCategory{
			Name: trans.Ptr("系统公告"),
			Code: trans.Ptr("svc_announcement"),
		},
	})
	require.NoError(t, err)

	// nil 请求 / 缺 Data：必须 BadRequest 而不是 panic
	_, err = svc.Create(opCtx, nil)
	require.Error(t, err, "nil 请求应报 BadRequest")
	_, err = svc.Create(opCtx, &internalMessageV1.CreateInternalMessageCategoryRequest{})
	require.Error(t, err, "缺 Data 应报 BadRequest")

	created, err := svc.repo.Get(sysCtx, &internalMessageV1.GetInternalMessageCategoryRequest{
		QueryBy: &internalMessageV1.GetInternalMessageCategoryRequest_Id{Id: 1},
	})
	require.NoError(t, err)
	require.NotNil(t, created.GetCreatedBy())
	require.Equal(t, int64(81), created.GetCreatedBy(), "CreatedBy 应落操作人")

	mask, err := fieldmaskpb.New(&internalMessageV1.InternalMessageCategory{}, "name")
	require.NoError(t, err)
	_, err = svc.Update(opCtx, &internalMessageV1.UpdateInternalMessageCategoryRequest{
		Id:         created.GetId(),
		UpdateMask: mask,
		Data:       &internalMessageV1.InternalMessageCategory{Name: trans.Ptr("系统公告改名")},
	})
	require.NoError(t, err)

	updated, err := svc.repo.Get(sysCtx, &internalMessageV1.GetInternalMessageCategoryRequest{
		QueryBy: &internalMessageV1.GetInternalMessageCategoryRequest_Id{Id: created.GetId()},
	})
	require.NoError(t, err)
	require.Equal(t, "系统公告改名", updated.GetName())
	require.NotNil(t, updated.GetUpdatedBy(), "UpdatedBy 应经追加掩码落库")

	// Update 守卫
	_, err = svc.Update(opCtx, nil)
	require.Error(t, err, "nil 请求应报 BadRequest")
	_, err = svc.Update(opCtx, &internalMessageV1.UpdateInternalMessageCategoryRequest{Id: 1})
	require.Error(t, err, "缺 Data 应报 BadRequest")
}

func TestInternalMessageCategoryService_DeletePassthrough(t *testing.T) {
	svc, opCtx := newInternalMessageCategoryServiceForTest(t)

	_, err := svc.Create(opCtx, &internalMessageV1.CreateInternalMessageCategoryRequest{
		Data: &internalMessageV1.InternalMessageCategory{
			Name: trans.Ptr("待删分类"),
			Code: trans.Ptr("svc_to_delete"),
		},
	})
	require.NoError(t, err)

	_, err = svc.Delete(opCtx, &internalMessageV1.DeleteInternalMessageCategoryRequest{
		QueryBy: &internalMessageV1.DeleteInternalMessageCategoryRequest_Id{Id: 1},
	})
	require.NoError(t, err)

	resp, err := svc.List(opCtx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Equal(t, int64(0), resp.GetTotal(), "删除后列表应为空")
}
