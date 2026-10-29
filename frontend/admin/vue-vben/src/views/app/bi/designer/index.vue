<template>
  <div class="bi-designer-page">
    <BiDesigner :demo="true" :initial-schema="loadedSchema" @change="onSchemaChange" />
  </div>
</template>

<script setup lang="ts">
/**
 * BI 设计器页面（vben 端）：@go-wind/bi-vue 的 BiDesigner 画布。
 * 与 dashboard 页共用 localStorage 键（bi:schema:default）：这里保存的 schema
 * 就是仪表盘页渲染的那一份。后续接后端 dashboard CRUD 时两页一起换。
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { BiDesigner, provideBiEnv, setBiLocale } from '@go-wind/bi-vue';
import '@go-wind/bi-vue/style.css';
import type { BiDashboardSchema } from '@go-wind/bi-core';
import { usePreferences } from '@vben/preferences';

defineOptions({ name: 'BiDesignerPage' });

/** 与 vue-element / react 端同一键 */
const STORAGE_KEY = 'bi:schema:default';
/** 落盘防抖：画布拖动会高频触发 change */
const SAVE_DEBOUNCE_MS = 1000;

const { locale: appLocale } = usePreferences();
const biLocale = computed(() =>
  (appLocale.value ?? 'zh-CN').startsWith('zh') ? 'zh-CN' : 'en-US',
);

provideBiEnv({});

const loadedSchema = ref<Partial<BiDashboardSchema> | undefined>(undefined);

onMounted(() => {
  setBiLocale(biLocale.value);
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved) {
    try {
      loadedSchema.value = JSON.parse(saved);
    } catch (error) {
      // 脏数据不阻塞页面：记原始错误，回落到内置演示 schema
      console.error('[bi-designer] 本地 schema 解析失败，改用演示 schema', error);
      loadedSchema.value = undefined;
    }
  }
});

watch(biLocale, (locale) => setBiLocale(locale));

let saveTimer: ReturnType<typeof setTimeout> | null = null;

function onSchemaChange(schema: BiDashboardSchema) {
  if (saveTimer) clearTimeout(saveTimer);
  saveTimer = setTimeout(() => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(schema));
  }, SAVE_DEBOUNCE_MS);
}

onBeforeUnmount(() => {
  if (saveTimer) clearTimeout(saveTimer);
  saveTimer = null;
});
</script>

<style scoped>
.bi-designer-page {
  height: 100%;
  padding: 12px 16px;
  box-sizing: border-box;
}
</style>
