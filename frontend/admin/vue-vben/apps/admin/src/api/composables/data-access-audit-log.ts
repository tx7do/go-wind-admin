import type {
  auditservicev1_DataAccessAuditLog_AccessType as AccessType,
  auditservicev1_DataAccessAuditLog,
  auditservicev1_GetDataAccessAuditLogRequest,
  auditservicev1_ListDataAccessAuditLogResponse,
} from '#/api/generated/admin/service/v1';
import type { PaginationQuery } from '#/transport/rest';

import { computed } from 'vue';

import { i18n } from '@vben/locales';

import { useQuery, type UseQueryOptions } from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';

const t = i18n.global.t;

// ==============================
// 数据访问审计日志
// ==============================

export function useListDataAccessAuditLogs(
  query: PaginationQuery,
  options?: UseQueryOptions<
    auditservicev1_ListDataAccessAuditLogResponse,
    Error
  >,
) {
  return useQuery({
    queryKey: ['listDataAccessAuditLogs', query],
    queryFn: () => apiClient.dataAccessAuditLogService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListDataAccessAuditLogs(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listDataAccessAuditLogs', params],
    queryFn: () => apiClient.dataAccessAuditLogService.List(params.toRawParams()),
    retry: 0,
  });
}

export function useGetDataAccessAuditLog(
  req: auditservicev1_GetDataAccessAuditLogRequest,
  options?: UseQueryOptions<auditservicev1_DataAccessAuditLog, Error>,
) {
  return useQuery({
    queryKey: ['getDataAccessAuditLog', req],
    queryFn: () => apiClient.dataAccessAuditLogService.Get(req),
    ...options,
  });
}

// ==============================
// 数据访问审计日志枚举与工具函数
// ==============================

export const dataAccessAuditLogAccessTypeList = computed(() => [
  { value: 'SELECT', label: t('enum.dataAccessAuditLog.accessType.SELECT') },
  { value: 'INSERT', label: t('enum.dataAccessAuditLog.accessType.INSERT') },
  { value: 'UPDATE', label: t('enum.dataAccessAuditLog.accessType.UPDATE') },
  { value: 'DELETE', label: t('enum.dataAccessAuditLog.accessType.DELETE') },
  { value: 'VIEW', label: t('enum.dataAccessAuditLog.accessType.VIEW') },
  {
    value: 'BULK_READ',
    label: t('enum.dataAccessAuditLog.accessType.BULK_READ'),
  },
  { value: 'EXPORT', label: t('enum.dataAccessAuditLog.accessType.EXPORT') },
  { value: 'IMPORT', label: t('enum.dataAccessAuditLog.accessType.IMPORT') },
  {
    value: 'DDL_CREATE',
    label: t('enum.dataAccessAuditLog.accessType.DDL_CREATE'),
  },
  {
    value: 'DDL_ALTER',
    label: t('enum.dataAccessAuditLog.accessType.DDL_ALTER'),
  },
  {
    value: 'DDL_DROP',
    label: t('enum.dataAccessAuditLog.accessType.DDL_DROP'),
  },
  {
    value: 'METADATA_READ',
    label: t('enum.dataAccessAuditLog.accessType.METADATA_READ'),
  },
  { value: 'SCAN', label: t('enum.dataAccessAuditLog.accessType.SCAN') },
  {
    value: 'ADMIN_OPERATION',
    label: t('enum.dataAccessAuditLog.accessType.ADMIN_OPERATION'),
  },
  { value: 'OTHER', label: t('enum.dataAccessAuditLog.accessType.OTHER') },
]);

const DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP: Record<string, string> = {
  SELECT: '#1677FF',
  INSERT: '#597EF7',
  UPDATE: '#597EF7',
  DELETE: '#FF4D4F',
  VIEW: '#6B7280',
  BULK_READ: '#6B7280',
  EXPORT: '#00B42A',
  IMPORT: '#36CFC9',
  DDL_CREATE: '#722ED1',
  DDL_ALTER: '#A855F7',
  DDL_DROP: '#FF4D4F',
  METADATA_READ: '#86909C',
  SCAN: '#86909C',
  ADMIN_OPERATION: '#722ED1',
  OTHER: '#86909C',
  DEFAULT: '#86909C',
};

export function dataAccessAuditLogAccessTypeToColor(accessType: AccessType) {
  return (
    DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP[accessType as string] ??
    DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP.DEFAULT ??
    '#86909C'
  );
}

export function dataAccessAuditLogAccessTypeToName(accessType: AccessType) {
  const values = dataAccessAuditLogAccessTypeList.value;
  const matchedItem = values.find((item) => item.value === accessType);
  return matchedItem ? matchedItem.label : '';
}
