<template>
  <div class="fallback-container" :data-animate="transitionEnabled ? 'on' : 'off'">
    <img v-if="image" :src="image" class="fallback-image" />
    <component :is="fallbackIcon" v-else-if="fallbackIcon" class="fallback-image" />
    <div class="fallback-content">
      <slot v-if="$slots.title" name="title"></slot>
      <p v-else-if="titleText" class="fallback-title">
        {{ titleText }}
      </p>
      <slot v-if="$slots.describe" name="describe"></slot>
      <p v-else-if="descText" class="fallback-desc">
        {{ descText }}
      </p>
      <slot v-if="$slots.action" name="action"></slot>
      <el-button v-else-if="showBack" type="primary" size="large" @click="back">
        <el-icon><ArrowLeft /></el-icon>
        {{ t("common.button.back") }}
      </el-button>
      <el-button v-else-if="showRefresh" type="primary" size="large" @click="refresh">
        <el-icon><Refresh /></el-icon>
        {{ t("common.button.refresh") }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FallbackProps } from "./fallback";

import { computed, defineAsyncComponent } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";

import { ArrowLeft, Refresh } from "@element-plus/icons-vue";

import { preferencesManager } from '@go-wind/admin-element';

defineOptions({
  name: "Fallback",
});

const props = withDefaults(defineProps<FallbackProps>(), {
  description: "",
  homePath: "/",
  image: "",
  showBack: true,
  status: "coming-soon",
  title: "",
});

const { t } = useI18n();

// 全局动效开关（设计语言 §2.7）：关闭时不播进场/悬浮动画
const transitionEnabled = computed(
  () => preferencesManager.getPreferences().transition.enable,
);

const Icon401 = defineAsyncComponent(() => import("./icons/icon-401.vue"));
const Icon403 = defineAsyncComponent(() => import("./icons/icon-403.vue"));
const Icon404 = defineAsyncComponent(() => import("./icons/icon-404.vue"));
const Icon500 = defineAsyncComponent(() => import("./icons/icon-500.vue"));
const IconHello = defineAsyncComponent(() => import("./icons/icon-coming-soon.vue"));
const IconOffline = defineAsyncComponent(() => import("./icons/icon-offline.vue"));

const titleText = computed(() => {
  if (props.title) {
    return props.title;
  }

  switch (props.status) {
    case "401": {
      return t("core.fallback.unauthorized");
    }
    case "403": {
      return t("core.fallback.forbidden");
    }
    case "404": {
      return t("core.fallback.pageNotFound");
    }
    case "500": {
      return t("core.fallback.internalError");
    }
    case "coming-soon": {
      return t("core.fallback.comingSoon");
    }
    case "offline": {
      return t("core.fallback.offlineError");
    }
    default: {
      return "";
    }
  }
});

const descText = computed(() => {
  if (props.description) {
    return props.description;
  }
  switch (props.status) {
    case "401": {
      return t("core.fallback.unauthorizedDesc");
    }
    case "403": {
      return t("core.fallback.forbiddenDesc");
    }
    case "404": {
      return t("core.fallback.pageNotFoundDesc");
    }
    case "500": {
      return t("core.fallback.internalErrorDesc");
    }
    case "offline": {
      return t("core.fallback.offlineErrorDesc");
    }
    default: {
      return "";
    }
  }
});

const fallbackIcon = computed(() => {
  switch (props.status) {
    case "401": {
      return Icon401;
    }
    case "403": {
      return Icon403;
    }
    case "404": {
      return Icon404;
    }
    case "500": {
      return Icon500;
    }
    case "coming-soon": {
      return IconHello;
    }
    case "offline": {
      return IconOffline;
    }
    default: {
      return null;
    }
  }
});

const showBack = computed(() => {
  return props.status === "401" || props.status === "403" || props.status === "404";
});

const showRefresh = computed(() => {
  return props.status === "500" || props.status === "offline";
});

const { push } = useRouter();

// 返回首页
function back() {
  push(props.homePath);
}

function refresh() {
  // 网络异常页不能 reload（会停留在 /offline 死循环），
  // 而是跳转首页，让路由守卫重新判断网络状态
  if (props.status === "offline") {
    push(props.homePath);
  } else {
    location.reload();
  }
}
</script>

