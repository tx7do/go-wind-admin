import { useMutation, useQuery, useQueryClient, type UseMutationOptions, type UseQueryOptions } from '@tanstack/react-query';
import {
  type aiservicev1_AiAgentApp,
  type aiservicev1_AiAgentRun,
  type aiservicev1_CreateAiAgentAppRequest,
  type aiservicev1_DeleteAiAgentAppRequest,
  type aiservicev1_GetAiAgentRunRequest,
  type aiservicev1_ListAiAgentAppsResponse,
  type aiservicev1_ListAiAgentRevisionsResponse,
  type aiservicev1_ListAiAgentRunsResponse,
  type aiservicev1_RestoreAiAgentRevisionRequest,
  type aiservicev1_RunAiAgentAppRequest,
  type aiservicev1_UpdateAiAgentAppRequest,
} from '@/api/generated/admin/service/v1';
import { PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// AI Agent 编排（可视化画布）
// ==============================

const LIST_KEY = 'listAiAgentApps';
const RUNS_KEY = 'listAiAgentRuns';
const REVISIONS_KEY = 'listAiAgentRevisions';

export function useListAiAgentApps(
  query: PaginationQuery,
  options?: UseQueryOptions<aiservicev1_ListAiAgentAppsResponse, Error>,
) {
  return useQuery({
    queryKey: [LIST_KEY, query],
    queryFn: () => apiClient.aiAgentService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListAiAgentApps(query: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: [LIST_KEY, query],
    queryFn: () => apiClient.aiAgentService.List(query.toRawParams()),
    retry: 0,
  });
}

export function useCreateAiAgentApp(
  options?: UseMutationOptions<aiservicev1_AiAgentApp, Error, aiservicev1_CreateAiAgentAppRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.aiAgentService.Create(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useUpdateAiAgentApp(
  options?: UseMutationOptions<{}, Error, aiservicev1_UpdateAiAgentAppRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.aiAgentService.Update(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useDeleteAiAgentApp(
  options?: UseMutationOptions<{}, Error, aiservicev1_DeleteAiAgentAppRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.aiAgentService.Delete(req),
    onSuccess: () => queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] }),
    ...options,
  });
}

export function useRunAiAgentApp(
  options?: UseMutationOptions<aiservicev1_AiAgentRun, Error, aiservicev1_RunAiAgentAppRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.aiAgentService.Run(req),
    ...options,
  });
}

export function useListAiAgentRuns(
  appId: number | undefined,
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
    ...options,
  });
}

export function useListAiAgentRevisions(
  appId: number | undefined,
  options?: UseQueryOptions<aiservicev1_ListAiAgentRevisionsResponse, Error>,
) {
  return useQuery({
    queryKey: [REVISIONS_KEY, appId],
    queryFn: () =>
      apiClient.aiAgentService.ListRevisions({
        appId,
        page: new PaginationQuery({ paging: { page: 1, pageSize: 50 } }).toRawParams(),
      }),
    enabled: !!appId,
    ...options,
  });
}

export function useRestoreAiAgentRevision(
  appId: number | undefined,
  options?: UseMutationOptions<{}, Error, aiservicev1_RestoreAiAgentRevisionRequest>,
) {
  const queryClientRef = useQueryClient();
  return useMutation({
    mutationFn: (req) => apiClient.aiAgentService.RestoreRevision(req),
    onSuccess: () => {
      queryClientRef.invalidateQueries({ queryKey: [REVISIONS_KEY, appId] });
      queryClientRef.invalidateQueries({ queryKey: [LIST_KEY] });
    },
    ...options,
  });
}
