import { useQuery, type UseQueryOptions } from "@tanstack/vue-query";
import type {
  InitialContextResponse,
  ListPermissionCodeResponse,
  ListRouteResponse,
} from "@/api/generated/admin/service/v1";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";

// 直接导出函数，供非 Vue 上下文使用
export async function getMyPermissionCode() {
  return apiClient.adminPortalService.GetMyPermissionCode({});
}

async function getNavigation() {
  return apiClient.adminPortalService.GetNavigation({});
}

async function getInitialContext() {
  return apiClient.adminPortalService.GetInitialContext({});
}

// ------------------------------
// 1. 获取导航路由（左侧菜单）
// ------------------------------
export function useGetNavigation(options?: UseQueryOptions<ListRouteResponse, Error>) {
  return useQuery({
    queryKey: ["getNavigation"],
    queryFn: () => getNavigation(),
    ...options,
  });
}

// ==============================================
// 获取导航路由 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchNavigation() {
  return queryClient.fetchQuery({
    queryKey: ["navigation"],
    queryFn: () => getNavigation(),
    staleTime: 0,
    retry: 0,
  });
}

// ------------------------------
// 2. 获取当前用户权限码
// ------------------------------
export function useGetMyPermissionCode(
  options?: UseQueryOptions<ListPermissionCodeResponse, Error>
) {
  return useQuery({
    queryKey: ["getMyPermissionCode"],
    queryFn: () => getMyPermissionCode(),
    ...options,
  });
}

// ==============================================
// 获取当前用户权限码 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchMyPermissionCode() {
  return queryClient.fetchQuery({
    queryKey: ["permissionCode"],
    queryFn: () => getMyPermissionCode(),
    staleTime: 0,
    retry: 0,
  });
}

// ------------------------------
// 3. 获取初始化上下文（进入后台一次性全量数据）
// ------------------------------
export function useGetInitialContext(options?: UseQueryOptions<InitialContextResponse, Error>) {
  return useQuery({
    queryKey: ["getInitialContext"],
    queryFn: () => getInitialContext(),
    ...options,
  });
}

// ==============================================
// 获取初始化上下文 【给 Store / 外部调用】不带 Hook 的方法
// ==============================================
export async function fetchInitialContext() {
  return queryClient.fetchQuery({
    queryKey: ["initialContext"],
    queryFn: () => getInitialContext(),
    staleTime: 0,
    retry: 0,
  });
}
