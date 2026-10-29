import {
  useMutation,
  useQuery,
  type UseMutationOptions,
} from '@tanstack/react-query';
import {
  type biservicev1_CopyBiDashboardRequest,
  type biservicev1_CreateBiDashboardRequest,
  type biservicev1_DeleteBiDashboardRequest,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// BI 仪表盘（设计器 schema 载体；payload 服务端不解析只透传）
// ==============================

export async function fetchListBiDashboards(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listBiDashboards', params],
    queryFn: () => apiClient.biDashboardService.List(params.toRawParams()),
    retry: 0,
  });
}

/** 按 id 取详情（命令式；独立设计器页加载用） */
export async function fetchBiDashboard(id: number) {
  return queryClient.fetchQuery({
    queryKey: ['biDashboard', id],
    queryFn: () => apiClient.biDashboardService.Get({ id }),
    retry: 0,
  });
}

/** 按 id 取详情（仪表盘页/设计器抽屉打开时拉 payload） */
export function useBiDashboard(id: number | undefined, enabled = true) {
  return useQuery({
    queryKey: ['biDashboard', id],
    queryFn: () => apiClient.biDashboardService.Get({ id }),
    enabled: enabled && !!id,
    retry: 0,
  });
}

export function useCreateBiDashboard(
  options?: UseMutationOptions<{}, Error, biservicev1_CreateBiDashboardRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.biDashboardService.Create(data),
    ...options,
  });
}

export function useUpdateBiDashboard(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.biDashboardService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

/**
 * 设计器保存：整份覆盖 payload。带 version 走乐观锁，
 * 后端对 stale 写返回 409（提示刷新重试）。
 */
export function useSaveBiDashboardPayload(
  options?: UseMutationOptions<{}, Error, { id: number; version: number; payload: string; chartCount: number; themeKey?: string }>,
) {
  return useMutation({
    mutationFn: ({ id, version, payload, chartCount, themeKey }) =>
      apiClient.biDashboardService.Update({
        id,
        data: { version, payload, chartCount, themeKey } as any,
        updateMask: makeUpdateMask(['payload', 'chartCount', 'themeKey', 'version']),
      }),
    ...options,
  });
}

export function useDeleteBiDashboard(
  options?: UseMutationOptions<{}, Error, biservicev1_DeleteBiDashboardRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.biDashboardService.Delete(req),
    ...options,
  });
}

export function useCopyBiDashboard(
  options?: UseMutationOptions<{}, Error, biservicev1_CopyBiDashboardRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.biDashboardService.Copy(req),
    ...options,
  });
}
