import { useMutation, type UseMutationOptions } from '@tanstack/react-query';
import {
  type permissionservicev1_PolicyEvaluationLog,
  type permissionservicev1_GetPolicyEvaluationLogRequest,
  type permissionservicev1_ListPolicyEvaluationLogResponse,
} from '@/api/generated/admin/service/v1';
import { type PaginationQuery } from '@/core';
import { listPolicyEvaluationLogs, getPolicyEvaluationLog } from '@/api/service/policy-evaluation-log';

// ==============================
// 策略评估日志管理
// ==============================

export function useListPolicyEvaluationLogs(
  options?: UseMutationOptions<
    permissionservicev1_ListPolicyEvaluationLogResponse,
    Error,
    PaginationQuery
  >,
) {
  return useMutation({
    mutationFn: (query) => listPolicyEvaluationLogs(query),
    ...options,
  });
}

export function useGetPolicyEvaluationLog(
  options?: UseMutationOptions<
    permissionservicev1_PolicyEvaluationLog,
    Error,
    permissionservicev1_GetPolicyEvaluationLogRequest
  >,
) {
  return useMutation({
    mutationFn: (data) => getPolicyEvaluationLog(data),
    ...options,
  });
}
