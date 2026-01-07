package service

import (
	"context"
	"sort"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"entgo.io/ent/dialect/sql"
	"github.com/getkin/kin-openapi/openapi3"
	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/cmd/server/assets"
	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/middleware/auth"
)

type RouteWalker interface {
	WalkRoute(fn http.WalkRouteFunc) error
}

type ApiResourceService struct {
	adminV1.ApiResourceServiceHTTPServer

	log *log.Helper

	repo        *data.ApiResourceRepo
	authorizer  *data.Authorizer
	routeWalker RouteWalker
}

func NewApiResourceService(
	ctx *bootstrap.Context,
	repo *data.ApiResourceRepo,
	authorizer *data.Authorizer,
) *ApiResourceService {
	svc := &ApiResourceService{
		log:        ctx.NewLoggerHelper("api-resource/service/admin-service"),
		repo:       repo,
		authorizer: authorizer,
	}

	svc.init()

	return svc
}

func (s *ApiResourceService) init() {
	ctx := context.Background()
	if count, _ := s.repo.Count(ctx, []func(s *sql.Selector){}); count == 0 {
		_, _ = s.SyncApiResources(ctx, &emptypb.Empty{})
	}
}

func (s *ApiResourceService) RegisterRouteWalker(routeWalker RouteWalker) {
	s.routeWalker = routeWalker
}

func (s *ApiResourceService) List(ctx context.Context, req *pagination.PagingRequest) (*permissionV1.ListApiResourceResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *ApiResourceService) Get(ctx context.Context, req *permissionV1.GetApiResourceRequest) (*permissionV1.ApiResource, error) {
	return s.repo.Get(ctx, req)
}

func (s *ApiResourceService) Create(ctx context.Context, req *permissionV1.CreateApiResourceRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
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

func (s *ApiResourceService) Update(ctx context.Context, req *permissionV1.UpdateApiResourceRequest) (*emptypb.Empty, error) {
	if req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.UpdatedBy = trans.Ptr(operator.UserId)

	if err = s.repo.Update(ctx, req); err != nil {
		return nil, err
	}

	// 重置权限策略
	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ApiResourceService) Delete(ctx context.Context, req *permissionV1.DeleteApiResourceRequest) (*emptypb.Empty, error) {
	if err := s.repo.Delete(ctx, req); err != nil {
		return nil, err
	}

	// 重置权限策略
	if err := s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ApiResourceService) SyncApiResources(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
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
func (s *ApiResourceService) syncWithOpenAPI(ctx context.Context) error {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(assets.OpenApiData)
	if err != nil {
		s.log.Fatalf("加载 OpenAPI 文档失败: %v", err)
		return adminV1.ErrorInternalServerError("load OpenAPI document failed")
	}

	if doc == nil {
		s.log.Fatal("OpenAPI 文档为空")
		return adminV1.ErrorInternalServerError("OpenAPI document is nil")
	}
	if doc.Paths == nil {
		s.log.Fatal("OpenAPI 文档的路径为空")
		return adminV1.ErrorInternalServerError("OpenAPI document paths is nil")
	}

	var count uint32 = 0
	var apiResourceList []*permissionV1.ApiResource

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

			count++

			apiResourceList = append(apiResourceList, &permissionV1.ApiResource{
				Id:                trans.Ptr(count),
				Path:              trans.Ptr(path),
				Method:            trans.Ptr(method),
				Module:            trans.Ptr(module),
				ModuleDescription: trans.Ptr(moduleDescription),
				Description:       trans.Ptr(operation.Description),
				Operation:         trans.Ptr(operation.OperationID),
			})
		}
	}

	for i, res := range apiResourceList {
		res.Id = trans.Ptr(uint32(i + 1))
		_ = s.repo.Update(ctx, &permissionV1.UpdateApiResourceRequest{
			AllowMissing: trans.Ptr(true),
			Data:         res,
		})
	}

	return nil
}

// syncWithWalkRoute 使用 WalkRoute 同步 API 资源
func (s *ApiResourceService) syncWithWalkRoute(ctx context.Context) error {
	if s.routeWalker == nil {
		return adminV1.ErrorInternalServerError("router walker is nil")
	}

	var count uint32 = 0

	var apiResourceList []*permissionV1.ApiResource

	if err := s.routeWalker.WalkRoute(func(info http.RouteInfo) error {
		//log.Infof("Path[%s] Method[%s]", info.Path, info.Method)
		count++

		apiResourceList = append(apiResourceList, &permissionV1.ApiResource{
			Id:     trans.Ptr(count),
			Path:   trans.Ptr(info.Path),
			Method: trans.Ptr(info.Method),
		})

		return nil
	}); err != nil {
		s.log.Errorf("failed to walk route: %v", err)
		return adminV1.ErrorInternalServerError("failed to walk route")
	}

	sort.SliceStable(apiResourceList, func(i, j int) bool {
		if apiResourceList[i].GetPath() == apiResourceList[j].GetPath() {
			return apiResourceList[i].GetMethod() < apiResourceList[j].GetMethod()
		}
		return apiResourceList[i].GetPath() < apiResourceList[j].GetPath()
	})

	for i, res := range apiResourceList {
		res.Id = trans.Ptr(uint32(i + 1))
		_ = s.repo.Update(ctx, &permissionV1.UpdateApiResourceRequest{
			AllowMissing: trans.Ptr(true),
			Data:         res,
		})
	}

	return nil
}

// GetWalkRouteData 获取通过 WalkRoute 获取的路由数据，用于调试
func (s *ApiResourceService) GetWalkRouteData(_ context.Context, _ *emptypb.Empty) (*permissionV1.ListApiResourceResponse, error) {
	if s.routeWalker == nil {
		return nil, adminV1.ErrorInternalServerError("router walker is nil")
	}

	resp := &permissionV1.ListApiResourceResponse{
		Items: []*permissionV1.ApiResource{},
	}
	var count uint32 = 0
	if err := s.routeWalker.WalkRoute(func(info http.RouteInfo) error {
		//log.Infof("Path[%s] Method[%s]", info.Path, info.Method)
		count++
		resp.Items = append(resp.Items, &permissionV1.ApiResource{
			Id:     trans.Ptr(count),
			Path:   trans.Ptr(info.Path),
			Method: trans.Ptr(info.Method),
			Status: trans.Ptr(permissionV1.ApiResource_ON),
		})
		return nil
	}); err != nil {
		s.log.Errorf("failed to walk route: %v", err)
		return nil, adminV1.ErrorInternalServerError("failed to walk route")
	}
	resp.Total = uint64(count)

	return resp, nil
}
