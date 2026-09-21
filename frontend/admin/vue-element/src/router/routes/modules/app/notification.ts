import type { RouteRecordRaw } from "vue-router";

import { Layout } from "@/layouts";

/**
 * 通知域路由：渠道配置、路由规则（事件 → 渠道）、投递台账。
 * 站内信只是本域的一个 INTERNAL 渠道，故这三页与 /internal-message 平级而非挂在其下。
 */
const notification: RouteRecordRaw[] = [
  {
    path: "/notification",
    name: "NotificationManagement",
    redirect: "/notification/channels",
    component: Layout,
    meta: {
      order: 2006,
      icon: "lucide:bell-ring",
      title: "routes.notification.moduleName",
      keepAlive: true,
      authority: ["sys:platform_admin"],
    },
    children: [
      {
        path: "channels",
        name: "NotificationChannelManagement",
        meta: {
          order: 1,
          icon: "lucide:mail",
          title: "routes.notification.channels",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/notification/channel/index.vue"),
      },

      {
        // 路由规则（事件 → 渠道 + 同步/异步）：投递时的唯一真相，删一行即停一条路由
        path: "rules",
        name: "NotificationRuleManagement",
        meta: {
          order: 2,
          icon: "lucide:git-branch",
          title: "routes.notification.rules",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/notification/rule/index.vue"),
      },

      {
        // 投递台账：每次投递一行，是规则 + 渠道配置跑出来的结果，故排在两者之后
        path: "deliveries",
        name: "NotificationDeliveryManagement",
        meta: {
          order: 3,
          icon: "lucide:send",
          title: "routes.notification.deliveries",
          authority: ["sys:platform_admin"],
        },
        component: () => import("@/pages/app/notification/delivery/index.vue"),
      },
    ],
  },
];

export default notification;
