/**
 * 权限鉴权 Hook
 * 对齐 Vue-Vben 版 use-access.ts，提供角色码/权限码独立鉴权
 *
 * 鉴权码来源：
 *   - 角色码：userInfo.roles → useUserStore.userRoles
 *   - 权限码：GetMyPermissionCode → useUserStore.accessCodes
 *   - UI 绑定：meta.authority（角色码和权限码的混合数组）
 */
import { useCallback } from 'react';
import { useUserStore } from '@/stores/user';

/**
 * 核心权限判断逻辑（纯函数，可复用）
 */
function checkAccessByRoles(userRoles: string[], requiredRoles: string[]): boolean {
  if (!requiredRoles.length) return true;
  const userRoleSet = new Set(userRoles);
  return requiredRoles.some((item) => userRoleSet.has(item));
}

function checkAccessByCodes(accessCodes: string[], requiredCodes: string[]): boolean {
  if (!requiredCodes.length) return true;
  const userCodesSet = new Set(accessCodes);
  return requiredCodes.some((item) => userCodesSet.has(item));
}

function checkAccessByAuthority(
  userRoles: string[],
  accessCodes: string[],
  authority: string[] | undefined,
): boolean {
  if (!authority?.length) return true;
  const allSet = new Set([...userRoles, ...accessCodes]);
  return authority.some((item) => allSet.has(item));
}

/**
 * React Hook：权限鉴权（组件内使用）
 *
 * @example
 * ```tsx
 * const { hasAccessByCodes, hasAccessByRoles, hasAccessByAuthority } = useAccess();
 *
 * // 按钮级权限
 * {hasAccessByCodes(['sys:user:create']) && <Button>新建</Button>}
 * {hasAccessByRoles(['admin']) && <Button>系统设置</Button>}
 * ```
 */
function useAccess() {
  const userRoles = useUserStore((s) => s.userRoles);
  const accessCodes = useUserStore((s) => s.accessCodes);

  /**
   * 基于角色码判断是否有权限（useCallback 稳定化引用）
   */
  const hasAccessByRoles = useCallback(
    (roles: string[]): boolean => checkAccessByRoles(userRoles, roles),
    [userRoles],
  );

  /**
   * 基于权限码判断是否有权限（useCallback 稳定化引用）
   */
  const hasAccessByCodes = useCallback(
    (codes: string[]): boolean => checkAccessByCodes(accessCodes, codes),
    [accessCodes],
  );

  /**
   * 基于 meta.authority 判断是否有权限（useCallback 稳定化引用）
   */
  const hasAccessByAuthority = useCallback(
    (authority: string[] | undefined): boolean =>
      checkAccessByAuthority(userRoles, accessCodes, authority),
    [userRoles, accessCodes],
  );

  /**
   * 获取合并后的完整权限列表（useCallback 稳定化引用）
   */
  const getAllPermissions = useCallback(
    (): string[] => [...userRoles, ...accessCodes],
    [userRoles, accessCodes],
  );

  return {
    /** 用户的角色码列表 */
    userRoles,
    /** 用户的权限码列表 */
    accessCodes,
    /** 基于角色码鉴权 */
    hasAccessByRoles,
    /** 基于权限码鉴权 */
    hasAccessByCodes,
    /** 基于 meta.authority 鉴权（角色码+权限码混合匹配） */
    hasAccessByAuthority,
    /** 获取合并权限列表 */
    getAllPermissions,
  };
}

/**
 * 非组件内的静态访问方法（用于路由生成、工具函数等非 React 场景）
 *
 * @example
 * ```ts
 * // 在路由生成中使用
 * const freshPermissions = getAccessStatic().getAllPermissions();
 * ```
 */
export function getAccessStatic() {
  const { userRoles, accessCodes } = useUserStore.getState();

  function hasAccessByRoles(roles: string[]): boolean {
    return checkAccessByRoles(userRoles, roles);
  }

  function hasAccessByCodes(codes: string[]): boolean {
    return checkAccessByCodes(accessCodes, codes);
  }

  function hasAccessByAuthority(authority: string[] | undefined): boolean {
    return checkAccessByAuthority(userRoles, accessCodes, authority);
  }

  /**
   * 获取合并后的完整权限列表（角色码 + 权限码）
   */
  function getAllPermissions(): string[] {
    return [...userRoles, ...accessCodes];
  }

  return {
    userRoles,
    accessCodes,
    hasAccessByRoles,
    hasAccessByCodes,
    hasAccessByAuthority,
    getAllPermissions,
  };
}

export { useAccess, checkAccessByRoles, checkAccessByCodes, checkAccessByAuthority };
