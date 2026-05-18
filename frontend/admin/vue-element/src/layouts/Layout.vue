<template>
  <div class="layout-wrapper">
    <component :is="currentLayoutComponent" />
    <Settings v-if="showSettings" />
  </div>
</template>

<script setup lang="ts">
import { useRoute } from "vue-router";
import { useLayout } from "./useLayout";

import LeftLayout from "./LeftLayout.vue";
import TopLayout from "./TopLayout.vue";
import MixLayout from "./MixLayout.vue";
import Settings from "./components/LayoutSettings.vue";

const route = useRoute();
const { currentLayout, showSettings } = useLayout();

// 设置面板可见性（全局状态）
const settingsVisible = ref(false);

// 提供给子组件使用
provide("settingsVisible", settingsVisible);

const currentLayoutComponent = computed(() => {
  const override = route.meta?.layout as LayoutType | undefined;
  const layout = override ?? currentLayout.value;

  switch (layout) {
    case "header-nav":
      return TopLayout;
    case "mixed-nav":
      return MixLayout;
    default:
      return LeftLayout;
  }
});
</script>

<style lang="scss" scoped>
.layout-wrapper {
  width: 100%;
  height: 100%;
}
</style>
