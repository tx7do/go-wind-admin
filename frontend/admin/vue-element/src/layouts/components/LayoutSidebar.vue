<!-- 菜单组件 -->
<template>
  <el-menu
    ref="menuRef"
    :default-active="activeMenuPath"
    :class="[`nav-style--${navigationStyle}`]"
    :collapse="effectiveCollapsed"
    :background-color="menuThemeProps.backgroundColor"
    :text-color="menuThemeProps.textColor"
    :active-text-color="menuThemeProps.activeTextColor"
    :popper-effect="theme"
    :unique-opened="navigationPreferences.accordion"
    :collapse-transition="false"
    :mode="menuMode"
    @open="onMenuOpen"
    @close="onMenuClose"
  >
    <!-- 菜单项 -->
    <LayoutSidebarItem
      v-for="route in data"
      :key="route.path"
      :item="route"
      :base-path="resolveFullPath(route.path)"
    />
  </el-menu>
</template>

<script lang="ts" setup>
import { useRoute } from "vue-router";
import path from "path-browserify";
import type { MenuInstance } from "element-plus";
import type { RouteRecordRaw } from "vue-router";

import { usePreferences } from "@/core/preferences";
import { preferences } from "@/core/preferences";
import { isExternal } from "@/utils";

import LayoutSidebarItem from "./LayoutSidebarItem.vue";
import variables from "@/styles/variables.module.scss";

const props = defineProps({
  data: {
    type: Array as PropType<RouteRecordRaw[]>,
    default: () => [],
  },
  basePath: {
    type: String,
    required: true,
    example: "/system",
  },
  menuMode: {
    type: String as PropType<"vertical" | "horizontal">,
    default: "vertical",
    validator: (value: string) => ["vertical", "horizontal"].includes(value),
  },
  /** 菜单折叠状态，由父组件传入视觉状态 */
  collapse: {
    type: Boolean,
    default: undefined,
  },
});

const menuRef = ref<MenuInstance>();
const currentRoute = useRoute();
const { sidebarCollapsed, theme, navigationPreferences } = usePreferences();

// 折叠状态：优先使用外部传入的 prop，否则回退到 preferences
const effectiveCollapsed = computed(() => {
  if (props.collapse !== undefined) return props.collapse;
  return sidebarCollapsed.value;
});

// 菜单风格类型
const navigationStyle = computed(() => navigationPreferences.value.styleType);

// 存储已展开的菜单项索引
const expandedMenuIndexes = ref<string[]>([]);

// 菜单主题属性
// 深色主题或半深色侧边栏模式下，使用深色菜单配色
const useDarkMenuColors = computed(() => {
  if (theme.value === "dark") return true;
  // 浅色主题下 semiDarkSidebar 开启时也使用深色配色
  return preferences.theme.semiDarkSidebar;
});
const menuThemeProps = computed(() => {
  if (!useDarkMenuColors.value) {
    return { backgroundColor: undefined, textColor: undefined, activeTextColor: undefined };
  }
  return {
    backgroundColor: variables["menu-background"],
    textColor: variables["menu-text"],
    activeTextColor: variables["menu-active-text"],
  };
});

// 计算当前激活的菜单项
const activeMenuPath = computed((): string => {
  const { meta, path } = currentRoute;

  // 如果路由 meta 中设置了 activePath，则使用它（用于处理一些特殊情况，如详情页等）
  if (meta?.activePath && typeof meta.activePath === "string") {
    return meta.activePath;
  }

  // 否则使用当前路由路径
  return path;
});

/**
 * 获取完整路径
 *
 * @param routePath 当前路由的相对路径 /user
 * @returns 完整的绝对路径 D://vue3-element-admin/system/user
 */
function resolveFullPath(routePath: string) {
  if (isExternal(routePath)) {
    return routePath;
  }
  if (isExternal(props.basePath)) {
    return props.basePath;
  }

  // 如果 basePath 为空（顶部布局），直接返回 routePath
  if (!props.basePath || props.basePath === "") {
    return routePath;
  }

  // 解析路径，生成完整的绝对路径
  return path.resolve(props.basePath, routePath);
}

/**
 * 打开菜单
 *
 * @param index 当前展开的菜单项索引
 */
const onMenuOpen = (index: string) => {
  expandedMenuIndexes.value.push(index);
};

/**
 * 关闭菜单
 *
 * @param index 当前收起的菜单项索引
 */
const onMenuClose = (index: string) => {
  expandedMenuIndexes.value = expandedMenuIndexes.value.filter((item) => item !== index);
};

/**
 * 监听展开的菜单项变化，更新父菜单样式
 */
watch(
  () => expandedMenuIndexes.value,
  () => {
    updateParentMenuStyles();
  }
);

/**
 * 监听菜单模式变化：当菜单模式切换为水平模式时，关闭所有展开的菜单项
 * 避免在水平模式下菜单项显示错位
 */
watch(
  () => props.menuMode,
  (newMode) => {
    if (newMode === "horizontal" && menuRef.value) {
      expandedMenuIndexes.value.forEach((item) => menuRef.value!.close(item));
    }
  }
);

/**
 * 监听激活菜单变化，为包含激活子菜单的父菜单添加样式
 */
watch(
  () => activeMenuPath.value,
  () => {
    nextTick(() => {
      updateParentMenuStyles();
    });
  },
  { immediate: true }
);

/**
 * 监听路由变化，确保菜单能随 TagsView 切换而正确激活
 */
