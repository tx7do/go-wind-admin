import { useEffect } from 'react';
import { matchRoutes } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import { usePreferencesStore } from '@go-wind/admin-react';
import type { AppRouteObject } from '@go-wind/admin-react';

import { translateRouteTitle } from '@/layouts/components/PageContainer/hooks/usePageTitle';

/** react-router 数据路由对象的最小结构面（仅用到订阅与当前 location） */
interface DataRouterLike {
  subscribe: (listener: () => void) => () => void;
  state?: { location?: { pathname?: string } };
}

/**
 * 全局浏览器标签标题：与 vue-element/vben 的路由守卫同型——按当前路由
 * meta.title 翻译并拼 ' - 应用名'。react 此前只有 PageContainer 内的
 * usePageTitle 会写标题，登录/错误页等不走 PageContainer 的路由标题一直是
 * 裸应用名（如登录页只有「GoWind Admin」，另两端是「登录 - GoWind Admin」）。
 *
 * 业务页 PageContainer 仍会写入：manual 标题时其 effect 挂载晚于本更新器、
 * 最终以它为准；非 manual 时两者同源（route.meta.title），值一致。
 * 匹配用静态 allRoutes：标题是静态配置、与权限过滤无关，且后端 accessMode
 * 下业务路由在静态 businessRoutes 镜像里有同路径同标题的副本。
 */
export function useDocumentTitle(
  router: DataRouterLike | null,
  routes: AppRouteObject[] | null,
) {
  const dynamicTitle = usePreferencesStore(
    (state) => state.preferences.app.dynamicTitle,
  );
  const appName = usePreferencesStore((state) => state.preferences.app.name);
  const { t: tMenu } = useTranslation('routes');

  useEffect(() => {
    if (!router || !routes || !dynamicTitle) return;

    const updateTitle = () => {
      const pathname = router.state?.location?.pathname;
      if (!pathname) return;
      // meta 是本仓路由对象的自定义字段，react-router 类型不含它——定向 any
      const matches = matchRoutes(routes as any, pathname) ?? [];
      const titleMeta = [...matches]
        .reverse()
        .find((match) => typeof match.route?.meta?.title === 'string')?.route
        ?.meta?.title as string | undefined;
      if (titleMeta) {
        document.title = `${translateRouteTitle(titleMeta, tMenu)} - ${
          appName || import.meta.env.VITE_APP_TITLE || ''
        }`;
      } else {
        // 无标题路由兜底为裸应用名（对齐 vue-element 守卫的无标题分支）
        document.title = appName || import.meta.env.VITE_APP_TITLE || '';
      }
    };

    updateTitle();
    return router.subscribe(updateTitle);
  }, [router, routes, dynamicTitle, appName, tMenu]);
}
