import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const messageRoutes: RouteRecordRaw[] = [
  {
    name: "Inbox",
    path: "/inbox",
    component: Layout,
    meta: {
      title: "routes.profile.internalMessage",
      hideInMenu: true,
    },
    children: [
      {
        path: "/inbox",
        name: "InboxPage",
        component: () => import("@/views/core/message/index.vue"),
        meta: {
          title: "routes.profile.internalMessage",
          icon: "lucide:message-circle-more",
          hideInMenu: true,
        },
      },
    ],
  },
];

export default messageRoutes;
