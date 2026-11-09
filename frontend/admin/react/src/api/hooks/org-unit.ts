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
import { apiClient } from '@/api/client';

// ==============================
// 组织架构管理
// ==============================

export function useListOrgUnits(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListOrgUnitResponse, Error>,
) {
  return useQuery({
    queryKey: ['listOrgUnits', query],
    queryFn: () => apiClient.orgUnitService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListOrgUnits(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listOrgUnits', params],
    queryFn: () => apiClient.orgUnitService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetOrgUnit(
  req: identityservicev1_GetOrgUnitRequest,
  options?: UseQueryOptions<identityservicev1_OrgUnit, Error>,
) {
  return useQuery({
    queryKey: ['getOrgUnit', req],
    queryFn: () => apiClient.orgUnitService.Get(req),
    ...options,
  });
}

export function useCreateOrgUnit(
  options?: UseMutationOptions<{}, Error, identityservicev1_CreateOrgUnitRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.orgUnitService.Create(data),
    ...options,
  });
}

export function useUpdateOrgUnit(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.orgUnitService.Update({
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
    mutationFn: (req) => apiClient.orgUnitService.Delete(req),
    ...options,
  });
}
