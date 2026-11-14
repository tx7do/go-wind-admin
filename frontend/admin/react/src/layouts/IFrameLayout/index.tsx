import { useState, useCallback, useEffect, useRef, useMemo } from 'react';
import { Spin, Alert } from 'antd';
import { useLocation } from 'react-router-dom';

import { usePreferences } from '@go-wind/admin-react';
import { useI18n } from '@go-wind/admin-react';

import './iframe-layout.less';

/**
 * 校验 iframe src 是否安全。
 * 允许：同源相对路径（以 / 开头且不以 // 开头），或 https 绝对 URL。
 * 拒绝：http、//、javascript:、data:、及其他任意 scheme（防 iframe 注入/钓鱼）。
 * 返回安全后的 src，不安全返回空串。
 */
function sanitizeIframeSrc(raw: string | undefined | null): string {
  if (!raw) return '';
  const trimmed = raw.trim();
  // 同源相对路径
  if (trimmed.startsWith('/') && !trimmed.startsWith('//')) {
    return trimmed;
  }
  // 仅允许 https 绝对 URL
  try {
    const u = new URL(trimmed);
    if (u.protocol === 'https:') return trimmed;
  } catch {
    // 非合法 URL
  }
  return '';
}

/**
 * Iframe 布局：嵌入外部系统（如报表、文档等）
 * 支持：自动高度、加载状态、跨域通信
 */
const IFrameLayout = () => {
  const location = useLocation();
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { isDark } = usePreferences();
  const { t } = useI18n('common');

  // 从路由 state 或 query 获取 iframe URL（经安全校验）
  const iframeSrc = useMemo(() => {
    const candidate =
      (location.state?.url as string | undefined) ??
      new URLSearchParams(location.search).get('url') ??
      '';
    // 无论是路由 state 还是 query，都需校验，避免注入任意 url（如 ?url=https://evil.com）
    return sanitizeIframeSrc(candidate);
  }, [location.state, location.search]);

  // 计算 iframe src 的 origin，用于校验 postMessage 来源/目标
  const iframeOrigin = useMemo(() => {
    if (!iframeSrc) return '';
    try {
      // 同源相对路径时 origin 即当前页 origin
      if (iframeSrc.startsWith('/')) return window.location.origin;
      return new URL(iframeSrc).origin;
    } catch {
      return '';
    }
  }, [iframeSrc]);

  // iframe 加载完成
  const handleLoad = useCallback(() => {
    setLoading(false);
    setError(null);
  }, []);

  // iframe 加载失败
  const handleError = useCallback(() => {
    setLoading(false);
    setError(t('iframe.loadErrorDesc'));
  }, [t]);

  // 跨域通信：监听 iframe 发来的消息
  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      // 校验来源 origin：只接受当前 iframe src 对应 origin 的消息，
      // 防止任意页面（包括其他标签页/弹窗）发 {type:'ready'} 隐藏 loading
      // 或 {type:'resize'} 篡改高度。
      if (!iframeOrigin || event.origin !== iframeOrigin) return;

      const { type, data } = event.data || {};

      switch (type) {
        case 'resize':
          // iframe 内部通知高度变化，实现自适应
          if (iframeRef.current && data?.height) {
            iframeRef.current.style.height = `${data.height}px`;
          }
          break;
        case 'route-change':
          // iframe 内部路由变化，可同步到主应用面包屑等
          console.log('[IFrame] route change:', data);
          break;
        case 'ready':
          // iframe 内部应用初始化完成
          setLoading(false);
          break;
        default:
          break;
      }
    };

    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, [iframeOrigin]);

  // 向 iframe 发送消息（跨域通信）
  const postMessageToIframe = useCallback(
    (type: string, data?: any) => {
      if (!iframeRef.current?.contentWindow || !iframeOrigin) return;
      // targetOrigin 指定具体 origin 而非 '*'，避免把主题模式等信息泄露给任意监听者
      iframeRef.current.contentWindow.postMessage({ type, data }, iframeOrigin);
    },
    [iframeOrigin],
  );

  // 主题切换时通知 iframe
  useEffect(() => {
    if (!loading) {
      postMessageToIframe('theme-change', { mode: isDark ? 'dark' : 'light' });
    }
  }, [isDark, loading, postMessageToIframe]);

  // 无 URL 时显示提示
  if (!iframeSrc) {
    return (
      <div className="iframe-layout-empty">
        <Alert
          type="warning"
          title={t('iframe.noUrlTitle')}
          description={t('iframe.noUrlDesc')}
          showIcon
        />
      </div>
    );
  }

  return (
    <div className="iframe-layout-wrapper">
      {loading && (
        <div className="iframe-layout-loading">
          <Spin size="large" description={t('iframe.loading')} />
        </div>
      )}

      {error && (
        <div className="iframe-layout-error">
          <Alert type="error" title={t('iframe.loadErrorTitle')} description={error} showIcon />
        </div>
      )}

      <iframe
        ref={iframeRef}
        src={iframeSrc}
        className="iframe-layout-content"
        onLoad={handleLoad}
        onError={handleError}
        title={t('iframe.embeddedPage')}
        sandbox="allow-scripts allow-same-origin allow-popups allow-forms allow-downloads"
        allow="clipboard-read; clipboard-write"
        style={{ visibility: loading ? 'hidden' : 'visible' }}
      />
    </div>
  );
};

export default IFrameLayout;
