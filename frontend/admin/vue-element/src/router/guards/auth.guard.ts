import type { Router } from "vue-router";

import { DEFAULT_HOME_PATH, LOGIN_PATH } from "@/constants";

import { accessRoutes, coreRouteNames } from "@/router/routes";
import { useAccessStore, useAppUserStore } from "@/stores";
import { useAuth } from "@/composables/use-auth";

import { generateAccess } from "@/router/route-generator";
import { fetchAllDictEntries } from "@/composables/use-dict-cache";

/**
 * 权限访问守卫配置
 * @param router
 */
function setupAccessGuard(router: Router) {
  router.beforeEach(async (to, from) => {
    const accessStore = useAccessStore();
    const userStore = useAppUserStore();
    const auth = useAuth();

    // 基本路由，这些路由不需要进入权限拦截
    if (coreRouteNames.includes(to.name as string)) {
      // 如果已经登录且访问登录页面，重定向到首页
      if (to.path === LOGIN_PATH && accessStore.accessToken) {
        return decodeURIComponent(
          (to.query?.redirect as string) || userStore.userInfo?.homePath || DEFAULT_HOME_PATH
        );
      }
      return true;
    }

    // accessToken 检查
    if (!accessStore.accessToken) {
      // 明确声明忽略权限访问权限，则可以访问
      if (to.meta.ignoreAccess) {
        return true;
      }

      // 没有访问权限，跳转登录页面
      if (to.path !== LOGIN_PATH) {
        return {
          path: LOGIN_PATH,
          // 如不需要，直接删除 query
          query:
            to.fullPath === DEFAULT_HOME_PATH ? {} : { redirect: encodeURIComponent(to.fullPath) },
          // 携带当前跳转的页面，登录后重新跳转该页面
          replace: true,
        };
      }
      // 已经在登录页面，直接通过
      return true;
    }

    // 是否已经生成过动态路由
    if (accessStore.isAccessChecked) {
      return true;
    }

    // 生成路由表
    // 分别获取角色码和权限码
    const userPermissionCodes = await auth.getUserPermissionCodes();
    if (!userPermissionCodes) {
      return false;
    }

    // 预先加载字典数据，部分页面可能会用到字典数据，如果没有预先加载，可能会导致页面闪烁
    await fetchAllDictEntries();

    // 生成菜单和路由
    const { accessibleMenus, accessibleRoutes } = await generateAccess({
      roles: userStore.userRoles,
      accessCodes: accessStore.accessCodes,
      router,
      // 则会在菜单中显示，但是访问会被重定向到403
      routes: accessRoutes,
    });

    // 保存菜单信息和路由信息
    accessStore.setAccessMenus(accessibleMenus);
    accessStore.setAccessRoutes(accessibleRoutes);
    accessStore.setIsAccessChecked(true);

    const redirectPath = (from.query.redirect ??
      (to.path === DEFAULT_HOME_PATH
        ? userStore.userInfo?.homePath || DEFAULT_HOME_PATH
        : to.fullPath)) as string;

    return {
      ...router.resolve(decodeURIComponent(redirectPath)),
      replace: true,
    };
  });
}

export { setupAccessGuard };
