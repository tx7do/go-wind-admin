import {useModel} from '@umijs/max';

/**
 * 全局配置 Model（组合模式）
 * 整合 theme、language、loading 等核心状态
 * 方便一次性获取所有全局配置
 */
export default function GlobalModel() {
  const themeModel = useModel('core.theme');
  const languageModel = useModel('core.language');
  const loadingModel = useModel('core.loading');

  // 主题切换快捷方法
  const toggleTheme = () => {
    themeModel.setMode(themeModel.mode === 'light' ? 'dark' : 'light');
  };

  return {
    // theme
    theme: themeModel.mode,
    setTheme: themeModel.setMode,
    toggleTheme,

    // language
    lang: languageModel.locale,
    setLang: languageModel.setLocale,

    // loading
    loading: loadingModel.isLoading,
    setLoading: (isLoading: boolean) => {
      isLoading ? loadingModel.startLoading() : loadingModel.finishLoading();
    },
    startLoading: loadingModel.startLoading,
    finishLoading: loadingModel.finishLoading,
    loadingError: loadingModel.loadingError,
  };
}
