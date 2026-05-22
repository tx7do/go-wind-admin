import { initI18n } from '@/core/i18n';
import { usePreferencesStore } from '@/core/preferences';

/**
 * 应用启动初始化
 */
export async function bootstrap() {
  await _initI18n();

  // 可放全局初始化逻辑
  console.log('✅ 应用启动初始化完成');
}

async function _initI18n() {
  // 从 preferences 获取初始语言
  const initialLocale = usePreferencesStore.getState().preferences.app.locale;

  // 初始化 i18n（传入初始语言）
  const i18nInstance = await initI18n(initialLocale);

  // 调试：打印 i18n 资源
  console.log(' i18n 初始化完成');
  console.log('📚 当前语言:', i18nInstance.language);
  console.log('📖 可用的命名空间:', i18nInstance.options.ns);
  // console.log('📦 common 命名空间资源:', i18nInstance.getResource('zh-CN', 'common'))
  // console.log(' auth 命名空间资源:', i18nInstance.getResource('zh-CN', 'auth'))
}
