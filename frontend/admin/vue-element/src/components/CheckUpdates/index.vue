<template>
  <ElDialog
    :model-value="open"
    :title="t('preferences.general.checkUpdatesTitle')"
    width="420px"
    align-center
    :show-close="false"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
  >
    <span>{{ t("preferences.general.checkUpdatesDescription") }}</span>
    <template #footer>
      <ElButton @click="handleDismiss">
        {{ t("common.button.cancel") }}
      </ElButton>
      <ElButton type="primary" @click="handleRefresh">
        {{ t("common.button.refresh") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from "vue";
import { useI18n } from "vue-i18n";
import { ElButton, ElDialog } from "element-plus";
import { usePreferences } from "@go-wind/admin-element";

/**
 * 热更新检测（部署新版本后提示用户刷新页面）。
 * 行为基准是 react 端 MainLayout/components/CheckUpdates（移植自 vben
 * framework/effects/layouts/widgets/check-updates）：每 checkUpdatesInterval
 * 分钟对 BASE_URL（index.html）发一次 HEAD 请求，以 etag / last-modified
 * 响应头作为版本指纹，指纹变化即弹窗提示刷新。
 * 本地开发（localhost/127.0.0.1）不检测；偏好面板「通用 → 检查更新」开关门控。
 */
const { t } = useI18n();
const { appPreferences } = usePreferences();

const open = ref(false);
let lastVersionTag = "";
let currentVersionTag = "";
let timer: ReturnType<typeof setInterval> | undefined;
let checking = false;

async function fetchVersionTag(): Promise<string | null> {
  const hostname = window.location.hostname;
  if (hostname === "localhost" || hostname === "127.0.0.1") {
    return null;
  }
  try {
    const response = await fetch(import.meta.env.BASE_URL || "/", {
      cache: "no-cache",
      method: "HEAD",
    });
    return response.headers.get("etag") || response.headers.get("last-modified");
  } catch (error) {
    // 必须带出原始错误对象：HEAD 失败（断网/代理异常）不该静默
    console.error("[CheckUpdates] fetch version tag failed:", error);
    return null;
  }
}

async function checkForUpdates() {
  const versionTag = await fetchVersionTag();
  if (!versionTag) {
    return;
  }
  // 首次运行只记基线，不提示更新
  if (!lastVersionTag) {
    lastVersionTag = versionTag;
    return;
  }
  if (lastVersionTag !== versionTag) {
    stop();
    currentVersionTag = versionTag;
    open.value = true;
  }
}

function start() {
  stop();
  const interval = appPreferences.value.checkUpdatesInterval;
  // 间隔 <= 0 视为关闭轮询
  if (interval <= 0) {
    return;
  }
  timer = setInterval(() => {
    void checkForUpdates();
  }, interval * 60 * 1000);
}

function stop() {
  if (timer !== undefined) {
    clearInterval(timer);
    timer = undefined;
  }
}

// 页面隐藏时暂停、重新可见时立即补查一次再续轮询
// （vben 同款语义：页签在后台时不白发请求）
function handleVisibilityChange() {
  if (document.hidden) {
    stop();
    return;
  }
  if (!checking) {
    checking = true;
    checkForUpdates()
      .finally(() => {
        checking = false;
        start();
      });
  }
}

watch(
  [() => appPreferences.value.enableCheckUpdates, () => appPreferences.value.checkUpdatesInterval],
  ([enabled]) => {
    stop();
    document.removeEventListener("visibilitychange", handleVisibilityChange);
    if (!enabled) {
      // 关闭时若弹窗还开着，一并收起
      open.value = false;
      return;
    }
    document.addEventListener("visibilitychange", handleVisibilityChange);
    start();
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  stop();
  document.removeEventListener("visibilitychange", handleVisibilityChange);
});

function handleDismiss() {
  // 取消后轮询保持停止，回到页签（visibilitychange）时会再次检测并重弹
  open.value = false;
}

function handleRefresh() {
  // 记下已确认的指纹再刷新：万一刷新被浏览器策略打断，也不会立即再弹
  lastVersionTag = currentVersionTag;
  window.location.reload();
}
</script>
