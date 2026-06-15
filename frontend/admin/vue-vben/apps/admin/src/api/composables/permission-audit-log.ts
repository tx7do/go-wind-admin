import type {
  auditservicev1_GetPermissionAuditLogRequest,
  auditservicev1_ListPermissionAuditLogResponse,
  auditservicev1_PermissionAuditLog,
  auditservicev1_PermissionAuditLog_ActionType as PermissionAuditActionType,
} from '#/api/generated/admin/service/v1';
import type { PaginationQuery } from '#/transport/rest';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

const t = i18n.global.t;

// ==============================
// 权限审计日志
// ==============================

export function useListPermissionAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<
    auditservicev1_ListPermissionAuditLogResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['listPermissionAuditLogs', query],
    queryFn: () => apiClient.permissionAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPermissionAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPermissionAuditLogs', params],
    queryFn: () => apiClient.permissionAuditLogService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetPermissionAuditLog(
  req: auditservicev1_GetPermissionAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_PermissionAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getPermissionAuditLog', req],
    queryFn: () => apiClient.permissionAuditLogService.Get(req),
    ...options,
  });
}

// ==============================
// 权限审计日志枚举与工具函数
// ==============================

export const permissionAuditLogActionList = computed(() => [
  { value: 'GRANT', label: t('enum.permissionAuditLog.action.GRANT') },
  { value: 'REVOKE', label: t('enum.permissionAuditLog.action.REVOKE') },
  { value: 'UPDATE', label: t('enum.permissionAuditLog.action.UPDATE') },
  { value: 'RESET', label: t('enum.permissionAuditLog.action.RESET') },
  { value: 'CREATE', label: t('enum.permissionAuditLog.action.CREATE') },
  { value: 'DELETE', label: t('enum.permissionAuditLog.action.DELETE') },
  { value: 'ASSIGN', label: t('enum.permissionAuditLog.action.ASSIGN') },
  { value: 'UNASSIGN', label: t('enum.permissionAuditLog.action.UNASSIGN') },
  {
    value: 'BULK_GRANT',
    label: t('enum.permissionAuditLog.action.BULK_GRANT'),
  },
  {
    value: 'BULK_REVOKE',
    label: t('enum.permissionAuditLog.action.BULK_REVOKE'),
  },
  { value: 'EXPIRE', label: t('enum.permissionAuditLog.action.EXPIRE') },
  { value: 'RESUME', label: t('enum.permissionAuditLog.action.RESUME') },
  { value: 'ROLLBACK', label: t('enum.permissionAuditLog.action.ROLLBACK') },
  { value: 'OTHER', label: t('enum.permissionAuditLog.action.OTHER') },
]);

const PERMISSION_AUDIT_LOG_ACTION_COLOR_MAP: Record<string, string> = {
  GRANT: '#1677FF',
  REVOKE: '#FF4D4F',
  UPDATE: '#597EF7',
  RESET: '#6B7280',
  CREATE: '#722ED1',
  DELETE: '#FF4D4F',
  ASSIGN: '#00B42A',
  UNASSIGN: '#FF7875',
  BULK_GRANT: '#36CFC9',
  BULK_REVOKE: '#FFC0C2',
  EXPIRE: '#FF4D4F',
  RESUME: '#00B42A',
  ROLLBACK: '#597EF7',
  OTHER: '#86909C',
  DEFAULT: '#86909C',
};

export function permissionAuditLogActionToColor(
  action: PermissionAuditActionType,
) {
  return (
    PERMISSION_AUDIT_LOG_ACTION_COLOR_MAP[action as string] ||
    PERMISSION_AUDIT_LOG_ACTION_COLOR_MAP.DEFAULT
  );
}

export function permissionAuditLogActionToName(
  action: PermissionAuditActionType,
) {
  const values = permissionAuditLogActionList.value;
  const matchedItem = values.find((item) => item.value === action);
  return matchedItem ? matchedItem.label : '';
}
