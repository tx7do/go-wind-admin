package viewer

import (
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"

	"github.com/tx7do/go-crud/viewer"
)

// UserViewer describes a user-viewer.
type UserViewer struct {
	uid         uint64
	tid         uint64
	ouid        uint64
	dataScopes  []viewer.DataScope
	roles       []string
	permissions []string
	traceID     string
}

func NewUserViewer(
	uid uint64,
	tid uint64,
	ouid uint64,
	dataScope permissionV1.DataScope,
) viewer.Context {
	return UserViewer{
		uid:  uid,
		tid:  tid,
		ouid: ouid,
	}
}

// UserID 返回当前用户ID
func (v UserViewer) UserID() uint64 {
	return v.uid
}

// TenantID 返回租户ID
func (v UserViewer) TenantID() uint64 {
	return v.tid
}

// OrgUnitID 返回当前身份挂载的组织单元 ID
func (v UserViewer) OrgUnitID() uint64 {
	return v.ouid
}

// Permissions 返回当前 Viewer 的权限列表（可用于细粒度判断）
func (v UserViewer) Permissions() []string {
	return v.permissions
}

// Roles 返回当前 Viewer 的角色列表（可选，用于审计或策略）
func (v UserViewer) Roles() []string {
	return v.roles
}

// DataScope 返回当前身份的数据权限范围（用于 SQL 拼接）
func (v UserViewer) DataScope() []viewer.DataScope {
	return v.dataScopes
}

// TraceID 返回当前请求的 Trace ID（用于日志跟踪）
func (v UserViewer) TraceID() string {
	return v.traceID
}

// HasPermission 判断是否具有某个动作/资源的权限（如 "update:user"）
func (v UserViewer) HasPermission(_, _ string) bool {
	return false
}

// IsPlatformContext 当前是否处于平台管理视图（tenant_id == 0）
func (v UserViewer) IsPlatformContext() bool {
	return v.tid == 0
}

// IsTenantContext 当前是否处于租户业务视图（tenant_id > 0）
func (v UserViewer) IsTenantContext() bool {
	return v.tid > 0
}

// IsSystemContext 判断是否为系统后台任务
func (v UserViewer) IsSystemContext() bool {
	return false
}

// ShouldAudit 返回是否需要记录审计日志（便于在中间件/Hook 中快速判断）
func (v UserViewer) ShouldAudit() bool {
	return false
}
