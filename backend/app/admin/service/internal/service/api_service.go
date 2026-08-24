package service

import (
	"context"

	bLogger "github.com/tx7do/kratos-bootstrap/logger"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"github.com/getkin/kin-openapi/openapi3"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/cmd/server/assets"
	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/authorizer"
	"go-wind-admin/pkg/constants"
	appViewer "go-wind-admin/pkg/entgo/viewer"
	"go-wind-admin/pkg/middleware/auth"
)

type RouteWalker interface {
	WalkRoute(fn http.WalkRouteFunc) error
}

type ApiService struct {
	adminV1.ApiServiceHTTPServer

	log *bLogger.Helper

	repo        *data.ApiRepo
	authorizer  *authorizer.Authorizer
	routeWalker RouteWalker
}

func NewApiService(
	ctx *bootstrap.Context,
	repo *data.ApiRepo,
	authorizer *authorizer.Authorizer,
) *ApiService {
	svc := &ApiService{
		log:        ctx.NewLoggerHelper("api/service/admin-service"),
		repo:       repo,
		authorizer: authorizer,
	}

	svc.init()

	return svc
}

func (s *ApiService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())
	if count, _ := s.repo.Count(ctx, nil); count.Count == 0 {
		_, _ = s.SyncApis(ctx, &emptypb.Empty{})
	}
}

func (s *ApiService) RegisterRouteWalker(routeWalker RouteWalker) {
	s.routeWalker = routeWalker
}

func (s *ApiService) List(ctx context.Context, req *paginationV1.PagingRequest) (*permissionV1.ListApiResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *ApiService) Get(ctx context.Context, req *permissionV1.GetApiRequest) (*permissionV1.Api, error) {
	return s.repo.Get(ctx, req)
}

func (s *ApiService) Create(ctx context.Context, req *permissionV1.CreateApiRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if err = s.repo.Create(ctx, req); err != nil {
		return nil, err
	}

	// 重置权限策略
	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ApiService) Update(ctx context.Context, req *permissionV1.UpdateApiRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	if err = s.repo.Update(ctx, req); err != nil {
		return nil, err
	}

	// 重置权限策略
	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ApiService) Delete(ctx context.Context, req *permissionV1.DeleteApiRequest) (*emptypb.Empty, error) {
	if err := s.repo.Delete(ctx, req); err != nil {
		return nil, err
	}

	// 重置权限策略
	if err := s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ApiService) SyncApis(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	_ = s.repo.Truncate(ctx)

	//if err := s.syncWithWalkRoute(ctx); err != nil {
	//	return nil, err
	//}

	if err := s.syncWithOpenAPI(ctx); err != nil {
		return nil, err
	}

	// 重置权限策略
	if err := s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// syncWithOpenAPI 使用 OpenAPI 文档同步 API 资源
func (s *ApiService) syncWithOpenAPI(ctx context.Context) error {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(assets.OpenApiData)
	if err != nil {
		// 此前用 log.Fatal（os.Exit 直接终止进程），其后的 return 是死代码。
		// 启动期同步失败应返回错误由上层决定，而非杀掉整个进程。
		s.log.Errorf(ctx, "加载 OpenAPI 文档失败: %v", err)
		return adminV1.ErrorInternalServerError("load OpenAPI document failed")
	}

	if doc == nil {
		s.log.Error(ctx, "OpenAPI 文档为空")
		return adminV1.ErrorInternalServerError("OpenAPI document is nil")
	}
	if doc.Paths == nil {
		s.log.Error(ctx, "OpenAPI 文档的路径为空")
		return adminV1.ErrorInternalServerError("OpenAPI document paths is nil")
	}

	var count uint32 = 0
	var apiList []*permissionV1.Api

	// 遍历所有路径和操作
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {

			var module string
			var moduleDescription string
			if len(operation.Tags) > 0 {
				tag := doc.Tags.Get(operation.Tags[0])
				if tag != nil {
					module = tag.Name
					moduleDescription = tag.Description
				}
			}

			var businessModule = identityV1.Module_MODULE_UNSPECIFIED
			if module != "" {
				if bm, ok := constants.ServiceTagToBusinessModule[module]; ok {
					businessModule = bm
				}
			}

			count++

			apiList = append(apiList, &permissionV1.Api{
				Id:                trans.Ptr(count),
				Path:              trans.Ptr(path),
				Method:            trans.Ptr(method),
				Module:            trans.Ptr(module),
				ModuleDescription: trans.Ptr(moduleDescription),
				BusinessModule:    trans.Ptr(businessModule),
				Description:       trans.Ptr(operation.Description),
				Operation:         trans.Ptr(operation.OperationID),
			})
		}
	}

	for i, res := range apiList {
		res.Id = trans.Ptr(uint32(i + 1))
		_ = s.repo.Update(ctx, &permissionV1.UpdateApiRequest{
			AllowMissing: trans.Ptr(true),
			Data:         res,
		})
	}

	return nil
}
// GetWalkRouteData 获取通过 WalkRoute 获取的路由数据，用于调试
func (s *ApiService) GetWalkRouteData(_ context.Context, _ *emptypb.Empty) (*permissionV1.ListApiResponse, error) {
	if s.routeWalker == nil {
		return nil, adminV1.ErrorInternalServerError("router walker is nil")
	}

	resp := &permissionV1.ListApiResponse{
		Items: []*permissionV1.Api{},
	}
	var count uint32 = 0
	if err := s.routeWalker.WalkRoute(func(info http.RouteInfo) error {
		//log.Infof("Path[%s] Method[%s]", info.Path, info.Method)
		count++
		resp.Items = append(resp.Items, &permissionV1.Api{
			Id:     trans.Ptr(count),
			Path:   trans.Ptr(info.Path),
			Method: trans.Ptr(info.Method),
			Status: trans.Ptr(permissionV1.Api_ON),
		})
		return nil
	}); err != nil {
		s.log.Errorf(context.Background(), "failed to walk route: %v", err)
		return nil, adminV1.ErrorInternalServerError("failed to walk route")
	}
	resp.Total = uint64(count)

	return resp, nil
}
