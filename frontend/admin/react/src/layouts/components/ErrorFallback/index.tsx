import { Button, Result, Typography, Space, Alert, message } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useEffect, useState } from 'react';
import { CopyOutlined, BugOutlined, ReloadOutlined, HomeOutlined } from '@ant-design/icons';
import { useI18n } from '@/core/i18n';
import { reportError } from './utils/report';

const { Text, Paragraph } = Typography;

export interface ErrorFallbackProps {
  /** 捕获的错误对象 */
  error: Error;
  /** 重置错误边界的回调（重试按钮调用） */
  resetErrorBoundary?: () => void;
  /** 组件堆栈信息（部分 ErrorBoundary 库提供） */
  componentStack?: string;
  /** 是否显示详细错误信息（默认开发环境显示） */
  showDetails?: boolean;
}

/**
 * 企业级错误兜底组件
 * 功能：防白屏崩溃 + 开发环境调试 + 生产环境监控 + 用户友好交互
 */
export const ErrorFallback = ({
  error,
  resetErrorBoundary,
  componentStack,
  showDetails = import.meta.env.DEV,
}: ErrorFallbackProps) => {
  const navigate = useNavigate();
  const [copied, setCopied] = useState(false);
  const { t } = useI18n('common');

  // 生产环境自动上报错误（收集端点由 .env 的 VITE_ERROR_REPORT_ENDPOINT 配置，见 utils/report.ts）
  useEffect(() => {
    if (!import.meta.env.DEV) {
      console.error('[ErrorFallback] Caught error:', error);
      reportError(error, componentStack);
    }
  }, [error, componentStack]);

  const handleCopy = async () => {
    const errorInfo = `${error.message}\n${error.stack || ''}\n${componentStack || ''}`;
    try {
      await navigator.clipboard.writeText(errorInfo);
      setCopied(true);
      message.success(t('fallback.errorFallback.copySuccess'));
      setTimeout(() => setCopied(false), 2000);
    } catch {
      message.error(t('fallback.errorFallback.copyFailed'));
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-6 bg-[var(--ant-color-bg-layout)]">
      <div className="w-full max-w-3xl">
        <Result
          status="error"
          title={
            <div className="flex items-center gap-2">
              <BugOutlined className="text-[var(--ant-color-error)] text-xl" />
              <span>{t('fallback.errorFallback.title')}</span>
            </div>
          }
          subTitle={t('fallback.errorFallback.subTitle')}
          extra={
            <Space wrap>
              {resetErrorBoundary && (
                <Button type="primary" icon={<ReloadOutlined />} onClick={resetErrorBoundary}>
                  {t('fallback.errorFallback.retry')}
                </Button>
              )}
              <Button icon={<HomeOutlined />} onClick={() => navigate('/')}>
                {t('fallback.errorFallback.backToHome')}
              </Button>
              <Button onClick={() => window.location.reload()}>
                {t('fallback.errorFallback.refreshPage')}
              </Button>
            </Space>
          }
        >
          {showDetails && (
            <div className="mt-6 text-left">
              <Alert
                type="error"
                message={t('fallback.errorFallback.errorDetails')}
                showIcon
                action={
                  <Button size="small" icon={<CopyOutlined />} onClick={handleCopy}>
                    {copied ? t('fallback.errorFallback.copied') : t('fallback.errorFallback.copy')}
                  </Button>
                }
                className="mb-3"
              />
              <div className="bg-[var(--ant-color-bg-container)] border border-[var(--ant-color-border)] rounded-lg p-4 max-h-80 overflow-auto">
                <Paragraph strong className="mb-2">
                  {t('fallback.errorFallback.errorMessage')}
                </Paragraph>
                <Text code type="danger" className="block mb-4 break-all">
                  {error.toString()}
                </Text>

                {error.stack && (
                  <>
                    <Paragraph strong className="mb-2">
                      {t('fallback.errorFallback.callStack')}
                    </Paragraph>
                    <pre className="text-xs font-mono text-[var(--ant-color-text-secondary)] whitespace-pre-wrap break-all">
                      {error.stack}
                    </pre>
                  </>
                )}

                {componentStack && (
                  <>
                    <Paragraph strong className="mb-2 mt-4">
                      {t('fallback.errorFallback.componentStack')}
                    </Paragraph>
                    <pre className="text-xs font-mono text-[var(--ant-color-text-secondary)] whitespace-pre-wrap break-all">
                      {componentStack}
                    </pre>
                  </>
                )}
              </div>
            </div>
          )}
        </Result>
      </div>
    </div>
  );
};

export default ErrorFallback;
