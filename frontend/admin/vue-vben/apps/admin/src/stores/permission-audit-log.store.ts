import { computed } from 'vue';

import { $t } from '@vben/locales';
import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  type auditservicev1_PermissionAuditLog_ActionType as ActionType,
  createPermissionAuditLogServiceClient,
} from '#/generated/api/admin/service/v1';
import { makeOrderBy, makeQueryString } from '#/utils/query';
import { type Paging, requestClientRequestHandler } from '#/utils/request';

export const usePermissionAuditLogStore = defineStore(
  'permission-audit-log',
  () => {
    const service = createPermissionAuditLogServiceClient(
      requestClientRequestHandler,
    );
    const userStore = useUserStore();

    /**
     * 查询权限变更审计日志列表
     */
    async function listPermissionAuditLog(
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
     * 查询权限变更审计日志日志
     */
    async function getPermissionAuditLog(id: number) {
      return await service.Get({ id });
    }

    function $reset() {}

    return {
      $reset,
      listPermissionAuditLog,
      getPermissionAuditLog,
    };
  },
);

export const permissionAuditLogActionList = computed(() => [
  { value: 'GRANT', label: $t('enum.permissionAuditLog.action.GRANT') },
  { value: 'REVOKE', label: $t('enum.permissionAuditLog.action.REVOKE') },
  { value: 'UPDATE', label: $t('enum.permissionAuditLog.action.UPDATE') },
  { value: 'RESET', label: $t('enum.permissionAuditLog.action.RESET') },
  { value: 'CREATE', label: $t('enum.permissionAuditLog.action.CREATE') },
  { value: 'DELETE', label: $t('enum.permissionAuditLog.action.DELETE') },
  { value: 'ASSIGN', label: $t('enum.permissionAuditLog.action.ASSIGN') },
  { value: 'UNASSIGN', label: $t('enum.permissionAuditLog.action.UNASSIGN') },
  {
    value: 'BULK_GRANT',
    label: $t('enum.permissionAuditLog.action.BULK_GRANT'),
  },
  {
    value: 'BULK_REVOKE',
    label: $t('enum.permissionAuditLog.action.BULK_REVOKE'),
  },
  { value: 'EXPIRE', label: $t('enum.permissionAuditLog.action.EXPIRE') },
  { value: 'RESUME', label: $t('enum.permissionAuditLog.action.RESUME') },
  { value: 'ROLLBACK', label: $t('enum.permissionAuditLog.action.ROLLBACK') },
  { value: 'OTHER', label: $t('enum.permissionAuditLog.action.OTHER') },
]);

const PERMISSION_AUDIT_LOG_ACTION_COLOR_MAP = {
  // ========== 核心权限管控（单条操作，按风险分级） ==========
  GRANT: '#1677FF', // 主蓝：授予权限（正向核心操作）
  REVOKE: '#FF4D4F', // 危险红：撤销权限（高危操作，修复原配色错误）
  UPDATE: '#597EF7', // 浅蓝：更新权限（常规编辑，移除原警示红）
  RESET: '#6B7280', // 中性灰：重置权限/密码（常规操作）

  // ========== 权限配置结构操作 ==========
  CREATE: '#722ED1', // 深紫：创建权限规则/角色
  DELETE: '#FF4D4F', // 危险红：删除权限配置（高危操作，修复原配色错误）

  // ========== 主体权限分配（用户/角色绑定） ==========
  ASSIGN: '#00B42A', // 成功绿：分配权限（正向操作）
  UNASSIGN: '#FF7875', // 浅红：取消分配（中风险操作，区分高危删除）

  // ========== 批量权限操作（浅色系，区分单条操作） ==========
  BULK_GRANT: '#36CFC9', // 青绿：批量授权
  BULK_REVOKE: '#FFC0C2', // 超浅红：批量撤销（中风险）

  // ========== 权限状态管理（按风险分级） ==========
  EXPIRE: '#FF4D4F', // 危险红：权限过期（强制失效，高危）
  SUSPEND: '#FF7875', // 浅红：暂停权限（中风险）
  RESUME: '#00B42A', // 成功绿：恢复权限（正向操作）
  ROLLBACK: '#597EF7', // 浅蓝：回滚权限配置（常规修复操作）

  // ========== 兜底默认值 ==========
  OTHER: '#86909C',
  DEFAULT: '#86909C',
} as const;

export function permissionAuditLogActionToColor(action: ActionType) {
  return (
    PERMISSION_AUDIT_LOG_ACTION_COLOR_MAP[
      action as keyof typeof PERMISSION_AUDIT_LOG_ACTION_COLOR_MAP
    ] || PERMISSION_AUDIT_LOG_ACTION_COLOR_MAP.DEFAULT
  );
}

export function permissionAuditLogActionToName(action: ActionType) {
  const values = permissionAuditLogActionList.value;
  const matchedItem = values.find((item) => item.value === action);
  return matchedItem ? matchedItem.label : '';
}
