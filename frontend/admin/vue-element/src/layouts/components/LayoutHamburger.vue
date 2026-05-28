<template>
  <div class="hamburger-wrapper" @click="toggleClick">
    <SvgIcon
      icon="collapse"
      :class="[{ hamburger: true, 'is-active': isActive }, hamburgerClass]"
    />
  </div>
</template>

<script setup lang="ts">
import { usePreferences } from "@/core/preferences";
import SvgIcon from "@/components/SvgIcon/index.vue";

defineProps({
  isActive: { type: Boolean, required: true },
});

const emit = defineEmits(["toggleClick"]);

const { appPreferences, isDark } = usePreferences();

const hamburgerClass = computed(() => {
  // 如果暗黑主题
  if (isDark) {
    return "hamburger--white";
  }

  // 如果是混合布局，使用白色图标
  if (appPreferences.value.layout === "mixed-nav") {
    return "hamburger--white";
  }

  // 默认返回空字符串
  return "";
});

function toggleClick() {
  emit("toggleClick");
}
</script>

<style scoped lang="scss">
.hamburger-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 15px;
  cursor: pointer;

  .hamburger {
    vertical-align: middle;
    transform: scaleX(-1);
    transition: transform 0.3s ease;

    &--white {
      color: #fff;
    }

    &.is-active {
      transform: scaleX(1);
    }
  }
}
</style>
