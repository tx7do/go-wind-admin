import type {
  identityservicev1_BindContactRequest,
  identityservicev1_ChangePasswordRequest,
  identityservicev1_UpdateUserRequest,
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

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask } from '#/transport/rest';

// 供非 Vue 上下文使用的纯函数
export async function getMe(): Promise<identityservicev1_User | null> {
  try {
    return await apiClient.userProfileService.GetUser({});
  } catch (error) {
    console.error('getMe failed:', error);
    return null;
  }
}

export async function updateMyUserInfo(
  request: identityservicev1_UpdateUserRequest,
) {
  return apiClient.userProfileService.UpdateUser(request);
}

export async function changeMyPassword(
  request: identityservicev1_ChangePasswordRequest,
) {
  return apiClient.userProfileService.ChangePassword(request);
}

export async function uploadMyAvatar(
  request: identityservicev1_UploadAvatarRequest,
) {
  return apiClient.userProfileService.UploadAvatar(request);
}

export async function deleteMyAvatar() {
  return apiClient.userProfileService.DeleteAvatar({});
}

export async function bindMyContact(
  request: identityservicev1_BindContactRequest,
) {
  return apiClient.userProfileService.BindContact(request);
}

export async function verifyMyContact(
  request: identityservicev1_VerifyContactRequest,
) {
  return apiClient.userProfileService.VerifyContact(request);
}

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
