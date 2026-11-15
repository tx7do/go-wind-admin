import type { App } from "vue";
import { defineAsyncComponent } from "vue";
import { VueQueryPlugin } from "@tanstack/vue-query";

// 全局 QueryClient 唯一实例在 @go-wind/admin-element（登出清理 queryClient.clear()
// 也在包内）——此处 re-export 保持消费点不变，避免双实例导致登出清不到缓存。
import { queryClient } from "@go-wind/admin-element";

// 唯一实例在 @go-wind/admin-element（登出清理在包内），此处再导出保持消费点
export { queryClient };

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
