<template>
  <div class="fullscreen-trigger" @click="toggle">
    <SvgIcon :icon="isFullscreen ? 'fullscreen-exit' : 'fullscreen'" />
  </div>
</template>

<script setup lang="ts">
import SvgIcon from "@/components/SvgIcon/index.vue";

const { isFullscreen, toggle } = useFullscreen();
</script>

<style lang="scss" scoped>
.fullscreen-trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}
</style>
