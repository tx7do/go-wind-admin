import type {
  taskservicev1_InspectSystemTasksRequest,
  taskservicev1_InspectSystemTasksResponse,
} from '#/api/generated/admin/service/v1';

import { useMutation } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';

// ==============================
// 系统级常驻任务监控（只读 asynq Inspector，方案 C' 零新表）
// ==============================

/** 一站式巡检：调度条目（cron/上次/下次入队）+ 各任务类型的队列状态与失败明细 */
export function useInspectSystemTasks(
  options?: Parameters<
    typeof useMutation<
      taskservicev1_InspectSystemTasksResponse,
      Error,
      taskservicev1_InspectSystemTasksRequest
    >
  >[0],
) {
  return useMutation({
    mutationFn: (req: taskservicev1_InspectSystemTasksRequest) =>
      apiClient.taskMonitorService.InspectSystemTasks(req),
    ...options,
  });
}
