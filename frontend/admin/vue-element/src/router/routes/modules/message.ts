import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";
import { i18n } from "@/i18n/setup";

const t = i18n.global.t;

const messageRoutes: RouteRecordRaw[] = [
  {
    name: "Inbox",
    path: "/inbox",
    component: Layout,
    meta: {
      title: t("menu.profile.internalMessage"),
      hideInMenu: true,
    },
    children: [
      {
        path: "/inbox",
        name: "InboxPage",
        component: () => import("@/views/core/message/index.vue"),
        meta: {
          title: t("menu.profile.internalMessage"),
          icon: "lucide:message-circle-more",
          hideInMenu: true,
        },
      },
    ],
  },
];

export default messageRoutes;
