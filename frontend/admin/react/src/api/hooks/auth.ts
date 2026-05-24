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
import { login, logout, register, refreshToken, generateCaptcha } from '@/api/service/auth';
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
    mutationFn: (req) => login(req),
    ...options,
  });
}

// ==============================================
// 登录 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchLogin(params: authenticationservicev1_LoginRequest) {
  return queryClient.fetchQuery({
    queryKey: ['login', params],
    queryFn: () => login(params),
    retry: 0,
  });
}

// ------------------------------
// 登出（Mutation）
// ------------------------------
export function useLogout(options?: UseMutationOptions<{}, Error, {}>) {
  return useMutation({
    mutationFn: () => logout(),
    ...options,
  });
}

// ==============================================
// 登出 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchLogout() {
  return queryClient.fetchQuery({
    queryKey: ['logout'],
    queryFn: () => logout(),
    retry: 0,
  });
}

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
    mutationFn: (req) => register(req.username || '', req.password || '', req.tenantCode),
    ...options,
  });
}

// ==============================================
// 注册用户 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchRegister(username: string, password: string, tenantCode: string = '') {
  return queryClient.fetchQuery({
    queryKey: ['register', { username, password, tenantCode }],
    queryFn: () => register(username, password, tenantCode),
    retry: 0,
  });
}

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

// ==============================================
// 刷新 Token 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchRefreshToken(refreshTokenValue: string) {
  return queryClient.fetchQuery({
    queryKey: ['refreshToken', refreshTokenValue],
    queryFn: () => refreshToken(refreshTokenValue),
    retry: 0,
  });
}

// ------------------------------
// 获取验证码（Query - GET）
// ------------------------------
export function useGenerateCaptcha(
  options?: UseQueryOptions<authenticationservicev1_GenerateCaptchaResponse, Error>,
) {
  return useQuery({
    queryKey: ['captcha'],
    queryFn: () => generateCaptcha(),
    ...options,
  });
}

// ==============================================
// 获取验证码 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchGenerateCaptcha() {
  return queryClient.fetchQuery({
    queryKey: ['generateCaptcha'],
    queryFn: () => generateCaptcha(),
    retry: 0,
  });
}
