package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	pagination "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

const defaultDomain = "*"

// AuthorizerData 权限数据
type AuthorizerData struct {
	Path   string
	Method string
	Domain string
}

type AuthorizerDataArray []AuthorizerData

// AuthorizerDataMap 权限数据映射
type AuthorizerDataMap map[string]AuthorizerDataArray

// AuthorizerProvider 权限数据提供者
type AuthorizerProvider struct {
	log *log.Helper

	roleRepo        *RoleRepo
	apiResourceRepo *ApiResourceRepo
}

func NewAuthorizerProvider(
	ctx *bootstrap.Context,
	roleRepo *RoleRepo,
	apiResourceRepo *ApiResourceRepo,
) *AuthorizerProvider {
	return &AuthorizerProvider{
		log:             ctx.NewLoggerHelper("authorizer-data-provider/data/admin-service"),
		roleRepo:        roleRepo,
		apiResourceRepo: apiResourceRepo,
	}
}

// Provide 提供权限数据
func (p *AuthorizerProvider) Provide(ctx context.Context) (AuthorizerDataMap, error) {
	roles, err := p.roleRepo.List(ctx, &pagination.PagingRequest{NoPaging: trans.Ptr(true)})
	if err != nil {
		p.log.Errorf("failed to list roles: %v", err)
		return nil, err
	}

	result := make(AuthorizerDataMap)
	var apiIDs []uint32
	var apiResources []*permissionV1.ApiResource
	for _, role := range roles.Items {
		apiIDs, err = p.roleRepo.GetRolePermissionApiIDs(ctx, role.GetId())
		if err != nil {
			p.log.Errorf("failed to get role [%d] permission api ids: %v", role.GetId(), err)
			continue
		}

		apiResources, err = p.apiResourceRepo.GetApiResourceByIDs(ctx, apiIDs)
		if err != nil {
			p.log.Errorf("failed to list api resources by ids: %v", err)
			continue
		}

		var authorizerDataArray AuthorizerDataArray
		for _, apiResource := range apiResources {
			if apiResource == nil {
				continue
			}
			if apiResource.GetPath() == "" || apiResource.GetMethod() == "" {
				continue
			}

			var data = AuthorizerData{
				Domain: defaultDomain,
				Path:   apiResource.GetPath(),
				Method: apiResource.GetMethod(),
			}
			authorizerDataArray = append(authorizerDataArray, data)
		}
		if len(authorizerDataArray) > 0 {
			result[role.GetCode()] = authorizerDataArray
		}
	}

	return result, nil
}
