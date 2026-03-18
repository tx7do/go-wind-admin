/**
 * 主题初始化脚本
 * 在 HTML 加载时立即执行，防止主题闪烁
 */
export default function initThemeScript() {
    const namespace = 'app'; // 默认值，与 caches/index.ts 一致
    const fullKey = `${namespace}-theme-mode`;

    // 检查是否有存储的主题偏好（StorageManager 存储的是 JSON 格式）
    let storedMode: string | null = null;
    try {
        const itemStr = localStorage.getItem(fullKey);
        if (itemStr) {
            try {
                const item = JSON.parse(itemStr);
                storedMode = item.value || item; // 兼容 JSON 格式和纯字符串格式
            } catch {
                storedMode = itemStr; // 如果是纯字符串，直接使用
            }
        }
    } catch (error) {
        console.warn('无法访问 localStorage:', error);
    }

    let themeToUse: string;

    if (storedMode && storedMode !== 'system') {
        // 如果有明确的主题偏好（light/dark），直接使用
        themeToUse = storedMode;
        console.log('[Init Theme] Using stored mode:', storedMode);
    } else if (storedMode === 'system') {
        // 如果用户选择了跟随系统，根据系统主题设置
        const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        themeToUse = isDark ? 'dark' : 'light';
        console.log('[Init Theme] Following system theme:', themeToUse);
    } else {
        // 如果没有存储，根据系统主题设置
        const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        themeToUse = isDark ? 'dark' : 'light';
        console.log('[Init Theme] Using default system theme:', themeToUse);
    }

    // 应用主题到 HTML
    document.documentElement.classList.add(themeToUse);
    console.log('[Init Theme] Applied theme class:', themeToUse);
}
