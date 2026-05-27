import { useMemo, useState, useCallback, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { PageContainer as ProPageContainer } from '@ant-design/pro-components';
import { Skeleton, Alert, Button, Space } from 'antd';
import { ReloadOutlined, FullscreenOutlined, FullscreenExitOutlined } from '@ant-design/icons';

import { usePageRefreshStore } from '@/stores/pageRefresh';
import { useAccess } from '@/core/access';

import { useI18n } from '@/core/i18n';
import { usePreferencesStore } from '@/core/preferences/store';

import { useBreadcrumb } from './hooks/useBreadcrumb';
import { usePageTitle } from './hooks/usePageTitle';
import { checkPagePermission } from './utils/permission';
import type { PageContainerProps } from './types';
import './PageContainer.style.css'; // 引入样式文件

/**
 * 企业级页面容器组件
 * 功能：自动面包屑 + 动态标题 + 权限控制 + 加载状态 + 内容区域增强
 */
export const PageContainer = ({
  // ProPageContainer 原生属性（透传）
  ghost,
  header,
  footer,
  extra,
  children,

  // 自定义属性
  title: manualTitle,
  breadcrumb: manualBreadcrumb,
  route,
  permission,
  forbiddenFallback,
  loading,
  loadingContent,
  content,
  contentPadding = true,
  contentClassName,
  keepAlive,
  pageKey: customPageKey,
  showRefresh,
  onRefresh,
  showFullscreen,
  render,
  ...restProps
}: PageContainerProps) => {
  const location = useLocation();

  // 用户权限（通过 useAccess 获取合并后的角色码 + 权限码）
  const { getAllPermissions } = useAccess();
  const userPermissions = useMemo(() => getAllPermissions(), [getAllPermissions]);
  const { t } = useI18n('common');

  // 全屏状态
  const [isFullscreen, setIsFullscreen] = useState(false);

  // 刷新加载状态
  const [refreshing, setRefreshing] = useState(false);

  // 页面刷新管理
  const { setRefreshCallback, removeRefreshCallback } = usePageRefreshStore();

  // 面包屑偏好设置
  const breadcrumbPreferences = usePreferencesStore((state) => state.preferences.breadcrumb);
  const breadcrumbStyleType = breadcrumbPreferences?.styleType ?? 'normal';

  // 生成页面唯一 key
  const pageKey = useMemo(() => {
    return customPageKey || location.pathname;
  }, [customPageKey, location.pathname]);

  // 注册/移除刷新回调
  useEffect(() => {
    if (onRefresh) {
      setRefreshCallback(pageKey, onRefresh);
    }

    return () => {
      removeRefreshCallback(pageKey);
    };
  }, [pageKey, onRefresh, setRefreshCallback, removeRefreshCallback]);

  // 处理刷新（用于 PageContainer 内部的刷新按钮）
  const handleRefresh = useCallback(async () => {
    if (!onRefresh) return;

    setRefreshing(true);
    try {
      await onRefresh();
    } finally {
      setRefreshing(false);
    }
  }, [onRefresh]);

  // 计算面包屑
  const breadcrumb = useBreadcrumb({
    manual: manualBreadcrumb === false ? false : undefined,
    route,
    showHomeIcon: breadcrumbPreferences?.showHome ?? true,
    showIcon: breadcrumbPreferences?.showIcon ?? true,
  });

  // 计算标题
  const pageTitle = usePageTitle({
    manual: manualTitle,
    routeTitle: route?.meta?.title,
    defaultTitle: t('pageContainer.defaultTitle'),
    updateDocumentTitle: true,
  });

  // 权限检查
  const hasPermission = useMemo(() => {
    return checkPagePermission(permission, userPermissions);
  }, [permission, userPermissions]);

  // 切换全屏
  const toggleFullscreen = useCallback(() => {
    setIsFullscreen((prev) => !prev);
  }, []);

  // 全屏时隐藏面包屑和标题
  const effectiveBreadcrumb = isFullscreen ? false : breadcrumb;
  const effectiveHeader = isFullscreen ? false : header;

  // 渲染面包屑项
  const renderBreadcrumbItems = useCallback((items: any[], styleType: string) => {
    return items.map((item) => ({
      title: (
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 4,
            ...(styleType === 'background'
              ? {
                  padding: '4px 8px',
                  borderRadius: 4,
                  backgroundColor: '#ffffff',
                  border: '1px solid #e5e7eb',
                }
              : {}),
          }}
        >
          {item.icon}
          <span>{item.breadcrumbName}</span>
        </span>
      ),
      onClick: item.onClick,
    }));
  }, []);

  // 面包屑配置
  const breadcrumbConfig = useMemo(() => {
    if (effectiveBreadcrumb === false) return undefined;
    return {
      items: renderBreadcrumbItems(effectiveBreadcrumb, breadcrumbStyleType),
    };
  }, [effectiveBreadcrumb, breadcrumbStyleType, renderBreadcrumbItems]);

  // 操作按钮
  const actionButtons = useMemo(() => {
    const buttons = [];

    if (showRefresh) {
      buttons.push(
        <Button
          key="refresh"
          type="text"
          icon={<ReloadOutlined spin={refreshing} />}
          onClick={handleRefresh}
          loading={refreshing}
        />,
      );
    }

    if (showFullscreen) {
      buttons.push(
        <Button
          key="fullscreen"
          type="text"
          icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
          onClick={toggleFullscreen}
        />,
      );
    }

    return buttons.length > 0 ? buttons : undefined;
  }, [showRefresh, showFullscreen, refreshing, isFullscreen, handleRefresh, toggleFullscreen]);

  // 全屏样式
  const fullscreenStyles = isFullscreen
    ? {
        position: 'fixed' as const,
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        zIndex: 9999,
        background: 'inherit',
        overflow: 'auto' as const,
      }
    : undefined;

  const renderForbidden = useMemo(() => {
    if (forbiddenFallback) return forbiddenFallback;

    return (
      <Alert
        type="warning"
        title={t('pageContainer.forbiddenTitle')}
        description={t('pageContainer.forbiddenDesc')}
        showIcon
        className="mt-4"
      />
    );
  }, [forbiddenFallback]);

  // 渲染加载状态
  const renderLoading = useMemo(() => {
    if (loadingContent) return loadingContent;

    return <Skeleton active paragraph={{ rows: 6 }} className="mt-4" />;
  }, [loadingContent]);

  // 自定义渲染（完全接管）
  if (render) {
    return (
      <>
        {render({
          title: pageTitle,
          breadcrumb,
          hasPermission,
          loading: loading || false,
        })}
      </>
    );
  }

  // 无权限：显示 fallback
  if (!hasPermission) {
    return (
      <ProPageContainer
        ghost={ghost}
        header={effectiveHeader === false ? undefined : (effectiveHeader as any)}
        title={effectiveHeader === false ? undefined : pageTitle}
        breadcrumb={breadcrumbConfig}
        extra={actionButtons ? <Space>{actionButtons}</Space> : extra}
        style={fullscreenStyles}
        {...restProps}
      >
        {renderForbidden}
      </ProPageContainer>
    );
  }

  // 加载中：显示骨架屏
  if (loading) {
    return (
      <ProPageContainer
        ghost={ghost}
        header={effectiveHeader === false ? undefined : (effectiveHeader as any)}
        title={effectiveHeader === false ? undefined : pageTitle}
        breadcrumb={breadcrumbConfig}
        extra={actionButtons ? <Space>{actionButtons}</Space> : extra}
        style={fullscreenStyles}
        {...restProps}
      >
        {renderLoading}
      </ProPageContainer>
    );
  }

  // 正常渲染
  return (
    <ProPageContainer
      className="page-container-root"
      ghost={ghost}
      header={effectiveHeader === false ? undefined : (effectiveHeader as any)}
      title={effectiveHeader === false ? undefined : pageTitle}
      breadcrumb={breadcrumbConfig}
      extra={actionButtons ? <Space>{actionButtons}</Space> : extra}
      footer={footer}
      style={{
        ...fullscreenStyles,
        position: 'relative',
        height: '100%',
      }}
      {...restProps}
    >
      {/*
       * 用绝对定位绕过 ProPageContainer 内部的未知 flex 层级
       * 这样不管 ProPageContainer 内部包了多少层 div，都不影响
       */}
      <div
        style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        {content ?? children}
      </div>
    </ProPageContainer>
  );
};

export default PageContainer;
