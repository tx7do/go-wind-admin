import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const tenant: RouteRecordRaw[] = [
  {
    path: "/tenant",
    name: "TenantManagement",
    component: Layout,
    redirect: "/tenant/members",
    meta: {
      order: 2000,
      icon: "lucide:building-2",
      title: "routes.tenant.moduleName",
      authority: ["sys:platform_admin"],
    },
    children: [
      {
        path: "members",
        name: "TenantMemberManagement",
        meta: {
          order: 1,
          icon: "lucide:users",
          title: "routes.tenant.member",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/views/app/tenant/tenant/index.vue"),
      },
    ],
  },
];

export default tenant;
