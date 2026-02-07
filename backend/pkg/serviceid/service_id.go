package serviceid

const (
	AdminService = "admin-service" // 后台服务
)

// NewDiscoveryName 构建服务发现名称
func NewDiscoveryName(serviceName string) string {
	return ProjectName + "/" + serviceName
}