watch(
  () => currentRoute.path,
  () => {
    nextTick(() => {
      updateParentMenuStyles();
    });
  }
);

/**
 * 更新父菜单样式 - 为包含激活子菜单的父菜单添加 has-active-child 类
 */
function updateParentMenuStyles() {
  if (!menuRef.value?.$el) return;

  nextTick(() => {
    try {
      const menuEl = menuRef.value?.$el as HTMLElement;
      if (!menuEl) return;

      // 移除所有现有的 has-active-child 类
      const allSubMenus = menuEl.querySelectorAll(".el-sub-menu");
      allSubMenus.forEach((subMenu) => {
        subMenu.classList.remove("has-active-child");
      });

      // 查找当前激活的菜单项
      const activeMenuItem = menuEl.querySelector(".el-menu-item.is-active");

      if (activeMenuItem) {
        // 向上查找父级 el-sub-menu 元素
        let parent = activeMenuItem.parentElement;
        while (parent && parent !== menuEl) {
          if (parent.classList.contains("el-sub-menu")) {
            parent.classList.add("has-active-child");
          }
          parent = parent.parentElement;
        }
      } else {
        // 水平模式下可能需要特殊处理
        if (props.menuMode === "horizontal") {
          // 对于水平菜单，使用路径匹配来找到父菜单
          const currentPath = activeMenuPath.value;

          // 查找所有父菜单项，检查哪个包含当前路径
          allSubMenus.forEach((subMenu) => {
            const subMenuEl = subMenu as HTMLElement;
            const subMenuPath =
              subMenuEl.getAttribute("data-path") ||
              subMenuEl.querySelector(".el-sub-menu__title")?.getAttribute("data-path");

            // 如果找到包含当前路径的父菜单，则添加激活类
            if (subMenuPath && currentPath.startsWith(subMenuPath)) {
              subMenuEl.classList.add("has-active-child");
            }
          });
        }
      }
    } catch (error) {
      console.error("Error updating parent menu styles:", error);
    }
  });
}

/**
 * 组件挂载后立即更新父菜单样式
 */
onMounted(() => {
  // 确保在组件挂载后更新样式，不依赖于异步操作
  updateParentMenuStyles();
});
</script>

<style lang="scss">
// ============================================
// 左侧菜单样式优化
// ============================================

.layout__sidebar {
  .el-menu {
    // 菜单项全局统一规范：36px 高度，圆角 6px，柔和交互
    .el-menu-item,
    .el-sub-menu__title {
      height: 36px !important;
      line-height: 36px !important;
      padding: 0 16px !important;
      margin: 4px 12px !important;
      border-radius: 6px !important;
      font-size: 13px !important;
      font-weight: 400 !important;
      transition:
        background-color 0.2s ease,
        color 0.2s ease;
    }

    // 图标与文字间距
    .el-menu-item .el-icon,
    .el-sub-menu__title .el-icon {
      margin-right: 10px !important;
      width: 18px !important;
      height: 18px !important;
      transition: color 0.2s ease;
    }

    // 亮色模式 hover：极浅灰背景，提供交互反馈
    .el-menu-item:hover,
    .el-sub-menu__title:hover {
      background-color: #f5f7fa !important;
    }

    // 选中菜单项：左侧亮色竖条 + 主色文字 + 加粗，与暗色模式一致
    .el-menu-item.is-active {
      background-color: var(--el-color-primary-light-9) !important;
      color: var(--el-color-primary) !important;
      font-weight: 500 !important;
      box-shadow: inset 3px 0 0 0 var(--el-color-primary) !important;

      .el-icon {
        color: var(--el-color-primary) !important;
      }
    }

    // 子菜单缩进
    .el-sub-menu .el-menu-item {
      padding-left: 48px !important;
    }

    // ============================================
    // navigation.styleType: rounded 风格
    // 激活菜单项带圆角背景色
    // ============================================
    &.nav-style--rounded {
      .el-menu-item.is-active {
        background-color: var(--el-color-primary-light-9) !important;
        color: var(--el-color-primary) !important;
        font-weight: 500 !important;
        box-shadow: inset 3px 0 0 0 var(--el-color-primary) !important;
      }
    }

    // ============================================
    // 暗黑模式
    // ============================================
    html.dark & {
      // 未选中菜单文字：亮灰色（#d9d9d9），清晰可读，像“活”的按钮
      .el-menu-item,
      .el-sub-menu__title {
        color: #d9d9d9 !important;
      }

      // hover 态：淡白背景 + 纯白文字，明确交互反馈
      .el-menu-item:hover,
      .el-sub-menu__title:hover {
        background-color: rgba(255, 255, 255, 0.05) !important;
        color: #ffffff !important;
      }

      // 选中菜单：品牌色透明度背景 + 左侧竖条 + 纯白文字
      .el-menu-item.is-active {
        background-color: rgba(24, 144, 255, 0.15) !important;
        color: #ffffff !important;
        font-weight: 500 !important;
        box-shadow: inset 3px 0 0 0 var(--el-color-primary) !important;

        .el-icon {
          color: var(--el-color-primary) !important;
        }
      }

      // rounded 风格暗黑模式
      &.nav-style--rounded .el-menu-item.is-active {
        background-color: rgba(24, 144, 255, 0.15) !important;
        color: #ffffff !important;
        font-weight: 500 !important;
        box-shadow: inset 3px 0 0 0 var(--el-color-primary) !important;
      }
    }
  }
}
</style>
