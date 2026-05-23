import { useMemo } from 'react';
import React from 'react';
import { useMatches, useNavigate } from 'react-router-dom';
import * as Icons from '@ant-design/icons';

import { useI18n } from '@/core/i18n';
import type { BreadcrumbItem } from '../types';

interface UseBreadcrumbOptions {
  /** 手动传入的面包屑（优先级最高） */
  manual?: BreadcrumbItem[] | false;

  /** 当前路由对象（用于获取 meta 信息） */
  route?: any;

  /** 是否显示首页图标 */
  showHomeIcon?: boolean;
}

export const useBreadcrumb = ({
  manual,
  route,
  showHomeIcon = true,
}: UseBreadcrumbOptions): BreadcrumbItem[] | false => {
  const navigate = useNavigate();
  const matches = useMatches();
  const { t } = useI18n('common');

  // 手动传入优先
  if (manual === false) return false;
  if (manual?.length) return manual;

  // 自动计算：基于 react-router-dom useMatches

  return useMemo(() => {
    // 类型定义：为 handle 扩展类型
    type MatchWithHandle = {
      pathname: string;
      params: { [key: string]: string | undefined };
      pathnameBase: string;
      handle?: {
        title?: string;
        icon?: string;
        [key: string]: any;
      };
      [key: string]: any;
    };

    const typedMatches = matches as MatchWithHandle[];
    
    console.log('useBreadcrumb - matches:', typedMatches);
    console.log('useBreadcrumb - handles:', typedMatches.map(m => ({ path: m.pathname, handle: m.handle })));

    // 过滤出有标题的匹配项
    const items = typedMatches
      .filter((match: MatchWithHandle) => match.handle?.title || match.pathname === '/')
      .map((match: MatchWithHandle, index: number, arr: MatchWithHandle[]) => {
        const isLast = index === arr.length - 1;
        const title = match.handle?.title || route?.meta?.title || t('pageContainer.defaultTitle');
        
        console.log(`useBreadcrumb - match ${index}:`, { path: match.pathname, handle: match.handle, title });
        
        // 将图标字符串转换为 React 组件
        let icon: React.ReactNode = undefined;
        if (match.handle?.icon) {
          const IconComponent = (Icons as any)[match.handle.icon];
          if (IconComponent) {
            icon = React.createElement(IconComponent);
            console.log(`useBreadcrumb - icon found for ${match.pathname}:`, match.handle.icon);
          } else {
            console.warn(`useBreadcrumb - icon component not found:`, match.handle.icon);
          }
        } else {
          console.log(`useBreadcrumb - no icon for ${match.pathname}`);
        }

        return {
          path: match.pathname,
          breadcrumbName: title,
          icon, // 添加图标
          // 最后一项不可点击（当前页）
          onClick: !isLast && match.pathname ? () => navigate(match.pathname) : undefined,
        };
      });

    // 确保首页存在
    if (showHomeIcon && items.length && items[0].path !== '/') {
      items.unshift({
        path: '/',
        breadcrumbName: t('home'),
        icon: React.createElement(Icons.HomeOutlined), // 首页图标
        onClick: () => navigate('/'),
      });
    }
    
    console.log('useBreadcrumb - final items:', items);

    return items;
  }, [matches, route?.meta?.title, navigate, showHomeIcon]);
};
