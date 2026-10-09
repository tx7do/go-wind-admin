import type {
  ActionDistributionResponse,
  DashboardOverviewResponse,
  LoginTrendResponse,
  StatusDistributionResponse,
} from '#/api/generated/admin/service/v1';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';

// =====================================================
// 首页分析概览（只读聚合统计）
// =====================================================

// ------------------------------
// 1. 概览统计卡片
// ------------------------------
export function useDashboardOverview(
  options?: UseQueryOptions<DashboardOverviewResponse, Error>,
) {
  return useQuery({
    queryKey: ['dashboardOverview'],
    queryFn: () => apiClient.dashboardService.GetOverview({}),
    ...options,
  });
}

// ------------------------------
// 2. 近 N 天登录趋势
// ------------------------------
export function useLoginTrend(
  days: number,
  options?: UseQueryOptions<LoginTrendResponse, Error>,
) {
  return useQuery({
    queryKey: ['loginTrend', days],
    queryFn: () => apiClient.dashboardService.GetLoginTrend({ days }),
    ...options,
  });
}

// ------------------------------
// 3. 操作审计按 action 分布
// ------------------------------
export function useOperationActionDistribution(
  options?: UseQueryOptions<ActionDistributionResponse, Error>,
) {
  return useQuery({
    queryKey: ['operationActionDistribution'],
    queryFn: () => apiClient.dashboardService.GetOperationActionDistribution({}),
    ...options,
  });
}

// ------------------------------
// 4. 登录审计按 status 分布
// ------------------------------
export function useLoginStatusDistribution(
  options?: UseQueryOptions<StatusDistributionResponse, Error>,
) {
  return useQuery({
    queryKey: ['loginStatusDistribution'],
    queryFn: () => apiClient.dashboardService.GetLoginStatusDistribution({}),
    ...options,
  });
}

