import type {
  identityservicev1_EditUserPasswordRequest,
  identityservicev1_GetUserRequest,
  identityservicev1_ListUserResponse,
  identityservicev1_User,
  identityservicev1_UserExistsRequest,
  identityservicev1_UserExistsResponse,
  identityservicev1_User_Gender as User_Gender,
  identityservicev1_User_Status as User_Status,
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
// 获取用户列表
// ==============================
export function useListUsers(
  query: PaginationQuery,
  options?: UseQueryOptions<identityservicev1_ListUserResponse, Error>,
) {
  return useQuery({
    queryKey: ['listUsers', query],
    queryFn: () => listUsers(query),
    ...options,
  });
}

// ==============================================
// 获取用户列表 【给 Store / 外部调用】不用 Hook 的方式
// ==============================================
export async function fetchListUsers(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listUsers', params],
    queryFn: () => listUsers(params),
    retry: 0,
  });
}

// ==============================
// 获取单个用户
// ==============================
export function useGetUser(
  req: identityservicev1_GetUserRequest,
  options?: UseQueryOptions<identityservicev1_User, Error>,
) {
  return useQuery({
    queryKey: ['getUser', req],
    queryFn: () => apiClient.userService.Get(req),
    ...options,
  });
}

// ==============================================
// 获取单个用户 【给 Store / 外部调用】不用 Hook 的方式
// ==============================================
export async function fetchUser(params: identityservicev1_GetUserRequest) {
  return queryClient.fetchQuery({
    queryKey: ['getUser', params],
    queryFn: () => apiClient.userService.Get(params),
    retry: 0,
  });
}

// ==============================
// 创建用户
// ==============================
export function useCreateUser(
  options?: UseMutationOptions<
    object,
    Error,
    { data: identityservicev1_User; password?: string }
  >,
) {
  return useMutation({
    mutationFn: ({ data, password }) =>
      apiClient.userService.Create({ data, password }),
    ...options,
  });
}

// ==============================
// 删除用户
// ==============================
export function useDeleteUser(
  options?: UseMutationOptions<object, Error, number>,
) {
  return useMutation({
    mutationFn: (id) => apiClient.userService.Delete({ id }),
    ...options,
  });
}

// ==============================
// 更新用户
// ==============================
export function useUpdateUser(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.userService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

// ==============================
// 检查用户是否存在
// ==============================
export function useUserExists(
  options?: UseMutationOptions<
    identityservicev1_UserExistsResponse,
    Error,
    identityservicev1_UserExistsRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.userService.UserExists(data),
    ...options,
  });
}

// ==============================
// 修改用户密码（管理员）
// ==============================
export function useEditUserPassword(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_EditUserPasswordRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => apiClient.userService.EditUserPassword(data),
    ...options,
  });
}

// ==============================
// 用户枚举与工具函数
// ==============================

export const userStatusList = computed(() => [
  { value: 'NORMAL', label: t('enum.user.status.NORMAL') },
  { value: 'DISABLED', label: t('enum.user.status.DISABLED') },
  { value: 'PENDING', label: t('enum.user.status.PENDING') },
  { value: 'LOCKED', label: t('enum.user.status.LOCKED') },
  { value: 'EXPIRED', label: t('enum.user.status.EXPIRED') },
  { value: 'CLOSED', label: t('enum.user.status.CLOSED') },
]);

const USER_STATUS_COLOR_MAP: Record<string, string> = {
  NORMAL: '#4096FF',
  DISABLED: '#909399',
  PENDING: '#FF9A2E',
  LOCKED: '#F56C6C',
  TERMINATED: '#F53F3F',
  EXPIRED: '#C9CDD4',
  CLOSED: '#86909C',
  DEFAULT: '#86909C',
};

export function userStatusToColor(status: User_Status) {
  return (
    USER_STATUS_COLOR_MAP[status as string] ??
    USER_STATUS_COLOR_MAP.DEFAULT ??
    '#86909C'
  );
}

export function userStatusToName(status?: User_Status) {
  const values = userStatusList.value;
  const matchedItem = values.find((item) => item.value === status);
  return matchedItem ? matchedItem.label : '';
}

export const genderList = computed(() => [
  { value: 'SECRET', label: t('enum.gender.SECRET') },
  { value: 'MALE', label: t('enum.gender.MALE') },
  { value: 'FEMALE', label: t('enum.gender.FEMALE') },
]);

export function genderToName(gender?: User_Gender) {
  const values = genderList.value;
  const matchedItem = values.find((item) => item.value === gender);
  return matchedItem ? matchedItem.label : '';
}

export function genderToColor(gender?: User_Gender) {
  switch (gender) {
    case 'FEMALE': {
      return '#F77272';
    }
    case 'MALE': {
      return '#4096FF';
    }
    case 'SECRET': {
      return '#86909C';
    }
    default: {
      return '#C9CDD4';
    }
  }
}

// ==============================
// 内部辅助：listUsers 需要清理 toRawParams 中不需要的字段
// ==============================
async function listUsers(query: PaginationQuery) {
  const params = query.toRawParams();
  const req = {
    ...params,
    sorting: undefined,
    offset: undefined,
    limit: undefined,
    token: undefined,
    filter: undefined,
    filterExpr: undefined,
  };
  return apiClient.userService.List(req);
}
