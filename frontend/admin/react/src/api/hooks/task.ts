import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type taskservicev1_CreateTaskRequest,
  type taskservicev1_DeleteTaskRequest,
  type taskservicev1_GetTaskRequest,
  type taskservicev1_ListTaskResponse,
  type taskservicev1_Task,
  type taskservicev1_UpdateTaskRequest,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listTasks, getTask, createTask, updateTask, deleteTask } from '@/api/service/task';

// ==============================
// 任务管理
// ==============================

export function useListTasks(
  options?: UseMutationOptions<taskservicev1_ListTaskResponse, Error, PaginationQuery>,
) {
  return useMutation({
    mutationFn: (query) => listTasks(query),
    ...options,
  });
}

export function useGetTask(
  options?: UseMutationOptions<taskservicev1_Task, Error, taskservicev1_GetTaskRequest>,
) {
  return useMutation({
    mutationFn: (req) => getTask(req),
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
  options?: UseMutationOptions<{}, Error, taskservicev1_UpdateTaskRequest>,
) {
  return useMutation({
    mutationFn: (data) => updateTask(data),
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
