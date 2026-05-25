import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  type taskservicev1_CreateTaskRequest,
  type taskservicev1_DeleteTaskRequest,
  type taskservicev1_GetTaskRequest,
  type taskservicev1_ListTaskResponse,
  type taskservicev1_Task,
} from '@/api/generated/admin/service/v1';
import { makeUpdateMask, type PaginationQuery, queryClient } from '@/core';
import { listTasks, getTask, createTask, updateTask, deleteTask } from '@/api/service/task';

// ==============================
// 任务管理
// ==============================

export function useListTasks(
  query: PaginationQuery,
  options?: UseQueryOptions<taskservicev1_ListTaskResponse, Error>,
) {
  return useQuery({
    queryKey: ['listTasks', query],
    queryFn: () => listTasks(query),
    ...options,
  });
}

export async function fetchListTasks(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listTasks', params],
    queryFn: () => listTasks(params),
    retry: 0,
  });
}

export function useGetTask(
  req: taskservicev1_GetTaskRequest,
  options?: UseQueryOptions<taskservicev1_Task, Error>,
) {
  return useQuery({
    queryKey: ['getTask', req],
    queryFn: () => getTask(req),
    ...options,
  });
}

export function useCreateTask(
  options?: UseMutationOptions<{}, Error, taskservicev1_CreateTaskRequest>,
) {
  return useMutation({
    mutationFn: (data) => createTask(data),
    ...options,
  });
}

export function useUpdateTask(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      updateTask({
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
    mutationFn: (req) => deleteTask(req),
    ...options,
  });
}
