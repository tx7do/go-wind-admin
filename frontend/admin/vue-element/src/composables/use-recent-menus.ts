import { ref } from "vue";
import { i18n } from "@/core/i18n";

const t = i18n.global.t;

/**
 * 最近访问菜单项
 */
export interface RecentMenuItem {
  path: string;
  title: string;
  icon?: string;
  visitedAt: number;
}

const STORAGE_KEY = "recent_menus";
const MAX_COUNT = 8;

// 全局状态
const recentMenus = ref<RecentMenuItem[]>([]);

/**
 * 从 localStorage 加载数据
 */
function loadFromStorage(): RecentMenuItem[] {
  try {
    const data = localStorage.getItem(STORAGE_KEY);
    return data ? JSON.parse(data) : [];
  } catch {
    return [];
  }
}

/**
 * 保存到 localStorage
 */
function saveToStorage(menus: RecentMenuItem[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(menus));
}

// 初始化
recentMenus.value = loadFromStorage();

/**
 * 最近访问菜单 composable
 */
export function useRecentMenus() {
  /**
   * 清空所有记录
   */
  function clearRecentMenus() {
    recentMenus.value = [];
    localStorage.removeItem(STORAGE_KEY);
  }

  /**
   * 格式化访问时间
   */
  function formatVisitTime(timestamp: number): string {
    const now = Date.now();
    const diff = now - timestamp;

    if (diff < 60000) return t("common.relativeTime.justNow");
    if (diff < 3600000) return t("common.relativeTime.minutesAgo", { n: Math.floor(diff / 60000) });
    if (diff < 86400000) return t("common.relativeTime.hoursAgo", { n: Math.floor(diff / 3600000) });
    if (diff < 604800000) return t("common.relativeTime.daysAgo", { n: Math.floor(diff / 86400000) });

    const date = new Date(timestamp);
    return `${date.getMonth() + 1}-${date.getDate()}`;
  }

  return {
    recentMenus,
    clearRecentMenus,
    formatVisitTime,
  };
}

/**
 * 添加最近访问记录（全局方法，供路由守卫调用）
 */
export function addRecentMenu(path: string, title: string, icon?: string) {
  if (!path || !title) return;

  // 过滤掉不需要记录的路径
  const excludePaths = ["/dashboard", "/404", "/401", "/login", "/"];
  if (excludePaths.some((p) => path === p || path.startsWith(p + "/"))) return;

  // 移除已存在的相同路径
  const filtered = recentMenus.value.filter((item) => item.path !== path);

  // 添加到开头
  const newItem: RecentMenuItem = {
    path,
    title,
    icon,
    visitedAt: Date.now(),
  };

  recentMenus.value = [newItem, ...filtered].slice(0, MAX_COUNT);
  saveToStorage(recentMenus.value);
}
