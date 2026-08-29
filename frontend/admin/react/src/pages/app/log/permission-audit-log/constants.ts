/**
 * 权限审计日志模块枚举映射常量
 */

type TFn = (key: string, options?: Record<string, any>) => string;

// ========== 操作类型 ==========

/** 操作类型枚举值列表（与后端 proto PermissionAuditLog.ActionType 一致） */
export const ACTION_VALUES = [
  'GRANT',
  'REVOKE',
  'UPDATE',
  'RESET',
  'CREATE',
  'DELETE',
  'ASSIGN',
  'UNASSIGN',
  'BULK_GRANT',
  'BULK_REVOKE',
  'EXPIRE',
  'SUSPEND',
  'RESUME',
  'ROLLBACK',
  'OTHER',
] as const;

/** 操作类型颜色映射 */
export const ACTION_COLORS: Record<string, string> = {
  GRANT: 'success',
  BULK_GRANT: 'success',
  RESUME: 'success',
  REVOKE: 'error',
  BULK_REVOKE: 'error',
  DELETE: 'error',
  SUSPEND: 'error',
  UPDATE: 'warning',
  EXPIRE: 'warning',
  ROLLBACK: 'warning',
  RESET: 'processing',
  CREATE: 'processing',
  ASSIGN: 'geekblue',
  UNASSIGN: 'default',
  OTHER: 'default',
};

/** 获取操作类型映射（text + color），供 render 使用 */
export function getActionMap(t: TFn) {
  const map: Record<string, { text: string; color: string }> = {};
  for (const value of ACTION_VALUES) {
    map[value] = { text: t(`action.${value}`), color: ACTION_COLORS[value] };
  }
  return map;
}

/** 获取操作类型选项列表，供搜索/Select 使用 */
export function getActionOptions(t: TFn) {
  return ACTION_VALUES.map((value) => ({
    label: t(`action.${value}`),
    value,
  }));
}
