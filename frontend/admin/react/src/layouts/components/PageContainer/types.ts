import type { ReactNode } from 'react';
import type { PageContainerProps as ProPageContainerProps } from '@ant-design/pro-components';

import type { AppRoute } from '@/core/router/types';

export interface PageContainerProps extends Omit<
  ProPageContainerProps,
  'title' | 'breadcrumb' | 'header'
> {
  // 核心配置
  /** 页面标题（优先级：传入 > route.meta.title > 默认） */
  title?: ReactNode;

  /** 面包屑配置（传入则覆盖自动计算） */
  breadcrumb?: ProPageContainerProps['breadcrumb'] | false;

  /** 页面头部配置（传入 false 则隐藏头部） */
  header?: ProPageContainerProps['header'] | false;

  /** 路由对象（用于自动计算标题/面包屑/权限） */
  route?: Partial<AppRoute> | null;

  // 权限控制
  /** 所需权限码（满足任一即可） */
  permission?: string | string[];

  /** 无权限时的 fallback 内容 */
  forbiddenFallback?: ReactNode;

  // 加载状态
  /** 是否显示加载骨架屏 */
  loading?: boolean;

  /** 加载时显示的内容（优先级高于 skeleton） */
  loadingContent?: ReactNode;

  // 内容区域增强
  /** 页面内容（替代 children，语义更清晰） */
  content?: ReactNode;

  /** 是否启用内容区域的内边距（默认 true） */
  contentPadding?: boolean;

  /** 内容区域自定义类名 */
  contentClassName?: string;

  // 高级功能
  /** 是否启用页面缓存标记（配合 react-activation） */
  keepAlive?: boolean;

  /** 页面唯一标识（用于缓存/埋点） */
  pageKey?: string;

  // 操作功能
  /** 是否显示刷新按钮 */
  showRefresh?: boolean;

  /** 刷新回调函数 */
  onRefresh?: () => void;

  /** 是否显示全屏按钮 */
  showFullscreen?: boolean;

  /** 自定义渲染函数（完全接管渲染） */
  render?: (props: {
    title: ReactNode;
    breadcrumb: BreadcrumbItem[] | false | undefined;
    hasPermission: boolean;
    loading: boolean;
  }) => ReactNode;
}

export interface BreadcrumbItem {
  path?: string;
  breadcrumbName: string;
  icon?: ReactNode;
  onClick?: () => void;
}
