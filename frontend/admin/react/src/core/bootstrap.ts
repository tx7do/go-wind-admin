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
  await initI18n(initialLocale);
}
