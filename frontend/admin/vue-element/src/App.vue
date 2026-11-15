<template>
  <el-config-provider :locale="locale" :size="size">
    <!-- 开启水印-->
    <el-watermark
      :font="{ color: fontColor }"
      :content="showWatermark ? watermarkContent : ''"
      :z-index="9999"
      class="wh-full"
    >
      <router-view />
    </el-watermark>
    <!-- Vue Query Devtools（仅开发环境） -->
    <TanstackQueryDevtools v-if="isDev" :button-position="'bottom-left'" />
  </el-config-provider>
</template>

<script setup lang="ts">
import { TanstackQueryDevtools } from "@/plugins/vue-query";
import { APP_PREFIX } from "@/constants";
import { preferences, usePreferences } from "@go-wind/admin-element";

const { isDark, getElementPlusLocale } = usePreferences();

const isDev = import.meta.env.DEV;

const locale = computed(() => getElementPlusLocale.value);
const showWatermark = computed(() => preferences.app.watermark);
const watermarkContent = APP_PREFIX;

/**
 * 组件尺寸：根据紧凑模式动态切换
 * compact=true → small（紧凑）
 * compact=false → default（默认）
 */
const size = computed(() => (preferences.app.compact ? "small" : "default"));

// 明亮/暗黑主题水印字体颜色适配
const fontColor = computed(() => {
  return isDark.value ? "rgba(255, 255, 255, .15)" : "rgba(0, 0, 0, .15)";
});
</script>