<style lang="scss">
// 全局样式：插画语义色板（SVG 填色统一走 --fb-*，禁用裸色值）。
// 主色锚点复用 update-css-variables 写在 <html> 上的 --primary-hsl（随偏好主色联动）。
.fallback-container {
  // 插画语义色板 · 亮色（纸墨日光版）
  --fb-primary: hsl(var(--primary-hsl));
  --fb-ink: #1f2937; // 线稿/数字 = 文字主色 gray-800
  --fb-paper: #ffffff; // 留白件（衬衫/纸面），两态同为白
  --fb-mist: #f2f2f2; // 最浅装饰件（云/山体）
  --fb-mist-2: #e4e4e4; // 次浅装饰件
  --fb-line: #cacaca; // 细线（地平线/弧线）
  --fb-navy: #3f3d56; // 深藏青物件
  --fb-navy-deep: #2f2e41; // 更深藏青物件
  --fb-skin: #a0616a; // 肤色，两态同值
  --fb-skin-light: #ffb6b6;
  --fb-coral: #ff6b6b; // 警示件（401 三角），两态同值——与 react Fallback.css 同步
  --fb-glow: color-mix(in srgb, var(--fb-primary) 14%, transparent);
}

// 插画语义色板 · 暗色（月夜版：装饰件压到画布上方一档，藏青物件提亮保形，
// 主色提亮一档保持插画可读——色板基调与 docs/design-language.md §2.3 一致）
.dark .fallback-container {
  --fb-primary: color-mix(in srgb, hsl(var(--primary-hsl)) 78%, white);
  --fb-ink: #f8fafc;
  --fb-mist: #182136;
  --fb-mist-2: #22304a;
  --fb-line: #33405e;
  --fb-navy: #46547a;
  --fb-navy-deep: #38456a;
  --fb-glow: color-mix(in srgb, var(--fb-primary) 26%, transparent);
}
</style>

<style lang="scss" scoped>
.fallback-container {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;

  // 整页底色取页面画布档（§2.2/§2.3）。改前 html/body/#app/本容器一路
  // rgba(0,0,0,0)，暗色下的底色由 color-scheme 的浏览器默认画布决定，不归设计语言管
  background: var(--el-bg-color-page);

  // 主色柔光晕：垫在插画后方，暗色下更明显
  &::before {
    content: "";
    position: absolute;
    top: 10%;
    left: 50%;
    z-index: 0;
    width: min(58vmin, 540px);
    aspect-ratio: 1;
    transform: translateX(-50%);
    background: radial-gradient(closest-side, var(--fb-glow), transparent 72%);
    border-radius: 50%;
    pointer-events: none;
  }
}

.fallback-image {
  position: relative;
  z-index: 1;
  width: 65%;
  max-width: 620px;
  height: auto;
  animation:
    fb-enter 0.45s cubic-bezier(0.22, 1, 0.36, 1) both,
    fb-float 6s ease-in-out 0.6s infinite;

  @media (min-width: 768px) {
    width: 50%;
  }

  @media (min-width: 1024px) {
    width: 42%;
  }

  svg {
    width: 100%;
    height: auto;
    max-height: 400px;
  }
}

.fallback-content {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.fallback-title {
  margin-top: 2rem;
  font-size: 1.5rem;
  color: var(--el-text-color-primary);
  animation: fb-enter 0.45s cubic-bezier(0.22, 1, 0.36, 1) 0.12s both;

  @media (min-width: 768px) {
    font-size: 1.875rem;
  }

  @media (min-width: 1024px) {
    font-size: 2.25rem;
  }
}

.fallback-desc {
  margin: 1rem 0;
  font-size: 1rem;
  color: var(--el-text-color-secondary);
  animation: fb-enter 0.45s cubic-bezier(0.22, 1, 0.36, 1) 0.2s both;

  @media (min-width: 768px) {
    font-size: 1.125rem;
  }

  @media (min-width: 1024px) {
    font-size: 1.25rem;
  }
}

.fallback-container :deep(.el-button) {
  padding: 0.625rem 1.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  border-radius: 9999px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  transition: all 0.2s;
  animation: fb-enter 0.45s cubic-bezier(0.22, 1, 0.36, 1) 0.28s both;

  .el-icon {
    margin-right: 0.25rem;
  }
}

// 进场：上浮淡入
@keyframes fb-enter {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

// 插画悬浮呼吸（与认证页品牌插画同款语言，幅度收敛）
@keyframes fb-float {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-12px);
  }
}

// 动效尊重全局开关（preferences.transition.enable）与系统减弱动效
.fallback-container[data-animate="off"] .fallback-image,
.fallback-container[data-animate="off"] .fallback-title,
.fallback-container[data-animate="off"] .fallback-desc,
.fallback-container[data-animate="off"] :deep(.el-button) {
  animation: none;
}

@media (prefers-reduced-motion: reduce) {
  .fallback-container .fallback-image,
  .fallback-container .fallback-title,
  .fallback-container .fallback-desc,
  .fallback-container :deep(.el-button) {
    animation: none;
  }
}
</style>
