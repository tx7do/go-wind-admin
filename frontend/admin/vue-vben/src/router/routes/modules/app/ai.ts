import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

/**
 * AI 域路由：对话助手（登录可用）+ 模型提供商管理（平台管理员）。
 * 与 react/ele 基准同构：providers 页挂 sys:platform_admin，chat 页不加 authority。
 */
const ai: RouteRecordRaw[] = [
  {
    path: '/ai',
    name: 'AiAssistant',
    component: BasicLayout,
    redirect: '/ai/chat',
    meta: {
      order: 2007,
      icon: 'lucide:sparkles',
      title: $t('menu.ai.moduleName'),
      keepAlive: true,
    },
    children: [
      {
        path: 'chat',
        name: 'AiChat',
        meta: {
          order: 1,
          icon: 'lucide:message-circle',
          title: $t('menu.ai.chat'),
          keepAlive: true,
        },
        component: () => import('#/views/app/ai/chat/index.vue'),
      },
      {
        path: 'knowledge',
        name: 'AiKnowledgeManagement',
        meta: {
          order: 3,
          icon: 'lucide:book-open',
          title: $t('menu.ai.knowledge'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/ai/knowledge/index.vue'),
      },
      {
        path: 'query',
        name: 'AiQuery',
        meta: {
          order: 2,
          icon: 'lucide:search-code',
          title: $t('menu.ai.query'),
        },
        component: () => import('#/views/app/ai/query/index.vue'),
      },
      {
        path: 'usage',
        name: 'AiUsage',
        meta: {
          order: 4,
          icon: 'lucide:bar-chart-3',
          title: $t('menu.ai.usage'),
        },
        component: () => import('#/views/app/ai/usage/index.vue'),
      },
      {
        path: 'providers',
        name: 'AiProviderManagement',
        meta: {
          order: 2,
          icon: 'lucide:cpu',
          title: $t('menu.ai.providers'),
          authority: ['sys:platform_admin'],
        },
        component: () => import('#/views/app/ai/provider/index.vue'),
      },
    ],
  },
];

export default ai;
