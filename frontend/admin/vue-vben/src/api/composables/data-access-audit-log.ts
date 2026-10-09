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
    staleTime: 0,
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

// 类型分类色使用 antd 预设调色板名（由主题 token 驱动，亮/暗自动切换），
// 保留各访问类型区分度，避免硬编码 hex 在暗黑下产生刺眼浅色块。
const DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP: Record<
  string,
  | 'blue'
  | 'geekblue'
  | 'red'
  | 'green'
  | 'cyan'
  | 'purple'
  | 'magenta'
  | 'default'
> = {
  SELECT: 'blue',
  INSERT: 'geekblue',
  UPDATE: 'geekblue',
  DELETE: 'red',
  VIEW: 'default',
  BULK_READ: 'default',
  EXPORT: 'green',
  IMPORT: 'cyan',
  DDL_CREATE: 'purple',
  DDL_ALTER: 'magenta',
  DDL_DROP: 'red',
  METADATA_READ: 'default',
  SCAN: 'default',
  ADMIN_OPERATION: 'purple',
  OTHER: 'default',
  DEFAULT: 'default',
};

export function dataAccessAuditLogAccessTypeToColor(
  accessType: AccessType,
):
  | 'blue'
  | 'geekblue'
  | 'red'
  | 'green'
  | 'cyan'
  | 'purple'
  | 'magenta'
  | 'default' {
  return (
    DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP[accessType as string] ??
    'default'
  );
}

export function dataAccessAuditLogAccessTypeToName(accessType: AccessType) {
  const values = dataAccessAuditLogAccessTypeList.value;
  const matchedItem = values.find((item) => item.value === accessType);
  return matchedItem ? matchedItem.label : '';
}

// 数据分类码与后端 pkg/audit.ClassifyTable 一一对应
export const dataAccessAuditLogCategoryList = computed(() => [
  {
    value: 'USER_DATA',
    label: t('enum.dataAccessAuditLog.dataCategory.USER_DATA'),
  },
  {
    value: 'ORG_DATA',
    label: t('enum.dataAccessAuditLog.dataCategory.ORG_DATA'),
  },
  {
    value: 'ACCESS_CONTROL',
    label: t('enum.dataAccessAuditLog.dataCategory.ACCESS_CONTROL'),
  },
  {
    value: 'TENANT_DATA',
    label: t('enum.dataAccessAuditLog.dataCategory.TENANT_DATA'),
  },
  {
    value: 'MESSAGE_DATA',
    label: t('enum.dataAccessAuditLog.dataCategory.MESSAGE_DATA'),
  },
  {
    value: 'AUDIT_LOG',
    label: t('enum.dataAccessAuditLog.dataCategory.AUDIT_LOG'),
  },
  {
    value: 'SYSTEM_CONFIG',
    label: t('enum.dataAccessAuditLog.dataCategory.SYSTEM_CONFIG'),
  },
  { value: 'UNKNOWN', label: t('enum.dataAccessAuditLog.dataCategory.UNKNOWN') },
]);

const DATA_ACCESS_AUDIT_LOG_CATEGORY_COLOR_MAP: Record<
  string,
  'blue' | 'geekblue' | 'red' | 'green' | 'cyan' | 'purple' | 'orange' | 'default'
> = {
  USER_DATA: 'blue',
  ORG_DATA: 'cyan',
  ACCESS_CONTROL: 'purple',
  TENANT_DATA: 'geekblue',
  MESSAGE_DATA: 'green',
  AUDIT_LOG: 'orange',
  SYSTEM_CONFIG: 'default',
  UNKNOWN: 'default',
};

export function dataAccessAuditLogCategoryToColor(
  category: string | undefined,
): 'blue' | 'geekblue' | 'red' | 'green' | 'cyan' | 'purple' | 'orange' | 'default' {
  return (
    DATA_ACCESS_AUDIT_LOG_CATEGORY_COLOR_MAP[category ?? ''] ?? 'default'
  );
}

export function dataAccessAuditLogCategoryToName(
  category: string | undefined,
) {
  if (!category) {
    return '';
  }
  const values = dataAccessAuditLogCategoryList.value;
  const matchedItem = values.find((item) => item.value === category);
  return matchedItem ? matchedItem.label : category;
}
