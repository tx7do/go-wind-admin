import type {
  identityservicev1_DeleteOrgUnitRequest,
  identityservicev1_GetOrgUnitRequest,
  identityservicev1_ListOrgUnitResponse,
  identityservicev1_OrgUnit as OrgUnit,
  identityservicev1_OrgUnit_Status as OrgUnit_Status,
  identityservicev1_OrgUnit_Type as OrgUnit_Type,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

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
    staleTime: 0,
    retry: 0,
  });
}

export function useGetOrgUnit(
  req: identityservicev1_GetOrgUnitRequest,
  options?: UseQueryOptions<OrgUnit, Error>,
) {
  return useQuery({
    queryKey: ['getOrgUnit', req],
    queryFn: () => apiClient.orgUnitService.Get(req),
    ...options,
  });
}

export function useCreateOrgUnit(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) => apiClient.orgUnitService.Create({ data: { ...values } as OrgUnit }),
    ...options,
  });
}

export function useUpdateOrgUnit(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.orgUnitService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteOrgUnit(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_DeleteOrgUnitRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.orgUnitService.Delete(req),
    ...options,
  });
}

// ==============================
// 组织单位枚举与工具函数
// ==============================

export const orgUnitStatusList = computed(() => [
  { value: 'ON', label: t('enum.status.ON') },
  { value: 'OFF', label: t('enum.status.OFF') },
]);

export function orgUnitStatusToName(status: OrgUnit_Status) {
  const values = orgUnitStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export function orgUnitStatusToColor(status: OrgUnit_Status) {
  switch (status) {
    case 'OFF': {
      return '#8C8C8C';
    }
    case 'ON': {
      return '#52C41A';
    }
    default: {
      return '#C9CDD4';
    }
  }
}

export const orgUnitTypeList = computed(() => {
  const typeOrder: OrgUnit_Type[] = [
    'COMPANY',
    'DIVISION',
    'DEPARTMENT',
    'TEAM',
    'PROJECT',
    'COMMITTEE',
    'REGION',
    'SUBSIDIARY',
    'BRANCH',
    'OTHER',
  ];
  return typeOrder.map((type) => ({
    value: type,
    label: t(`enum.orgUnit.type.${type}`),
  }));
});

export const orgUnitTypeListForQuery = computed(() => {
  const queryAllowTypes: OrgUnit_Type[] = [
    'BRANCH',
    'COMMITTEE',
    'COMPANY',
    'DEPARTMENT',
    'DIVISION',
    'OTHER',
    'PROJECT',
    'REGION',
    'SUBSIDIARY',
    'TEAM',
  ];
  const allowTypeSet = new Set(queryAllowTypes);
  return orgUnitTypeList.value.filter((item) => allowTypeSet.has(item.value));
});

export function orgUnitTypeToName(orgUnitType: OrgUnit_Type) {
  const values = orgUnitTypeList.value;
  const matchedItem = values.find((item) => item.value === orgUnitType);
  return matchedItem ? matchedItem.label : '';
}

const ORG_UNIT_COLOR_MAP: Record<string, string> = {
  BRANCH: '#4096FF',
  COMMITTEE: '#00B42A',
  COMPANY: '#165DFF',
  DEPARTMENT: '#722ED1',
  DIVISION: '#FF7D00',
  OTHER: '#86909C',
  PROJECT: '#F53F3F',
  REGION: '#14C9C9',
  SUBSIDIARY: '#6B778C',
  TEAM: '#FFC53D',
  DEFAULT: '#C9CDD4',
};

export function orgUnitTypeToColor(orgUnitType: OrgUnit_Type) {
  return (
    ORG_UNIT_COLOR_MAP[orgUnitType as string] || ORG_UNIT_COLOR_MAP.DEFAULT
  );
}

export const findOrgUnit = (
  list: OrgUnit[],
  id: number,
): null | OrgUnit | undefined => {
  for (const item of list) {
    if (item.id === id) return item;
    if (item.children && item.children.length > 0) {
      const found = findOrgUnit(item.children, id);
      if (found) return found;
    }
  }
  return null;
};
