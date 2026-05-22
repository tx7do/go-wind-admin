import i18n from 'i18next';
import type { SupportedLocale } from '@/locales';

/**
 * 动态加载本地 JSON 文案（Code Splitting）
 * @param lang 语言代码
 * @param namespace 命名空间标识
 */
export const loadDynamicLocale = async (
  lang: string,
  namespace: string,
): Promise<boolean> => {
  try {
    // 使用相对路径，从 src/core/i18n/utils/ 到 src/locales/
    // 需要向上3级：utils -> i18n -> core -> src
    const modulePath = `../../../locales/${lang}/_modules/${namespace}.json`;
    const translations = await import(/* @vite-ignore */ modulePath);

    // 合并到 i18next
    i18n.addResourceBundle(
      lang,
      namespace,
      translations.default || translations,
      true,
      true,
    );

    console.debug(`[i18n] Loaded dynamic locale: ${namespace}@${lang}`);
    return true;
  } catch (error) {
    console.warn(`[i18n] Failed to load ${namespace}@${lang}:`, error);

    // 尝试加载 fallback（如 zh-CN）
    if (lang !== 'zh-CN') {
      return loadDynamicLocale('zh-CN', namespace);
    }

    return false;
  }
};

/**
 * 批量加载多个模块
 */
export const loadDynamicLocales = async (
  lang: string,
  namespaces: string[],
): Promise<void> => {
  await Promise.all(namespaces.map((ns) => loadDynamicLocale(lang, ns)));
};

/**
 * 检查模块是否已加载
 */
export const isLocaleLoaded = (lang: string, namespace: string): boolean => {
  return !!i18n.getResourceBundle(lang, namespace);
};
