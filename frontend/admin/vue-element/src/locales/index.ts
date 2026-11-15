/**
 * 语言包映射组装（i18n 资源收集已从包内 glob 移到宿主侧——包 lib 构建看不见宿主目录）。
 * 语言级深合并：包 core.json（框架文案）与宿主多命名空间文件各加载后按语言合并，
 * 同名命名空间宿主覆盖包内置。
 */
import { loadLocalesMapFromDir, coreLocaleModules } from "@go-wind/admin-element";

// 宿主业务文案：语言目录/命名空间.json
const modules = import.meta.glob("./**/*.json");
const hostLocalesMap = loadLocalesMapFromDir(/\.\/([^/]+)\/(.*)\.json$/, modules);

/** 传给 setupI18n({ localesMap }) 的合并映射 */
export const localesMap: Record<string, () => Promise<{ default: unknown }>> = Object.fromEntries(
  [...new Set([...Object.keys(coreLocaleModules), ...Object.keys(hostLocalesMap)])].map((lang) => [
    lang,
    async () => {
      const core = (await coreLocaleModules[lang]?.()) as { default?: unknown } | undefined;
      const host = (await hostLocalesMap[lang]?.()) as { default?: unknown } | undefined;
      return {
        default: {
          ...((core?.default as Record<string, unknown>) ?? {}),
          ...((host?.default as Record<string, unknown>) ?? {}),
        },
      };
    },
  ]),
);
