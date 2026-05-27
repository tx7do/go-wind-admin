<template>
  <section class="app-main" :style="{ height: appMainHeight }">
    <router-view>
      <template #default="{ Component, route }">
        <transition :name="transitionName" mode="out-in">
          <keep-alive :include="cachedViews">
            <component :is="currentComponent(Component, route)" :key="route.fullPath" />
          </keep-alive>
        </transition>
      </template>
    </router-view>

    <!-- 返回顶部按钮 -->
    <el-backtop target=".app-main">
      <div class="i-svg:backtop w-6 h-6" />
    </el-backtop>
  </section>
</template>

<script setup lang="ts">
import { type RouteLocationNormalized } from "vue-router";
import { useTagsViewStore } from "@/stores";
import { preferences, usePreferences } from "@/core/preferences";
import variables from "@/styles/variables.module.scss";
import Error404 from "@/views/core/error/404.vue";

const { cachedViews } = toRefs(useTagsViewStore());
const { tabbarPreferences } = usePreferences();

// 当前组件
const wrapperMap = new Map<string, Component>();
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
</script>

<style lang="scss" scoped>
.app-main {
  position: relative;
  overflow-y: auto;
  scrollbar-gutter: stable;
  background-color: var(--el-bg-color-page);
  width: 100%;
  min-width: 0;
}
</style>

<style lang="scss">
/* 页面过渡动画 - 不能使用 scoped，否则类名无法应用到 transition 子元素 */
.app-main {
  /* fade */
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease-in-out;
  }
  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }

  /* fade-slide */
  .fade-slide-leave-active,
  .fade-slide-enter-active {
    transition: all 0.3s;
  }
  .fade-slide-enter-from {
    opacity: 0;
    transform: translateX(-30px);
  }
  .fade-slide-leave-to {
    opacity: 0;
    transform: translateX(30px);
  }

  /* fade-scale */
  .fade-scale-leave-active,
  .fade-scale-enter-active {
    transition: all 0.28s;
  }
  .fade-scale-enter-from {
    opacity: 0;
    transform: scale(1.2);
  }
  .fade-scale-leave-to {
    opacity: 0;
    transform: scale(0.8);
  }

  /* fade-down */
  .fade-down-leave-active,
  .fade-down-enter-active {
    transition: all 0.3s;
  }
  .fade-down-enter-from {
    opacity: 0;
    transform: translateY(-30px);
  }
  .fade-down-leave-to {
    opacity: 0;
    transform: translateY(30px);
  }

  /* fade-up */
  .fade-up-leave-active,
  .fade-up-enter-active {
    transition: all 0.3s;
  }
  .fade-up-enter-from {
    opacity: 0;
    transform: translateY(30px);
  }
  .fade-up-leave-to {
    opacity: 0;
    transform: translateY(-30px);
  }
}
</style>
