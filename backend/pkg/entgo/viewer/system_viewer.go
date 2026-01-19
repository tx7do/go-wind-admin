package viewer

import (
	"context"

	"github.com/tx7do/go-crud/viewer"
)

// SystemViewer describes a system-viewer.
type SystemViewer struct {
}

func NewSystemViewer() viewer.Context {
	return SystemViewer{}
}

func NewSystemViewerContext(ctx context.Context) context.Context {
	return viewer.WithContext(ctx, NewSystemViewer())
}

func (v SystemViewer) ShouldAudit() bool {
	return false
}

// UserID 返回当前用户ID
func (v SystemViewer) UserID() uint64 {
	return 0
}

// TenantID 返回租户ID
func (v SystemViewer) TenantID() uint64 {
	return 0
}

// OrgUnitID 返回当前身份挂载的组织单元 ID
func (v SystemViewer) OrgUnitID() uint64 {
	return 0
}

// Permissions 返回当前 Viewer 的权限列表（可用于细粒度判断）
func (v SystemViewer) Permissions() []string {
	return []string{}
}

// Roles 返回当前 Viewer 的角色列表（可选，用于审计或策略）
func (v SystemViewer) Roles() []string {
	return []string{}
}

// DataScope 返回当前身份的数据权限范围（用于 SQL 拼接）
func (v SystemViewer) DataScope() []viewer.DataScope {
	return []viewer.DataScope{}
}

// TraceID 返回当前请求的 Trace ID（用于日志跟踪）
func (v SystemViewer) TraceID() string {
	return ""
}

// HasPermission 判断是否具有某个动作/资源的权限（如 "update:user"）
func (v SystemViewer) HasPermission(action, resource string) bool {
	return true
}

// IsPlatformContext 当前是否处于平台管理视图（tenant_id == 0）
func (v SystemViewer) IsPlatformContext() bool {
	return true
}

// IsTenantContext 当前是否处于租户业务视图（tenant_id > 0）
func (v SystemViewer) IsTenantContext() bool {
	return false
}

// IsSystemContext 判断是否为系统后台任务
func (v SystemViewer) IsSystemContext() bool {
	return true
}
