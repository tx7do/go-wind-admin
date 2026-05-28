import type { Router, RouteLocationNormalizedLoaded } from "vue-router";

import { effectScope, watch } from "vue";

import { startProgress, stopProgress } from "@/utils";
import { preferences } from "@/core/preferences";
import { i18n, translateRouteTitle } from "@/core/i18n";

/**
 * 根据当前路由和 i18n 状态更新浏览器标签页标题
 */
function updateDocumentTitle(route: RouteLocationNormalizedLoaded) {
  if (!preferences.app.dynamicTitle) return;
  const routeTitle = route.meta?.title as string | undefined;
  if (routeTitle) {
    const translatedTitle = translateRouteTitle(routeTitle);
    const appTitle = preferences.app.name || import.meta.env.VITE_APP_TITLE;
    document.title = `${translatedTitle} - ${appTitle}`;
  } else {
    document.title = preferences.app.name || import.meta.env.VITE_APP_TITLE || "";
  }
}

/**
 * 通用守卫配置
 * @param router
 */
function setupCommonGuard(router: Router) {
  // 记录已经加载的页面
  const loadedPaths = new Set<string>();

  router.beforeEach((to) => {
    to.meta.loaded = loadedPaths.has(to.path);

    // 页面加载进度条
    if (!to.meta.loaded && preferences.transition.progress) {
      startProgress();
    }
    return true;
  });

  router.afterEach((to) => {
    // 记录页面是否加载,如果已经加载，后续的页面切换动画等效果不在重复执行
    loadedPaths.add(to.path);

    // 关闭页面加载进度条
    if (preferences.transition.progress) {
      stopProgress();
    }

    // 动态更新页面标题
    updateDocumentTitle(to);
  });

  // 监听 i18n locale 变化，重新翻译当前路由标题
  // 解决：语言切换 / 消息延迟加载导致标题显示 i18n 原始 key 的问题
  effectScope().run(() => {
    watch(
      () => i18n.global.locale.value,
      () => {
        updateDocumentTitle(router.currentRoute.value);
      }
    );
  });
}

export { setupCommonGuard };
