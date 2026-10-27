import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const system: RouteRecordRaw[] = [
  {
    path: "/system",
    name: "System",
    component: Layout,
    redirect: "/system/dict",
    meta: {
      order: 2005,
      icon: "lucide:settings",
      title: "routes.system.moduleName",
      keepAlive: true,
      authority: ["sys:platform_admin", "sys:tenant_manager"],
    },
    children: [
      {
        path: "dict",
        name: "DictManagement",
        meta: {
          order: 1,
          icon: "lucide:library-big",
          title: "routes.system.dict",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/system/dict/index.vue"),
      },

      {
        path: "files",
        name: "FileManagement",
        meta: {
          order: 2,
          icon: "lucide:file-search",
          title: "routes.system.file",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
        component: () => import("@/pages/app/system/file/index.vue"),
      },

      {
        path: "tasks",
        name: "TaskManagement",
        meta: {
          order: 3,
          icon: "lucide:list-todo",
          title: "routes.system.task",
          authority: ["sys:platform_admin", "sys:tenant_manager"],
        },
        component: () => import("@/pages/app/system/task/index.vue"),
      },

      {
        path: "login-policies",
        name: "LoginPolicyManagement",
        meta: {
          order: 4,
          icon: "lucide:shield-x",
          title: "routes.system.loginPolicy",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/system/login_policy/index.vue"),
      },

      {
        path: "languages",
        name: "LanguageManagement",
        meta: {
          order: 5,
          icon: "lucide:globe",
          title: "routes.system.language",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/system/language/index.vue"),
      },

      {
        path: "online-sessions",
        name: "OnlineSessionManagement",
        meta: {
          order: 6,
          icon: "lucide:monitor",
          title: "routes.system.onlineSessions",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/system/online_session/index.vue"),
      },

      {
        path: "server-monitor",
        name: "ServerMonitor",
        meta: {
          order: 7,
          icon: "lucide:activity",
          title: "routes.system.serverMonitor",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/system/server_monitor/index.vue"),
      },

      {
        // 监控告警规则：指标阈值 → 触发通知（联动通知域，平台管理员）
        path: "monitor-alerts",
        name: "MonitorAlertRule",
        meta: {
          order: 8,
          icon: "lucide:bell-plus",
          title: "routes.system.monitorAlerts",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/system/monitor_alert/index.vue"),
      },

      {
        path: "scripts",
        name: "ScriptManagement",
        meta: {
          order: 9,
          icon: "lucide:file-code-2",
          title: "routes.system.scripts",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/system/script/index.vue"),
      },

      {
        path: "configs",
        name: "ConfigManagement",
        meta: {
          order: 10,
          icon: "lucide:sliders-horizontal",
          title: "routes.system.config",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/system/config/index.vue"),
      },

      {
        path: "access-keys",
        name: "AccessKeyManagement",
        meta: {
          order: 11,
          icon: "lucide:key-round",
          title: "routes.system.accessKeys",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/system/access_key/index.vue"),
      },

      {
        path: "print-templates",
        name: "PrintTemplateManagement",
        meta: {
          order: 12,
          icon: "lucide:printer",
          title: "routes.system.printTemplates",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/system/print_template/index.vue"),
      },
    ],
  },
];

export default system;
