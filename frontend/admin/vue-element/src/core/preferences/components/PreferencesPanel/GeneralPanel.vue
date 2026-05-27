<template>
  <div class="general-panel">
    <section class="general-section">
      <h3 class="section-title">{{ t("preferences.general.general") }}</h3>

      <div class="preference-item">
        <span>{{ t("preferences.general.language") }}</span>
        <ElSelect
          :model-value="preferences.app.locale"
          style="width: 200px"
          @change="
            (val: string | number | boolean) => updatePreferences({ app: { locale: val as any } })
          "
        >
          <ElOption
            v-for="opt in LANGUAGE_OPTIONS"
            :key="opt.value"
            :label="t(`preferences.${opt.label}`)"
            :value="opt.value"
          />
        </ElSelect>
      </div>

      <div class="preference-item">
        <span>{{ t("preferences.general.dynamicTitle") }}</span>
        <ElSwitch
          :model-value="preferences.app.dynamicTitle"
          @change="
            (val: string | number | boolean) =>
              updatePreferences({ app: { dynamicTitle: val as boolean } })
          "
        />
      </div>

      <div class="preference-item">
        <span>{{ t("preferences.general.watermark") }}</span>
        <ElSwitch
          :model-value="preferences.app.watermark"
          @change="
            (val: string | number | boolean) =>
              updatePreferences({ app: { watermark: val as boolean } })
          "
        />
      </div>

      <div class="preference-item">
        <span>{{ t("preferences.general.checkUpdates") }}</span>
        <ElSwitch
          :model-value="preferences.app.enableCheckUpdates"
          @change="
            (val: string | number | boolean) =>
              updatePreferences({ app: { enableCheckUpdates: val as boolean } })
          "
        />
      </div>
    </section>

    <section class="general-section">
      <h3 class="section-title">{{ t("preferences.general.animation") }}</h3>

      <div class="preference-item">
        <span>{{ t("preferences.general.progressBar") }}</span>
        <ElSwitch
          :model-value="preferences.transition.progress"
          @change="
            (val: string | number | boolean) =>
              updatePreferences({ transition: { progress: val as boolean } })
          "
        />
      </div>

      <div class="preference-item">
        <span>{{ t("preferences.general.loading") }}</span>
        <ElSwitch
          :model-value="preferences.transition.loading"
          @change="
            (val: string | number | boolean) =>
              updatePreferences({ transition: { loading: val as boolean } })
          "
        />
      </div>

      <div class="preference-item">
        <span>{{ t("preferences.general.pageTransition") }}</span>
        <ElSwitch
          :model-value="preferences.transition.enable"
          @change="
            (val: string | number | boolean) =>
              updatePreferences({ transition: { enable: val as boolean } })
          "
        />
      </div>

      <div class="transition-animations" :class="{ disabled: !preferences.transition.enable }">
        <div class="transition-grid">
          <div
            v-for="option in TRANSITION_OPTIONS"
            :key="option.value"
            class="transition-wrapper"
            @click="
              preferences.transition.enable &&
              updatePreferences({ transition: { name: option.value } })
            "
          >
            <div
              class="transition-item"
              :class="{ active: preferences.transition.name === option.value }"
            >
              <div class="transition-preview">
                <div class="animation-box" :class="option.value" />
              </div>
            </div>
            <span class="transition-label">{{ t(`preferences.${option.label}`) }}</span>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ElSwitch, ElSelect, ElOption } from "element-plus";
import { useI18n } from "@/i18n";
import { preferences, updatePreferences } from "../../index";

const { t } = useI18n();

const TRANSITION_OPTIONS = [
  { label: "general.transitions.fade", value: "fade" },
  { label: "general.transitions.fadeDown", value: "fade-down" },
  { label: "general.transitions.fadeSlide", value: "fade-slide" },
  { label: "general.transitions.fadeUp", value: "fade-up" },
];

const LANGUAGE_OPTIONS = [
  { label: "general.languages.zhCN", value: "zh-CN" },
  { label: "general.languages.enUS", value: "en-US" },
];
</script>

<style scoped lang="scss">
.general-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.general-section {
  display: flex;
  flex-direction: column;
  gap: 8px;

  .section-title {
    margin: 0 0 8px;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
}

.preference-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
  font-size: 13px;
}

.transition-animations {
  &.disabled {
    opacity: 0.4;
    pointer-events: none;
  }
}

.transition-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.transition-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  cursor: pointer;
}

.transition-item {
  width: 100%;
  aspect-ratio: 16 / 9;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;

  &:hover {
    border-color: var(--el-color-primary);
  }

  &.active {
    border: 2px solid var(--el-color-primary);
  }
}

.transition-preview {
  width: 80%;
  height: 60%;
  background: var(--el-fill-color);
  border-radius: 4px;
  overflow: hidden;
}

.transition-label {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

/* 过渡动画预览效果 */
.animation-box {
  width: 100%;
  height: 100%;
  background: var(--el-color-primary-light-7);
  border-radius: 3px;
}

/* fade 动画 */
.animation-box.fade {
  animation: fadePreview 2s ease-in-out infinite;
}

@keyframes fadePreview {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0;
  }
}

/* fade-down 动画 */
.animation-box.fade-down {
  animation: fadeDownPreview 2s ease-in-out infinite;
}

@keyframes fadeDownPreview {
  0%,
  100% {
    opacity: 1;
    transform: translateY(0);
  }
  50% {
    opacity: 0;
    transform: translateY(-10px);
  }
}

/* fade-slide 动画 */
.animation-box.fade-slide {
  animation: fadeSlidePreview 2s ease-in-out infinite;
}

@keyframes fadeSlidePreview {
  0%,
  100% {
    opacity: 1;
    transform: translateX(0);
  }
  50% {
    opacity: 0;
    transform: translateX(-10px);
  }
}

/* fade-up 动画 */
.animation-box.fade-up {
  animation: fadeUpPreview 2s ease-in-out infinite;
}

@keyframes fadeUpPreview {
  0%,
  100% {
    opacity: 1;
    transform: translateY(0);
  }
  50% {
    opacity: 0;
    transform: translateY(10px);
  }
}
</style>
