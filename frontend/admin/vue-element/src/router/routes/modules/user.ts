import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";
import { i18n } from "@/i18n/setup";

const t = i18n.global.t;

const userRoutes: RouteRecordRaw[] = [
  {
    name: "Profile",
    path: "/profile",
    component: Layout,
    meta: {
      title: t("menu.profile.settings"),
      hideInMenu: true,
    },

    children: [
      {
        path: "/profile",
        name: "ProfilePage",
        component: () => import("@/views/core/profile/index.vue"),
        meta: {
          title: t("menu.profile.settings"),
          icon: "lucide:user-pen",
          hideInMenu: true,
        },
      },
    ],
  },
];

export default userRoutes;
