<template>
  <div class="tabs-bar" :style="{ height: tabHeight + 'px' }">
    <!-- 左侧滚动按钮 -->
    <span
      v-show="showScrollButton"
      :class="{
        'is-disabled': scrollIsAtLeft,
      }"
      class="tabs-bar__scroll-btn"
      @click="scrollDirection('left')"
    >
      <ElIcon :size="14"><ArrowLeft /></ElIcon>
    </span>

    <!-- 标签内容区域 -->
    <div
      ref="scrollbarRef"
      class="tabs-bar__content"
      :class="{
        'tabs-bar__content--chrome': styleType === 'chrome',
      }"
      @wheel.prevent="handleWheel"
    >
      <div ref="scrollInnerRef" class="tabs-bar__inner">
        <TransitionGroup name="tab-slide">
          <div
            v-for="(tab, i) in visitedViews"
            :key="tab.fullPath"
            :class="[
              'tabs-bar__item',
              `tabs-bar__item--${styleType}`,
              {
                'is-active': isActive(tab),
                'is-affix': tab.affix,
                'is-draggable': draggable && !tab.affix,
              },
            ]"
            @click="handleTabClick(tab)"
            @click.middle.prevent="handleMiddleClick(tab)"
            @contextmenu.prevent="(e: MouseEvent) => openContextMenu(tab, e)"
          >
            <!-- Chrome 风格的 SVG 背景 -->
            <template v-if="styleType === 'chrome'">
              <div v-if="i !== 0 && !isActive(tab)" class="tabs-bar__divider" />
              <div class="tabs-bar__chrome-bg">
                <div class="tabs-bar__chrome-bg__content" />
                <svg class="tabs-bar__chrome-bg__before" width="7" height="7">
                  <path d="M 0 7 A 7 7 0 0 0 7 0 L 7 7 Z" />
                </svg>
                <svg class="tabs-bar__chrome-bg__after" width="7" height="7">
                  <path d="M 0 0 A 7 7 0 0 0 7 7 L 0 7 Z" />
                </svg>
              </div>
            </template>

            <!-- 标签内容 -->
            <div class="tabs-bar__item-main">
              <TabIcon v-if="showIcon && tab.icon" :icon="tab.icon" class="tabs-bar__icon" />
              <span class="tabs-bar__title">{{ translateRouteTitle(tab.title) }}</span>
            </div>

            <!-- 关闭/固定按钮 -->
            <div class="tabs-bar__extra">
              <ElIcon
                v-if="!tab.affix && visitedViews.length > 1"
                :size="12"
                class="tabs-bar__close"
                @click.stop="closeSelectedTag(tab)"
              >
                <Close />
              </ElIcon>
              <ElIcon v-if="tab.affix && visitedViews.length > 1" :size="12" class="tabs-bar__pin">
                <svg viewBox="0 0 24 24" fill="currentColor" width="1em" height="1em">
                  <path d="M16 12V4h1V2H7v2h1v8l-2 2v2h5.2v6h1.6v-6H18v-2l-2-2z" />
                </svg>
              </ElIcon>
            </div>
          </div>
        </TransitionGroup>
      </div>
    </div>

    <!-- 右侧滚动按钮 -->
    <span
      v-show="showScrollButton"
      :class="{
        'is-disabled': scrollIsAtRight,
      }"
      class="tabs-bar__scroll-btn"
      @click="scrollDirection('right')"
    >
      <ElIcon :size="14"><ArrowRight /></ElIcon>
    </span>

    <!-- 工具按钮 -->
    <div class="tabs-bar__tools">
      <ElTooltip v-if="showMaximize" :content="isMaximized ? '还原' : '最大化'" placement="bottom">
        <ElIcon class="tabs-bar__tool-btn" @click="toggleMaximize">
          <FullScreen v-if="!isMaximized" />
          <ScaleToOriginal v-else />
        </ElIcon>
      </ElTooltip>
      <ElDropdown v-if="showMore" trigger="click" @command="handleCommand">
        <ElIcon class="tabs-bar__tool-btn">
          <ArrowDown />
        </ElIcon>
        <template #dropdown>
          <ElDropdownMenu>
            <ElDropdownItem command="reload" :icon="RefreshRight">
              {{ t("preferences.tabbar.contextMenu.reload") || "重新加载" }}
            </ElDropdownItem>
            <ElDropdownItem
              command="close"
              :icon="Close"
              :disabled="!activeTag || !!activeTag.affix"
            >
              {{ t("preferences.tabbar.contextMenu.close") || "关闭" }}
            </ElDropdownItem>
            <ElDropdownItem command="pin">
              <ElIcon>
                <svg viewBox="0 0 24 24" fill="currentColor" width="1em" height="1em">
                  <path d="M16 12V4h1V2H7v2h1v8l-2 2v2h5.2v6h1.6v-6H18v-2l-2-2z" />
                </svg>
              </ElIcon>
              {{
                activeTag?.affix
                  ? t("preferences.tabbar.contextMenu.unpin") || "取消固定"
                  : t("preferences.tabbar.contextMenu.pin") || "固定"
              }}
            </ElDropdownItem>
            <ElDropdownItem command="maximize" :icon="FullScreen">
              {{
                isMaximized
                  ? t("preferences.tabbar.contextMenu.restoreMaximize") || "退出最大化"
                  : t("preferences.tabbar.contextMenu.maximize") || "最大化"
              }}
            </ElDropdownItem>
            <ElDropdownItem command="openInNewWindow" :icon="Promotion">
              {{ t("preferences.tabbar.contextMenu.openInNewWindow") || "在新窗口打开" }}
            </ElDropdownItem>
            <ElDropdownItem
              command="closeLeft"
              divided
              :icon="DArrowLeft"
              :disabled="isFirstViewOf(activeTag)"
            >
              {{ t("preferences.tabbar.contextMenu.closeLeft") || "关闭左侧标签页" }}
            </ElDropdownItem>
            <ElDropdownItem
              command="closeRight"
              :icon="DArrowRight"
              :disabled="isLastViewOf(activeTag)"
            >
              {{ t("preferences.tabbar.contextMenu.closeRight") || "关闭右侧标签页" }}
            </ElDropdownItem>
            <ElDropdownItem
              command="closeOther"
              divided
              :icon="SemiSelect"
              :disabled="closableCount <= 0"
            >
              {{ t("preferences.tabbar.contextMenu.closeOther") || "关闭其他标签页" }}
            </ElDropdownItem>
            <ElDropdownItem command="closeAll" :icon="CircleClose" :disabled="closableCount <= 0">
              {{ t("preferences.tabbar.contextMenu.closeAll") || "关闭全部标签页" }}
            </ElDropdownItem>
          </ElDropdownMenu>
        </template>
      </ElDropdown>
    </div>

    <!-- 右键菜单 -->
    <Teleport to="body">
      <ul
        v-show="contextMenu.visible"
        class="tabs-bar__contextmenu"
        :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
      >
        <li @click="refreshSelectedTag(contextMenuTag)">
          <ElIcon :size="14"><RefreshRight /></ElIcon>
          {{ t("preferences.tabbar.contextMenu.reload") || "重新加载" }}
        </li>
        <li
          :class="{ 'is-disabled': !contextMenuTag || !!contextMenuTag?.affix }"
          @click="closeSelectedTag(contextMenuTag)"
        >
          <ElIcon :size="14"><Close /></ElIcon>
          {{ t("preferences.tabbar.contextMenu.close") || "关闭" }}
        </li>
        <li @click="togglePin(contextMenuTag)">
          <ElIcon :size="14">
            <svg viewBox="0 0 24 24" fill="currentColor" width="1em" height="1em">
              <path d="M16 12V4h1V2H7v2h1v8l-2 2v2h5.2v6h1.6v-6H18v-2l-2-2z" />
            </svg>
          </ElIcon>
          {{
            contextMenuTag?.affix
              ? t("preferences.tabbar.contextMenu.unpin") || "取消固定"
              : t("preferences.tabbar.contextMenu.pin") || "固定"
          }}
        </li>
        <li @click="toggleMaximize">
          <ElIcon :size="14">
            <FullScreen v-if="!isMaximized" />
            <ScaleToOriginal v-else />
          </ElIcon>
          {{
            isMaximized
              ? t("preferences.tabbar.contextMenu.restoreMaximize") || "退出最大化"
              : t("preferences.tabbar.contextMenu.maximize") || "最大化"
          }}
        </li>
        <li @click="openInNewWindow(contextMenuTag)">
          <ElIcon :size="14"><Promotion /></ElIcon>
          {{ t("preferences.tabbar.contextMenu.openInNewWindow") || "在新窗口打开" }}
        </li>
        <li class="tabs-bar__contextmenu-divider" />
        <li
          :class="{ 'is-disabled': isFirstViewOf(contextMenuTag) }"
          @click="closeLeftTags(contextMenuTag)"
        >
          <ElIcon :size="14"><DArrowLeft /></ElIcon>
          {{ t("preferences.tabbar.contextMenu.closeLeft") || "关闭左侧标签页" }}
        </li>
        <li
          :class="{ 'is-disabled': isLastViewOf(contextMenuTag) }"
          @click="closeRightTags(contextMenuTag)"
        >
          <ElIcon :size="14"><DArrowRight /></ElIcon>
          {{ t("preferences.tabbar.contextMenu.closeRight") || "关闭右侧标签页" }}
        </li>
        <li class="tabs-bar__contextmenu-divider" />
        <li :class="{ 'is-disabled': closableCount <= 0 }" @click="closeOtherTags(contextMenuTag)">
          <ElIcon :size="14"><SemiSelect /></ElIcon>
          {{ t("preferences.tabbar.contextMenu.closeOther") || "关闭其他标签页" }}
        </li>
        <li :class="{ 'is-disabled': closableCount <= 0 }" @click="closeAllTags(contextMenuTag)">
          <ElIcon :size="14"><CircleClose /></ElIcon>
          {{ t("preferences.tabbar.contextMenu.closeAll") || "关闭全部标签页" }}
        </li>
      </ul>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { useRoute, useRouter, type RouteRecordRaw } from "vue-router";
