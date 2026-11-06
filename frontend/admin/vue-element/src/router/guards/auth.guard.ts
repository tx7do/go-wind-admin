import type { Router } from "vue-router";

import { DEFAULT_HOME_PATH, LOGIN_PATH } from "@/constants";

import { accessRoutes, coreRouteNames } from "@/router/routes";
import { useAccessStore, useAppUserStore } from "@/stores";
import { useAuth, NetworkError } from "@/composables/use-auth";

import { generateAccess } from "@/router/route-generator";
import { fetchAllDictEntries } from "@/composables/use-dict-cache";

/**
 * P2 修复：redirect 同源校验（对齐 react 端 GuestGuard 口径）。
 * redirect 直接来自 URL query，未校验时攻击者可构造 ?redirect=//evil.com
 * 诱导登录后跳转外站（开放重定向）。必须是站内相对路径（以 / 开头且不以
 * // 开头），否则回退默认首页。
 */
function safeRedirectPath(raw: string | undefined | null, fallback: string): string {
  let decoded = "";
  try {
    decoded = raw ? decodeURIComponent(raw) : "";
  } catch {
    decoded = "";
  }
  return decoded.startsWith("/") && !decoded.startsWith("//") ? decoded : fallback;
}

/**
 * 权限访问守卫配置
 * @param router
 */
function setupAccessGuard(router: Router) {
  router.beforeEach(async (to, from) => {
    const accessStore = useAccessStore();
    const userStore = useAppUserStore();
    const auth = useAuth();

    // MFA 待验证态收敛：mfaOperationId 非空表示密码已通过但需二次验证。
    // 此时用户无有效 token，唯一允许的目标是 MfaChallenge；其余一律强制跳过去。
    if (accessStore.mfaOperationId && to.name !== "MfaChallenge") {
      return { name: "MfaChallenge", replace: true } as any;
    }

    // 过期的 access token 视同无 token 统一跳登录，避免“仅未访问页面首次发请求
    // 被 401 拦截才登出”的不一致。治标：dev 明文 HTTP 拒收 Secure cookie 致续期
    // 链路断裂，token 过期无法刷新；此检查统一登出时机，不解决会话丢失根因。
    const tokenExpired = accessStore.checkAccessTokenExpired();

    // 基本路由，这些路由不需要进入权限拦截
    if (coreRouteNames.includes(to.name as string)) {
      // 已经有效登录且访问登录页面，重定向到首页（token 过期则放行到登录页）
      if (to.path === LOGIN_PATH && accessStore.accessToken && !tokenExpired) {
        return safeRedirectPath(
          to.query?.redirect as string | undefined,
          userStore.userInfo?.homePath || DEFAULT_HOME_PATH
        );
      }
      return true;
    }

    // accessToken 检查（含过期判定：过期视同无 token，统一跳登录页）
    if (!accessStore.accessToken || tokenExpired) {
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
    let userPermissionCodes: boolean;
    try {
      userPermissionCodes = await auth.getUserPermissionCodes();
    } catch (error: unknown) {
      // 网络异常：跳转 offline 错误页，避免白屏
      if (error instanceof NetworkError) {
        return { name: "FallbackOffline", replace: true };
      }
      // 其他异常：也跳转网络异常页
      console.error("Unexpected error in access guard:", error);
      return { name: "FallbackOffline", replace: true };
    }

    if (!userPermissionCodes) {
      // 认证失败（如 token 过期），跳转登录页
      // 注意：此时 getUserPermissionCodes 内部已通过 logoutToLoginPage 处理了跳转
      // 返回 false 阻止当前导航即可
      return false;
    }

    // 预先加载字典数据，部分页面可能会用到字典数据，如果没有预先加载，可能会导致页面闪烁。
    // 失败不阻断导航：页面退化为未翻译字典码（warn 带出原始错误便于排查）
    try {
      await fetchAllDictEntries();
    } catch (error) {
      console.warn("[Guard] 字典预加载失败，页面将以未翻译字典码降级渲染:", error);
    }

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
      ...router.resolve(safeRedirectPath(redirectPath, DEFAULT_HOME_PATH)),
      replace: true,
    };
  });
}

export { setupAccessGuard };
