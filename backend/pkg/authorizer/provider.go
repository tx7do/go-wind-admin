package authorizer

import "context"

// PermissionData 权限数据
type PermissionData struct {
	Path   string
	Method string
	Domain string
}

type PermissionDataArray []PermissionData

// PermissionDataMap 权限数据映射
type PermissionDataMap map[string]PermissionDataArray

// ModelDataMap 模型数据映射
type ModelDataMap map[string][]byte

// Provider 权限数据提供者接口
type Provider interface {
	// ProvideModels 提供模型数据
	ProvideModels(engineName string) ModelDataMap

	// ProvidePolicies 提供策略数据
	ProvidePolicies(ctx context.Context) (PermissionDataMap, error)
}