import { resolve } from "path-browserify";
import { useSortable, type Sortable } from "@/composables/use-sortable";
import { translateRouteTitle } from "@/i18n";
import { useAccessStore, useTagsViewStore } from "@/stores";
import { preferences, updatePreferences } from "@/core/preferences";

import {
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  CircleClose,
  Close,
  DArrowLeft,
  DArrowRight,
  FullScreen,
  Promotion,
  RefreshRight,
  ScaleToOriginal,
  SemiSelect,
} from "@element-plus/icons-vue";
import { ElIcon } from "element-plus";

// 标签图标渲染组件（与侧边栏 MenuIcon 逻辑一致）
const TabIcon = defineComponent({
  props: { icon: { type: String, default: "" } },
  setup(props) {
    const isLucideIcon = computed(() => props.icon?.startsWith("lucide:"));
    const isElIcon = computed(() => props.icon?.startsWith("el-icon"));
    const lucideName = computed(() => props.icon?.replace("lucide:", ""));
    const iconName = computed(() => props.icon?.replace("el-icon-", ""));

    return () => {
      if (!props.icon) return null;

      // Lucide 图标（通过 UnoCSS）
      if (isLucideIcon.value) {
        return h("div", { class: `i-lucide:${lucideName.value}` });
      }

      // Element Plus 图标
      if (isElIcon.value) {
        return h(ElIcon, { size: 14 }, () => h(resolveComponent(iconName.value!)));
      }

      // SVG 图标（通过 UnoCSS）
      return h("div", { class: `i-svg:${props.icon}` });
    };
  },
});

