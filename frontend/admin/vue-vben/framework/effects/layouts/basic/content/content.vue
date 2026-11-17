<script lang="ts" setup>
import type {
  RouteLocationNormalizedLoaded,
  RouteLocationNormalizedLoadedGeneric,
} from 'vue-router';

import { type VNode } from 'vue';
import { RouterView } from 'vue-router';

import { preferences, usePreferences } from '@vben/preferences';
import { storeToRefs, useTabbarStore } from '@vben/stores';

import { IFrameRouterView } from '../../iframe';

defineOptions({ name: 'LayoutContent' });

const tabbarStore = useTabbarStore();
const { keepAlive } = usePreferences();

const { getCachedTabs, getExcludeCachedTabs, renderRouteView } =
  storeToRefs(tabbarStore);

// 页面切换动画
function getTransitionName(_route: RouteLocationNormalizedLoaded) {
  // 如果偏好设置未设置，则不使用动画
  const { tabbar, transition } = preferences;
  const transitionName = transition.name;
  if (!transitionName || !transition.enable) {
    return;
  }

  // 标签页未启用或者未开启缓存，则使用全局配置动画
  if (!tabbar.enable || !keepAlive) {
    return transitionName;
  }

  // 如果页面已经加载过，则不使用动画
  // if (route.meta.loaded) {
  //   return;
  // }
  // 已经打开且已经加载过的页面不使用动画
  // const inTabs = getCachedTabs.value.includes(route.name as string);

  // return inTabs && route.meta.loaded ? undefined : transitionName;
  return transitionName;
}

/**
 * 转换组件，自动添加 name
 * @param component
 */
function transformComponent(
  component: VNode,
  route: RouteLocationNormalizedLoadedGeneric,
) {
  // 组件视图未找到，如果有设置后备视图，则返回后备视图，如果没有，则抛出错误
  if (!component) {
    console.error(
      'Component view not found，please check the route configuration',
    );
    return undefined;
  }

  const routeName = route.name as string;
  // 如果组件没有 name，则直接返回
  if (!routeName) {
    return component;
  }
  const componentName = (component?.type as any)?.name;

  // 已经设置过 name，则直接返回
  if (componentName) {
    return component;
  }

  // componentName 与 routeName 一致，则直接返回
  if (componentName === routeName) {
    return component;
  }

  // 设置 name
  component.type ||= {};
  (component.type as any).name = routeName;

  return component;
}
</script>

<template>
  <div class="relative h-full">
    <IFrameRouterView />
    <RouterView v-slot="{ Component, route }">
      <!-- 显式 duration：Transition 结束判定走 Vue 内置计时器兜底，不依赖
           transitionend/animationend 事件——部分内嵌 webview（软件渲染路径）
           不派发这两类事件，纯 CSS 判定会让路由视图永久冻结在 leave 阶段，
           换页后 URL 已变而画面停在旧页。毫秒值与 fade-slide 的 CSS 时长
           （0.3s）对齐。 -->
      <Transition
        :duration="300"
        :name="getTransitionName(route)"
        appear
        mode="out-in"
      >
        <KeepAlive
          v-if="keepAlive"
          :exclude="getExcludeCachedTabs"
          :include="getCachedTabs"
        >
          <component
            :is="transformComponent(Component, route)"
            v-if="renderRouteView"
            v-show="!route.meta.iframeSrc"
            :key="route.fullPath"
          />
        </KeepAlive>
        <component
          :is="Component"
          v-else-if="renderRouteView"
          :key="route.fullPath"
        />
      </Transition>
    </RouterView>
  </div>
</template>
