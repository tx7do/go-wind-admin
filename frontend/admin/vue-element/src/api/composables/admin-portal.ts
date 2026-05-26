import { useQuery, type UseQueryOptions } from "@tanstack/vue-query";
import {
  type InitialContextResponse,
  type ListPermissionCodeResponse,
  type ListRouteResponse,
} from "@/api/generated/admin/service/v1";
import { getNavigation, getMyPermissionCode, getInitialContext } from "@/api/service/admin-portal";
import { queryClient } from "@/plugins/vue-query";

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
    retry: 0,
  });
}
