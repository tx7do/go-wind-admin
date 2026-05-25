import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { resources, allNamespaces, type SupportedLocale } from '@/locales';

export const initI18n = async (initialLang: SupportedLocale) => {
  await i18n.use(initReactI18next).init({
    lng: initialLang,
    resources, // 核心 + 扩展模块全部预加载
    fallbackLng: 'zh-CN',
    supportedLngs: ['zh-CN', 'en-US'],

    defaultNS: 'common',
    ns: allNamespaces, // 声明所有命名空间

    // 缺失 key 处理（开发环境）
    missingKeyHandler: import.meta.env.DEV
      ? (lngs, ns, key) => {
          console.warn(`[i18n] Missing: "${key}" in "${ns}" for "${lngs[0]}"`);
        }
      : undefined,
  });

  return i18n;
};
