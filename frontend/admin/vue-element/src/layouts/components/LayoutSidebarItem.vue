<template>
  <div v-if="!item.meta || !item.meta.hideInMenu">
    <!--【叶子节点】无可见子节点 -->
    <template v-if="!hasVisibleChildren(item.children)">
      <AppLink
        :to="{
          path: resolvePath(item.path),
          query: item.meta?.params as LocationQueryRaw,
        }"
      >
        <el-menu-item
          :index="resolvePath(item.path)"
          :class="{ 'submenu-title-noDropdown': !isNest }"
        >
          <MenuIcon :icon="item.meta?.icon" />
          <span v-if="item.meta?.title" class="ml-1">
            {{ translateRouteTitle(item.meta.title) }}
          </span>
        </el-menu-item>
      </AppLink>
    </template>

    <!--【非叶子节点】有可见子节点 -->
    <el-sub-menu v-else :index="resolvePath(item.path)" :data-path="item.path" teleported>
      <template #title>
        <template v-if="item.meta">
          <MenuIcon :icon="item.meta.icon" />
          <span v-if="item.meta.title" class="ml-1">
            {{ translateRouteTitle(item.meta.title) }}
          </span>
        </template>
      </template>

      <LayoutSidebarItem
        v-for="child in item.children"
        :key="child.path"
        :is-nest="true"
        :item="child"
        :base-path="resolvePath(child.path)"
      />
    </el-sub-menu>
  </div>
</template>

<script setup lang="ts">
import path from "path-browserify";
import { RouteRecordRaw, LocationQueryRaw } from "vue-router";
import { isExternal } from "@/utils";
import { translateRouteTitle } from "@/i18n";
import { ElIcon } from "element-plus";

defineOptions({
  name: "LayoutSidebarItem",
  inheritAttrs: false,
});

// 菜单图标组件
const MenuIcon = defineComponent({
  props: { icon: String },
  setup(props) {
    const isElIcon = computed(() => props.icon?.startsWith("el-icon"));
    const isLucideIcon = computed(() => props.icon?.startsWith("lucide:"));
    const iconName = computed(() => props.icon?.replace("el-icon-", ""));
    const lucideName = computed(() => props.icon?.replace("lucide:", ""));

    return () => {
      if (!props.icon) {
        return h("div", { class: "i-svg:menu" });
      }

      // Element Plus 图标
      if (isElIcon.value) {
        return h(ElIcon, null, () => h(resolveComponent(iconName.value!)));
      }

      // Lucide 图标
      if (isLucideIcon.value) {
        return h("div", { class: `i-lucide:${lucideName.value}` });
      }

      // SVG 图标
      return h("div", { class: `i-svg:${props.icon}` });
    };
  },
});

const props = defineProps({
  /**
   * 当前路由对象
   */
  item: {
    type: Object as PropType<RouteRecordRaw>,
    required: true,
  },

  /**
   * 父级完整路径
   */
  basePath: {
    type: String,
    required: true,
  },

  /**
   * 是否为嵌套路由
   */
  isNest: {
    type: Boolean,
    default: false,
  },
});

/**
 * 判断是否有可见的子节点
 */
const hasVisibleChildren = (children?: RouteRecordRaw[]) => {
  return children?.some((child) => !child.meta?.hideInMenu);
};

/**
 * 获取完整路径，适配外部链接
 *
 * @param routePath 路由路径
 * @returns 绝对路径
 */
function resolvePath(routePath: string) {
  if (isExternal(routePath)) return routePath;
  if (isExternal(props.basePath)) return props.basePath;

  // 拼接父路径和当前路径
  return path.resolve(props.basePath, routePath);
}
</script>

<style lang="scss">
/* stylelint-disable no-descending-specificity */
/* 菜单图标统一样式 */
.el-menu-item,
.el-sub-menu__title {
  .el-icon {
    width: 1em !important;
    margin-right: 0 !important;
    font-size: 18px;
    color: currentcolor;
  }

  [class^="i-svg:"],
  [class^="i-lucide:"] {
    width: 18px;
    height: 18px;
    font-size: 18px;
    color: currentcolor !important;
  }
}

/* 折叠状态下的图标样式 - 确保 SVG 图标不被压缩 */
.el-menu--collapse {
  .el-menu-item,
  .el-sub-menu > .el-sub-menu__title {
    [class^="i-svg:"],
    [class^="i-lucide:"] {
      width: 18px !important;
      min-width: 18px !important;
      height: 18px !important;
      font-size: 18px !important;
    }
  }

  /* tooltip 弹出层中的图标 */
  .el-tooltip__trigger {
    [class^="i-svg:"],
    [class^="i-lucide:"] {
      width: 18px !important;
      min-width: 18px !important;
      height: 18px !important;
      font-size: 18px !important;
    }
  }
}

/* hideSidebar 状态下的图标 */
.hideSidebar {
  [class^="i-svg:"],
  [class^="i-lucide:"] {
    width: 18px !important;
    min-width: 18px !important;
    height: 18px !important;
    font-size: 18px !important;
  }

  .submenu-title-noDropdown {
    position: relative;

    & > span {
      display: inline-block;
      visibility: hidden;
      width: 0;
      height: 0;
      overflow: hidden;
    }
  }

  .el-sub-menu {
    overflow: hidden;

    & > .el-sub-menu__title {
      .sub-el-icon {
        margin-left: 19px;
      }

      .el-sub-menu__icon-arrow {
        display: none;
      }
    }
  }

  .el-menu--collapse {
    width: $sidebar-width-collapsed;

    .el-sub-menu {
      & > .el-sub-menu__title > span {
        display: inline-block;
        visibility: hidden;
        width: 0;
        height: 0;
        overflow: hidden;
      }
    }
  }
}

html.dark {
  .el-menu-item:hover {
    background-color: $menu-hover;
  }
}

html.sidebar-color-blue {
  .el-menu-item:hover {
    background-color: $menu-hover;
  }
}

// 父菜单激活状态样式 - 当子菜单激活时，父菜单显示激活状态
.el-sub-menu {
  // 当父菜单包含激活子菜单时的样式
  &.has-active-child > .el-sub-menu__title {
    color: var(--el-color-primary) !important;
    background-color: var(--el-color-primary-light-9) !important;

    .menu-icon {
      color: var(--el-color-primary) !important;
    }
  }

  // 深色主题下的父菜单激活状态"
  html.dark & {
    &.has-active-child > .el-sub-menu__title {
      color: var(--el-color-primary-light-3) !important;
      background-color: rgba(64, 128, 255, 0.15) !important;

      .menu-icon {
        color: var(--el-color-primary-light-3) !important;
      }
    }
  }

  // 深蓝色侧边栏配色下的父菜单激活状态"
  html.sidebar-color-blue & {
    &.has-active-child > .el-sub-menu__title {
      color: var(--el-color-primary-light-3) !important;
      background-color: rgba(64, 128, 255, 0.2) !important;

      .menu-icon {
        color: var(--el-color-primary-light-3) !important;
      }
    }
  }
}
/* stylelint-enable no-descending-specificity */
</style>
