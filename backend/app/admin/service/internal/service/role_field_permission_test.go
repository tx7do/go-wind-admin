package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	authenticationV1 "go-wind-admin/api/gen/go/authentication/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
	"go-wind-admin/pkg/middleware/auth"
)

// innerRoleStub 被装饰内层服务桩：只覆写装饰器会调用的五个方法。
type innerRoleStub struct {
	permissionV1.RoleServiceServer

	gotListReq   *paginationV1.PagingRequest
	gotGetReq    *permissionV1.GetRoleRequest
	gotCreateReq *permissionV1.CreateRoleRequest
	gotUpdateReq *permissionV1.UpdateRoleRequest
	gotDeleteReq *permissionV1.DeleteRoleRequest
}

func (s *innerRoleStub) List(_ context.Context, in *paginationV1.PagingRequest) (*permissionV1.ListRoleResponse, error) {
	s.gotListReq = in
	return &permissionV1.ListRoleResponse{
		Items: []*permissionV1.Role{
			{Name: trans.Ptr("桩角色甲"), Permissions: []int64{1, 2, 3}},
		},
		Total: 1,
	}, nil
}

func (s *innerRoleStub) Get(_ context.Context, in *permissionV1.GetRoleRequest) (*permissionV1.Role, error) {
	s.gotGetReq = in
	return &permissionV1.Role{Name: trans.Ptr("桩角色甲"), Permissions: []int64{1, 2, 3}}, nil
}

func (s *innerRoleStub) Create(_ context.Context, in *permissionV1.CreateRoleRequest) (*emptypb.Empty, error) {
	s.gotCreateReq = in
	return &emptypb.Empty{}, nil
}

func (s *innerRoleStub) Update(_ context.Context, in *permissionV1.UpdateRoleRequest) (*emptypb.Empty, error) {
	s.gotUpdateReq = in
	return &emptypb.Empty{}, nil
}

func (s *innerRoleStub) Delete(_ context.Context, in *permissionV1.DeleteRoleRequest) (*emptypb.Empty, error) {
	s.gotDeleteReq = in
	return &emptypb.Empty{}, nil
}

// roleHiddenCtx 构造带 Role.permissions 隐藏字段声明的令牌上下文。
func roleHiddenCtx(ctx context.Context) context.Context {
	return auth.NewContext(ctx, &authenticationV1.UserTokenPayload{
		UserId:       9,
		HiddenFields: []string{"Role.permissions", "User.phone"},
	})
}

// TestFieldPermissionRole_ReadMask 读路径：permissions 被清空（不可见），
// 其他字段保持；User 资源的隐藏声明不串扰 Role 资源。
func TestFieldPermissionRole_ReadMask(t *testing.T) {
	inner := &innerRoleStub{}
	wrapped := NewFieldPermissionRoleServiceServer(inner)
	ctx := roleHiddenCtx(context.Background())

	listResp, err := wrapped.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Empty(t, listResp.GetItems()[0].GetPermissions(), "列表中 permissions 应被清空")
	require.Equal(t, "桩角色甲", listResp.GetItems()[0].GetName(), "非受控字段保持")

	getResp, err := wrapped.Get(ctx, &permissionV1.GetRoleRequest{QueryBy: &permissionV1.GetRoleRequest_Id{Id: 1}})
	require.NoError(t, err)
	require.Empty(t, getResp.GetPermissions())
}

// TestFieldPermissionRole_WriteStrip 写路径：payload 中 permissions 被剥离、
// updateMask 中对应路径被同步剔除。
func TestFieldPermissionRole_WriteStrip(t *testing.T) {
	inner := &innerRoleStub{}
	wrapped := NewFieldPermissionRoleServiceServer(inner)
	ctx := roleHiddenCtx(context.Background())

	// Create：permissions 剥离
	_, err := wrapped.Create(ctx, &permissionV1.CreateRoleRequest{
		Data: &permissionV1.Role{
			Name:        trans.Ptr("新角色"),
			Permissions: []int64{9, 9, 9},
		},
	})
	require.NoError(t, err)
	require.Empty(t, inner.gotCreateReq.GetData().GetPermissions(), "越权 permissions 应被剥离")
	require.Equal(t, "新角色", inner.gotCreateReq.GetData().GetName())

	// Update：mask 含 permissions → 剔除；不含 → 不新增
	mask := &fieldmaskpb.FieldMask{Paths: []string{"name", "permissions"}}
	_, err = wrapped.Update(ctx, &permissionV1.UpdateRoleRequest{
		Id:         1,
		Data:       &permissionV1.Role{Name: trans.Ptr("改名"), Permissions: []int64{7}},
		UpdateMask: mask,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"name"}, inner.gotUpdateReq.GetUpdateMask().GetPaths(),
		"越权字段应从 mask 中剔除")
}

// TestFieldPermissionRole_NoConfigPassThrough 无隐藏配置：完全不裁剪。
func TestFieldPermissionRole_NoConfigPassThrough(t *testing.T) {
	inner := &innerRoleStub{}
	wrapped := NewFieldPermissionRoleServiceServer(inner)
	// 无令牌上下文
	ctx := context.Background()

	resp, err := wrapped.List(ctx, &paginationV1.PagingRequest{})
	require.NoError(t, err)
	require.Len(t, resp.GetItems()[0].GetPermissions(), 3, "无配置时不裁剪")
}
