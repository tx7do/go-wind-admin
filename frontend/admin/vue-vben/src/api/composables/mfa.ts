import type {
  authenticationservicev1_ConfirmEnrollMethodRequest,
  authenticationservicev1_ConfirmEnrollMethodResponse,
  authenticationservicev1_DisableMFARequest,
  authenticationservicev1_GetMFAStatusResponse,
  authenticationservicev1_ListEnrolledMethodsResponse,
  authenticationservicev1_StartEnrollMethodRequest,
  authenticationservicev1_StartEnrollMethodResponse,
  authenticationservicev1_VerifyMFAChallengeRequest,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

// 供非 Vue 上下文使用的纯函数
export async function verifyMfaChallenge(
  request: authenticationservicev1_VerifyMFAChallengeRequest,
) {
  return apiClient.mfaService.VerifyMFAChallenge(request);
}

export async function getMfaStatus() {
  return apiClient.mfaService.GetMFAStatus({});
}

export async function listEnrolledMethods() {
  return apiClient.mfaService.ListEnrolledMethods({});
}

export async function startEnrollMfa(
  request: authenticationservicev1_StartEnrollMethodRequest,
) {
  return apiClient.mfaService.StartEnrollMethod(request);
}

export async function confirmEnrollMfa(
  request: authenticationservicev1_ConfirmEnrollMethodRequest,
) {
  return apiClient.mfaService.ConfirmEnrollMethod(request);
}

export async function disableMfa(request: authenticationservicev1_DisableMFARequest) {
  return apiClient.mfaService.DisableMFA(request);
}

// ------------------------------
// MFA 挑战验证（Mutation - 给 Store 调用）
// ------------------------------
export const verifyMfaMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ['mfa-verify'],
  mutationFn: verifyMfaChallenge,
  retry: 0,
});

// ------------------------------
// 获取 MFA 状态（Query）
// ------------------------------
export function useGetMfaStatus(
  options?: UseQueryOptions<authenticationservicev1_GetMFAStatusResponse, Error>,
) {
  return useQuery({
    queryKey: ['mfa-status'],
    queryFn: () => getMfaStatus(),
    ...options,
  });
}

export function useListEnrolledMethods(
  options?: UseQueryOptions<authenticationservicev1_ListEnrolledMethodsResponse, Error>,
) {
  return useQuery({
    queryKey: ['mfa-enrolled'],
    queryFn: () => listEnrolledMethods(),
    ...options,
  });
}

export function useStartEnrollMfa(
  options?: UseMutationOptions<
    authenticationservicev1_StartEnrollMethodResponse,
    Error,
    authenticationservicev1_StartEnrollMethodRequest
  >,
) {
  return useMutation({
    mutationFn: (req: authenticationservicev1_StartEnrollMethodRequest) =>
      startEnrollMfa(req),
    ...options,
  });
}

export function useConfirmEnrollMfa(
  options?: UseMutationOptions<
    authenticationservicev1_ConfirmEnrollMethodResponse,
    Error,
    authenticationservicev1_ConfirmEnrollMethodRequest
  >,
) {
  return useMutation({
    mutationFn: (req: authenticationservicev1_ConfirmEnrollMethodRequest) =>
      confirmEnrollMfa(req),
    ...options,
  });
}

export function useDisableMfa(
  options?: UseMutationOptions<{}, Error, authenticationservicev1_DisableMFARequest>,
) {
  return useMutation({
    mutationFn: (req: authenticationservicev1_DisableMFARequest) => disableMfa(req),
    ...options,
  });
}
