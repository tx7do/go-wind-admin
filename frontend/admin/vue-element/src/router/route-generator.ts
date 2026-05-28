import { ElMessage } from "element-plus";

import { BasicLayout, Layout } from "@/layouts";
import { generateAccessible } from "@/core/router";
import { preferences } from "@/core/preferences";
import { fetchNavigation } from "@/api/composables";
import type {
  ComponentRecordType,
  GenerateMenuAndRoutesOptions,
  RouteRecordStringComponent,
} from "@/core/router/types";

import { i18n } from "@/core/i18n";

const t = i18n.global.t;

const forbiddenComponent = () => import("@/pages/core/error/403.vue");

/**
 * 后端返回的路由数据结构兼容 RouteRecordStringComponent
 * （path、name、component、children、meta 等字段一致），
 * 但 TypeScript 类型不匹配，需要通过此函数显式转换。
 */
function asRouteRecords(data: Record<string, any>): RouteRecordStringComponent[] {
  return (data.items ?? []) as RouteRecordStringComponent[];
}

async function generateAccess(options: GenerateMenuAndRoutesOptions) {
  const pageMap: ComponentRecordType = import.meta.glob("../pages/**/*.vue");

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
        return asRouteRecords(data);
      } finally {
        loadingMessage.close();
      }
    },
    forbiddenComponent,
    layoutMap,
    pageMap,
  });
}

export { generateAccess };
