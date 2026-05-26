import {
  useMutation,
  useQuery,
  type UseMutationOptions,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  authenticationservicev1_GenerateCaptchaResponse,
  authenticationservicev1_LoginRequest,
  authenticationservicev1_LoginResponse,
  authenticationservicev1_RegisterUserRequest,
  authenticationservicev1_RegisterUserResponse,
} from "@/api/generated/admin/service/v1";
import { login, logout, refreshToken, generateCaptcha, registerUser } from "@/api/service/auth";
import { queryClient } from "@/plugins/vue-query";

// ------------------------------
// 登录（Mutation）
// ------------------------------
export function useLogin(
  options?: UseMutationOptions<
    authenticationservicev1_LoginResponse,
    Error,
    authenticationservicev1_LoginRequest
  >
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
  mutationKey: ["login"],
  mutationFn: login,
  retry: 0,
});

// ------------------------------
// 登出（Mutation）
// ------------------------------
export function useLogout(options?: UseMutationOptions<{}, Error, {}>) {
  return useMutation({
    mutationFn: () => logout(),
    ...options,
  });
}

// ------------------------------
// 登出（Mutation - GET）
// ------------------------------
export const logoutMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ["logout"],
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
  >
) {
  return useMutation({
    mutationFn: (req) => registerUser(req),
    ...options,
  });
}

// ------------------------------
// 注册用户（Mutation - GET）
// ------------------------------
export const registerMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ["register"],
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
  >
) {
  return useMutation({
    mutationFn: (req) => refreshToken(req.refresh_token ?? ""),
    ...options,
  });
}

// ------------------------------
// 刷新 Token（Mutation - GET）
// ------------------------------
export const refreshTokenMutation = queryClient.getMutationCache().build(queryClient, {
  mutationKey: ["refreshToken"],
  mutationFn: refreshToken,
  retry: 0,
});

// ------------------------------
// 获取验证码（Query - GET）
// ------------------------------
export function useGenerateCaptcha(
  options?: UseQueryOptions<authenticationservicev1_GenerateCaptchaResponse, Error>
) {
  return useQuery({
    queryKey: ["captcha"],
    queryFn: () => generateCaptcha(),
    ...options,
  });
}

// ==============================================
// 获取验证码 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchGenerateCaptcha() {
  return queryClient.fetchQuery({
    queryKey: ["generateCaptcha"],
    queryFn: () => generateCaptcha(),
    retry: 0,
  });
}
