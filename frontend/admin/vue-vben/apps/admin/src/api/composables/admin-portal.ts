import type {
  InitialContextResponse,
  ListPermissionCodeResponse,
  ListRouteResponse,
} from '#/api/generated/admin/service/v1';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

// 供非 Vue 上下文使用的纯函数
export async function getMyPermissionCode() {
  return apiClient.adminPortalService.GetMyPermissionCode({});
}

// ------------------------------
// 1. 获取导航路由（左侧菜单）
// ------------------------------
export function useGetNavigation(
  options?: UseQueryOptions<ListRouteResponse, Error>,
) {
  return useQuery({
    queryKey: ['getNavigation'],
    queryFn: () => apiClient.adminPortalService.GetNavigation({}),
    ...options,
  });
}

// ==============================================
// 获取导航路由 【给 Store / 外部调用】不用 Hook 的方式
// ==============================================
export async function fetchNavigation() {
  return queryClient.fetchQuery({
    queryKey: ['navigation'],
    queryFn: () => apiClient.adminPortalService.GetNavigation({}),
    retry: 0,
  });
}

// ------------------------------
// 2. 获取当前用户权限码
// ------------------------------
export function useGetMyPermissionCode(
  options?: UseQueryOptions<ListPermissionCodeResponse, Error>,
) {
  return useQuery({
    queryKey: ['getMyPermissionCode'],
    queryFn: () => getMyPermissionCode(),
    ...options,
  });
}

// ==============================================
// 获取当前用户权限码 【给 Store / 外部调用】不用 Hook 的方式
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
export function useGetInitialContext(
  options?: UseQueryOptions<InitialContextResponse, Error>,
) {
  return useQuery({
    queryKey: ['getInitialContext'],
    queryFn: () => apiClient.adminPortalService.GetInitialContext({}),
    ...options,
  });
}

// ==============================================
// 获取初始化上下文 【给 Store / 外部调用】不用 Hook 的方式
// ==============================================
export async function fetchInitialContext() {
  return queryClient.fetchQuery({
    queryKey: ['initialContext'],
    queryFn: () => apiClient.adminPortalService.GetInitialContext({}),
    retry: 0,
  });
}
