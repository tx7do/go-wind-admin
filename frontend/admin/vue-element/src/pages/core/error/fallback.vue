<template>
  <div class="fallback-container">
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
// 全局样式，不使用 scoped，让 SVG 可以继承 CSS 变量
.fallback-container {
  // 定义主题色变量，供 SVG 使用
  --fallback-primary: #006be6; // 规范主色 hsl(212 100% 45%)
  --fallback-foreground: #ffffff;
}

// 深色主题下的颜色
.dark .fallback-container {
  --fallback-primary: hsl(212, 100%, 60%); // 暗色下主色提亮一档保持插画可读
  --fallback-foreground: #f8fafc; // 规范暗色主文字
}
</style>

<style lang="scss" scoped>
.fallback-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.fallback-image {
  width: 65%;
  max-width: 620px;
  height: auto;

  @media (min-width: 768px) {
    width: 50%;
  }

  @media (min-width: 1024px) {
    width: 42%;
  }

  svg,
  svg * {
    fill: revert !important;
    stroke: revert !important;
  }
}

.fallback-content {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.fallback-title {
  margin-top: 2rem;
  font-size: 1.5rem;
  color: var(--el-text-color-primary);

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

  .el-icon {
    margin-right: 0.25rem;
  }
}
</style>
