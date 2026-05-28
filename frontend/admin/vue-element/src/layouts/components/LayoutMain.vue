<template>
  <section class="app-main" :class="mainClass" :style="{ height: appMainHeight }">
    <router-view>
      <template #default="{ Component, route }">
        <transition :name="transitionName" mode="out-in">
          <keep-alive :include="cachedViews">
            <component
              :is="currentComponent(Component, route)"
              :key="contentRefreshKey ? `${route.fullPath}__${contentRefreshKey}` : route.fullPath"
            />
          </keep-alive>
        </transition>
      </template>
    </router-view>

    <!-- 返回顶部按钮 -->
    <el-backtop target=".app-main">
      <SvgIcon icon="backtop" class="w-6 h-6" />
    </el-backtop>
  </section>
</template>

<script setup lang="ts">
import { type RouteLocationNormalized } from "vue-router";
import { useTagsViewStore } from "./useTagsViewStore";
import SvgIcon from "@/components/SvgIcon/index.vue";
import { preferences, usePreferences } from "@/core/preferences";
import variables from "@/styles/variables.module.scss";
import Error404 from "@/pages/core/error/404.vue";

const { cachedViews } = toRefs(useTagsViewStore());
const { tabbarPreferences } = usePreferences();

// 注入刷新状态
const contentRefreshing = inject<Ref<boolean>>("contentRefreshing", ref(false));

// 刷新 key：每次刷新递增，强制组件重建
const contentRefreshKey = inject<Ref<number>>("contentRefreshKey", ref(0));

// 当前组件
const wrapperMap = new Map<string, Component>();

// 刷新时清理 wrapperMap，确保组件完全重建
watch(contentRefreshing, (val) => {
  if (val) {
    wrapperMap.clear();
  }
});
const currentComponent = (component: Component, route: RouteLocationNormalized) => {
  if (!component) return;

  const { fullPath: componentName } = route; // 使用路由路径作为组件名称
  let wrapper = wrapperMap.get(componentName);

  if (!wrapper) {
    wrapper = {
      name: componentName,
      render: () => {
        try {
          return h(component);
        } catch (error) {
          console.error(`Error rendering component for route: ${componentName}`, error);
          return h(Error404);
        }
      },
    };
    wrapperMap.set(componentName, wrapper);
  }

  // 添加组件数量限制
  if (wrapperMap.size > 100) {
    const firstKey = wrapperMap.keys().next().value;
    if (firstKey) {
      wrapperMap.delete(firstKey);
    }
  }

  return h(wrapper);
};

const appMainHeight = computed(() => {
  if (tabbarPreferences.value.enable) {
    return `calc(100vh - ${variables["navbar-height"]} - ${variables["tags-view-height"]})`;
  } else {
    return `calc(100vh - ${variables["navbar-height"]})`;
  }
});

// 页面切换动画名称
const transitionName = computed(() => {
  if (!preferences.transition.enable) return "";
  return preferences.transition.name ?? "";
});

// 根据 contentCompact 设置主容器类名
const mainClass = computed(() => {
  return {
    "app-main--compact": preferences.app.contentCompact === "compact",
    "app-main--wide": preferences.app.contentCompact === "wide",
  };
});
</script>

<style lang="scss" scoped>
.app-main {
  position: relative;
  overflow-y: auto;
  background-color: var(--el-bg-color-page);
  width: 100%;
  min-width: 0;
}

// 紧凑模式：限制最大宽度并居中
.app-main--compact {
  max-width: 1200px;
  margin-left: auto;
  margin-right: auto;
}

// 宽屏模式：占满整个宽度
.app-main--wide {
  max-width: 100%;
}
</style>

<style lang="scss">
/* 页面过渡动画 - 不能使用 scoped，否则类名无法应用到 transition 子元素 */
.app-main {
  /* fade */
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.2s ease;
  }
  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }

  /* fade-slide */
  .fade-slide-leave-active,
  .fade-slide-enter-active {
    transition:
      opacity 0.2s ease,
      transform 0.2s ease;
  }
  .fade-slide-enter-from {
    opacity: 0;
    transform: translateX(-10px);
  }
  .fade-slide-leave-to {
    opacity: 0;
    transform: translateX(10px);
  }

  /* fade-down */
  .fade-down-leave-active,
  .fade-down-enter-active {
    transition:
      opacity 0.2s ease,
      transform 0.2s ease;
  }
  .fade-down-enter-from {
    opacity: 0;
    transform: translateY(-10px);
  }
  .fade-down-leave-to {
    opacity: 0;
    transform: translateY(10px);
  }

  /* fade-up */
  .fade-up-leave-active,
  .fade-up-enter-active {
    transition:
      opacity 0.2s ease,
      transform 0.2s ease;
  }
  .fade-up-enter-from {
    opacity: 0;
    transform: translateY(10px);
  }
  .fade-up-leave-to {
    opacity: 0;
    transform: translateY(-10px);
  }
}
</style>
