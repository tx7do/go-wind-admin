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
        icon="lucide:house"
        :size="16"
        class="breadcrumb__icon"
      />
      {{ $t("common.breadcrumb.home") }}
    </el-breadcrumb-item>
    <el-breadcrumb-item v-for="(item, index) in breadcrumbs" :key="item.path">
      <span
        v-if="item.redirect === 'noredirect' || index === breadcrumbs.length - 1"
        class="breadcrumb__current"
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
}
</style>

<!--
  面包屑颜色/字重样式用非 scoped 全局块，确保 :last-child 等选择器
  的特异性能覆盖 Element Plus 默认样式，不再依赖 scoped + :deep() 组合
-->
<style lang="scss">
.breadcrumb {
  // 与按钮高度匹配，line-height: normal 让 flex 的 align-items: center 负责垂直居中
  line-height: normal !important;

  // home图标居中对齐
  .breadcrumb__home .el-breadcrumb__inner {
    display: inline-flex !important;
    align-items: center !important;
  }

  // 非当前页：中灰 + 常规字重
  .el-breadcrumb__inner,
  .el-breadcrumb__inner a {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-weight: 400 !important;
    font-size: 15px !important; // 从 14px 增大到 15px
    color: #909399 !important;
    transition: color 0.2s ease;
    line-height: normal !important;
    height: auto !important;
    padding: 0 4px; // 与按钮视觉对齐
  }

  // 当前页（最后一项）：深黑 + 半粗体，视觉焦点
  .el-breadcrumb__item:last-child .el-breadcrumb__inner {
    color: #303133 !important;
    font-weight: 600 !important;
  }

  // 当前页（最后一项）内部 .breadcrumb__current span 也必须深色
  .el-breadcrumb__item:last-child .breadcrumb__current {
    color: #303133 !important;
    font-weight: 600 !important;
  }

  // 可点击链接 hover
  .el-breadcrumb__inner a:hover {
    color: var(--el-color-primary) !important;
  }

  // 分隔符
  .el-breadcrumb__separator {
    color: #c0c4cc;
  }

  // background 风格暗色模式
  &.breadcrumb--background {
    .el-breadcrumb__item:not(:last-child) .el-breadcrumb__inner {
      background-color: rgba(255, 255, 255, 0.06);
    }
    .el-breadcrumb__item:not(:last-child) .el-breadcrumb__inner:hover {
      background-color: rgba(255, 255, 255, 0.1);
    }
  }

  // ======== 暗色模式 ========
  html.dark & {
    .el-breadcrumb__inner,
    .el-breadcrumb__inner a {
      color: #d9d9d9 !important;
    }

    .el-breadcrumb__item:last-child .el-breadcrumb__inner {
      color: #ffffff !important;
      font-weight: 600 !important;
    }

    .el-breadcrumb__item:last-child .breadcrumb__current {
      color: #ffffff !important;
      font-weight: 600 !important;
    }

    .el-breadcrumb__separator {
      color: #595959 !important;
    }

    .el-breadcrumb__inner a:hover {
      color: var(--el-color-primary) !important;
    }

    &.breadcrumb--background {
      .el-breadcrumb__item:not(:last-child) .el-breadcrumb__inner {
        background-color: rgba(255, 255, 255, 0.06);
      }
      .el-breadcrumb__item:not(:last-child) .el-breadcrumb__inner:hover {
        background-color: rgba(255, 255, 255, 0.1);
      }
    }
  }
}
</style>
