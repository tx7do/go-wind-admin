import {zhCN, type ZhCNResources, allNamespaces as zhCNNamespaces} from './zh-CN';
import {enUS, type EnUSResources} from './en-US';

// 所有命名空间列表（供 i18next ns 配置）
const allNamespaces = zhCNNamespaces;

// 资源映射（供 i18next 初始化，核心 + 扩展全部预加载）
export const resources = {
    'zh-CN': zhCN,
    'en-US': enUS,
} as const;

// 支持的语言
export const supportedLocales = ['zh-CN', 'en-US'] as const;
export type SupportedLocale = typeof supportedLocales[number];

// 核心资源类型联合（用于 t() 类型推导）
export type CoreResources = ZhCNResources | EnUSResources;
export type CoreNamespace = keyof CoreResources;
export type CoreKey<N extends CoreNamespace> = keyof CoreResources[N];

// 类型安全 t() 函数（仅核心命名空间）
export type CoreTFunction = <N extends CoreNamespace>(
    namespace: N,
    key: CoreKey<N>,
    params?: Record<string, any>
) => string;

// 导出命名空间列表供 i18n 初始化使用
export {allNamespaces};
