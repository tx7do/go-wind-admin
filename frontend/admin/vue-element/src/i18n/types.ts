/**
 * 支持的语言类型
 * 与应用使用的语言值保持一致
 */
export type SupportedLanguagesType = "zh-cn" | "en";

/**
 * 语言文件目录映射
 * 将应用语言值映射到实际的语言文件目录名
 */
export const LANGUAGE_FILE_MAP: Record<SupportedLanguagesType, string> = {
  "zh-cn": "zh-CN",
  en: "en-US",
};

export type LoadMessageFn = (
  lang: SupportedLanguagesType
) => Promise<Record<string, string> | undefined>;

export interface LocaleSetupOptions {
  /**
   * Default language
   * @default zh-CN
   */
  defaultLocale?: SupportedLanguagesType;
  /**
   * Load message function
   * @param lang
   * @returns
   */
  loadMessages?: LoadMessageFn;
  /**
   * Whether to warn when the key is not found
   */
  missingWarn?: boolean;
}

export type ImportLocaleFn = () => Promise<{ default: Record<string, string> }>;
