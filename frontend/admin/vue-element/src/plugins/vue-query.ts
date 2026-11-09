import type { App } from "vue";
import { defineAsyncComponent } from "vue";
import { QueryClient, VueQueryPlugin } from "@tanstack/vue-query";

/** 全局 QueryClient 实例，供 hooks 外部（Store、路由守卫等）调用 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // 管理后台不缓存查询响应：每次组件挂载都向后端重新请求
      staleTime: 0,
      // 查询在无观察者（组件卸载）后立即回收，跨挂载不残留缓存
      gcTime: 0,
      retry: 1, // 网络波动时自动重试 1 次
      refetchOnWindowFocus: false,
      refetchOnReconnect: true, // 网络恢复时自动重新请求
    },
    mutations: {
      retry: 0, // 变更操作不重试，避免重复提交
    },
  },
});

/** Vue Query Devtools 组件（动态导入，生产构建不打包；
 *  另设 VITE_QUERY_DEVTOOLS=false 可在 dev 下关闭——悬浮球会遮挡右下角弹层按钮） */
export const TanstackQueryDevtools =
  import.meta.env.DEV && import.meta.env.VITE_QUERY_DEVTOOLS !== "false"
    ? defineAsyncComponent(() =>
        import("@tanstack/vue-query-devtools").then((m) => m.VueQueryDevtools)
      )
    : defineAsyncComponent(() => Promise.resolve({ render: () => null }));

export function setupVueQuery(app: App) {
  app.use(VueQueryPlugin, {
    queryClient,
  });
}
