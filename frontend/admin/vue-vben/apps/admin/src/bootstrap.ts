import { createApp, watchEffect } from 'vue';

import { registerAccessDirective } from '@vben/access';
import { preferences } from '@vben/preferences';
import { initStores, useAccessStore } from '@vben/stores';
import '@vben/styles';
import '@vben/styles/antd';

import lucide from '@iconify/json/json/lucide.json';
import { addCollection } from '@iconify/vue';
import { useTitle } from '@vueuse/core';

import { $t, setupI18n } from '#/locales';
import { setupVueQuery } from '#/plugins/vue-query';
import { RequestClient } from '#/transport/rest';

import { initComponentAdapter } from './adapter/component';
import App from './app.vue';
import { registerGlobComp } from './registerGlobComp';
import { router } from './router';
import { useAuthStore } from './stores';

// 菜单 meta.icon 统一使用 lucide:* 图标。@iconify/vue 默认从 api.iconify.design 在线加载，
// 网络不可达时（内网/国内环境常见）未被浏览器缓存的图标会渲染失败——新加的菜单尤其容易中招。
// 这里在启动时离线注册完整 lucide 集合，保证菜单图标不依赖外网（menu-drawer 的图标选择器同理复用）。
addCollection(lucide);

async function bootstrap(namespace: string) {
  // 初始化组件适配器
  await initComponentAdapter();

  const app = createApp(App);

  // 注册全局组件
  registerGlobComp(app);

  // 初始化 Vue Query
  setupVueQuery(app);

  // 国际化 i18n 配置
  await setupI18n(app);

  // 配置 pinia-store
  await initStores(app, { namespace });

  // 注入 RequestClient 回调（业务层 → 基础设施层）
  // 必须在 initStores 之后，因为 getToken 依赖 accessStore
  const accessStore = useAccessStore();
  const authStore = useAuthStore();
  RequestClient.init(import.meta.env.VITE_GLOB_API_URL, {
    getToken: () => accessStore.accessToken,
    refreshToken: () => authStore.refreshToken(),
    onReAuthenticate: () => authStore.reauthenticate(),
    onError: (msg) => console.error('[RequestClient]', msg),
  });

  // 安装权限指令
  registerAccessDirective(app);

  // 配置路由及路由守卫
  app.use(router);

  // 动态更新标题
  watchEffect(() => {
    if (preferences.app.dynamicTitle) {
      const routeTitle = router.currentRoute.value.meta?.title;
      const pageTitle =
        (routeTitle ? `${$t(routeTitle)} - ` : '') + preferences.app.name;
      useTitle(pageTitle);
    }
  });

  // 页面刷新后的会话恢复：
  // access token 仅存内存，刷新页面后丢失。refresh token 现以 HttpOnly Cookie 传输，
  // 刷新后仍在。若检测到有效的 refresh_exp cookie，静默调 /refresh-token 换回新 access token，
  // 使用户无感知地恢复会话，无需重新登录。
  if (accessStore.accessToken) {
    // 内存中仍有 access token（如 SPA 内导航），仅需恢复定时器
    authStore.startRefreshTimer();
  } else {
    // 内存无 access token（页面刷新），尝试静默恢复
    const refreshExpireAt = getRefreshExpireAt();
    const now = Date.now();
    if (refreshExpireAt && refreshExpireAt > now) {
      try {
        // refresh token cookie 由浏览器自动携带，无需前端传参
        const newToken = await authStore.refreshToken();
        if (newToken) {
          // getUserPermissionCodes 内部会 fetchUserInfo + fetchAccessCodes，
          // 并启动定时器 + 连接 SSE。返回权限码供路由守卫使用。
          await authStore.getUserPermissionCodes();
          console.log('[Bootstrap] session silently restored via refresh cookie');
        }
      } catch (e) {
        console.log('[Bootstrap] silent restore skipped: refresh cookie invalid or expired');
      }
    }
  }

  app.mount('#app');
}

/**
 * 从 refresh_exp cookie 读取 refresh token 的过期时间戳（Unix 秒）。
 * 返回毫秒级时间戳或 null（cookie 不存在/已过期）。
 */
function getRefreshExpireAt(): number | null {
  const match = document.cookie
    .split('; ')
    .find((row) => row.startsWith('refresh_exp='));
  if (!match) return null;
  const parts = match.split('=');
  const raw = parts.length >= 2 ? parts[1] : undefined;
  if (raw === undefined) return null;
  const val = parseInt(raw, 10);
  if (!Number.isFinite(val) || val <= 0) return null;
  return val * 1000;
}

export { bootstrap };