interface ContextMenu {
  visible: boolean;
  x: number;
  y: number;
}

const router = useRouter();
const route = useRoute();

// 状态管理
const accessStore = useAccessStore();
const tagsViewStore = useTagsViewStore();
const { t } = useI18n();

const { visitedViews } = storeToRefs(tagsViewStore);

// 偏好设置
const styleType = computed(() => preferences.tabbar.styleType || "chrome");
const showIcon = computed(() => preferences.tabbar.showIcon ?? false);
const showMore = computed(() => preferences.tabbar.showMore ?? true);
const showMaximize = computed(() => preferences.tabbar.showMaximize ?? true);
const draggable = computed(() => preferences.tabbar.draggable ?? true);
const tabHeight = computed(() => preferences.tabbar.height || 38);

// 最大化状态
const isMaximized = ref(false);

const toggleMaximize = () => {
  isMaximized.value = !isMaximized.value;
  // 隐藏/显示 header 和 sidebar
  updatePreferences({
    header: { hidden: isMaximized.value },
    sidebar: { hidden: isMaximized.value },
  });
};

// 监听外部最大化状态变化
watch(
  () => preferences.header.hidden && preferences.sidebar.hidden,
  (val) => {
    isMaximized.value = val;
  }
);

// 当前选中的标签
const selectedTag = ref<TagView | null>(null);

