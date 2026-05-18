import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const routes: RouteRecordRaw[] = [
  {
    name: "Dashboard",
    path: "/dashboard",
    component: Layout,
    meta: {
      icon: "lucide:layout-dashboard",
      order: -1,
      title: "routes.dashboard.title",
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
          title: "routes.dashboard.analytics",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
      },
    ],
  },
];

export default routes;
