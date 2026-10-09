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
import { encryptPassword } from '#/utils';

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
    staleTime: 0,
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
    staleTime: 0,
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
    { data: Record<string, any>; password?: string }
  >,
) {
  return useMutation({
    mutationFn: ({ data, password }) =>
      apiClient.userService.Create({
        data: data as identityservicev1_User,
        password,
      }),
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
    // proto UpdateUserRequest.password 是顶层字段且要求 AES 密文：
    // 留在 data/updateMask 里后端读不到，会静默跳过重置密码（200 但密码未变）
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) => {
      const { password, ...data } = values ?? {};
      return apiClient.userService.Update({
        id,
        data: data as any,
        password: password ? encryptPassword(String(password)) : undefined,
        updateMask: makeUpdateMask(Object.keys(data)),
      });
    },
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
    // 后端 NeedDecrypt 要求 AES 密文传输（与登录同规），明文会被当密文解密导致校验必败
    mutationFn: (data) =>
      apiClient.userService.EditUserPassword({
        ...data,
        newPassword: data.newPassword ? encryptPassword(data.newPassword) : data.newPassword,
      }),
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

// 状态色使用 antd 预设状态名（由主题 token 驱动，亮/暗自动切换）。
// 语义：NORMAL=success，PENDING=warning，LOCKED=error，其余/default=default。
const USER_STATUS_COLOR_MAP: Record<
  string,
  'success' | 'warning' | 'error' | 'default'
> = {
  NORMAL: 'success',
  DISABLED: 'default',
  PENDING: 'warning',
  LOCKED: 'error',
  TERMINATED: 'default',
  EXPIRED: 'default',
  CLOSED: 'default',
  DEFAULT: 'default',
};

export function userStatusToColor(
  status: User_Status,
): 'success' | 'warning' | 'error' | 'default' {
  return USER_STATUS_COLOR_MAP[status as string] ?? 'default';
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

// 性别为无敏感语义的二元分类，使用 antd 预设调色板名（由主题 token 驱动）。
export function genderToColor(
  gender?: User_Gender,
): 'magenta' | 'blue' | 'default' {
  switch (gender) {
    case 'FEMALE': {
      return 'magenta';
    }
    case 'MALE': {
      return 'blue';
    }
    default: {
      return 'default';
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
