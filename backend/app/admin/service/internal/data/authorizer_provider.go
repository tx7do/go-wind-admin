package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/admin/service/cmd/server/assets"

	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"go-wind-admin/pkg/authorizer"
	"go-wind-admin/pkg/constants"
)

const defaultDomain = "*"

// AuthorizerProvider 权限数据提供者
type AuthorizerProvider struct {
	log *log.Helper

	roleRepo *RoleRepo
	apiRepo  *ApiRepo
}

func NewAuthorizerProvider(
	ctx *bootstrap.Context,
	roleRepo *RoleRepo,
	apiRepo *ApiRepo,
) authorizer.Provider {
	return &AuthorizerProvider{
		log:      ctx.NewLoggerHelper("authorizer-data-provider/data/admin-service"),
		roleRepo: roleRepo,
		apiRepo:  apiRepo,
	}
}

// ProvideModels 提供模型数据
func (p *AuthorizerProvider) ProvideModels(engineName string) authorizer.ModelDataMap {
	switch engineName {
	case "casbin":
		return make(map[string][]byte)
	case "opa":
		return map[string][]byte{
			"rbac.rego": assets.OpaRbacRego,
		}
	}
	return nil
}

// ProvidePolicies 提供策略数据
func (p *AuthorizerProvider) ProvidePolicies(ctx context.Context) (authorizer.PermissionDataMap, error) {
	roles, err := p.roleRepo.List(ctx, &paginationV1.PagingRequest{NoPaging: trans.Ptr(true)})
	if err != nil {
		p.log.Errorf("failed to list roles: %v", err)
		return nil, err
	}

	result := make(authorizer.PermissionDataMap)
	var apiIDs []uint32
	var apis []*permissionV1.Api
	for _, role := range roles.Items {
		//p.log.Infof("processing role: %s", role.GetCode())
		if role == nil {
			continue
		}
		if role.GetCode() == "" {
			continue
		}
		if constants.IsTemplateRoleCode(role.GetCode()) {
			continue
		}

		apiIDs, err = p.roleRepo.GetRolePermissionApiIDs(ctx, role.GetId())
		if err != nil {
			p.log.Errorf("failed to get role [%d] permission api ids: %v", role.GetId(), err)
			continue
		}

		apis, err = p.apiRepo.GetApiByIDs(ctx, apiIDs)
		if err != nil {
			p.log.Errorf("failed to list apis by ids: %v", err)
			continue
		}

		var authorizerDataArray authorizer.PermissionDataArray
		for _, api := range apis {
			if api == nil {
				continue
			}
			if api.GetPath() == "" || api.GetMethod() == "" {
				continue
			}

			var data = authorizer.PermissionData{
				Domain: defaultDomain,
				Path:   api.GetPath(),
				Method: api.GetMethod(),
			}
			authorizerDataArray = append(authorizerDataArray, data)
		}
		if len(authorizerDataArray) > 0 {
			result[role.GetCode()] = authorizerDataArray
		}
	}

	return result, nil
}
