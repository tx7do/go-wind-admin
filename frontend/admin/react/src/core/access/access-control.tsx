/**
 * AccessControl 权限控制组件
 * 对齐 Vue 版 <AccessControl>，无权限时不渲染子元素
 *
 * @example
 * ```tsx
 * // 基于权限码
 * <AccessControl codes={['sys:user:create']} type="code">
 *   <Button>新建用户</Button>
 * </AccessControl>
 *
 * // 基于角色码
 * <AccessControl codes={['ADMIN']} type="role">
 *   <Button>管理员操作</Button>
 * </AccessControl>
 *
 * // 混合匹配（meta.authority 模式）
 * <AccessControl codes={['admin', 'sys:user:export']} type="authority">
 *   <Button>导出</Button>
 * </AccessControl>
 *
 * // 自定义 fallback
 * <AccessControl codes={['sys:user:delete']} type="code" fallback={<span>无权限</span>}>
 *   <Button danger>删除</Button>
 * </AccessControl>
 * ```
 */
import type { ReactNode } from 'react';
import { useAccess } from './use-access';

export interface AccessControlProps {
  /** 子元素 */
  children: ReactNode;
  /** 所需权限码/角色码列表 */
  codes: string[];
  /** 权限类型 */
  type?: 'code' | 'role' | 'authority';
  /** 无权限时的替代内容（默认不渲染） */
  fallback?: ReactNode;
}

export const AccessControl = ({
  children,
  codes,
  type = 'code',
  fallback = null,
}: AccessControlProps) => {
  const { hasAccessByCodes, hasAccessByRoles, hasAccessByAuthority } = useAccess();

  let hasAccess: boolean;

  switch (type) {
    case 'role':
      hasAccess = hasAccessByRoles(codes);
      break;
    case 'authority':
      hasAccess = hasAccessByAuthority(codes);
      break;
    case 'code':
    default:
      hasAccess = hasAccessByCodes(codes);
      break;
  }

  return hasAccess ? <>{children}</> : <>{fallback}</>;
};