// 右键菜单对应的标签
const contextMenuTag = computed(() => selectedTag.value);

// 当前路由对应的标签（下拉菜单上下文）
const activeTag = computed(() => {
  return visitedViews.value.find((v: TagView) => v.path === route.path) || null;
});

/** 可关闭的标签数量（非固定） */
const closableCount = computed(() => visitedViews.value.filter((v: TagView) => !v.affix).length);

/** 是否没有可关闭的左侧标签 */
function isFirstViewOf(tag: TagView | null): boolean {
  if (!tag) return true;
  const idx = visitedViews.value.findIndex((v: TagView) => v.fullPath === tag.fullPath);
  if (idx <= 0) return true;
  return visitedViews.value.slice(0, idx).every((v: TagView) => v.affix);
}

/** 是否没有可关闭的右侧标签 */
function isLastViewOf(tag: TagView | null): boolean {
  if (!tag) return true;
  const idx = visitedViews.value.findIndex((v: TagView) => v.fullPath === tag.fullPath);
  if (idx < 0 || idx >= visitedViews.value.length - 1) return true;
  return visitedViews.value.slice(idx + 1).every((v: TagView) => v.affix);
}

// 右键菜单状态
const contextMenu = reactive<ContextMenu>({
  visible: false,
  x: 0,
  y: 0,
});

// 滚动相关
const scrollbarRef = ref<HTMLElement | null>(null);
const scrollInnerRef = ref<HTMLElement | null>(null);
const showScrollButton = ref(false);
const scrollIsAtLeft = ref(true);
const scrollIsAtRight = ref(false);

// 判断标签是否激活
const isActive = (tag: TagView) => {
  return tag.path === route.path;
};

// 路由映射缓存
const routePathMap = computed(() => {
  const map = new Map<string, TagView>();
  visitedViews.value.forEach((tag: { path: string }) => {
    map.set(tag.path, tag);
  });
  return map;
});

// ==================== 滚动功能 ====================

function calcShowScrollButton() {
  if (!scrollbarRef.value || !scrollInnerRef.value) return;
  const containerWidth = scrollbarRef.value.clientWidth;
  const innerWidth = scrollInnerRef.value.scrollWidth;
  showScrollButton.value = innerWidth > containerWidth;
  updateScrollState();
}

function updateScrollState() {
  if (!scrollbarRef.value || !scrollInnerRef.value) return;
  const el = scrollbarRef.value;
  scrollIsAtLeft.value = el.scrollLeft <= 0;
  scrollIsAtRight.value = el.scrollLeft + el.clientWidth >= el.scrollWidth - 1;
}

function scrollDirection(direction: "left" | "right") {
  if (!scrollbarRef.value) return;
  const distance = 150;
  scrollbarRef.value.scrollBy({
    behavior: "smooth",
    left: direction === "left" ? -distance : distance,
  });
}

function handleWheel(e: WheelEvent) {
  if (!scrollbarRef.value) return;
  scrollbarRef.value.scrollBy({
    behavior: "smooth",
    left: e.deltaY || e.deltaX,
  });
}

function scrollToActiveTab() {
  nextTick(() => {
    if (!scrollbarRef.value) return;
    const activeTab = scrollbarRef.value.querySelector(".is-active") as HTMLElement;
    if (activeTab) {
      activeTab.scrollIntoView({ behavior: "smooth", inline: "nearest", block: "nearest" });
    }
    calcShowScrollButton();
  });
}

// ==================== 标签操作 ====================

/**
 * 递归提取固定标签
 */
