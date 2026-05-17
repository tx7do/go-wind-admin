import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";
import { i18n } from "@/i18n/setup";

const t = i18n.global.t;

const routes: RouteRecordRaw[] = [
  {
    name: "Dashboard",
    path: "/dashboard",
    component: Layout,
    meta: {
      icon: "lucide:layout-dashboard",
      order: -1,
      title: t("page.dashboard.title"),
      authority: ["sys:platform_admin", "sys:tenant_manager"],
    },

    children: [
      {
        name: "Analytics",
        path: "/analytics",
        component: () => import("@/views/app/dashboard/analytics/index.vue"),
        meta: {
          affixTab: true,
          icon: "lucide:area-chart",
          title: t("page.dashboard.analytics"),
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
      },
    ],
  },
];

export default routes;
