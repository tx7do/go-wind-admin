import { useMemo, useState, useCallback, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { PageContainer as ProPageContainer } from '@ant-design/pro-components';
import { Skeleton, Alert, Button, Space } from 'antd';
import { ReloadOutlined, FullscreenOutlined, FullscreenExitOutlined } from '@ant-design/icons';
import { useUserStore } from '@/stores/user';
import { useI18n } from '@/core/i18n';
import { usePageRefreshStore } from '@/stores/pageRefresh';
import { useBreadcrumb } from './hooks/useBreadcrumb';
import { usePageTitle } from './hooks/usePageTitle';
import { checkPagePermission } from './utils/permission';
import type { PageContainerProps } from './types';

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
  
  // 用户权限（从 userInfo 中获取 permissions）
  const { userInfo } = useUserStore();
  const userPermissions = userInfo?.permissions || [];
  const { t } = useI18n('common');

  // 全屏状态
  const [isFullscreen, setIsFullscreen] = useState(false);

  // 刷新加载状态
  const [refreshing, setRefreshing] = useState(false);

  // 页面刷新管理
  const { setRefreshCallback, removeRefreshCallback } = usePageRefreshStore();

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

  // 切换全屏
  const toggleFullscreen = useCallback(() => {
    setIsFullscreen(prev => !prev);
  }, []);

  // 计算面包屑
  const breadcrumb = useBreadcrumb({
    manual: manualBreadcrumb === false ? false : undefined,
    route,
    showHomeIcon: true,
  });
  
  // 调试：查看面包屑数据
  console.log('PageContainer - breadcrumb:', breadcrumb);

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
        />
      );
    }

    if (showFullscreen) {
      buttons.push(
        <Button
          key="fullscreen"
          type="text"
          icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
          onClick={toggleFullscreen}
        />
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

  // 全屏时隐藏面包屑和标题
  const effectiveBreadcrumb = isFullscreen ? false : breadcrumb;
  const effectiveHeader = isFullscreen ? false : header;
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
        breadcrumb={
          effectiveBreadcrumb === false
            ? undefined
            : {
                items: effectiveBreadcrumb.map((item) => ({
                  title: (
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                      {item.icon}
                      <span>{item.breadcrumbName}</span>
                    </span>
                  ),
                  onClick: item.onClick,
                })),
              }
        }
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
        breadcrumb={
          effectiveBreadcrumb === false
            ? undefined
            : {
                items: effectiveBreadcrumb.map((item) => ({
                  title: (
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                      {item.icon}
                      <span>{item.breadcrumbName}</span>
                    </span>
                  ),
                  onClick: item.onClick,
                })),
              }
        }
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
      ghost={ghost}
      header={effectiveHeader === false ? undefined : (effectiveHeader as any)}
      title={effectiveHeader === false ? undefined : pageTitle}
      breadcrumb={
        effectiveBreadcrumb === false
          ? undefined
          : {
              // 使用 items 而不是 routes（Ant Design v6）
              items: effectiveBreadcrumb.map((item) => ({
                title: (
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                    {item.icon}
                    <span>{item.breadcrumbName}</span>
                  </span>
                ),
                onClick: item.onClick,
              })),
            }
      }
      extra={actionButtons ? <Space>{actionButtons}</Space> : extra}
      footer={footer}
      style={fullscreenStyles}
      {...restProps}
    >
      {/* 内容区域增强 */}
      <div
        className={`
          ${isFullscreen ? '' : contentPadding ? 'p-4 md:p-6' : ''}
          ${contentClassName || ''}
        `.trim()}
        data-page-key={pageKey}
        data-keep-alive={keepAlive || undefined}
      >
        {/* 优先使用 content，否则用 children */}
        {content ?? children}
      </div>
    </ProPageContainer>
  );
};

export default PageContainer;
