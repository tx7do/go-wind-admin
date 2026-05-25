import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type identityservicev1_CreateOrgUnitRequest,
  type identityservicev1_DeleteOrgUnitRequest,
  type identityservicev1_GetOrgUnitRequest,
  type identityservicev1_ListOrgUnitResponse,
  type identityservicev1_OrgUnit,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import {
  listOrgUnits,
  getOrgUnit,
  createOrgUnit,
  updateOrgUnit,
  deleteOrgUnit,
} from '@/api/service/org-unit';

// ==============================
// 组织架构管理
// ==============================

export function useListOrgUnits(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListOrgUnitResponse, Error>,
) {
  return useQuery({
    queryKey: ['listOrgUnits', query],
    queryFn: () => listOrgUnits(query),
    ...options,
  });
}

export async function fetchListOrgUnits(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listOrgUnits', params],
    queryFn: () => listOrgUnits(params),
    retry: 0,
  });
}

export function useGetOrgUnit(
  req: identityservicev1_GetOrgUnitRequest,
  options?: UseQueryOptions<identityservicev1_OrgUnit, Error>,
) {
  return useQuery({
    queryKey: ['getOrgUnit', req],
    queryFn: () => getOrgUnit(req),
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
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateOrgUnit({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
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
