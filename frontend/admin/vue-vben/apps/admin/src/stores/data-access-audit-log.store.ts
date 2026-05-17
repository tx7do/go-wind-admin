import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  type auditservicev1_DataAccessAuditLog_AccessType as AccessType,
  createDataAccessAuditLogServiceClient,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useDataAccessAuditLogStore = defineStore(
  'data-access-audit-log',
  () => {
    const service = createDataAccessAuditLogServiceClient(
      requestClientRequestHandler,
    );

    const userStore = useUserStore();

    /**
     * 查询数据访问审计日志列表
     */
    async function listDataAccessAuditLog(
      paging?: Paging,
      formValues?: null | object,
      fieldMask?: null | string,
      orderBy?: null | string[],
    ) {
      const noPaging =
        paging?.page === undefined && paging?.pageSize === undefined;
      return await service.List({
        // @ts-ignore proto generated code is error.
        fieldMask,
        orderBy: makeOrderBy(orderBy),
        query: makeQueryString(formValues, userStore.isTenantUser()),
        page: paging?.page,
        pageSize: paging?.pageSize,
        noPaging,
      });
    }

    /**
     * 查询数据访问审计日志日志
     */
    async function getDataAccessAuditLog(id: number) {
      return await service.Get({ id });
    }

    function $reset() {}

    return {
      $reset,
      listDataAccessAuditLog,
      getDataAccessAuditLog,
    };
  },
);

export const dataAccessAuditLogAccessTypeList = computed(() => [
  { value: 'SELECT', label: $t('enum.dataAccessAuditLog.accessType.SELECT') },
  { value: 'INSERT', label: $t('enum.dataAccessAuditLog.accessType.INSERT') },
  { value: 'UPDATE', label: $t('enum.dataAccessAuditLog.accessType.UPDATE') },
  { value: 'DELETE', label: $t('enum.dataAccessAuditLog.accessType.DELETE') },
  { value: 'VIEW', label: $t('enum.dataAccessAuditLog.accessType.VIEW') },
  {
    value: 'BULK_READ',
    label: $t('enum.dataAccessAuditLog.accessType.BULK_READ'),
  },
  { value: 'EXPORT', label: $t('enum.dataAccessAuditLog.accessType.EXPORT') },
  { value: 'IMPORT', label: $t('enum.dataAccessAuditLog.accessType.IMPORT') },
  {
    value: 'DDL_CREATE',
    label: $t('enum.dataAccessAuditLog.accessType.DDL_CREATE'),
  },
  {
    value: 'DDL_ALTER',
    label: $t('enum.dataAccessAuditLog.accessType.DDL_ALTER'),
  },
  {
    value: 'DDL_DROP',
    label: $t('enum.dataAccessAuditLog.accessType.DDL_DROP'),
  },
  {
    value: 'METADATA_READ',
    label: $t('enum.dataAccessAuditLog.accessType.METADATA_READ'),
  },
  { value: 'SCAN', label: $t('enum.dataAccessAuditLog.accessType.SCAN') },
  {
    value: 'ADMIN_OPERATION',
    label: $t('enum.dataAccessAuditLog.accessType.ADMIN_OPERATION'),
  },
  { value: 'OTHER', label: $t('enum.dataAccessAuditLog.accessType.OTHER') },
]);

// 数据访问审计日志 - 访问类型颜色映射
const DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP = {
  // ========== 基础DML数据操作（核心读写，蓝系主色） ==========
  SELECT: '#1677FF', // 主蓝：标准查询（核心读操作）
  INSERT: '#597EF7', // 浅蓝：数据插入（新增写操作）
  UPDATE: '#597EF7', // 浅蓝：数据更新（中风险写操作，修复原警示红错误）
  DELETE: '#FF4D4F', // 危险红：数据删除（最高风险操作，修复原中性灰错误）
  VIEW: '#6B7280', // 中性灰：视图查询
  BULK_READ: '#6B7280', // 中性灰：批量读取

  // ========== 数据流转操作（绿色系，标识数据导入导出） ==========
  EXPORT: '#00B42A', // 成功绿：数据导出
  IMPORT: '#36CFC9', // 青绿：数据导入

  // ========== DDL/元数据/管理操作（紫色系，区分风险等级） ==========
  DDL_CREATE: '#722ED1', // 深紫：创建库表结构（高风险DDL）
  DDL_ALTER: '#A855F7', // 浅紫：修改结构（中风险DDL）
  DDL_DROP: '#FF4D4F', // 危险红：删除结构（最高风险DDL，独立配色警示）
  METADATA_READ: '#86909C', // 中性灰：元数据查询
  SCAN: '#86909C', // 中性灰：全表扫描
  ADMIN_OPERATION: '#722ED1', // 深紫：管理员操作（高权限操作）

  // ========== 兜底默认值（统一中性灰） ==========
  OTHER: '#86909C',
  DEFAULT: '#86909C',
} as const;

export function dataAccessAuditLogAccessTypeToColor(accessType: AccessType) {
  return (
    DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP[
      accessType as keyof typeof DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP
    ] || DATA_ACCESS_AUDIT_LOG_ACCESS_TYPE_COLOR_MAP.DEFAULT
  );
}

export function dataAccessAuditLogAccessTypeToName(accessType: AccessType) {
  const values = dataAccessAuditLogAccessTypeList.value;
  const matchedItem = values.find((item) => item.value === accessType);
  return matchedItem ? matchedItem.label : '';
}
