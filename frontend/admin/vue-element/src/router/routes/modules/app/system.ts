import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const system: RouteRecordRaw[] = [
  {
    path: "/system",
    name: "System",
    component: Layout,
    redirect: "/system/menus",
    meta: {
      order: 2005,
      icon: "lucide:settings",
      title: "routes.system.moduleName",
      keepAlive: true,
      authority: ["sys:platform_admin", "sys:tenant_manager"],
    },
    children: [
      {
        path: "menus",
        name: "MenuManagement",
        meta: {
          order: 1,
          icon: "lucide:square-menu",
          title: "routes.system.menu",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/views/app/system/menu/index.vue"),
      },

      {
        path: "apis",
        name: "APIManagement",
        meta: {
          order: 2,
          icon: "lucide:route",
          title: "routes.system.api",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/views/app/system/api/index.vue"),
      },

      {
        path: "dict",
        name: "DictManagement",
        meta: {
          order: 3,
          icon: "lucide:library-big",
          title: "routes.system.dict",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/views/app/system/dict/index.vue"),
      },

      {
        path: "files",
        name: "FileManagement",
        meta: {
          order: 4,
          icon: "lucide:file-search",
          title: "routes.system.file",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
        component: () => import("@/views/app/system/file/index.vue"),
      },

      {
        path: "tasks",
        name: "TaskManagement",
        meta: {
          order: 5,
          icon: "lucide:list-todo",
          title: "routes.system.task",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
        component: () => import("@/views/app/system/task/index.vue"),
      },

      {
        path: "login-policies",
        name: "LoginPolicyManagement",
        meta: {
          order: 6,
          icon: "lucide:shield-x",
          title: "routes.system.loginPolicy",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/views/app/system/login_policy/index.vue"),
      },

      {
        path: "languages",
        name: "LanguageManagement",
        meta: {
          order: 7,
          icon: "lucide:globe",
          title: "routes.system.language",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/views/app/system/language/index.vue"),
      },
    ],
  },
];

export default system;
