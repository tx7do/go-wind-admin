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

import {
  getOperationAuditLog,
  listOperationAuditLogs,
} from '#/api/service/operation-audit-log';
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
    queryFn: () => listOperationAuditLogs(query),
    ...options,
  });
}

export async function fetchListOperationAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listOperationAuditLogs', params],
    queryFn: () => listOperationAuditLogs(params),
    retry: 0,
  });
}

export function useGetOperationAuditLog(
  req: auditservicev1_GetOperationAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_OperationAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getOperationAuditLog', req],
    queryFn: () => getOperationAuditLog(req),
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

const OPERATION_AUDIT_LOG_ACTION_COLOR_MAP: Record<string, string> = {
  CREATE: '#1677FF',
  UPDATE: '#597EF7',
  DELETE: '#FF4D4F',
  READ: '#6B7280',
  ASSIGN: '#722ED1',
  UNASSIGN: '#A855F7',
  EXPORT: '#00B42A',
  IMPORT: '#36CFC9',
  OTHER: '#86909C',
  DEFAULT: '#86909C',
};

export function operationAuditLogActionToColor(action: OperationActionType) {
  return (
    OPERATION_AUDIT_LOG_ACTION_COLOR_MAP[action as string] ||
    OPERATION_AUDIT_LOG_ACTION_COLOR_MAP.DEFAULT
  );
}

export function operationAuditLogActionToName(action: OperationActionType) {
  const values = operationAuditLogActionList.value;
  const matchedItem = values.find((item) => item.value === action);
  return matchedItem ? matchedItem.label : '';
}
