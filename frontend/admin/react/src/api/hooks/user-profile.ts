import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type identityservicev1_User,
  type identityservicev1_UpdateUserRequest,
  type identityservicev1_ChangePasswordRequest,
  type identityservicev1_UploadAvatarRequest,
  type identityservicev1_UploadAvatarResponse,
  type identityservicev1_BindContactRequest,
  type identityservicev1_VerifyContactRequest,
} from '@/api/generated/admin/service/v1';
import {
  bindMyContact,
  changeMyPassword,
  deleteMyAvatar,
  getMe,
  updateMyUserInfo,
  uploadMyAvatar,
  verifyMyContact,
} from '@/api/service/user-profile';
import { queryClient } from '@/core';

// ==============================
// React Hooks（用于组件中，支持缓存和重试）
// ==============================

export function useGetUserProfile(
  options?: UseMutationOptions<identityservicev1_User | null, Error>,
) {
  return useMutation({
    mutationFn: () => getMe(),
    ...options,
  });
}

// ==============================================
// 获取用户资料 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchUserProfile() {
  return queryClient.fetchQuery({
    queryKey: ['userProfile'],
    queryFn: () => getMe(),
    retry: 0,
  });
}

export function useUpdateUserProfile(
  options?: UseMutationOptions<{}, Error, identityservicev1_UpdateUserRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateMyUserInfo(data),
    ...options,
  });
}

// ==============================================
// 更新用户资料 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchUpdateUserProfile(params: identityservicev1_UpdateUserRequest) {
  return queryClient.fetchQuery({
    queryKey: ['updateUserProfile', params],
    queryFn: () => updateMyUserInfo(params),
    retry: 0,
  });
}

export function useChangePassword(
  options?: UseMutationOptions<{}, Error, identityservicev1_ChangePasswordRequest>,
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

export function useDeleteAvatar(options?: UseMutationOptions<{}, Error, void>) {
  return useMutation({
    mutationFn: () => deleteMyAvatar(),
    ...options,
  });
}

export function useBindContact(
  options?: UseMutationOptions<{}, Error, identityservicev1_BindContactRequest>,
) {
  return useMutation({
    mutationFn: (data) => bindMyContact(data),
    ...options,
  });
}

export function useVerifyContact(
  options?: UseMutationOptions<{}, Error, identityservicev1_VerifyContactRequest>,
) {
  return useMutation({
    mutationFn: (data) => verifyMyContact(data),
    ...options,
  });
}
