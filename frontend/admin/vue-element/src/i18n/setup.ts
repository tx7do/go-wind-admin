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
  const { defaultLocale = "zh-cn" } = options;
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
  // 将应用语言值转换为语言文件目录名（zh-cn -> zh-CN, en-US -> en-US）
  const langDir = lang === "zh-cn" ? "zh-CN" : lang;

  if (unref(i18n.global.locale) === lang) {
    return setI18nLanguage(lang);
  }

  const message = await localesMap[langDir]?.();

  if (message?.default) {
    // 调试：打印加载的消息结构
    console.log("[i18n] Loaded messages for", lang, ":", Object.keys(message.default));
    console.log(
      "[i18n] routes keys:",
      message.default.routes ? Object.keys(message.default.routes) : "not found"
    );
    // 合并所有 JSON 文件的翻译内容
    i18n.global.mergeLocaleMessage(lang, message.default);
    // 验证合并后的结果
    console.log(
      "[i18n] After merge, te('routes.dashboard.title'):",
      i18n.global.te("routes.dashboard.title")
    );
  }

  const mergeMessage = await loadMessages(lang);
  i18n.global.mergeLocaleMessage(lang, mergeMessage);

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

export {
  i18n,
  loadLocaleMessages,
  loadLocalesMap,
  loadLocalesMapFromDir,
  setupI18n,
  translateRouteTitle,
};
