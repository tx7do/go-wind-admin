import { useMutation, useQuery, type UseMutationOptions, type UseQueryOptions } from '@tanstack/react-query';
import {
  type aiservicev1_GetAiAgentRunRequest,
  type aiservicev1_ListAiAgentRunsResponse,
  type aiservicev1_AiAgentRun,
} from '@/api/generated/admin/service/v1';
import { PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// AI Agent 运行台账（某应用的历次试运行记录）
// ==============================

const RUNS_KEY = 'listAiAgentRuns';

export function useListAiAgentRuns(
  appId: string | undefined,
  query: PaginationQuery,
  options?: UseQueryOptions<aiservicev1_ListAiAgentRunsResponse, Error>,
) {
  return useQuery({
    queryKey: [RUNS_KEY, appId, query],
    // appId 走 formValues → 通用 eq 过滤（camel→snake 归一由后端机制处理）
    queryFn: () =>
      apiClient.aiAgentService.ListRuns(
        new PaginationQuery({
          paging: { page: query.paging?.page, pageSize: query.paging?.pageSize },
          formValues: appId ? { appId: String(appId) } : undefined,
        }).toRawParams(),
      ),
    enabled: !!appId,
    ...options,
  });
}

export function useGetAiAgentRun(
  options?: UseMutationOptions<aiservicev1_AiAgentRun, Error, aiservicev1_GetAiAgentRunRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.aiAgentService.GetRun(req),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [RUNS_KEY] }),
    ...options,
  });
}
