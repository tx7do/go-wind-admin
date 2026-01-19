package metadata

import (
	permissionV1 "go-wind-admin/api/gen/go/permission/service/v1"
)

const (
	mdOperator = "x-md-operator"
)

// OperatorInfo is a struct for operator metadata.
type OperatorInfo struct {
	UserID    *uint64                 `json:"uid,omitempty"`
	TenantID  *uint64                 `json:"tid,omitempty"`
	OrgUnitID *uint64                 `json:"ouid,omitempty"`
	DataScope *permissionV1.DataScope `json:"ds,omitempty"`
}

func NewOperatorInfo(
	uid uint64,
	tid uint64,
	ouid uint64,
	dataScope permissionV1.DataScope,
) *OperatorInfo {
	return &OperatorInfo{
		UserID:    &uid,
		TenantID:  &tid,
		OrgUnitID: &ouid,
		DataScope: &dataScope,
	}
}

func (oi *OperatorInfo) GetUserID() uint64 {
	if oi != nil && oi.UserID != nil {
		return *oi.UserID
	}
	return 0
}

func (oi *OperatorInfo) GetTenantID() uint64 {
	if oi != nil && oi.TenantID != nil {
		return *oi.TenantID
	}
	return 0
}

func (oi *OperatorInfo) GetOrgUnitID() uint64 {
	if oi != nil && oi.OrgUnitID != nil {
		return *oi.OrgUnitID
	}
	return 0
}

func (oi *OperatorInfo) GetDataScope() permissionV1.DataScope {
	if oi != nil && oi.DataScope != nil {
		return *oi.DataScope
	}
	return permissionV1.DataScope_DATA_SCOPE_UNSPECIFIED
}
