<template>
  <div class="layout-wrapper">
    <component :is="currentLayoutComponent" />
    <PreferencesPanel :open="settingsVisible" @close="settingsVisible = false" />
    <LockScreen :avatar="avatar" @to-login="handleLockToLogin" />
    <!-- 热更新检测：组件内按 preferences.app.enableCheckUpdates 自行门控 -->
    <CheckUpdates />
  </div>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from "vue-router";

import { PreferencesPanel } from '@go-wind/admin-element';
import { LayoutType, preferences } from "@go-wind/admin-element";
import { useAuth } from "@go-wind/admin-element";
import { useAppUserStore } from '@go-wind/admin-element';
import { useLockStore } from '@/stores';

import { useLayout } from "./useLayout";

import LeftLayout from "./LeftLayout.vue";
import TopLayout from "./TopLayout.vue";
import MixLayout from "./MixLayout.vue";
import LockScreen from "@/components/LockScreen/index.vue";
import CheckUpdates from "@/components/CheckUpdates/index.vue";

const route = useRoute();
const router = useRouter();
const { currentLayout } = useLayout();
const userStore = useAppUserStore();
const authStore = useAuth();
const lockStore = useLockStore();

const avatar = computed(
  () => userStore.userInfo?.avatar || preferences.app.defaultAvatar || "/default-avatar.png"
);

/**
 * 锁屏遮罩上的「返回登录」：必须先撤遮罩——logout 不清 lock-storage，
 * 否则重新登录后遮罩依旧在。
 */
async function handleLockToLogin() {
  lockStore.unlockScreen();
  await authStore.logout();
  router.push(`/login?redirect=${route.fullPath}`);
}

// 设置面板可见性（全局状态）
const settingsVisible = ref(false);

// 内容区刷新状态（true 表示正在刷新，用于清理 wrapperMap）
const contentRefreshing = ref(false);

// 刷新 key：每次刷新递增，改变组件 key 强制重建
const contentRefreshKey = ref(0);

// 提供给子组件使用
provide("settingsVisible", settingsVisible);
provide("contentRefreshing", contentRefreshing);
provide("contentRefreshKey", contentRefreshKey);

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
