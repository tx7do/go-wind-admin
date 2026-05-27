import { computed } from "vue";

import { preferences, updatePreferences } from "@/core/preferences";
import { useAccessStore, useAppUserStore } from "@/stores";

function useAccess() {
  const accessStore = useAccessStore();
  const userStore = useAppUserStore();
  const accessMode = computed(() => {
    return preferences.app.accessMode;
  });

  /**
   * 基于角色码判断是否有权限（精确匹配）
   * 角色码来源：getMe().roles → userStore.userRoles
   * @param roles - 所需的角色码列表
   */
  function hasAccessByRoles(roles: string[]) {
    const userRoleSet = new Set(userStore.userRoles);
    const intersection = roles.filter((item) => userRoleSet.has(item));
    return intersection.length > 0;
  }

  /**
   * 基于权限码判断是否有权限（支持前缀匹配）
   * 权限码来源：GetMyPermissionCode → accessStore.accessCodes
   *
   * 匹配规则：
   * 1. 精确匹配：用户权限码 === 所需权限码
   * 2. 前缀匹配：用户权限码是所需权限码的前缀
   *    例如用户有 sys:manage_tenants，则拥有 sys:manage_tenants:add 等
   *
   * @param codes - 所需的权限码列表
   */
  function hasAccessByCodes(codes: string[]) {
    const userCodesSet = new Set(accessStore.accessCodes);

    // 精确匹配
    const exactMatch = codes.filter((item) => userCodesSet.has(item));
    if (exactMatch.length > 0) {
      return true;
    }

    // 前缀匹配
    for (const requiredCode of codes) {
      for (const userCode of userCodesSet) {
        if (requiredCode.startsWith(userCode + ":")) {
          return true;
        }
      }
    }

    return false;
  }

  /**
   * 综合权限检查：同时检查角色码和权限码（取并集）
   * 用于路由 meta.authority 等混合标识场景
   * @param authority - 所需的权限标识列表（可包含角色码和权限码）
   */
  function hasAccess(authority: string[]) {
    return hasAccessByRoles(authority) || hasAccessByCodes(authority);
  }

  async function toggleAccessMode() {
    updatePreferences({
      app: {
        accessMode: preferences.app.accessMode === "frontend" ? "backend" : "frontend",
      },
    });
  }

  return {
    accessMode,
    hasAccess,
    hasAccessByCodes,
    hasAccessByRoles,
    toggleAccessMode,
  };
}

export { useAccess };
