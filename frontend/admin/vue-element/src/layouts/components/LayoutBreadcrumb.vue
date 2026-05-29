<template>
  <el-breadcrumb v-show="visible" class="breadcrumb" :class="breadcrumbClass">
    <!-- 首页图标 -->
    <el-breadcrumb-item
      v-if="breadcrumbPrefs.showHome"
      :to="{ path: '/' }"
      class="breadcrumb__home"
    >
      <SvgIcon
        v-if="breadcrumbPrefs.showIcon"
        icon="homepage"
        :size="16"
        class="breadcrumb__icon"
      />
      {{ $t("common.breadcrumb.home") }}
    </el-breadcrumb-item>
    <el-breadcrumb-item v-for="(item, index) in breadcrumbs" :key="item.path">
      <span
        v-if="item.redirect === 'noredirect' || index === breadcrumbs.length - 1"
        class="color-gray-400"
      >
        <SvgIcon
          v-if="breadcrumbPrefs.showIcon && item.meta?.icon"
          :icon="item.meta.icon as string"
          :size="16"
          class="breadcrumb__item-icon"
        />
        {{ translateRouteTitle((item.meta.title as string) ?? "") }}
      </span>
      <a v-else @click.prevent="handleLink(item)">
        <SvgIcon
          v-if="breadcrumbPrefs.showIcon && item.meta?.icon"
          :icon="item.meta.icon as string"
          :size="16"
          class="breadcrumb__item-icon"
        />
        {{ translateRouteTitle((item.meta.title as string) ?? "") }}
      </a>
    </el-breadcrumb-item>
  </el-breadcrumb>
</template>

<script setup lang="ts">
import { RouteLocationMatched } from "vue-router";
import { compile } from "path-to-regexp";

import { router } from "@/router";
import { translateRouteTitle } from "@/core/i18n";
import { preferences } from "@/core/preferences";
import SvgIcon from "@/components/SvgIcon/index.vue";

const currentRoute = useRoute();
const pathCompile = (path: string) => {
  const { params } = currentRoute;
  const toPath = compile(path);
  return toPath(params);
};

// 面包屑偏好
const breadcrumbPrefs = computed(() => preferences.breadcrumb);

const breadcrumbs = ref<Array<RouteLocationMatched>>([]);

// 是否可见：启用 + 不只有一个时隐藏检查
const visible = computed(() => {
  if (!breadcrumbPrefs.value.enable) return false;
  return !(breadcrumbPrefs.value.hideOnlyOne && breadcrumbs.value.length <= 1);
});

// 面包屑样式类
const breadcrumbClass = computed(() => {
  return {
    "breadcrumb--background": breadcrumbPrefs.value.styleType === "background",
  };
});

function getBreadcrumb() {
  breadcrumbs.value = currentRoute.matched.filter(
    (item) =>
      item.meta && item.meta.title && item.meta.breadcrumb !== false && !item.meta.hideInBreadcrumb
  );
}

function handleLink(item: any) {
  const { redirect, path } = item;
  if (redirect) {
    router.push(redirect).then(
      () => {},
      (err) => {
        console.warn(err);
      }
    );
    return;
  }
  router.push(pathCompile(path)).then(
    () => {},
    (err) => {
      console.warn(err);
    }
  );
}

watch(
  () => currentRoute.path,
  () => {
    getBreadcrumb();
  }
);

onBeforeMount(() => {
  getBreadcrumb();
});
</script>

<style lang="scss" scoped>
.breadcrumb {
  display: flex;
  align-items: center;
  font-size: 14px;

  // 覆盖 element-plus 的样式
  // 亮色模式：非当前页 #909399（中灰），当前页 #303133（深黑）+ 加粗
  :deep(.el-breadcrumb__inner),
  :deep(.el-breadcrumb__inner a) {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-weight: 400 !important;
    font-size: 14px !important;
    color: #909399 !important;
    transition: color 0.2s ease;
  }

  // 当前页（最后一项）：深黑色 + 加粗，强化"当前位置"视觉焦点
  :deep(.el-breadcrumb__item:last-child .el-breadcrumb__inner) {
    color: #303133 !important;
    font-weight: 500 !important;
  }

  // 可点击链接 hover
  :deep(.el-breadcrumb__inner a:hover) {
    color: var(--el-color-primary) !important;
  }

  // 分隔符颜色：比非当前页文字略浅
  :deep(.el-breadcrumb__separator) {
    color: #c0c4cc;
  }

  // background 风格
  &--background {
    :deep(.el-breadcrumb__item) {
      .el-breadcrumb__inner {
        padding: 2px 8px;
        border-radius: 4px;
        transition: background-color 0.2s;
      }

      &:not(:last-child) .el-breadcrumb__inner {
        background-color: var(--el-fill-color-light);
      }

      &:not(:last-child) .el-breadcrumb__inner:hover {
        background-color: var(--el-fill-color);
      }
    }
  }

  &__home {
    :deep(.el-breadcrumb__inner) {
      display: inline-flex;
      align-items: center;
    }
  }

  &__icon,
  &__item-icon {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
    color: currentColor;
  }

  // ======== 暗色模式适配 ========
  :global(html.dark) {
    // 非当前页文字：亮灰色（#d9d9d9），清晰可读但不抢当前页焦点
    :deep(.el-breadcrumb__inner),
    :deep(.el-breadcrumb__inner a) {
      color: #d9d9d9 !important;
    }
  
    // 当前页（最后一项）：纯白 + 加粗，强化"当前位置"视觉焦点
    :deep(.el-breadcrumb__item:last-child .el-breadcrumb__inner) {
      color: #ffffff !important;
      font-weight: 500 !important;
    }
  
    // 分隔符：较暗灰色，作为视觉分割不抢文字焦点
    :deep(.el-breadcrumb__separator) {
      color: #595959 !important;
    }

    // 可点击链接 hover
    :deep(.el-breadcrumb__inner a:hover) {
      color: var(--el-color-primary) !important;
    }

    // background 风格暗色模式
    &.breadcrumb--background {
      :deep(.el-breadcrumb__item) {
        &:not(:last-child) .el-breadcrumb__inner {
          background-color: rgba(255, 255, 255, 0.06);
        }

        &:not(:last-child) .el-breadcrumb__inner:hover {
          background-color: rgba(255, 255, 255, 0.1);
        }
      }
    }
  }
}
</style>
