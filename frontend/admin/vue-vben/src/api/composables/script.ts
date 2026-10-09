import type {
  scriptservicev1_DeleteScriptRequest,
  scriptservicev1_PurgeScriptLogsRequest,
  scriptservicev1_PurgeScriptLogsResponse,
  scriptservicev1_GetScriptRequest,
  scriptservicev1_ListScriptsResponse,
  scriptservicev1_Script,
  scriptservicev1_TestRunScriptRequest,
  scriptservicev1_TestRunScriptResponse,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

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
    staleTime: 0,
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
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.scriptService.Create({ data: { ...values } as scriptservicev1_Script }),
    ...options,
  });
}

export function useUpdateScript(
  options?: UseMutationOptions<object, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.scriptService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function usePurgeScriptLogs(
  options?: UseMutationOptions<
    scriptservicev1_PurgeScriptLogsResponse,
    Error,
    scriptservicev1_PurgeScriptLogsRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.scriptLogService.Purge(req),
    ...options,
  });
}

export function useDeleteScript(
  options?: UseMutationOptions<object, Error, scriptservicev1_DeleteScriptRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.scriptService.Delete(req),
    ...options,
  });
}

export function useTestRunScript(
  options?: UseMutationOptions<
    scriptservicev1_TestRunScriptResponse,
    Error,
    scriptservicev1_TestRunScriptRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.scriptService.TestRun(req),
    ...options,
  });
}

export async function fetchListScriptLogs(params: any) {
  return queryClient.fetchQuery({
    queryKey: ['listScriptLogs', params],
    queryFn: () => apiClient.scriptLogService.List(params),
    staleTime: 0,
    retry: 0,
  });
}

export async function fetchHookPoints() {
  return queryClient.fetchQuery({
    queryKey: ['listHookPoints'],
    queryFn: () => apiClient.scriptService.ListHookPoints({}),
    staleTime: 30_000,
    retry: 0,
  });
}

// ==============================
// 脚本语言枚举与工具
// ==============================

export const scriptLanguageList = computed(() => [
  { value: 'LUA', label: t('page.script.languageLua') },
  { value: 'JAVASCRIPT', label: t('page.script.languageJavascript') },
]);

export function scriptLanguageToName(language: any) {
  const matched = scriptLanguageList.value.find((item) => item.value === language);
  return matched ? matched.label : language || '';
}

const SCRIPT_LANGUAGE_COLOR_MAP: Record<string, string> = {
  LUA: 'blue',
  JAVASCRIPT: 'purple',
  DEFAULT: 'default',
};

export function scriptLanguageToColor(language: any): string {
  return SCRIPT_LANGUAGE_COLOR_MAP[language as string] ?? 'default';
}
