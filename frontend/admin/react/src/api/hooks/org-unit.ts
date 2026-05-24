import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type identityservicev1_CreateOrgUnitRequest,
  type identityservicev1_DeleteOrgUnitRequest,
  type identityservicev1_GetOrgUnitRequest,
  type identityservicev1_ListOrgUnitResponse,
  type identityservicev1_OrgUnit,
  type identityservicev1_UpdateOrgUnitRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listOrgUnits, getOrgUnit, createOrgUnit, updateOrgUnit, deleteOrgUnit } from '@/api/service/org-unit';

// ==============================
// 组织架构管理
// ==============================

export function useListOrgUnits(
  options?: UseMutationOptions<identityservicev1_ListOrgUnitResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listOrgUnits(query),
    ...options,
  });
}

export function useGetOrgUnit(
  options?: UseMutationOptions<
    identityservicev1_OrgUnit,
    Error,
    identityservicev1_GetOrgUnitRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => getOrgUnit(req),
    ...options,
  });
}

export function useCreateOrgUnit(
  options?: UseMutationOptions<{}, Error, identityservicev1_CreateOrgUnitRequest>,
) {
  return useMutation({
    mutationFn: (data) => createOrgUnit(data),
    ...options,
  });
}

export function useUpdateOrgUnit(
  options?: UseMutationOptions<{}, Error, identityservicev1_UpdateOrgUnitRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateOrgUnit(data),
    ...options,
  });
}

export function useDeleteOrgUnit(
  options?: UseMutationOptions<{}, Error, identityservicev1_DeleteOrgUnitRequest>,
) {
  return useMutation({
    mutationFn: (req) => deleteOrgUnit(req),
    ...options,
  });
}
