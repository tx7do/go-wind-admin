import type {
  auditservicev1_GetOperationAuditLogRequest,
  auditservicev1_ListOperationAuditLogResponse,
  auditservicev1_OperationAuditLog,
  auditservicev1_OperationAuditLog_ActionType as OperationActionType,
} from '#/api/generated/admin/service/v1';
import type { PaginationQuery } from '#/transport/rest';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

const t = i18n.global.t;

// ==============================
// 操作审计日志
// ==============================

export function useListOperationAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<
    auditservicev1_ListOperationAuditLogResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['listOperationAuditLogs', query],
    queryFn: () => apiClient.operationAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListOperationAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listOperationAuditLogs', params],
    queryFn: () => apiClient.operationAuditLogService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetOperationAuditLog(
  req: auditservicev1_GetOperationAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_OperationAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getOperationAuditLog', req],
    queryFn: () => apiClient.operationAuditLogService.Get(req),
    ...options,
  });
}

// ==============================
// 操作审计日志枚举与工具函数
// ==============================

export const operationAuditLogActionList = computed(() => [
  { value: 'CREATE', label: t('enum.operationAuditLog.action.CREATE') },
  { value: 'UPDATE', label: t('enum.operationAuditLog.action.UPDATE') },
  { value: 'DELETE', label: t('enum.operationAuditLog.action.DELETE') },
  { value: 'READ', label: t('enum.operationAuditLog.action.READ') },
  { value: 'ASSIGN', label: t('enum.operationAuditLog.action.ASSIGN') },
  { value: 'UNASSIGN', label: t('enum.operationAuditLog.action.UNASSIGN') },
  { value: 'EXPORT', label: t('enum.operationAuditLog.action.EXPORT') },
  { value: 'IMPORT', label: t('enum.operationAuditLog.action.IMPORT') },
  { value: 'OTHER', label: t('enum.operationAuditLog.action.OTHER') },
]);

// 类型分类色使用 antd 预设调色板名（由主题 token 驱动，亮/暗自动切换）。
const OPERATION_AUDIT_LOG_ACTION_COLOR_MAP: Record<
  string,
  | 'blue'
  | 'geekblue'
  | 'red'
  | 'purple'
  | 'magenta'
  | 'green'
  | 'cyan'
  | 'default'
> = {
  CREATE: 'blue',
  UPDATE: 'geekblue',
  DELETE: 'red',
  READ: 'default',
  ASSIGN: 'purple',
  UNASSIGN: 'magenta',
  EXPORT: 'green',
  IMPORT: 'cyan',
  OTHER: 'default',
  DEFAULT: 'default',
};

export function operationAuditLogActionToColor(
  action: OperationActionType,
):
  | 'blue'
  | 'geekblue'
  | 'red'
  | 'purple'
  | 'magenta'
  | 'green'
  | 'cyan'
  | 'default' {
  return (
    OPERATION_AUDIT_LOG_ACTION_COLOR_MAP[action as string] ?? 'default'
  );
}

export function operationAuditLogActionToName(action: OperationActionType) {
  const values = operationAuditLogActionList.value;
  const matchedItem = values.find((item) => item.value === action);
  return matchedItem ? matchedItem.label : '';
}
