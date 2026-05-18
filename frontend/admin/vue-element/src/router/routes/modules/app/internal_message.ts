import type { RouteRecordRaw } from "vue-router";

import { Layout } from "@/layouts";

const internal_message: RouteRecordRaw[] = [
  {
    path: "/internal-message",
    name: "InternalMessageManagement",
    redirect: "/internal-message/messages",
    component: Layout,
    meta: {
      order: 2003,
      icon: "lucide:mail",
      title: "routes.internalMessage.moduleName",
      keepAlive: true,
      authority: ["sys:platform_admin", "sys:tenant_manager"],
    },
    children: [
      {
        path: "messages",
        name: "InternalMessageList",
        meta: {
          order: 1,
          icon: "lucide:message-circle-more",
          title: "routes.internalMessage.internalMessage",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
        component: () => import("@/views/app/internal_message/message/index.vue"),
      },

      {
        path: "categories",
        name: "InternalMessageCategoryManagement",
        meta: {
          order: 2,
          icon: "lucide:calendar-check",
          title: "routes.internalMessage.internalMessageCategory",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/views/app/internal_message/category/index.vue"),
      },
    ],
  },
];

export default internal_message;
