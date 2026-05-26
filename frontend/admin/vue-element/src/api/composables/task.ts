import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  taskservicev1_CreateTaskRequest,
  taskservicev1_DeleteTaskRequest,
  taskservicev1_GetTaskRequest,
  taskservicev1_ListTaskResponse,
  taskservicev1_Task,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import {
  listTasks,
  getTask,
  createTask,
  updateTask,
  deleteTask,
  listTaskTypeNames,
  controlTask,
  startAllTasks,
  stopAllTasks,
  restartAllTasks,
} from "@/api/service/task";
import { queryClient } from "@/plugins/vue-query";

// ==============================
// 任务管理
// ==============================

export function useListTasks(
  query: PaginationQuery,
  options?: UseQueryOptions<taskservicev1_ListTaskResponse, Error>
) {
  return useQuery({
    queryKey: ["listTasks", query],
    queryFn: () => listTasks(query),
    ...options,
  });
}

export async function fetchListTasks(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listTasks", params],
    queryFn: () => listTasks(params),
    retry: 0,
  });
}

export function useGetTask(
  req: taskservicev1_GetTaskRequest,
  options?: UseQueryOptions<taskservicev1_Task, Error>
) {
  return useQuery({
    queryKey: ["getTask", req],
    queryFn: () => getTask(req),
    ...options,
  });
}

export function useCreateTask(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => createTask({ data: { ...values } as taskservicev1_Task }),
    ...options,
  });
}

export function useUpdateTask(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
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
  options?: UseMutationOptions<{}, Error, taskservicev1_DeleteTaskRequest>
) {
  return useMutation({
    mutationFn: (req) => deleteTask(req),
    ...options,
  });
}

// ==============================
// 任务控制
// ==============================

/** 获取任务类型名称列表 */
export async function fetchListTaskTypeNames() {
  return queryClient.fetchQuery({
    queryKey: ["listTaskTypeNames"],
    queryFn: () => listTaskTypeNames(),
    retry: 0,
  });
}

/** 控制单个任务 */
export function useControlTask(
  options?: UseMutationOptions<{}, Error, { typeName: string; controlType: string }>
) {
  return useMutation({
    mutationFn: ({ typeName, controlType }) => controlTask(typeName, controlType),
    ...options,
  });
}

/** 启动所有任务 */
export function useStartAllTasks(options?: UseMutationOptions<{}, Error, void>) {
  return useMutation({
    mutationFn: () => startAllTasks(),
    ...options,
  });
}

/** 停止所有任务 */
export function useStopAllTasks(options?: UseMutationOptions<{}, Error, void>) {
  return useMutation({
    mutationFn: () => stopAllTasks(),
    ...options,
  });
}

/** 重启所有任务 */
export function useRestartAllTasks(options?: UseMutationOptions<{}, Error, void>) {
  return useMutation({
    mutationFn: () => restartAllTasks(),
    ...options,
  });
}
