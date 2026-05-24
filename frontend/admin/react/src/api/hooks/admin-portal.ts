import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type InitialContextResponse,
  type ListPermissionCodeResponse,
  type ListRouteResponse,
} from '@/api/generated/admin/service/v1';
import { getNavigation, getMyPermissionCode, getInitialContext } from '@/api/service/admin-portal';
import { queryClient } from '@/core';

// ------------------------------
// 1. 获取导航路由（左侧菜单）
// ------------------------------
export function useGetNavigation(options?: UseMutationOptions<ListRouteResponse, Error>) {
  return useMutation({
    mutationFn: () => getNavigation(),
    ...options,
  });
}

// ==============================================
// 获取导航路由 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchNavigation() {
  return queryClient.fetchQuery({
    queryKey: ['navigation'],
    queryFn: () => getNavigation(),
    retry: 0,
  });
}

// ------------------------------
// 2. 获取当前用户权限码
// ------------------------------
export function useGetMyPermissionCode(
  options?: UseMutationOptions<ListPermissionCodeResponse, Error>,
) {
  return useMutation({
    mutationFn: () => getMyPermissionCode(),
    ...options,
  });
}

// ==============================================
// 获取当前用户权限码 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchMyPermissionCode() {
  return queryClient.fetchQuery({
    queryKey: ['permissionCode'],
    queryFn: () => getMyPermissionCode(),
    retry: 0,
  });
}

// ------------------------------
// 3. 获取初始化上下文（进入后台一次性全量数据）
// ------------------------------
export function useGetInitialContext(options?: UseMutationOptions<InitialContextResponse, Error>) {
  return useMutation({
    mutationFn: () => getInitialContext(),
    ...options,
  });
}

// ==============================================
// 获取初始化上下文 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchInitialContext() {
  return queryClient.fetchQuery({
    queryKey: ['initialContext'],
    queryFn: () => getInitialContext(),
    retry: 0,
  });
}
