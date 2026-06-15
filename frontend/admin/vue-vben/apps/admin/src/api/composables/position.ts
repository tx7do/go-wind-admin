import type {
  identityservicev1_DeletePositionRequest,
  identityservicev1_GetPositionRequest,
  identityservicev1_ListPositionResponse,
  identityservicev1_Position,
  identityservicev1_Position_Status as Position_Status,
  identityservicev1_Position_Type as Position_Type,
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
// 职位管理
// ==============================

export function useListPositions(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListPositionResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPositions', query],
    queryFn: () => apiClient.positionService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPositions(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPositions', params],
    queryFn: () => apiClient.positionService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetPosition(
  req: identityservicev1_GetPositionRequest,
  options?: UseQueryOptions<identityservicev1_Position, Error>,
) {
  return useQuery({
    queryKey: ['getPosition', req],
    queryFn: () => apiClient.positionService.Get(req),
    ...options,
  });
}

export function useCreatePosition(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.positionService.Create({ data: { ...values } as identityservicev1_Position }),
    ...options,
  });
}

export function useUpdatePosition(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.positionService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeletePosition(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_DeletePositionRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.positionService.Delete(req),
    ...options,
  });
}

// ==============================
// 职位枚举与工具函数
// ==============================

export const membershipPositionStatusList = computed(() => [
  { value: 'PROBATION', label: t('enum.membershipPosition.status.PROBATION') },
  { value: 'ACTIVE', label: t('enum.membershipPosition.status.ACTIVE') },
  { value: 'LEAVE', label: t('enum.membershipPosition.status.LEAVE') },
  { value: 'RESIGNED', label: t('enum.membershipPosition.status.RESIGNED') },
  {
    value: 'TERMINATED',
    label: t('enum.membershipPosition.status.TERMINATED'),
  },
  { value: 'EXPIRED', label: t('enum.membershipPosition.status.EXPIRED') },
]);

export function membershipPositionStatusToName(status: any) {
  const values = membershipPositionStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

const MEMBERSHIP_POSITION_STATUS_COLOR_MAP: Record<string, string> = {
  PROBATION: '#4096FF',
  ACTIVE: '#00B42A',
  LEAVE: '#FF9A2E',
  RESIGNED: '#F56C6C',
  TERMINATED: '#F53F3F',
  EXPIRED: '#909399',
  DEFAULT: '#C9CDD4',
};

export function membershipPositionStatusToColor(status: Position_Status) {
  return (
    MEMBERSHIP_POSITION_STATUS_COLOR_MAP[status as string] ??
    MEMBERSHIP_POSITION_STATUS_COLOR_MAP.DEFAULT ??
    '#C9CDD4'
  );
}

export const positionTypeList = computed(() => [
  { value: 'REGULAR', label: t('enum.position.type.REGULAR') },
  { value: 'LEADER', label: t('enum.position.type.LEADER') },
  { value: 'MANAGER', label: t('enum.position.type.MANAGER') },
  { value: 'INTERN', label: t('enum.position.type.INTERN') },
  { value: 'CONTRACT', label: t('enum.position.type.CONTRACT') },
  { value: 'OTHER', label: t('enum.position.type.OTHER') },
]);

export function positionTypeToName(status: Position_Status) {
  const values = positionTypeList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

const POSITION_TYPE_COLOR_THEME: Record<string, Record<string, string>> = {
  light: {
    REGULAR: '#165DFF',
    LEADER: '#722ED1',
    MANAGER: '#FF7D00',
    INTERN: '#52C41A',
    CONTRACT: '#14C9C9',
    OTHER: '#86909C',
    DEFAULT: '#C9CDD4',
  },
  dark: {
    REGULAR: '#2F77FF',
    LEADER: '#8542E7',
    MANAGER: '#FF9529',
    INTERN: '#67E037',
    CONTRACT: '#20E0E0',
    OTHER: '#9BA3AD',
    DEFAULT: '#DCE0E6',
  },
};

export function positionTypeToColor(
  positionType: Position_Type,
  theme: 'dark' | 'light' = 'light',
): string {
  const colorMap = POSITION_TYPE_COLOR_THEME[theme] ?? {
    DEFAULT: '#C9CDD4',
  };
  return colorMap[positionType as string] ?? colorMap.DEFAULT ?? '#C9CDD4';
}
