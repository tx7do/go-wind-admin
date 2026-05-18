import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const userRoutes: RouteRecordRaw[] = [
  {
    name: "Profile",
    path: "/profile",
    component: Layout,
    meta: {
      title: "routes.profile.settings",
      hideInMenu: true,
    },

    children: [
      {
        path: "/profile",
        name: "ProfilePage",
        component: () => import("@/views/core/profile/index.vue"),
        meta: {
          title: "routes.profile.settings",
          icon: "lucide:user-pen",
          hideInMenu: true,
        },
      },
    ],
  },
];

export default userRoutes;
