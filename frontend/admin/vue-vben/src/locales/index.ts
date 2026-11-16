import type { LocaleSetupOptions, SupportedLanguagesType } from '@vben/locales';
import type { Locale } from 'ant-design-vue/es/locale';

import type { App } from 'vue';
import { ref } from 'vue';

import {
  $t,
  $te,
  setupI18n as coreSetup,
  loadLocalesMapFromDir,
} from '@vben/locales';
import { preferences } from '@vben/preferences';

import antdEnLocale from 'ant-design-vue/es/locale/en_US';
import antdDefaultLocale from 'ant-design-vue/es/locale/zh_CN';
import dayjs from 'dayjs';

const antdLocale = ref<Locale>(antdDefaultLocale);

const modules = import.meta.glob('./langs/**/*.json');

const localesMap = loadLocalesMapFromDir(
  /\.\/langs\/([^/]+)\/(.*)\.json$/,
  modules,
);

/**
 * loadLocalesMapFromDir 把「路径段（含 /）」当单个顶层键（如 "menu/system"），
 * 拆分到子目录的语言包（menu/<组>.json、page/<模块>.json）需还原成嵌套树并深合并，
 * $t('menu.system.*')、$t('page.<模块>.*') 才能解析。新增模块只加文件，不改共享大 JSON。
 */
function denormalizeSlashKeys(
  messages: Record<string, any>,
): Record<string, any> {
  const mergeValue = (target: Record<string, any>, key: string, value: any) => {
    const existing = target[key];
    if (
      existing &&
      typeof existing === 'object' &&
      value &&
      typeof value === 'object'
    ) {
      for (const [k, v] of Object.entries(value)) mergeValue(existing, k, v);
    } else {
      target[key] = value;
    }
  };
  const tree: Record<string, any> = {};
  for (const [key, value] of Object.entries(messages)) {
    const segs = key.split('/');
    let node = tree;
    for (let i = 0; i < segs.length - 1; i++) {
      const seg = segs[i]!;
      if (!(node[seg] instanceof Object)) node[seg] = {};
      node = node[seg];
    }
    mergeValue(node, segs[segs.length - 1]!, value);
  }
  return tree;
}

/**
 * 加载应用特有的语言包
 * 这里也可以改造为从服务端获取翻译数据
 * @param lang
 */
async function loadMessages(lang: SupportedLanguagesType) {
  const [appLocaleMessages] = await Promise.all([
    localesMap[lang]?.(),
    loadThirdPartyMessage(lang),
  ]);
  return denormalizeSlashKeys((appLocaleMessages?.default ?? {}) as Record<
    string,
    any
  >);
}

/**
 * 加载第三方组件库的语言包
 * @param lang
 */
async function loadThirdPartyMessage(lang: SupportedLanguagesType) {
  await Promise.all([loadAntdLocale(lang), loadDayjsLocale(lang)]);
}

/**
 * 加载dayjs的语言包
 * @param lang
 */
async function loadDayjsLocale(lang: SupportedLanguagesType) {
  let locale;
  switch (lang) {
    case 'en-US': {
      locale = await import('dayjs/locale/en');
      break;
    }
    case 'zh-CN': {
      locale = await import('dayjs/locale/zh-cn');
      break;
    }
    // 默认使用英语
    default: {
      locale = await import('dayjs/locale/en');
    }
  }
  if (locale) {
    dayjs.locale(locale);
  } else {
    console.error(`Failed to load dayjs locale for ${lang}`);
  }
}

/**
 * 加载antd的语言包
 * @param lang
 */
async function loadAntdLocale(lang: SupportedLanguagesType) {
  switch (lang) {
    case 'en-US': {
      antdLocale.value = antdEnLocale;
      break;
    }
    case 'zh-CN': {
      antdLocale.value = antdDefaultLocale;
      break;
    }
  }
}

async function setupI18n(app: App, options: LocaleSetupOptions = {}) {
  await coreSetup(app, {
    defaultLocale: preferences.app.locale,
    loadMessages,
    missingWarn: !import.meta.env.PROD,
    ...options,
  });
}

export { $t, $te, antdLocale, setupI18n };
