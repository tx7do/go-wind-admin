package viewer

import (
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

// Viewer describes the query/mutation viewer-context.
type Viewer interface {
	// UserId 返回当前用户ID
	UserId() uint32

	// Tenant 返回租户ID
	Tenant() (uint32, bool)

	// --- 作用域判定 ---

	// IsPlatformContext 当前是否处于平台管理视图（tenant_id == 0）
	IsPlatformContext() bool

	// IsTenantContext 当前是否处于租户业务视图（tenant_id > 0）
	IsTenantContext() bool

	// --- 数据权限（关键新增） ---

	// GetDataScope 返回当前身份的数据权限范围（用于 SQL 拼接）
	GetDataScope() permissionV1.DataScope // 对应 DataScope 枚举

	// GetOrgUnitId 返回当前身份挂载的组织单元 ID
	GetOrgUnitId() uint32
}

// UserViewer describes a user-viewer.
type UserViewer struct {
	uid       uint32
	tid       uint32
	ouid      uint32
	dataScope permissionV1.DataScope
}

func NewUserViewer(
	uid uint32,
	tid uint32,
	ouid uint32,
	dataScope permissionV1.DataScope,
) UserViewer {
	return UserViewer{
		uid:       uid,
		tid:       tid,
		ouid:      ouid,
		dataScope: dataScope,
	}
}

func (v UserViewer) UserId() uint32 {
	return v.uid
}

func (v UserViewer) Tenant() (uint32, bool) {
	return v.tid, v.tid > 0
}

func (v UserViewer) GetDataScope() permissionV1.DataScope {
	return v.dataScope
}

func (v UserViewer) GetOrgUnitId() uint32 {
	return v.ouid
}

func (v UserViewer) IsTenantContext() bool {
	return v.tid > 0
}

func (v UserViewer) IsPlatformContext() bool {
	return v.tid == 0
}
