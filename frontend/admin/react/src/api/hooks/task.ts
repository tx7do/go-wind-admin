import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type taskservicev1_ControlTaskRequest,
  type taskservicev1_CreateTaskRequest,
  type taskservicev1_DeleteTaskRequest,
  type taskservicev1_GetTaskRequest,
  type taskservicev1_ListTaskResponse,
  type taskservicev1_Task,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { apiClient } from '@/api/client';

// ==============================
// 任务管理
// ==============================

export function useListTasks(
  query: PaginationQuery,
  options?: UseQueryOptions<taskservicev1_ListTaskResponse, Error>,
) {
  return useQuery({
    queryKey: ['listTasks', query],
    queryFn: () => apiClient.taskService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListTasks(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listTasks', params],
    queryFn: () => apiClient.taskService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetTask(
  req: taskservicev1_GetTaskRequest,
  options?: UseQueryOptions<taskservicev1_Task, Error>,
) {
  return useQuery({
    queryKey: ['getTask', req],
    queryFn: () => apiClient.taskService.Get(req),
    ...options,
  });
}

export function useCreateTask(
  options?: UseMutationOptions<{}, Error, taskservicev1_CreateTaskRequest>,
) {
  return useMutation({
    mutationFn: (data) => apiClient.taskService.Create(data),
    ...options,
  });
}

export function useUpdateTask(
  options?: UseMutationOptions<{}, Error, { id: string; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: string; values: Record<string, any> }) =>
      apiClient.taskService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      }),
    ...options,
  });
}

export function useDeleteTask(
  options?: UseMutationOptions<{}, Error, taskservicev1_DeleteTaskRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.taskService.Delete(req),
    ...options,
  });
}

// ==============================
// 任务控制
// ==============================

/** 获取任务类型名称列表 */
export async function fetchListTaskTypeNames() {
  return queryClient.fetchQuery({
    queryKey: ['listTaskTypeNames'],
    queryFn: () => apiClient.taskService.ListTaskTypeName({}),
    retry: 0,
  });
}

/** 控制单个任务 */
export function useControlTask(
  options?: UseMutationOptions<{}, Error, { typeName: string; controlType: string }>,
) {
  return useMutation({
    mutationFn: ({ typeName, controlType }) =>
      apiClient.taskService.ControlTask({
        typeName,
        controlType: controlType as taskservicev1_ControlTaskRequest['controlType'],
      }),
    ...options,
  });
}

/** 启动所有任务 */
export function useStartAllTasks(options?: UseMutationOptions<{}, Error, void>) {
  return useMutation({
    mutationFn: () => apiClient.taskService.StartAllTask({}),
    ...options,
  });
}

/** 停止所有任务 */
export function useStopAllTasks(options?: UseMutationOptions<{}, Error, void>) {
  return useMutation({
    mutationFn: () => apiClient.taskService.StopAllTask({}),
    ...options,
  });
}

/** 重启所有任务 */
export function useRestartAllTasks(options?: UseMutationOptions<{}, Error, void>) {
  return useMutation({
    mutationFn: () => apiClient.taskService.RestartAllTask({}),
    ...options,
  });
}
