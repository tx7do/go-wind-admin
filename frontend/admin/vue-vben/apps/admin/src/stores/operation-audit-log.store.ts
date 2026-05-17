import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  type auditservicev1_OperationAuditLog_ActionType as ActionType,
  createOperationAuditLogServiceClient,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const useOperationAuditLogStore = defineStore(
  'operation-audit-log',
  () => {
    const service = createOperationAuditLogServiceClient(
      requestClientRequestHandler,
    );

    const userStore = useUserStore();

    /**
     * 查询操作审计日志列表
     */
    async function listOperationAuditLog(
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
     * 查询操作审计日志日志
     */
    async function getOperationAuditLog(id: number) {
      return await service.Get({ id });
    }

    function $reset() {}

    return {
      $reset,
      listOperationAuditLog,
      getOperationAuditLog,
    };
  },
);

export const operationAuditLogActionList = computed(() => [
  { value: 'CREATE', label: $t('enum.operationAuditLog.action.CREATE') },
  { value: 'UPDATE', label: $t('enum.operationAuditLog.action.UPDATE') },
  { value: 'DELETE', label: $t('enum.operationAuditLog.action.DELETE') },
  { value: 'READ', label: $t('enum.operationAuditLog.action.READ') },
  { value: 'ASSIGN', label: $t('enum.operationAuditLog.action.ASSIGN') },
  { value: 'UNASSIGN', label: $t('enum.operationAuditLog.action.UNASSIGN') },
  { value: 'EXPORT', label: $t('enum.operationAuditLog.action.EXPORT') },
  { value: 'IMPORT', label: $t('enum.operationAuditLog.action.IMPORT') },
  { value: 'OTHER', label: $t('enum.operationAuditLog.action.OTHER') },
]);

const OPERATION_AUDIT_LOG_ACTION_COLOR_MAP = {
  CREATE: '#1677FF', // 主蓝：新增/创建（AntD 标准主色）
  UPDATE: '#597EF7', // 浅蓝：编辑/更新（区分新增，中性正向）
  DELETE: '#FF4D4F', // 警示红：删除（高风险操作，标准警示色）
  READ: '#6B7280', // 中性灰：查询/查看（无风险，替换原误导性红色）

  // 权限管理操作：统一紫色系，区分层级
  ASSIGN: '#722ED1', // 深紫：分配权限
  UNASSIGN: '#A855F7', // 浅紫：取消分配

  // 数据导入导出：绿色系，标识数据流转
  EXPORT: '#00B42A', // 深绿：导出
  IMPORT: '#36CFC9', // 青绿：导入

  // 兜底类型：统一中性灰
  OTHER: '#86909C',
  DEFAULT: '#86909C',
} as const;

export function operationAuditLogActionToColor(action: ActionType) {
  return (
    OPERATION_AUDIT_LOG_ACTION_COLOR_MAP[
      action as keyof typeof OPERATION_AUDIT_LOG_ACTION_COLOR_MAP
    ] || OPERATION_AUDIT_LOG_ACTION_COLOR_MAP.DEFAULT
  );
}

export function operationAuditLogActionToName(action: ActionType) {
  const values = operationAuditLogActionList.value;
  const matchedItem = values.find((item) => item.value === action);
  return matchedItem ? matchedItem.label : '';
}
