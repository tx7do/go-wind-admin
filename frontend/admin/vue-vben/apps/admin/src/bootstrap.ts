import { createApp, watchEffect } from 'vue';

import { registerAccessDirective } from '@vben/access';
import { preferences } from '@vben/preferences';
import { initStores, useAccessStore } from '@vben/stores';
import '@vben/styles';
import '@vben/styles/antd';

import { useTitle } from '@vueuse/core';

import { $t, setupI18n } from '#/locales';
import { setupVueQuery } from '#/plugins/vue-query';
import { RequestClient } from '#/transport/rest';

import { initComponentAdapter } from './adapter/component';
import App from './app.vue';
import { registerGlobComp } from './registerGlobComp';
import { router } from './router';
import { useAuthStore } from './stores';

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
    onReAuthenticate: () => authStore.logout(true),
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

  app.mount('#app');
}

export { bootstrap };