const extractAffixTags = (routes: RouteRecordRaw[], basePath = "/"): TagView[] => {
  const affixTags: TagView[] = [];

  const traverse = (routeList: RouteRecordRaw[], currentBasePath: string) => {
    routeList.forEach((r) => {
      const fullPath = resolve(currentBasePath, r.path);
      if (r.meta?.affix) {
        affixTags.push({
          path: fullPath,
          fullPath,
          name: String(r.name || ""),
          title: r.meta.title || "no-name",
          icon: r.meta.icon as string | undefined,
          affix: true,
          keepAlive: r.meta.keepAlive || false,
        });
      }
      if (r.children?.length) {
        traverse(r.children, fullPath);
      }
    });
  };

  traverse(routes, basePath);
  return affixTags;
};

const initAffixTags = () => {
  const affixTags = extractAffixTags(accessStore.accessRoutes);
  affixTags.forEach((tag) => {
    if (tag.name) {
      tagsViewStore.addVisitedView(tag);
    }
  });
};

const addCurrentTag = () => {
  if (!route.meta?.title) return;
  tagsViewStore.addView({
    name: route.name as string,
    title: route.meta.title,
    path: route.path,
    fullPath: route.fullPath,
    icon: route.meta.icon as string | undefined,
    affix: route.meta.affix || false,
    keepAlive: route.meta.keepAlive || false,
    query: route.query,
  });
};

const updateCurrentTag = () => {
  nextTick(() => {
    const currentTag = routePathMap.value.get(route.path);
    if (currentTag && currentTag.fullPath !== route.fullPath) {
      tagsViewStore.updateVisitedView({
        name: route.name as string,
        title: route.meta?.title || "",
        path: route.path,
        fullPath: route.fullPath,
        icon: route.meta?.icon as string | undefined,
        affix: route.meta?.affix || false,
        keepAlive: route.meta?.keepAlive || false,
        query: route.query,
      });
    }
  });
};

const handleTabClick = (tag: TagView) => {
  router.push({
    path: tag.fullPath,
    query: tag.query,
  });
};

const handleMiddleClick = (tag: TagView) => {
  if (!tag.affix) {
    closeSelectedTag(tag);
  }
};

const openContextMenu = (tag: TagView, event: MouseEvent) => {
  contextMenu.x = event.clientX;
  contextMenu.y = event.clientY;
  contextMenu.visible = true;
  selectedTag.value = tag;
};

const closeContextMenu = () => {
  contextMenu.visible = false;
};

const refreshSelectedTag = (tag: TagView | null) => {
  if (!tag) return;
  closeContextMenu();
  tagsViewStore.delCachedView(tag);
  nextTick(() => {
    router.replace("/redirect" + tag.fullPath);
  });
};

const closeSelectedTag = (tag: TagView | null) => {
  if (!tag || tag.affix) return;
  closeContextMenu();
  tagsViewStore.delView(tag).then((result: any) => {
    if (tagsViewStore.isActive(tag)) {
      tagsViewStore.toLastView(result.visitedViews, tag);
    }
  });
};

const closeLeftTags = (tag?: TagView | null) => {
  const target = tag ?? selectedTag.value;
  if (!target) return;
  closeContextMenu();
  tagsViewStore.delLeftViews(target).then((result: any) => {
    const hasCurrentRoute = result.visitedViews.some((item: TagView) => item.path === route.path);
    if (!hasCurrentRoute) {
      tagsViewStore.toLastView(result.visitedViews);
    }
  });
};

const closeRightTags = (tag?: TagView | null) => {
  const target = tag ?? selectedTag.value;
  if (!target) return;
  closeContextMenu();
  tagsViewStore.delRightViews(target).then((result: any) => {
    const hasCurrentRoute = result.visitedViews.some((item: TagView) => item.path === route.path);
    if (!hasCurrentRoute) {
      tagsViewStore.toLastView(result.visitedViews);
    }
  });
};

const closeOtherTags = (tag?: TagView | null) => {
  const target = tag ?? selectedTag.value;
  if (!target) return;
  closeContextMenu();
  router.push(target);
  tagsViewStore.delOtherViews(target).then(() => {
    updateCurrentTag();
  });
};

