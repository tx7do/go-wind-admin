import {
  createOrgUnitServiceClient,
  type identityservicev1_CreateOrgUnitRequest,
  type identityservicev1_DeleteOrgUnitRequest,
  type identityservicev1_GetOrgUnitRequest,
  type identityservicev1_UpdateOrgUnitRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery, requestApi } from '@/core';

let _orgUnitInstance: ReturnType<typeof createOrgUnitServiceClient> | null = null;

export function getOrgUnitService() {
  if (!_orgUnitInstance) {
    _orgUnitInstance = createOrgUnitServiceClient(requestApi);
  }
  return _orgUnitInstance;
}

// ==============================
// 组织架构管理
// ==============================

export async function listOrgUnits(query: PaginationQuery) {
  const params = query.toRawParams();
  return getOrgUnitService().List(params);
}

export async function getOrgUnit(request: identityservicev1_GetOrgUnitRequest) {
  return getOrgUnitService().Get(request);
}

export async function createOrgUnit(request: identityservicev1_CreateOrgUnitRequest) {
  return getOrgUnitService().Create(request);
}

export async function updateOrgUnit(request: identityservicev1_UpdateOrgUnitRequest) {
  return getOrgUnitService().Update(request);
}

export async function deleteOrgUnit(request: identityservicev1_DeleteOrgUnitRequest) {
  return getOrgUnitService().Delete(request);
}
