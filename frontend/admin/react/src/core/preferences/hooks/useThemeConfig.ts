import {useMemo, useState, useEffect} from 'react';
import {theme as antdTheme, type ThemeConfig, type MappingAlgorithm} from 'antd';

import {darkThemeComponents, darkThemeTokens} from '../config';
import {usePreferencesStore} from '../store';

export const useThemeConfig = (): ThemeConfig => {
    const {theme, app} = usePreferencesStore((state) => state.preferences);

    // 响应式跟踪系统暗色模式（仅 auto 模式需要）
    const [systemIsDark, setSystemIsDark] = useState(
        () => window.matchMedia('(prefers-color-scheme: dark)').matches,
    );

    useEffect(() => {
        const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
        const handler = (e: MediaQueryListEvent) => setSystemIsDark(e.matches);
        mediaQuery.addEventListener('change', handler);
        return () => mediaQuery.removeEventListener('change', handler);
    }, []);

    // 计算有效主题：auto 模式下跟随系统偏好
    const effectiveIsDark = useMemo(() => {
        if (theme.mode === 'dark') return true;
        if (theme.mode === 'light') return false;
        if (theme.mode === 'auto') return systemIsDark;
        return false;
    }, [theme.mode, systemIsDark]);

    return useMemo((): ThemeConfig => {
        const algorithms: MappingAlgorithm[] = [];
        if (effectiveIsDark) algorithms.push(antdTheme.darkAlgorithm);
        if (app.compact) algorithms.push(antdTheme.compactAlgorithm);

        const tokens: ThemeConfig['token'] = {
            colorPrimary: theme.colorPrimary,
            colorSuccess: theme.colorSuccess,
            colorWarning: theme.colorWarning,
            colorError: theme.colorDestructive,
            borderRadius: Number.parseInt(theme.radius) || 6,
        };
        // 与全局 ThemeProvider 保持同一份暗色色阶，避免内层 ConfigProvider 覆盖外层
        if (effectiveIsDark) Object.assign(tokens, darkThemeTokens);

        return {
            algorithm: algorithms.length > 0 ? algorithms : antdTheme.defaultAlgorithm,
            token: tokens,
            components: effectiveIsDark ? darkThemeComponents : undefined,
        };
    }, [effectiveIsDark, theme, app]);
};