const closeAllTags = (tag: TagView | null) => {
  closeContextMenu();
  tagsViewStore.delAllViews().then((result: any) => {
    tagsViewStore.toLastView(result.visitedViews, tag || undefined);
  });
};

/** 固定/取消固定标签 */
const togglePin = (tag: TagView | null) => {
  if (!tag) return;
  closeContextMenu();
  const found = visitedViews.value.find((v: TagView) => v.fullPath === tag.fullPath);
  if (found) {
    found.affix = !found.affix;
  }
};

/** 在新窗口打开标签页 */
const openInNewWindow = (tag: TagView | null) => {
  if (!tag) return;
  closeContextMenu();
  const url = router.resolve(tag.fullPath).href;
  window.open(url, "_blank");
};

const handleCommand = (command: string) => {
  const tag = activeTag.value;
  switch (command) {
    case "reload":
      refreshSelectedTag(tag);
      break;
    case "close":
      closeSelectedTag(tag);
      break;
    case "pin":
      togglePin(tag);
      break;
    case "maximize":
      toggleMaximize();
      break;
    case "openInNewWindow":
      openInNewWindow(tag);
      break;
    case "closeLeft":
      closeLeftTags(tag);
      break;
    case "closeRight":
      closeRightTags(tag);
      break;
    case "closeOther":
      closeOtherTags(tag);
      break;
    case "closeAll":
      closeAllTags(tag);
      break;
  }
};

// ==================== 生命周期 ====================

// 右键菜单外部点击关闭
const handleOutsideClick = () => {
  closeContextMenu();
};

watchEffect(() => {
  if (contextMenu.visible) {
    document.addEventListener("click", handleOutsideClick);
  } else {
    document.removeEventListener("click", handleOutsideClick);
  }
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleOutsideClick);
});

// 监听路由变化
watch(
  route,
  () => {
    addCurrentTag();
    updateCurrentTag();
    nextTick(() => {
      scrollToActiveTab();
    });
  },
  { immediate: true }
);

// 监听 visitedViews 变化，重新计算滚动按钮
watch(
  () => visitedViews.value.length,
  () => {
    nextTick(() => {
      calcShowScrollButton();
    });
  }
);

// 监听标签栏样式变化，重新计算
watch(styleType, () => {
  nextTick(() => {
    calcShowScrollButton();
  });
});

// ==================== 拖拽排序 ====================

let sortableInstance: Sortable | null = null;

function initSortable() {
  if (!scrollInnerRef.value || !draggable.value) return;

  useSortable(scrollInnerRef.value, {
    animation: 200,
    delay: 300,
    delayOnTouchOnly: true,
    filter: ".is-affix",
    onEnd: (evt: any) => {
      const { oldIndex, newIndex } = evt;
      if (oldIndex === undefined || newIndex === undefined || oldIndex === newIndex) return;

      // 还原 SortableJS 的 DOM 操作，让 Vue 接管 DOM 更新
      const { item, from } = evt;
      from.removeChild(item);
      from.insertBefore(item, from.children[oldIndex] || null);

      // 更新响应式数据
      const views = [...visitedViews.value];
      const [moved] = views.splice(oldIndex, 1);
      views.splice(newIndex, 0, moved);
      visitedViews.value = views;
    },
  })
    .initializeSortable()
    .then((instance) => {
      sortableInstance = instance;
    });
}

// 监听 draggable 变化
watch(draggable, (val) => {
  if (val) {
    nextTick(() => initSortable());
  } else if (sortableInstance) {
    sortableInstance.destroy();
    sortableInstance = null;
  }
});

// 初始化
onMounted(() => {
  initAffixTags();

  // 如果未开启标签持久化，刷新后清除非固定标签
  if (!preferences.tabbar.persist) {
    tagsViewStore.clearNonAffixViews();
  }

  calcShowScrollButton();

  // 初始化拖拽排序
  if (draggable.value) {
    initSortable();
  }
});
</script>

<style lang="scss" scoped>
$tab-gap: 6px;
$chrome-radius: 7px;

