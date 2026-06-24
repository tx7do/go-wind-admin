import type {
  authenticationservicev1_GenerateCaptchaResponse,
  authenticationservicev1_LoginRequest,
  authenticationservicev1_LoginResponse,
  authenticationservicev1_RegisterUserRequest,
  authenticationservicev1_RegisterUserResponse,
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
export async function login(request: authenticationservicev1_LoginRequest) {
  return apiClient.authenticationService.Login(request);
}

export async function logout() {
  return apiClient.authenticationService.Logout({});
}

export async function registerUser(
  request: authenticationservicev1_RegisterUserRequest,
) {
  return apiClient.authenticationService.RegisterUser(request);
}

export async function generateCaptcha() {
  return apiClient.authenticationService.GenerateCaptcha({});
}

export async function refreshToken(refreshToken: string) {
  return apiClient.authenticationService.RefreshToken({
    grant_type: 'refresh_token',
    refresh_token: refreshToken ?? '',
  });
}

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
    mutationFn: (req) => login(req),
    ...options,
  });
}

// ------------------------------
// 登录（Mutation - GET）
// ------------------------------
export const loginMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ['login'],
  mutationFn: login,
  retry: 0,
});

// ------------------------------
// 登出（Mutation）
// ------------------------------
export function useLogout(options?: UseMutationOptions<object, Error, object>) {
  return useMutation({
    mutationFn: () => logout(),
    ...options,
  });
}

// ------------------------------
// 登出（Mutation - GET）
// ------------------------------
export const logoutMutation = queryClient
  .getMutationCache()
  .build(queryClient, {
    mutationKey: ['logout'],
    mutationFn: logout,
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
    mutationFn: (req) => registerUser(req),
    ...options,
  });
}

// ------------------------------
// 注册用户（Mutation - GET）
// ------------------------------
export const registerMutation = queryClient
  .getMutationCache()
  .build(queryClient, {
    mutationKey: ['register'],
    mutationFn: registerUser,
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
    mutationFn: (req) => refreshToken(req.refresh_token ?? ''),
    ...options,
  });
}

// ------------------------------
// 刷新 Token（Mutation - GET）
// ------------------------------
export const refreshTokenMutation = queryClient
  .getMutationCache()
  .build(queryClient, {
    mutationKey: ['refreshToken'],
    mutationFn: refreshToken,
    retry: 0,
  });

// ------------------------------
// 获取验证码（Query - GET）
// ------------------------------
export function useGenerateCaptcha(
  options?: UseQueryOptions<
    authenticationservicev1_GenerateCaptchaResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['captcha'],
    queryFn: () => generateCaptcha(),
    ...options,
  });
}

// ==============================================
// 获取验证码 【给 Store / 外部调用】不用 Hook 的方式
// ==============================================
export async function fetchGenerateCaptcha() {
  return queryClient.fetchQuery({
    queryKey: ['generateCaptcha'],
    queryFn: () => generateCaptcha(),
    staleTime: 0,
    retry: 0,
  });
}
