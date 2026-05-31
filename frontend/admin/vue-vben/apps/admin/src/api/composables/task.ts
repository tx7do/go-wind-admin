import type {
  taskservicev1_Task_Type as Task_Type,
  taskservicev1_DeleteTaskRequest,
  taskservicev1_GetTaskRequest,
  taskservicev1_ListTaskResponse,
  taskservicev1_Task,
} from '#/api/generated/admin/service/v1';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import {
  controlTask,
  createTask,
  deleteTask,
  getTask,
  listTasks,
  listTaskTypeNames,
  restartAllTasks,
  startAllTasks,
  stopAllTasks,
  updateTask,
} from '#/api/service/task';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

const t = i18n.global.t;

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
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      createTask({ data: { ...values } as taskservicev1_Task }),
    ...options,
  });
}

export function useUpdateTask(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
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
  options?: UseMutationOptions<object, Error, taskservicev1_DeleteTaskRequest>,
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
    queryKey: ['listTaskTypeNames'],
    queryFn: () => listTaskTypeNames(),
    retry: 0,
  });
}

/** 控制单个任务 */
export function useControlTask(
  options?: UseMutationOptions<
    object,
    Error,
    { controlType: string; typeName: string }
  >,
) {
  return useMutation({
    mutationFn: ({ typeName, controlType }) =>
      controlTask(typeName, controlType),
    ...options,
  });
}

/** 启动所有任务 */
export function useStartAllTasks(
  options?: UseMutationOptions<object, Error, void>,
) {
  return useMutation({
    mutationFn: () => startAllTasks(),
    ...options,
  });
}

/** 停止所有任务 */
export function useStopAllTasks(
  options?: UseMutationOptions<object, Error, void>,
) {
  return useMutation({
    mutationFn: () => stopAllTasks(),
    ...options,
  });
}

/** 重启所有任务 */
export function useRestartAllTasks(
  options?: UseMutationOptions<object, Error, void>,
) {
  return useMutation({
    mutationFn: () => restartAllTasks(),
    ...options,
  });
}

// ==============================
// 任务枚举与工具函数
// ==============================

export const taskTypeList = computed(() => [
  { value: 'PERIODIC', label: t('enum.task.type.Periodic') },
  { value: 'DELAY', label: t('enum.task.type.Delay') },
  { value: 'WAIT_RESULT', label: t('enum.task.type.WaitResult') },
]);

export function taskTypeToName(taskType: Task_Type) {
  const values = taskTypeList.value;
  const matchedItem = values.find((item) => item.value === taskType);
  return matchedItem ? matchedItem.label : '';
}

export function taskTypeToColor(taskType: Task_Type) {
  switch (taskType) {
    case 'DELAY': {
      return 'blue';
    }
    case 'PERIODIC': {
      return 'orange';
    }
    case 'WAIT_RESULT': {
      return 'purple';
    }
    default: {
      return 'gray';
    }
  }
}
