/**
 * 菜单搜索逻辑
 */
import { ref, onMounted, onBeforeUnmount, toRaw } from "vue";
import { RouteRecordRaw, LocationQueryRaw } from "vue-router";
import { router } from "@/router";
import { useAccessStore } from '@go-wind/admin-element';
import { isExternal } from "@/utils";
import { i18n } from "@go-wind/admin-element";
import { fetchSemanticSearch } from "@/api/composables/ai-content";

const t = i18n.global.t;

/** 搜索项类型 */
interface SearchItem {
  title: string;
  path: string;
  name?: string;
  icon?: string;
  redirect?: string;
  params?: LocationQueryRaw;
  /** 语义搜索命中项（区别于本地菜单关键字匹配，UI 上打 AI 标） */
  semantic?: boolean;
}

const STORAGE_KEY = "menu_search_history";
const MAX_HISTORY = 5;

export function useCommandPalette() {
  const accessStore = useAccessStore();

  // 状态
  const visible = ref(false);
  const keyword = ref("");
  const activeIndex = ref(-1);
  const inputRef = ref<HTMLInputElement>();
  const menuItems = ref<SearchItem[]>([]);
  const results = ref<SearchItem[]>([]);
  const history = ref<SearchItem[]>([]);

  // 排除的路由
  const excludedPaths = ["/login", "/401", "/404"];

  // ============================================
  // 弹窗控制
  // ============================================

  function open() {
    // 每次打开都重建菜单索引：标题在索引期翻译（loadRoutes 存的是 key，
    // 不重建则换语言后仍是旧文案；路由后到时也能补上）
    menuItems.value = [];
    loadRoutes(accessStore.accessRoutes);
    keyword.value = "";
    results.value = [];
    semanticResults.value = [];
    activeIndex.value = -1;
    visible.value = true;
    setTimeout(() => inputRef.value?.focus(), 100);
  }

  function close() {
    visible.value = false;
  }

  // ============================================
  // 搜索逻辑
  // ============================================

  function onSearch() {
    activeIndex.value = -1;
    if (!keyword.value.trim()) {
      results.value = [];
      semanticResults.value = [];
      clearTimeout(semanticTimer);
      return;
    }
    const kw = keyword.value.toLowerCase();
    // 与 react 端同口径：标题或路径 contains（key 已在索引期翻译为文案）
    results.value = menuItems.value.filter(
      (item) =>
        item.title.toLowerCase().includes(kw) || item.path.toLowerCase().includes(kw)
    );
    queueSemanticSearch(keyword.value.trim());
  }

  // ── 语义搜索（pgvector，500ms 防抖；尽力而为，失败只清空不打断本地搜索）──
  let semanticTimer: ReturnType<typeof setTimeout> | undefined;
  const semanticResults = ref<SearchItem[]>([]);
  const semanticLoading = ref(false);

  function queueSemanticSearch(query: string) {
    clearTimeout(semanticTimer);
    semanticTimer = setTimeout(async () => {
      semanticLoading.value = true;
      try {
        const resp = await fetchSemanticSearch(query, 8);
        semanticResults.value = (resp.items ?? [])
          .filter((item) => item.route)
          .map((item) => ({
            title: item.title ?? item.route!,
            path: item.route!,
            semantic: true,
          }));
      } catch (error) {
        // 不吞错：带出原始错误对象；语义搜索降级为仅本地结果
        console.error("semantic search failed:", error);
        semanticResults.value = [];
      } finally {
        semanticLoading.value = false;
      }
    }, 500);
  }

  function getDisplayList() {
    // 键盘选择列表必须与模板展示列表一致（关键词态=本地+语义合并，空关键词=历史）：
    // 此前只回本地结果或历史，语义命中的行无法用键盘选中，本地无结果时回车甚至会跳去历史项。
    if (keyword.value.trim()) {
      return [...results.value, ...semanticResults.value];
    }
    return history.value;
  }

  function onSelect() {
    const list = getDisplayList();
    if (list.length === 0) return;
    if (activeIndex.value < 0) return;
    const item = list[activeIndex.value];
    if (!item) return;
    onGo(item);
  }

  function onNavigate(direction: "up" | "down") {
    const list = getDisplayList();
    if (list.length === 0) return;

    if (direction === "up") {
      activeIndex.value = activeIndex.value <= 0 ? list.length - 1 : activeIndex.value - 1;
    } else {
      activeIndex.value = activeIndex.value >= list.length - 1 ? 0 : activeIndex.value + 1;
    }
  }

  function onGo(item: SearchItem) {
    close();
    addHistory(item);

    if (isExternal(item.path)) {
      window.open(item.path, "_blank");
    } else {
      router.push({ path: item.path, query: item.params });
    }
  }

  // ============================================
  // 历史记录
  // ============================================

  function loadHistory() {
    try {
      const data = localStorage.getItem(STORAGE_KEY);
      history.value = data ? JSON.parse(data) : [];
    } catch {
      history.value = [];
    }
  }

  function saveHistory() {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(history.value));
  }

  function addHistory(item: SearchItem) {
    // 去重
    const idx = history.value.findIndex((i) => i.path === item.path);
    if (idx !== -1) history.value.splice(idx, 1);

    // 添加到开头
    history.value.unshift(item);

    // 限制数量
    if (history.value.length > MAX_HISTORY) {
      history.value = history.value.slice(0, MAX_HISTORY);
    }

    saveHistory();
  }

  function removeHistory(index: number) {
    history.value.splice(index, 1);
    saveHistory();
  }

  function clearHistory() {
    history.value = [];
    localStorage.removeItem(STORAGE_KEY);
  }

  // ============================================
  // 路由解析
  // ============================================

  function loadRoutes(routes: RouteRecordRaw[], parentPath = "") {
    routes.forEach((route) => {
      const path = route.path.startsWith("/")
        ? route.path
        : `${parentPath}${parentPath.endsWith("/") ? "" : "/"}${route.path}`;

      if (excludedPaths.includes(route.path) || isExternal(route.path)) return;

      if (route.children) {
        loadRoutes(route.children, path);
      } else if (route.meta?.title && typeof route.meta.title === "string") {
        menuItems.value.push({
          // meta.title 是 i18n key（如 routes.tenant.member），索引期必须翻译，
          // 否则展示裸 key 且中文永远匹配不上；t() 对不存在的 key 原样返回
          title:
            route.meta.title === "dashboard"
              ? t("common.text.home")
              : t(route.meta.title),
          path,
          name: typeof route.name === "string" ? route.name : undefined,
          icon: typeof route.meta.icon === "string" ? route.meta.icon : undefined,
          redirect: typeof route.redirect === "string" ? route.redirect : undefined,
          // RouteMeta 的字段是 query（非 params），此前读 route.meta.params 恒为 undefined，
          // 导致路由预设的 meta.query 在 router.push 时丢失。
          params: route.meta.query
            ? JSON.parse(JSON.stringify(toRaw(route.meta.query)))
            : undefined,
        });
      }
    });
  }

  // ============================================
  // 快捷键
  // ============================================

  function handleKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      open();
    }
  }

  // ============================================
  // 生命周期
  // ============================================

  onMounted(() => {
    loadRoutes(accessStore.accessRoutes);
    loadHistory();
    document.addEventListener("keydown", handleKeydown);
  });

  onBeforeUnmount(() => {
    document.removeEventListener("keydown", handleKeydown);
    clearTimeout(semanticTimer);
  });

  return {
    visible,
    keyword,
    results,
    semanticResults,
    semanticLoading,
    history,
    activeIndex,
    inputRef,
    open,
    close,
    onSearch,
    onSelect,
    onNavigate,
    onGo,
    removeHistory,
    clearHistory,
  };
}
