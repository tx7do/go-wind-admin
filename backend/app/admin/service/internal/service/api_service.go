package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"github.com/getkin/kin-openapi/openapi3"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/admin/service/cmd/server/assets"
	"go-wind-admin/app/admin/service/internal/data"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	identityV1 "go-wind-admin/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/authorizer"
	"go-wind-admin/pkg/constants"
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
	ctx := viewer.WithSystemContext(context.Background())
	if count, _ := s.repo.Count(ctx, nil); count.Count == 0 {
		// 启动期播种走内部函数而非 HTTP 方法 SyncApis：后者已加平台管理员闸门，
		// 系统上下文没有操作者，会被自家闸门挡掉。空表场景 MERGE 即全量插入。
		if apis, err := s.loadApisFromOpenAPI(ctx); err != nil {
			s.log.Errorf(ctx, "启动期接口同步失败: %v", err)
		} else if _, _, _, err = s.repo.SyncMerge(ctx, apis, 0); err != nil {
			s.log.Errorf(ctx, "启动期接口同步失败: %v", err)
		}
		if err := s.authorizer.ResetPolicies(ctx); err != nil {
			s.log.Errorf(ctx, "reset policies error: %v", err)
		}
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

	// Api 表是租户闸门（套餐模块白名单）的数据源，增删改等于改闸门本身，
	// 仅平台管理员可操作。
	if err = requirePlatformAdmin(ctx, s.log, "api management"); err != nil {
		return nil, err
	}

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

	// 与 Create 同口径：改 Api 行可把任意端点归入已白名单模块，仅平台管理员。
	if err = requirePlatformAdmin(ctx, s.log, "api management"); err != nil {
		return nil, err
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
	if err := requirePlatformAdmin(ctx, s.log, "api management"); err != nil {
		return nil, err
	}

	// 先取行拿端点身份：删行后按 (path, method) 级联清理权限绑定快照，
	// 消灭孤儿绑定（孤儿绑定曾会在 id 重编号后指到无关端点，造成越权放行）
	api, err := s.repo.Get(ctx, &permissionV1.GetApiRequest{
		QueryBy: &permissionV1.GetApiRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		return nil, err
	}

	if api.GetPath() != "" && api.GetMethod() != "" {
		if err = s.repo.CascadeDeleteBindings(ctx, api.GetPath(), api.GetMethod()); err != nil {
			return nil, err
		}
	}

	// 重置权限策略
	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ApiService) SyncApis(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	// 同步会增删 API 行（租户闸门与授权策略的数据源），是平台级运维操作，
	// 仅平台管理员可触发。
	if err := requirePlatformAdmin(ctx, s.log, "api management"); err != nil {
		return nil, err
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	var apis []*permissionV1.Api
	if apis, err = s.loadApisFromOpenAPI(ctx); err != nil {
		return nil, err
	}

	// MERGE 按自然键 (path, method) 单事务增量对齐。旧的 Truncate + 顺序重编号
	// 造成过两类事故：同步窗口内租户闸门 fail-closed 全租户 403；id 全表洗牌让
	// api_id 绑定漂移（授权悄悄换到别的端点）。现 id 稳定、绑定存端点快照，
	// 授权零漂移；移除段同事务级联清绑定。
	added, updated, removed, err := s.repo.SyncMerge(ctx, apis, operator.UserId)
	if err != nil {
		return nil, err
	}
	s.log.Infof(ctx, "接口同步完成：新增 %d、更新 %d、移除 %d", added, updated, removed)

	// 重置权限策略
	if err = s.authorizer.ResetPolicies(ctx); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// loadApisFromOpenAPI 解析打进二进制的 OpenAPI 文档为端点清单（不含 id）：
// (path, method) 即身份，id 由 SyncMerge 插入时按自增生成。
func (s *ApiService) loadApisFromOpenAPI(ctx context.Context) ([]*permissionV1.Api, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(assets.OpenApiData)
	if err != nil {
		// 此前用 log.Fatal（os.Exit 直接终止进程），其后的 return 是死代码。
		// 启动期同步失败应返回错误由上层决定，而非杀掉整个进程。
		s.log.Errorf(ctx, "加载 OpenAPI 文档失败: %v", err)
		return nil, adminV1.ErrorInternalServerError("load OpenAPI document failed")
	}

	if doc == nil {
		s.log.Error(ctx, "OpenAPI 文档为空")
		return nil, adminV1.ErrorInternalServerError("OpenAPI document is nil")
	}
	if doc.Paths == nil {
		s.log.Error(ctx, "OpenAPI 文档的路径为空")
		return nil, adminV1.ErrorInternalServerError("OpenAPI document paths is nil")
	}

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

			apiList = append(apiList, &permissionV1.Api{
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

	return apiList, nil
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
