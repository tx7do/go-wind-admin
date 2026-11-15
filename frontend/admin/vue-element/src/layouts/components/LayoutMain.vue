<template>
  <section class="app-main" :class="mainClass" :style="{ height: appMainHeight }">
    <router-view>
      <template #default="{ Component, route: slotRoute }">
        <!-- mode="out-in" 与 vue-router 5 懒加载路由组合有白屏竞态：
             懒 chunk 解析完成时旧页被直接移除而新页永不挂载（2026-09 实测定位）。
             改用同帧交叉淡入淡出，动画时长不变。 -->
        <transition :name="transitionName" :duration="150">
          <keep-alive :include="cachedViews">
            <component
              :is="currentComponent(Component, slotRoute)"
              :key="
                contentRefreshKey
                  ? `${slotRoute.fullPath}__${contentRefreshKey}`
                  : slotRoute.fullPath
              "
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
import { type RouteLocationNormalized, useRoute } from "vue-router";
import { useTagsViewStore } from "./useTagsViewStore";
import SvgIcon from "@/components/SvgIcon/index.vue";
import { preferences, usePreferences } from "@go-wind/admin-element";
import variables from "@/styles/variables.module.scss";
import Error404 from "@/pages/core/error/404.vue";

const { cachedViews } = toRefs(useTagsViewStore());
const { tabbarPreferences } = usePreferences();

// 注入刷新状态
const contentRefreshing = inject<Ref<boolean>>("contentRefreshing", ref(false));

// 刷新 key：每次刷新递增，强制组件重建
const contentRefreshKey = inject<Ref<number>>("contentRefreshKey", ref(0));

// 当前组件
const wrapperMap = new Map<string, ComponentWrapper>();

// 刷新时清理 wrapperMap，确保组件完全重建
watch(contentRefreshing, (val) => {
  if (val) {
    wrapperMap.clear();
  }
});
// vue-router 5 的 slot Component 是预构建 vnode（携带 ref 与 routeProps），
// 不能在闭包里固化首次渲染的引用——否则后续渲染克隆的是过期 vnode，
// 与 out-in transition/keep-alive 组合会在多次导航后白屏（router-view 塌空）。
// wrapper 仅承担「以 fullPath 作为组件名供 keep-alive include 匹配」职责，
// 每次渲染同步最新 Component。
interface ComponentWrapper {
  name: string;
  current: Component;
  render: () => VNode;
}

const currentComponent = (component: Component, route: RouteLocationNormalized) => {
  if (!component) return;

  const { fullPath: componentName } = route; // 使用路由路径作为组件名称
  let wrapper = wrapperMap.get(componentName);

  if (!wrapper) {
    const created: ComponentWrapper = {
      name: componentName,
      current: component,
      render: () => {
        try {
          return h(created.current);
        } catch (error) {
          console.error(`Error rendering component for route: ${componentName}`, error);
          return h(Error404);
        }
      },
    };
    wrapperMap.set(componentName, created);
    wrapper = created;
  }

  // 关键：每次渲染都同步 slot 给到的最新 vnode/组件，绝不能复用旧引用
  wrapper.current = component;

  // 添加组件数量限制
  if (wrapperMap.size > 100) {
    const firstKey = wrapperMap.keys().next().value;
    if (firstKey) {
      wrapperMap.delete(firstKey);
    }
  }

  return h(wrapper);
};

// 页面高度
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
    transition: opacity 0.15s ease;
  }
  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }

  /* fade-slide */
  .fade-slide-leave-active,
  .fade-slide-enter-active {
    transition:
      opacity 0.15s ease,
      transform 0.15s ease;
  }
  .fade-slide-enter-from {
    opacity: 0;
    transform: translateX(-8px);
  }
  .fade-slide-leave-to {
    opacity: 0;
    transform: translateX(8px);
  }

  /* fade-down */
  .fade-down-leave-active,
  .fade-down-enter-active {
    transition:
      opacity 0.15s ease,
      transform 0.15s ease;
  }
  .fade-down-enter-from {
    opacity: 0;
    transform: translateY(-8px);
  }
  .fade-down-leave-to {
    opacity: 0;
    transform: translateY(8px);
  }

  /* fade-up */
  .fade-up-leave-active,
  .fade-up-enter-active {
    transition:
      opacity 0.15s ease,
      transform 0.15s ease;
  }
  .fade-up-enter-from {
    opacity: 0;
    transform: translateY(8px);
  }
  .fade-up-leave-to {
    opacity: 0;
    transform: translateY(-8px);
  }
}
</style>
