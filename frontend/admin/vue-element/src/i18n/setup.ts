import { createI18n, Locale } from "vue-i18n";
import { App } from "vue";

import { LoadMessageFn, LocaleSetupOptions } from "@/i18n/types";
import { loadLocalesMap, loadLocalesMapFromDir } from "@/i18n/utils";

const i18n = createI18n({
  globalInjection: true,
  legacy: false,
  locale: "",
  messages: {},
});

const modules = import.meta.glob("./langs/**/*.json");

const localesMap = loadLocalesMapFromDir(/\.\/langs\/([^/]+)\/(.*)\.json$/, modules);
let loadMessages: LoadMessageFn;

/**
 * Set i18n language
 * @param locale
 */
function setI18nLanguage(locale: Locale) {
  i18n.global.locale.value = locale;

  document?.querySelector("html")?.setAttribute("lang", locale);
}

async function setupI18n(app: App, options: LocaleSetupOptions = {}) {
  const { defaultLocale = "zh-CN" } = options;
  // app可以自行扩展一些第三方库和组件库的国际化
  loadMessages = options.loadMessages || (async () => ({}));
  app.use(i18n);
  await loadLocaleMessages(defaultLocale);

  // 在控制台打印警告
  i18n.global.setMissingHandler((locale, key) => {
    if (options.missingWarn && key.includes(".")) {
      console.warn(`[intlify] Not found '${key}' key in '${locale}' locale messages.`);
    }
  });
}

async function loadLocaleMessages(lang: SupportedLanguagesType) {
  // 将应用语言值转换为语言文件目录名（zh-CN -> zh-CN, en-US -> en-US）
  const langDir = lang;

  if (unref(i18n.global.locale) === lang) {
    return setI18nLanguage(lang);
  }

  const message = await localesMap[langDir]?.();

  if (message?.default) {
    // 设置语言包（先清空再设置，确保完全替换）
    i18n.global.setLocaleMessage(lang, message.default);
  } else {
    console.error(`[intlify] Failed to load '${lang}' locale messages.`);
  }

  const mergeMessage = await loadMessages(lang);
  if (mergeMessage && Object.keys(mergeMessage).length > 0) {
    i18n.global.mergeLocaleMessage(lang, mergeMessage);
  }

  return setI18nLanguage(lang);
}

/**
 * 翻译路由标题
 * 用于面包屑、侧边栏、标签页等场景
 * @param title 翻译键（如 "routes.dashboard.title"）
 */
function translateRouteTitle(title: string): string {
  if (!title) return "";
  // 直接尝试翻译，如果不存在则返回原文
  return i18n.global.te(title) ? i18n.global.t(title) : title;
}

const $t = i18n.global.t;
const $te = i18n.global.te;

export {
  $t,
  $te,
  i18n,
  loadLocaleMessages,
  loadLocalesMap,
  loadLocalesMapFromDir,
  setupI18n,
  translateRouteTitle,
};

export { useI18n } from "vue-i18n";

export type { Locale } from "vue-i18n";
