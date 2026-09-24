package service

import (
	"context"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	"go-wind-admin/pkg/fieldperm"
	"go-wind-admin/pkg/middleware/auth"
)

// fieldPermissionUserResource 字段权限的资源名（proto 消息简单名）。
// 角色字段权限配置按此名挂载；扩展新受控资源时在此追加并包一层同型包装器。
const fieldPermissionUserResource = "User"

// FieldPermissionUserServiceServer 字段权限装饰器：
// 依据请求令牌中的隐藏字段集（hfs claim），对用户管理的读写两侧做字段裁剪。
//
//   - List/Get 响应：清除命中字段。protojson 不输出未填充字段，清值即从
//     JSON 响应中消失，前端拿不到明文。
//   - Create/Update 请求：剥离命中字段的值并同步剔除 field_mask 路径，
//     语义为"静默忽略"（与读路径一致，不暴露字段受控事实）。
//
// 刻意不裁剪 /admin/v1/me 系端点（UserProfileService）：本人资料页需要
// 完整字段才能自编辑，字段权限管控的是"对他人/管理视角的可见性"。
// 平台管理员（tenantId==0）登录期即不聚合隐藏集，此处天然放行。
type FieldPermissionUserServiceServer struct {
	adminV1.UnsafeUserServiceServer
	inner adminV1.UserServiceServer
}

// NewFieldPermissionUserServiceServer 包装用户管理服务。
func NewFieldPermissionUserServiceServer(inner adminV1.UserServiceServer) adminV1.UserServiceServer {
	return &FieldPermissionUserServiceServer{inner: inner}
}

// hiddenFields 取当前请求的 User 资源隐藏字段集；取不到令牌或无配置返回 nil（不裁剪）。
func (s *FieldPermissionUserServiceServer) hiddenFields(ctx context.Context) map[string]struct{} {
	tokenPayload, err := auth.FromContext(ctx)
	if err != nil || tokenPayload == nil {
		return nil
	}
	return fieldperm.HiddenFieldsOf(tokenPayload.GetHiddenFields(), fieldPermissionUserResource)
}

func (s *FieldPermissionUserServiceServer) List(ctx context.Context, in *paginationV1.PagingRequest) (*identityV1.ListUserResponse, error) {
	res, err := s.inner.List(ctx, in)
	if err != nil {
		return res, err
	}
	if hidden := s.hiddenFields(ctx); !fieldperm.IsEmpty(hidden) && res != nil {
		fieldperm.ApplyReadMaskList(res.GetItems(), hidden)
	}
	return res, nil
}

func (s *FieldPermissionUserServiceServer) Get(ctx context.Context, in *identityV1.GetUserRequest) (*identityV1.User, error) {
	res, err := s.inner.Get(ctx, in)
	if err != nil {
		return res, err
	}
	fieldperm.ApplyReadMask(res, s.hiddenFields(ctx))
	return res, nil
}

func (s *FieldPermissionUserServiceServer) Create(ctx context.Context, in *identityV1.CreateUserRequest) (*emptypb.Empty, error) {
	fieldperm.StripWriteFields(in.GetData(), nil, s.hiddenFields(ctx))
	return s.inner.Create(ctx, in)
}

func (s *FieldPermissionUserServiceServer) Update(ctx context.Context, in *identityV1.UpdateUserRequest) (*emptypb.Empty, error) {
	fieldperm.StripWriteFields(in.GetData(), in.GetUpdateMask(), s.hiddenFields(ctx))
	return s.inner.Update(ctx, in)
}

func (s *FieldPermissionUserServiceServer) Delete(ctx context.Context, in *identityV1.DeleteUserRequest) (*emptypb.Empty, error) {
	return s.inner.Delete(ctx, in)
}

func (s *FieldPermissionUserServiceServer) UserExists(ctx context.Context, in *identityV1.UserExistsRequest) (*identityV1.UserExistsResponse, error) {
	return s.inner.UserExists(ctx, in)
}

func (s *FieldPermissionUserServiceServer) EditUserPassword(ctx context.Context, in *identityV1.EditUserPasswordRequest) (*emptypb.Empty, error) {
	return s.inner.EditUserPassword(ctx, in)
}
