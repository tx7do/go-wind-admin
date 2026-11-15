<template>
  <div v-if="visible" class="lock-screen">
    <div v-if="!showUnlockForm" class="lock-screen__clock-view">
      <div class="lock-screen__unlock-hint" @click="toggleForm">
        <SvgIcon icon="lucide:lock-keyhole" :size="20" />
        <span>{{ t("common.lockScreen.unlock") }}</span>
      </div>
      <div class="lock-screen__clock">
        {{ hour }}:{{ minute }}
        <span class="lock-screen__meridiem">{{ meridiem }}</span>
      </div>
    </div>

    <div v-else class="lock-screen__form-view" @keydown.enter.prevent="handleSubmit">
      <div class="lock-screen__form-inner">
        <el-avatar :size="80" :src="avatar" />
        <el-input
          v-model="password"
          class="lock-screen__input"
          type="password"
          show-password
          autocomplete="off"
          size="large"
          :placeholder="t('common.lockScreen.placeholder')"
          @input="error = ''"
        />
        <p v-if="error" class="lock-screen__error">{{ error }}</p>
        <el-button class="lock-screen__btn" type="primary" size="large" @click="handleSubmit">
          {{ t("common.lockScreen.entry") }}
        </el-button>
        <el-button class="lock-screen__btn" size="large" @click="emit('toLogin')">
          {{ t("common.lockScreen.backToLogin") }}
        </el-button>
        <el-button class="lock-screen__btn" text size="large" @click="toggleForm">
          {{ t("common.button.back") }}
        </el-button>
      </div>
    </div>

    <div class="lock-screen__date">{{ dateText }}</div>
  </div>
</template>

<script setup lang="ts">
import { useLockStore } from '@/stores';
/**
 * 全屏锁屏遮罩（行为基准：vue-vben framework/effects/layouts/widgets/lock-screen）
 *
 * 锁屏只是界面遮罩，不是认证边界：解锁成功仅撤掉遮罩，真正的鉴权仍由后端令牌负责。
 */
import { computed, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import SvgIcon from "@/components/SvgIcon/index.vue";
import { preferences } from "@go-wind/admin-element";

const props = withDefaults(
  defineProps<{
    avatar?: string;
  }>(),
  { avatar: "" }
);

const emit = defineEmits<{ toLogin: [] }>();

const { t } = useI18n();
const lockStore = useLockStore();

const visible = computed(() => preferences.widget.lockScreen && lockStore.isLockScreen);

const now = ref(new Date());
const showUnlockForm = ref(false);
const password = ref("");
const error = ref("");

const hour = computed(() => String(now.value.getHours()).padStart(2, "0"));
const minute = computed(() => String(now.value.getMinutes()).padStart(2, "0"));
const meridiem = computed(() => (now.value.getHours() < 12 ? "AM" : "PM"));
const dateText = computed(() =>
  now.value.toLocaleDateString(preferences.app.locale === "zh-CN" ? "zh-CN" : "en-US", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    weekday: "long",
  })
);

let timer: ReturnType<typeof setInterval> | undefined;

watch(
  visible,
  (locked) => {
    if (timer) {
      clearInterval(timer);
      timer = undefined;
    }
    if (!locked) return;
    // 遮罩弹出时从当前时刻起算，避免沿用上一次挂起的旧时间
    now.value = new Date();
    timer = setInterval(() => {
      now.value = new Date();
    }, 1000);
  },
  { immediate: true }
);

onUnmounted(() => {
  if (timer) clearInterval(timer);
});

function toggleForm() {
  showUnlockForm.value = !showUnlockForm.value;
  password.value = "";
  error.value = "";
}

function handleSubmit() {
  if (!password.value) return;
  if (password.value === lockStore.lockScreenPassword) {
    lockStore.unlockScreen();
  } else {
    error.value = t("common.lockScreen.errorTip");
  }
}
</script>

<style lang="scss" scoped>
.lock-screen {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background-color: var(--el-bg-color);

  &__clock-view {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  &__unlock-hint {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    margin-bottom: 24px;
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    cursor: pointer;
    transition: color 0.2s ease;

    &:hover {
      color: var(--gowind-primary-text);
    }
  }

  &__clock {
    position: relative;
    font-size: 160px;
    font-weight: 700;
    line-height: 1;
    color: var(--el-text-color-primary);
  }

  &__meridiem {
    position: absolute;
    top: 12px;
    margin-left: 16px;
    font-size: 24px;
    font-weight: 500;
    color: var(--el-text-color-secondary);
  }

  &__form-view {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  &__form-inner {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 300px;
  }

  &__input {
    width: 100%;
    margin-top: 24px;
  }

  &__error {
    width: 100%;
    margin: 8px 0 0;
    font-size: 12px;
    color: var(--el-color-danger);
  }

  &__btn {
    width: 100%;
    margin-top: 8px;

    &:first-of-type {
      margin-top: 16px;
    }
  }

  &__date {
    position: absolute;
    bottom: 24px;
    width: 100%;
    font-size: 24px;
    text-align: center;
    color: var(--el-text-color-secondary);
  }
}
</style>
