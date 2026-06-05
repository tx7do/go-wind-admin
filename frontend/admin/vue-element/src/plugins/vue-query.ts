import type { App } from "vue";
import { defineAsyncComponent } from "vue";
import { QueryClient, VueQueryPlugin } from "@tanstack/vue-query";

/** 全局 QueryClient 实例，供 hooks 外部（Store、路由守卫等）调用 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60_000,
      retry: 1, // 网络波动时自动重试 1 次
      refetchOnWindowFocus: false,
      refetchOnReconnect: true, // 网络恢复时自动重新请求
    },
    mutations: {
      retry: 0, // 变更操作不重试，避免重复提交
    },
  },
});

/** Vue Query Devtools 组件（动态导入，生产构建不打包） */
export const TanstackQueryDevtools = defineAsyncComponent(() =>
  import("@tanstack/vue-query-devtools").then((m) => m.VueQueryDevtools)
);

export function setupVueQuery(app: App) {
  app.use(VueQueryPlugin, {
    queryClient,
  });
}
