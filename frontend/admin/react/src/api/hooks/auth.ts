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
  type authenticationservicev1_RegisterUserRequest,
  type authenticationservicev1_RegisterUserResponse,
} from '@/api/generated/admin/service/v1';
import { apiClient } from '@/api/client';
import { queryClient } from '@/core';

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
// 注册用户（Mutation）
// ------------------------------
export function useRegisterUser(
  options?: UseMutationOptions<
    authenticationservicev1_RegisterUserResponse,
    Error,
    authenticationservicev1_RegisterUserRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.authenticationService.RegisterUser(req),
    ...options,
  });
}

// ------------------------------
// 注册用户（Mutation - GET）
// ------------------------------
export const registerMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ['register'],
  mutationFn: apiClient.authenticationService.RegisterUser,
  retry: 0,
});

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
    mutationFn: (req) =>
      apiClient.authenticationService.RefreshToken({
        grant_type: 'refresh_token',
        refresh_token: req.refresh_token ?? '',
      }),
    ...options,
  });
}

// ------------------------------
// 刷新 Token（Mutation - GET）
// ------------------------------
export const refreshTokenMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ['refreshToken'],
  mutationFn: (token: string) =>
    apiClient.authenticationService.RefreshToken({
      grant_type: 'refresh_token',
      refresh_token: token ?? '',
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
