import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type scriptservicev1_DeleteScriptRequest,
  type scriptservicev1_GetScriptRequest,
  type scriptservicev1_ListHookPointsResponse,
  type scriptservicev1_ListScriptsResponse,
  type scriptservicev1_CreateScriptRequest,
  type scriptservicev1_Script,
  type scriptservicev1_TestRunScriptRequest,
  type scriptservicev1_TestRunScriptResponse,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 脚本管理
// ==============================

export function useListScripts(
  query: PaginationQuery,
  options?: UseQueryOptions<scriptservicev1_ListScriptsResponse, Error>,
) {
  return useQuery({
    queryKey: ['listScripts', query],
    queryFn: () => apiClient.scriptService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListScripts(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listScripts', params],
    queryFn: () => apiClient.scriptService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetScript(
  req: scriptservicev1_GetScriptRequest,
  options?: UseQueryOptions<scriptservicev1_Script, Error>,
) {
  return useQuery({
    queryKey: ['getScript', req],
    queryFn: () => apiClient.scriptService.Get(req),
    ...options,
  });
}

export function useCreateScript(
  options?: UseMutationOptions<{}, Error, scriptservicev1_CreateScriptRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.scriptService.Create(data),
    ...options,
  });
}

export function useUpdateScript(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.scriptService.Update({
        id,
        data: { ...values } as any,
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteScript(
  options?: UseMutationOptions<{}, Error, scriptservicev1_DeleteScriptRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.scriptService.Delete(req),
    ...options,
  });
}

export function useTestRunScript(
  options?: UseMutationOptions<scriptservicev1_TestRunScriptResponse, Error, scriptservicev1_TestRunScriptRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.scriptService.TestRun(req),
    ...options,
  });
}

export function useListHookPoints(
  options?: Omit<
    UseQueryOptions<scriptservicev1_ListHookPointsResponse, Error>,
    'queryKey' | 'queryFn'
  >,
) {
  return useQuery({
    queryKey: ['listHookPoints'],
    queryFn: () => apiClient.scriptService.ListHookPoints({}),
    ...options,
  });
}
