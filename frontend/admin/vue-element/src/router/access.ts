import { ElMessage } from "element-plus";

import { createAdminPortalServiceClient } from "@/api/generated/admin/service/v1";
import { BasicLayout, Layout } from "@/layouts";
import { requestClientRequestHandler } from "@/transport/rest";
import { generateAccessible } from "@/router/accessible";
import { defaultPreferences } from "@/settings";

import { i18n } from "@/i18n/setup";

const t = i18n.global.t;

const adminPortalService = createAdminPortalServiceClient(requestClientRequestHandler);

const forbiddenComponent = () => import("@/views/core/error/403.vue");

async function getAllMenusApi(): Promise<RouteRecordStringComponent[]> {
  const data = (await adminPortalService.GetNavigation({})) ?? [];
  return <RouteRecordStringComponent[]>data.items ?? [];
}

async function generateAccess(options: GenerateMenuAndRoutesOptions) {
  const pageMap: ComponentRecordType = import.meta.glob("../views/**/*.vue");

  const layoutMap: ComponentRecordType = {
    BasicLayout,
    Layout,
  };

  return await generateAccessible(defaultPreferences.accessMode, {
    ...options,
    fetchMenuListAsync: async () => {
      const loadingMessage = ElMessage({
        message: `${t("common.loadingMenu")}...`,
        type: "info",
        duration: 0,
      });
      try {
        return await getAllMenusApi();
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
