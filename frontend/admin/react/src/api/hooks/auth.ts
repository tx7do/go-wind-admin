import {
  useMutation,
  useQuery,
  type UseMutationOptions,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type authenticationservicev1_GenerateCaptchaResponse,
  type authenticationservicev1_LoginRequest,
  type authenticationservicev1_LoginResponse,
} from '@/api/generated/admin/service/v1';
import { apiClient } from '@/api/client';
import { queryClient } from '@go-wind/admin-react';

// ------------------------------
// 登录（Mutation）
// ------------------------------
export function useLogin(
  options?: UseMutationOptions<
    authenticationservicev1_LoginResponse,
    Error,
    authenticationservicev1_LoginRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.authenticationService.Login(req),
    ...options,
  });
}

// ------------------------------
// 登录（Mutation - GET）
// ------------------------------
export const loginMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ['login'],
  mutationFn: apiClient.authenticationService.Login,
  retry: 0,
});

// ------------------------------
// MFA 挑战验证（Mutation）
// 提交 operation_id + TOTP 码，后端校验通过后返回 LoginResponse（含真 access_token）。
// ------------------------------
export const verifyMfaMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ['mfa-verify'],
  mutationFn: apiClient.mfaService.VerifyMFAChallenge,
  retry: 0,
});

// ------------------------------
// 登出（Mutation）
// ------------------------------
export function useLogout(options?: UseMutationOptions<{}, Error, {}>) {
  return useMutation({
    mutationFn: () => apiClient.authenticationService.Logout({}),
    ...options,
  });
}

// ------------------------------
// 登出（Mutation - GET）
// ------------------------------
export const logoutMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ['logout'],
  mutationFn: () => apiClient.authenticationService.Logout({}),
  retry: 0,
});

// ------------------------------
// ------------------------------
// 刷新 Token（Mutation）
// ------------------------------
export function useRefreshToken(
  options?: UseMutationOptions<
    authenticationservicev1_LoginResponse,
    Error,
    authenticationservicev1_LoginRequest
  >,
) {
  return useMutation({
    mutationFn: () =>
      apiClient.authenticationService.RefreshToken({
        grant_type: 'refresh_token',
      }),
    ...options,
  });
}

// ------------------------------
// 刷新 Token（Mutation - GET）
// ------------------------------
export const refreshTokenMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ['refreshToken'],
  mutationFn: () =>
    apiClient.authenticationService.RefreshToken({
      grant_type: 'refresh_token',
    }),
  retry: 0,
});

// ------------------------------
// 获取验证码（Query - GET）
// ------------------------------
export function useGenerateCaptcha(
  options?: UseQueryOptions<authenticationservicev1_GenerateCaptchaResponse, Error>,
) {
  return useQuery({
    queryKey: ['captcha'],
    queryFn: () => apiClient.authenticationService.GenerateCaptcha({}),
    ...options,
  });
}

// ==============================================
// 获取验证码 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchGenerateCaptcha() {
  return queryClient.fetchQuery({
    queryKey: ['generateCaptcha'],
    queryFn: () => apiClient.authenticationService.GenerateCaptcha({}),
    staleTime: 0,
    retry: 0,
  });
}
