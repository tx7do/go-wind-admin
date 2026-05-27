import { ElMessage } from "element-plus";

import { BasicLayout, Layout } from "@/layouts";
import { generateAccessible } from "@/router/accessible";
import { preferences } from "@/core/preferences";
import { fetchNavigation } from "@/api/composables";

import { i18n } from "@/i18n/setup";

const t = i18n.global.t;

const forbiddenComponent = () => import("@/views/core/error/403.vue");

async function generateAccess(options: GenerateMenuAndRoutesOptions) {
  const pageMap: ComponentRecordType = import.meta.glob("../views/**/*.vue");

  const layoutMap: ComponentRecordType = {
    BasicLayout,
    Layout,
  };

  return await generateAccessible(preferences.app.accessMode, {
    ...options,
    fetchMenuListAsync: async () => {
      const loadingMessage = ElMessage({
        message: `${t("common.loadingMenu")}...`,
        type: "info",
        duration: 0,
      });
      try {
        const data = (await fetchNavigation()) ?? {};
        return <RouteRecordStringComponent[]>(data.items ?? []);
      } finally {
        loadingMessage.close();
      }
    },
    // 可以指定没有权限跳转403页面
    forbiddenComponent,
    // 如果 route.meta.menuVisibleWithForbidden = true
    layoutMap,
    pageMap,
  });
}

export { generateAccess };
