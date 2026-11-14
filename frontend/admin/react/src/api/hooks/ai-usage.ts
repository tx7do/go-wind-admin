import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/api/client';
import { PaginationQuery, queryClient } from '@go-wind/admin-react';

// useGetAiUsageSummary — 当月用量汇总（tokens/调用次数/配额上限）
export function useGetAiUsageSummary() {
  return useQuery({
    queryKey: ['aiUsageSummary'],
    queryFn: () => apiClient.aiUsageLogService.GetUsageSummary({}),
    staleTime: 60_000,
  });
}

// fetchListAiUsageLogs — ProTable request 回调用
export async function fetchListAiUsageLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listAiUsageLogs', params],
    queryFn: () => apiClient.aiUsageLogService.List(params.toRawParams()),
    retry: 0,
  });
}
