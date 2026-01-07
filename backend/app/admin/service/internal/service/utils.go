package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	"go-wind-admin/app/admin/service/internal/data"

	userV1 "go-wind-admin/api/gen/go/user/service/v1"

	"go-wind-admin/pkg/utils/name_set"
)

// QueryUserInfoFromRepo queries user information from user repository and fills the nameSetMap.
func QueryUserInfoFromRepo(ctx context.Context, userRepo data.UserRepo, nameSetMap *name_set.UserNameSetMap) {
	userIds := make([]uint32, 0, len(*nameSetMap))
	for userId := range *nameSetMap {
		userIds = append(userIds, userId)
	}

	users, err := userRepo.ListUsersByIds(ctx, userIds)
	if err != nil {
		log.Errorf("query users err: %v", err)
		return
	}

	for _, user := range users {
		(*nameSetMap)[user.GetId()] = &name_set.UserNameSet{
			UserName: user.GetUsername(),
			RealName: user.GetRealname(),
			NickName: user.GetNickname(),
		}
	}
}

func QueryTenantInfoFromRepo(ctx context.Context, tenantRepo *data.TenantRepo, nameSetMap *name_set.UserNameSetMap) {
	var ids []uint32
	for id := range *nameSetMap {
		ids = append(ids, id)
	}

	tenants, err := tenantRepo.ListTenantsByIds(ctx, ids)
	if err != nil {
		log.Errorf("query tenants err: %v", err)
		return
	}

	for _, o := range tenants {
		(*nameSetMap)[o.GetId()] = &name_set.UserNameSet{
			UserName: o.GetName(),
		}
	}
}

func QueryOrgUnitInfoFromRepo(ctx context.Context, orgUnitRepo *data.OrgUnitRepo, nameSetMap *name_set.UserNameSetMap) {
	var ids []uint32
	for id := range *nameSetMap {
		ids = append(ids, id)
	}

	orgs, err := orgUnitRepo.ListOrgUnitsByIds(ctx, ids)
	if err != nil {
		log.Errorf("query orgUnits err: %v", err)
		return
	}

	for _, o := range orgs {
		(*nameSetMap)[o.GetId()] = &name_set.UserNameSet{
			UserName: o.GetName(),
		}
	}
}

func InitOrgUnitNameSetMap(depts []*userV1.OrgUnit, orgUnitSet *name_set.UserNameSetMap) {
	for _, v := range depts {
		(*orgUnitSet)[v.GetLeaderId()] = nil
		(*orgUnitSet)[v.GetContactUserId()] = nil

		for _, c := range v.Children {
			InitOrgUnitNameSetMap(c.Children, orgUnitSet)
		}
	}
}

func InitPositionNameSetMap(positions []*userV1.Position, orgUnitSet *name_set.UserNameSetMap, deptSet *name_set.UserNameSetMap) {
	for _, v := range positions {
		if v.OrgUnitId != nil {
			(*orgUnitSet)[v.GetOrgUnitId()] = nil
		}
	}
}

func FillPositionOrgUnitInfo(positions []*userV1.Position, orgSet *name_set.UserNameSetMap) {
	for k, v := range *orgSet {
		if v == nil {
			continue
		}

		for i := 0; i < len(positions); i++ {
			if positions[i].OrgUnitId != nil && positions[i].GetOrgUnitId() == k {
				positions[i].OrgUnitName = &v.UserName
			}
		}
	}
}

func QueryPositionInfoFromRepo(ctx context.Context, positionRepo *data.PositionRepo, nameSetMap *name_set.UserNameSetMap) {
	var posIds []uint32
	for posId := range *nameSetMap {
		posIds = append(posIds, posId)
	}

	poss, err := positionRepo.ListPositionByIds(ctx, posIds)
	if err != nil {
		log.Errorf("query positions err: %v", err)
		return
	}

	for _, position := range poss {
		(*nameSetMap)[position.GetId()] = &name_set.UserNameSet{
			UserName: position.GetName(),
		}
	}
}

func QueryRoleInfoFromRepo(ctx context.Context, roleRepo *data.RoleRepo, nameSetMap *name_set.UserNameSetMap) {
	var roleIds []uint32
	for roleId := range *nameSetMap {
		roleIds = append(roleIds, roleId)
	}

	roles, err := roleRepo.ListRolesByRoleIds(ctx, roleIds)
	if err != nil {
		log.Errorf("query roles err: %v", err)
		return
	}

	for _, role := range roles {
		(*nameSetMap)[role.GetId()] = &name_set.UserNameSet{
			UserName: role.GetName(),
			Code:     role.GetCode(),
		}
	}
}

func FillOrgUnitInfo(orgUnits []*userV1.OrgUnit, orgSet *name_set.UserNameSetMap) {
	for k, v := range *orgSet {
		if v == nil {
			continue
		}

		for i := 0; i < len(orgUnits); i++ {
			if orgUnits[i].GetLeaderId() == k {
				orgUnits[i].LeaderName = &v.UserName
			}
			if orgUnits[i].GetContactUserId() == k {
				orgUnits[i].ContactUserName = &v.UserName
			}

			FillOrgUnitInfo(orgUnits[i].Children, orgSet)
		}
	}
}
