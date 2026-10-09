import type { App } from 'vue';
import { defineAsyncComponent } from 'vue';

import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';

/** 全局 QueryClient 实例，供 hooks 外部（Store、路由守卫等）调用 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // 管理后台不缓存查询响应：每次组件挂载都向后端重新请求
      staleTime: 0,
      // 查询在无观察者（组件卸载）后立即回收，跨挂载不残留缓存
      gcTime: 0,
      retry: false,
      refetchOnWindowFocus: false,
      refetchOnReconnect: false,
    },
  },
});

/** Vue Query Devtools 组件（仅开发环境加载，生产环境为 null） */
export const TanstackQueryDevtools = import.meta.env.DEV
  ? defineAsyncComponent(async () => {
      const m = await import('@tanstack/vue-query-devtools');
      return m.VueQueryDevtools;
    })
  : null;

export function setupVueQuery(app: App) {
  app.use(VueQueryPlugin, {
    queryClient,
  });
}
