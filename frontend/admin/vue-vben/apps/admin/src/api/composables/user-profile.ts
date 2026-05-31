import type {
  identityservicev1_BindContactRequest,
  identityservicev1_ChangePasswordRequest,
  identityservicev1_UploadAvatarRequest,
  identityservicev1_UploadAvatarResponse,
  identityservicev1_User,
  identityservicev1_VerifyContactRequest,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import {
  bindMyContact,
  changeMyPassword,
  deleteMyAvatar,
  getMe,
  updateMyUserInfo,
  uploadMyAvatar,
  verifyMyContact,
} from '#/api/service/user-profile';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask } from '#/transport/rest';

// 直接导出 service 层函数，供非 Vue 上下文使用
export { getMe };

export function useGetUserProfile(
  options?: UseQueryOptions<identityservicev1_User | null, Error>,
) {
  return useQuery({
    queryKey: ['getMe'],
    queryFn: () => getMe(),
    ...options,
  });
}

// ==============================================
// 获取用户资料 【给 Store / 外部调用】不用 Hook 的方式
// ==============================================
export async function fetchUserProfile() {
  return queryClient.fetchQuery({
    queryKey: ['userProfile'],
    queryFn: () => getMe(),
    retry: 0,
  });
}

export function useUpdateUserProfile(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateMyUserInfo({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useChangePassword(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_ChangePasswordRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => changeMyPassword(data),
    ...options,
  });
}

export function useUploadAvatar(
  options?: UseMutationOptions<
    identityservicev1_UploadAvatarResponse,
    Error,
    identityservicev1_UploadAvatarRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => uploadMyAvatar(data),
    ...options,
  });
}

export function useDeleteAvatar(
  options?: UseMutationOptions<object, Error, void>,
) {
  return useMutation({
    mutationFn: () => deleteMyAvatar(),
    ...options,
  });
}

export function useBindContact(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_BindContactRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => bindMyContact(data),
    ...options,
  });
}

export function useVerifyContact(
  options?: UseMutationOptions<
    object,
    Error,
    identityservicev1_VerifyContactRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => verifyMyContact(data),
    ...options,
  });
}
