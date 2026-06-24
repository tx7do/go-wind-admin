import { computed } from "vue";
import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from "@tanstack/vue-query";
import type {
  taskservicev1_ControlTaskRequest,
  taskservicev1_DeleteTaskRequest,
  taskservicev1_GetTaskRequest,
  taskservicev1_ListTaskResponse,
  taskservicev1_Task,
  taskservicev1_Task_Type as Task_Type,
} from "@/api/generated/admin/service/v1";
import { makeUpdateMask, type PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { queryClient } from "@/plugins/vue-query";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

// ==============================
// 任务管理
// ==============================

export function useListTasks(
  query: PaginationQuery,
  options?: UseQueryOptions<taskservicev1_ListTaskResponse, Error>
) {
  return useQuery({
    queryKey: ["listTasks", query],
    queryFn: () => apiClient.taskService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListTasks(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ["listTasks", params],
    queryFn: () => apiClient.taskService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetTask(
  req: taskservicev1_GetTaskRequest,
  options?: UseQueryOptions<taskservicev1_Task, Error>
) {
  return useQuery({
    queryKey: ["getTask", req],
    queryFn: () => apiClient.taskService.Get(req),
    ...options,
  });
}

export function useCreateTask(options?: UseMutationOptions<{}, Error, Record<string, any>>) {
  return useMutation({
    mutationFn: (values) => apiClient.taskService.Create({ data: { ...values } as taskservicev1_Task }),
    ...options,
  });
}

export function useUpdateTask(
  options?: UseMutationOptions<{}, Error, { id: number; values: Record<string, any> }>
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.taskService.Update({
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
    queryKey: ["listTaskTypeNames"],
    queryFn: () => apiClient.taskService.ListTaskTypeName({}),
    staleTime: 0,
    retry: 0,
  });
}

/** 控制单个任务 */
export function useControlTask(
  options?: UseMutationOptions<{}, Error, { typeName: string; controlType: string }>
) {
  return useMutation({
    mutationFn: ({ typeName, controlType }) => apiClient.taskService.ControlTask({
      typeName,
      controlType: controlType as taskservicev1_ControlTaskRequest["controlType"],
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

// ==============================
// 任务枚举与工具函数
// ==============================

export const taskTypeList = computed(() => [
  { value: "PERIODIC", label: t("enum.task.type.Periodic") },
  { value: "DELAY", label: t("enum.task.type.Delay") },
  { value: "WAIT_RESULT", label: t("enum.task.type.WaitResult") },
]);

export function taskTypeToName(taskType: Task_Type) {
  const values = taskTypeList.value;
  const matchedItem = values.find((item) => item.value === taskType);
  return matchedItem ? matchedItem.label : "";
}

export function taskTypeToColor(taskType: Task_Type) {
  switch (taskType) {
    case "DELAY":
      return "blue";
    case "PERIODIC":
      return "orange";
    case "WAIT_RESULT":
      return "purple";
    default:
      return "gray";
  }
}
