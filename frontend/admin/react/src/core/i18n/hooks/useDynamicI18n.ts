import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { loadDynamicLocale, isLocaleLoaded } from '../utils/dynamicLoader';

export interface UseDynamicI18nOptions {
  /** 模块标识（必需） */
  namespace: string;

  /** 是否自动加载（默认 true） */
  autoLoad?: boolean;

  /** 加载依赖（变化时重新加载） */
  dependencies?: any[];
}

/**
 * 动态 i18n Hook（本地 JSON 懒加载）
 * 用法：const { t, loading } = useDynamicI18n({ namespace: 'preferences' });
 */
export const useDynamicI18n = (options: UseDynamicI18nOptions) => {
  const { namespace, autoLoad = true, dependencies = [] } = options;

  const { i18n, t: staticT } = useTranslation();
  const [loading, setLoading] = useState(autoLoad);
  const [loaded, setLoaded] = useState(false);

  // 加载动态文案
  const load = useCallback(async () => {
    const lang = i18n.language || i18n.options.lng || 'zh-CN';

    // 检查是否已加载
    if (isLocaleLoaded(lang, namespace)) {
      setLoaded(true);
      setLoading(false);
      return true;
    }

    setLoading(true);
    try {
      await loadDynamicLocale(lang, namespace);
      setLoaded(true);
      return true;
    } catch (error) {
      console.error(`[i18n] Failed to load module ${namespace}:`, error);
      return false;
    } finally {
      setLoading(false);
    }
  }, [namespace, i18n.language, i18n.options.lng]);

  // 自动加载
  useEffect(() => {
    if (autoLoad) {
      load();
    }
  }, [autoLoad, load, ...dependencies]);

  // 语言变更时重新加载
  useEffect(() => {
    const handleLangChange = () => {
      if (loaded) {
        setLoaded(false);
        load();
      }
    };

    i18n.on('languageChanged', handleLangChange);
    return () => i18n.off('languageChanged', handleLangChange);
  }, [i18n, load, loaded]);

  // 合并 t 函数：优先动态文案，降级到静态
  const t = useCallback(
    (key: string, options?: any): string => {
      // 尝试动态命名空间
      const dynamicKey = `${namespace}:${key}`;
      const dynamicValue = staticT(dynamicKey, { ...options, defaultValue: undefined });

      // 如果动态文案存在且不是 key 本身，返回动态值
      if (dynamicValue && dynamicValue !== dynamicKey) {
        return String(dynamicValue);
      }

      // 降级：使用静态文案
      return String(staticT(key, options));
    },
    [namespace, staticT],
  );

  return {
    t,
    loading,
    loaded,
    reload: load,
  };
};
