package data

import (
	"context"
	"strconv"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	bLogger "github.com/tx7do/kratos-bootstrap/logger"

	"go-wind-admin/app/admin/service/cmd/server/assets"

	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/authorizer"
	"go-wind-admin/pkg/constants"
)

// AuthorizerProvider 权限数据提供者
type AuthorizerProvider struct {
	log *bLogger.Helper

	roleRepo *RoleRepo
}

func NewAuthorizerProvider(
	ctx *bootstrap.Context,
	roleRepo *RoleRepo,
) authorizer.Provider {
	return &AuthorizerProvider{
		log:      ctx.NewLoggerHelper("authorizer-data-provider/data/admin-service"),
		roleRepo: roleRepo,
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
func (p *AuthorizerProvider) ProvidePolicies(_ context.Context) (authorizer.PermissionDataMap, error) {
	// 策略装载需要全量角色/权限数据，统一以 SystemContext 跑，忽略调用方 ctx 的 viewer
	ctx := viewer.WithSystemContext(context.Background())

	roles, err := p.roleRepo.List(ctx, &paginationV1.PagingRequest{NoPaging: trans.Ptr(true)})
	if err != nil {
		p.log.Errorf(context.Background(), "failed to list roles: %v", err)
		return nil, err
	}

	result := make(authorizer.PermissionDataMap)
	var endpoints []Endpoint
	for _, role := range roles.Items {
		if role == nil {
			continue
		}
		if role.GetCode() == "" {
			continue
		}
		if constants.IsTemplateRoleCode(role.GetCode()) {
			continue
		}

		// 绑定存 (path, method) 快照：绑定即策略，直接装载，不再回查 sys_apis。
		// 端点的删除/重建（API id 漂移）不影响策略有效性。
		endpoints, err = p.roleRepo.GetRolePermissionApiEndpoints(ctx, role.GetId())
		if err != nil {
			p.log.Errorf(context.Background(), "failed to get role [%d] permission api endpoints: %v", role.GetId(), err)
			continue
		}

		// 角色未绑定任何 API 时直接跳过，逐角色刷 ERROR 日志纯属误导。
		if len(endpoints) == 0 {
			continue
		}

		var authorizerDataArray authorizer.PermissionDataArray
		for _, e := range endpoints {
			if e.Path == "" || e.Method == "" {
				continue
			}

			var data = authorizer.PermissionData{
				Domain: strconv.FormatUint(uint64(role.GetTenantId()), 10),
				Path:   e.Path,
				Method: e.Method,
			}
			authorizerDataArray = append(authorizerDataArray, data)
		}
		if len(authorizerDataArray) > 0 {
			result[role.GetCode()] = authorizerDataArray
		}
	}

	return result, nil
}
