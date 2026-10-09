import type {
  ComponentRecordType,
  GenerateMenuAndRoutesOptions,
  RouteRecordStringComponent,
} from '@vben/types';

import { generateAccessible } from '@vben/access';
import { preferences } from '@vben/preferences';

import { message } from 'ant-design-vue';

import { fetchNavigation } from '#/api';
import { BasicLayout, IFrameView } from '#/layouts';
import { $t } from '#/locales';

/**
 * 从后端获取路由列表
 * 兼容两种响应格式:
 * 1. 直接返回 { items: [...] }
 * 2. 包装在 data 字段中 { data: { items: [...] } }
 */
async function getAllMenusApi(): Promise<RouteRecordStringComponent[]> {
  const data = (await fetchNavigation()) ?? [];
  const unwrapped = (data as any)?.data ?? data;
  const items = unwrapped?.items ?? [];
  return items as RouteRecordStringComponent[];
}

const forbiddenComponent = () => import('#/views/_core/fallback/forbidden.vue');

async function generateAccess(options: GenerateMenuAndRoutesOptions) {
  const pageMap: ComponentRecordType = import.meta.glob('../views/**/*.vue');

  const layoutMap: ComponentRecordType = {
    BasicLayout,
    IFrameView,
  };

  // 后端模式: 先尝试从后端获取路由，如果为空或失败则降级到前端模式
  let effectiveMode = preferences.app.accessMode;
  let cachedBackendRoutes: RouteRecordStringComponent[] | null = null;

  if (effectiveMode === 'backend') {
    message.loading({
      content: `${$t('common.loadingMenu')}...`,
      duration: 1.5,
    });
    try {
      cachedBackendRoutes = await getAllMenusApi();
      if (cachedBackendRoutes.length === 0) {
        effectiveMode = 'frontend';
      }
    } catch {
      effectiveMode = 'frontend';
    }
  }

  const result = await generateAccessible(effectiveMode, {
    ...options,
    fetchMenuListAsync: async () => {
      // 使用缓存的预检结果，避免重复调用 API
      if (cachedBackendRoutes !== null) {
        return cachedBackendRoutes;
      }
      try {
        return await getAllMenusApi();
      } catch {
        return [];
      }
    },
    // 可以指定没有权限跳转403页面
    forbiddenComponent,
    // 如果 route.meta.menuVisibleWithForbidden = true
    layoutMap,
    pageMap,
  });

  return result;
}

export { generateAccess };
