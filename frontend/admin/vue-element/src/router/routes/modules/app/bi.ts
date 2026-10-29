import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const routes: RouteRecordRaw[] = [
  {
    name: "BI",
    path: "/bi",
    component: Layout,
    meta: {
      icon: "lucide:bar-chart-3",
      order: 2,
      title: "routes.bi.title",
      authority: ["sys:platform_admin", "sys:tenant_manager"],
    },
    children: [
      {
        name: "BiDashboard",
        path: "/bi/dashboard",
        component: () => import("@/pages/app/bi/dashboard/index.vue"),
        meta: {
          order: 1,
          icon: "lucide:bar-chart-big",
          title: "routes.bi.dashboard",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
      },
      {
        name: "BiDesigner",
        path: "/bi/designer",
        component: () => import("@/pages/app/bi/designer/index.vue"),
        meta: {
          order: 2,
          icon: "lucide:pencil-ruler",
          title: "routes.bi.designer",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
      },
    ],
  },
];

export default routes;
