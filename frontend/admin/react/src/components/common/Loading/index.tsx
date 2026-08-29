import {useMemo} from 'react';
import {ConfigProvider, Spin, Progress, type ThemeConfig} from 'antd';
import {LoadingOutlined} from '@ant-design/icons';
import {usePreferencesStore, useThemeConfig} from '@/core/preferences';
import {useI18n} from '@/core/i18n';

export interface ThemeLoadingProps {
    text?: string;
    subText?: string;
    fullScreen?: boolean;
    className?: string;
    style?: React.CSSProperties;

    /** 进度值 0-100 */
    progress?: number;

    /** 加载状态：'loading' | 'success' | 'error' */
    status?: 'loading' | 'success' | 'error';

    /** 自动隐藏（当 progress=100 时） */
    autoHide?: boolean;

    /** 自定义主题覆盖 */
    themeOverride?: Partial<ReturnType<typeof useThemeConfig>>;
}

/**
 * 主题化 Loading 组件
 * 自动同步应用主题配置（颜色、圆角、暗黑模式等）
 */
export const ThemeLoading = ({
                                 text,
                                 subText,
                                 fullScreen = false,
                                 className = '',
                                 style,
                                 progress,
                                 status = 'loading',
                                 autoHide = true,
                                 themeOverride,
                             }: ThemeLoadingProps) => {
    const {theme: prefTheme} = usePreferencesStore(state => state.preferences);
    const defaultTheme = useThemeConfig();
    const {t} = useI18n('common');

    // 合并主题配置
    const themeConfig = useMemo((): ThemeConfig => {
        return {
            ...defaultTheme,
            token: {
                ...defaultTheme.token,
                ...themeOverride?.token,
            },
        };
    }, [defaultTheme, themeOverride]);

    // 自动隐藏逻辑
    if (autoHide && progress === 100 && status === 'loading') {
        return null;
    }

    // 状态对应的图标/颜色
    const statusConfig = useMemo(() => {
        switch (status) {
            case 'success':
                return {
                    icon: <span className="text-green-500 text-4xl">✓</span>,
                    color: themeConfig.token?.colorSuccess,
                    text: text || t('loading.success'),
                };
            case 'error':
                return {
                    icon: <span className="text-red-500 text-4xl">✕</span>,
                    color: themeConfig.token?.colorError,
                    text: text || t('loading.failed'),
                };
            default:
                return {
                    icon: (
                        <LoadingOutlined
                            style={{fontSize: 40, color: themeConfig.token?.colorPrimary}}
                            spin
                        />
                    ),
                    color: themeConfig.token?.colorPrimary,
                    text: text || t('loading.text'),
                };
        }
    }, [status, themeConfig.token, text, t]);

    const content = (
        <ConfigProvider theme={themeConfig}>
            <div className="flex flex-col items-center justify-center gap-4 animate-fade-in">
                {/* 状态图标 */}
                <div className="transition-transform duration-300 hover:scale-105">
                    {status === 'loading' ? (
                        <Spin indicator={statusConfig.icon} size="large"/>
                    ) : (
                        <div className="w-10 h-10 flex items-center justify-center">
                            {statusConfig.icon}
                        </div>
                    )}
                </div>

                {/* 文字区域 */}
                <div className="text-center space-y-1">
                    <div
                        className="font-medium"
                        style={{color: statusConfig.color}}
                    >
                        {statusConfig.text}
                    </div>
                    {subText && (
                        <div className="text-gray-400 text-xs opacity-80">
                            {subText}
                        </div>
                    )}
                </div>

                {/* 进度环（仅 loading 状态） */}
                {status === 'loading' && typeof progress === 'number' && progress < 100 && (
                    <Progress
                        type="circle"
                        percent={progress}
                        size={72}
                        strokeColor={statusConfig.color}
                        trailColor="var(--ant-color-fill-tertiary)"
                        format={() => `${Math.round(progress)}%`}
                        className="mt-2"
                    />
                )}
            </div>
        </ConfigProvider>
    );

    const baseClasses = `
    flex items-center justify-center
    ${fullScreen
        ? 'fixed inset-0 z-[1000] bg-[var(--ant-color-bg-container)]/95 backdrop-blur-md'
        : 'py-6'
    }
    ${className}
  `.trim();

    return (
        <div
            className={baseClasses}
            style={{
                ...style,
                // 暗黑模式适配
                backgroundColor: prefTheme.mode === 'dark'
                    ? 'rgba(0, 0, 0, 0.85)'
                    : undefined,
            }}
            role="status"
            aria-live="polite"
            aria-label={statusConfig.text}
        >
            {content}
        </div>
    );
};

export default ThemeLoading;