.tabs-bar {
  display: flex;
  align-items: center;
  width: 100%;
  border-top: 1px solid var(--el-border-color-lighter);
  background-color: var(--el-bg-color);

  // ==================== 滚动按钮 ====================
  &__scroll-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 100%;
    flex-shrink: 0;
    cursor: pointer;
    color: var(--el-text-color-secondary);
    transition: all 0.2s;

    &:hover {
      color: var(--el-text-color-primary);
      background-color: var(--el-fill-color-light);
    }

    &.is-disabled {
      opacity: 0.3;
      pointer-events: none;
    }
  }

  // ==================== 内容区域 ====================
  &__content {
    flex: 1;
    height: 100%;
    overflow: hidden;
    position: relative;

    &--chrome {
      padding-top: 3px;
    }
  }

  &__inner {
    display: flex;
    align-items: stretch;
    height: 100%;
    width: max-content;
    padding-right: 8px;
  }

  // ==================== 单个标签项 ====================
  &__item {
    display: flex;
    align-items: center;
    position: relative;
    cursor: pointer;
    user-select: none;
    transition: all 0.15s ease;
    flex-shrink: 0;
    height: 100%;

    // ---- Plain 样式 ----
    &--plain {
      padding: 0 12px;
      border-right: 1px solid var(--el-border-color-lighter);

      &:first-child {
        border-left: none;
      }

      &.is-active {
        color: var(--el-color-primary);
        background-color: var(--el-color-primary-light-9);
      }

      &:not(.is-active):hover {
        background-color: var(--el-fill-color-light);
      }
    }

    // ---- Card 样式 ----
    &--card {
      padding: 0 12px;
      margin: 3px 0 3px 4px;
      height: calc(100% - 6px);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 4px;

      &.is-active {
        color: var(--el-color-primary);
        background-color: var(--el-color-primary-light-9);
        border-color: var(--el-color-primary-light-5);
      }

      &:not(.is-active):hover {
        background-color: var(--el-fill-color-light);
      }
    }

    // ---- Brisk 样式 ----
    &--brisk {
      padding: 0 12px;
      position: relative;

      &:not(:first-child) {
        border-left: 1px solid var(--el-border-color-lighter);
      }

      // 底部指示线
      &::after {
        content: "";
        position: absolute;
        bottom: 0;
        left: 0;
        width: 100%;
        height: 2px;
        background-color: var(--el-color-primary);
        transform: scaleX(0);
        transition: transform 0.3s ease;
        transform-origin: left;
      }

      &:hover::after {
        transform: scaleX(1);
      }

      &.is-active {
        color: var(--el-color-primary);

        &::after {
          transform: scaleX(1);
        }
      }

      &:not(.is-active):hover {
        background-color: var(--el-fill-color-light);
      }
    }

    // ---- Chrome 样式 ----
    &--chrome {
      margin-right: -$chrome-radius;
      padding: 0 calc(#{$chrome-radius} * 2 + 2px);
      z-index: 1;

      &:hover {
        z-index: 2;
      }

      &.is-active {
        z-index: 3;
      }

      &.is-active + & {
        .tabs-bar__divider {
          opacity: 0 !important;
        }
      }

      &:not(.is-active):hover {
        & + .tabs-bar__item--chrome {
          .tabs-bar__divider {
            opacity: 0;
          }
        }

        .tabs-bar__chrome-bg__content {
          background-color: var(--el-fill-color);
          margin: 0 2px;
          border-radius: 4px;
        }
      }

      &.is-active {
        color: var(--el-color-primary);
      }
    }
  }

  // ==================== Chrome 风格特殊元素 ====================

  // 分隔线
  &__divider {
    position: absolute;
    left: $chrome-radius;
    top: 50%;
    transform: translateY(-50%);
    width: 1px;
    height: 14px;
    background-color: var(--el-border-color);
    z-index: 0;
    transition: opacity 0.15s;
  }

  // Chrome 背景
  &__chrome-bg {
    position: absolute;
    inset: 0;
    z-index: -1;
    padding: 0 calc(#{$chrome-radius} - 1px);

    &__content {
      height: 100%;
      border-top-left-radius: $chrome-radius;
      border-top-right-radius: $chrome-radius;
      transition: all 0.15s;
    }

    &__before,
    &__after {
      position: absolute;
      bottom: 0;
      fill: transparent;
      transition: all 0.15s;
    }

    &__before {
      left: -1px;
    }

    &__after {
      right: -1px;
    }

    .is-active & {
      &__content {
        background-color: var(--el-color-primary-light-9);
      }

      &__before,
      &__after {
        fill: var(--el-color-primary-light-9);
      }
    }
  }

  // ==================== 标签内容 ====================
  &__item-main {
    display: flex;
    align-items: center;
    gap: 4px;
    height: 100%;
    padding-right: 4px;
    z-index: 2;
    overflow: hidden;
  }

  &__icon {
    flex-shrink: 0;
    width: 14px;
    height: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  &__title {
    font-size: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 120px;
    line-height: 1;
  }

  // ==================== 关闭/固定按钮 ====================
  &__extra {
    position: relative;
    z-index: 3;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    margin-left: 2px;
    border-radius: 50%;
    transition: all 0.15s;
  }

  &__close {
    color: var(--el-text-color-secondary);
    opacity: 0;
    transition: all 0.15s;

    .tabs-bar__item:hover & {
      opacity: 1;
    }

    .tabs-bar__item.is-active & {
      opacity: 1;
      color: var(--el-color-primary);
    }

    &:hover {
      color: var(--el-color-primary);
      background-color: var(--el-color-primary-light-9);
    }
  }

  &__pin {
    color: var(--el-text-color-secondary);
    .tabs-bar__item.is-active & {
      color: var(--el-color-primary);
    }
  }

  // ==================== 工具按钮区域 ====================
  &__tools {
    display: flex;
    align-items: center;
    gap: 2px;
    height: 100%;
    padding: 0 8px;
    flex-shrink: 0;
    border-left: 1px solid var(--el-border-color-lighter);
  }

  &__tool-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    cursor: pointer;
    border-radius: 4px;
    color: var(--el-text-color-secondary);
    transition: all 0.2s;

    &:hover {
      color: var(--el-text-color-primary);
      background-color: var(--el-fill-color-light);
    }
  }

  // ==================== 右键菜单 ====================
  &__contextmenu {
    position: fixed;
    z-index: 3000;
    padding: 4px 0;
    margin: 0;
    font-size: 12px;
    color: var(--el-text-color-primary);
    list-style: none;
    background: var(--el-bg-color-overlay);
    border-radius: 4px;
    box-shadow: var(--el-box-shadow-light);

    li {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 6px 16px;
      cursor: pointer;
      transition: background-color 0.15s;

      &:hover {
        background-color: var(--el-fill-color-light);
      }

      &.is-disabled {
        opacity: 0.45;
        cursor: not-allowed;
        pointer-events: none;
      }
    }

    &-divider {
      height: 1px !important;
      margin: 4px 0 !important;
      padding: 0 !important;
      background-color: var(--el-border-color-lighter) !important;
      min-height: 1px !important;
      line-height: 0 !important;
      font-size: 0 !important;
      overflow: hidden;
      cursor: default;
      pointer-events: none;
      user-select: none;

      &:hover {
        background-color: var(--el-border-color-lighter) !important;
      }
    }
  }
}

// ==================== Element Plus Dropdown 分割线修正 ====================
// Element Plus 的 divided 属性会额外插入一个独立的 <li> 元素（class: el-dropdown-menu__item--divided），
// 该元素是一个 flex 容器，默认有 padding/line-height，视觉上会撑开。
// 此处覆盖为细分割线：只保留 border-top，其他全部隐藏。
:global(.el-dropdown-menu__item--divided) {
  margin: 4px 0 !important;
  padding: 0 !important;
  border-top: 1px solid var(--el-border-color-lighter) !important;
  min-height: 0 !important;
  height: 0 !important;
  line-height: 0 !important;
  font-size: 0 !important;
  overflow: hidden;

  &::before,
  &::after {
    display: none !important;
    content: none !important;
  }
}

// ==================== Tab 切换动画 ====================
.tab-slide-enter-active {
  transition: all 0.2s ease-out;
}

.tab-slide-leave-active {
  transition: all 0.15s ease-in;
}

.tab-slide-enter-from {
  opacity: 0;
  transform: translateX(-10px);
}

.tab-slide-leave-to {
  opacity: 0;
  transform: translateX(10px);
}
</style>
