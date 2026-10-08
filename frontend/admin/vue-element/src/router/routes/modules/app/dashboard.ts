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
        component: () => import("@/pages/app/dashboard/analytics/index.vue"),
        meta: {
          affixTab: true,
          icon: "lucide:area-chart",
          title: "routes.dashboard.analytics",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
      },
      {
        name: "Workspace",
        path: "/workspace",
        component: () => import("@/pages/app/dashboard/workspace/index.vue"),
        meta: {
          icon: "lucide:armchair",
          title: "routes.dashboard.workspace",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
      },
    ],
  },
];

export default routes;
