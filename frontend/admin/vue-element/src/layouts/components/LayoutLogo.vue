<template>
  <div class="logo">
    <transition enter-active-class="animate__animated animate__fadeInLeft">
      <router-link :key="+collapse" class="wh-full flex-center" to="/">
        <img :src="logo" class="logo-icon" />
        <span v-if="!collapse" class="title">
          {{ preferences.app.name }}
        </span>
      </router-link>
    </transition>
  </div>
</template>

<script lang="ts" setup>
import { preferences } from "@/core/preferences";
import logo from "@/assets/images/logo.png";

defineProps({
  collapse: {
    type: Boolean,
    required: true,
  },
});
</script>

<style lang="scss" scoped>
.logo {
  width: 100%;
  height: 64px;
  background-color: $sidebar-logo-background;
  border-bottom: 1px solid var(--sidebar-logo-border-color);

  .logo-icon {
    width: 32px;
    height: 32px;
    flex-shrink: 0;
  }

  .title {
    flex-shrink: 0;
    margin-left: 12px;
    padding-right: 8px;
    font-size: 16px;
    font-weight: 700;
    letter-spacing: 0.5px;
    color: $sidebar-logo-text-color;
    white-space: nowrap;
  }
}
</style>

<style lang="scss">
// 顶部布局和混合布局的特殊处理
.layout-top,
.layout-mix {
  .logo {
    background-color: transparent !important;

    .title {
      color: var(--menu-text);
    }
  }
}

// 宽屏时：openSidebar 状态下显示完整Logo+文字
.openSidebar {
  &.layout-top .layout__header-left .logo,
  &.layout-mix .layout__header-logo .logo {
    width: $sidebar-width; // 210px，显示logo+文字
  }
}

// 窄屏时：hideSidebar 状态下只显示Logo图标
.hideSidebar {
  &.layout-top .layout__header-left .logo,
  &.layout-mix .layout__header-logo .logo {
    width: $sidebar-width-collapsed; // 54px，只显示logo
  }

  // 隐藏文字，只显示图标
  .logo .title {
    display: none;
  }
}
</style>
