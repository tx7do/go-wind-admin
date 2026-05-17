<template>
  <el-dropdown trigger="click" @command="handleLanguageChange">
    <div class="i-svg:language" :class="size" />
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="item in langOptions"
          :key="item.value"
          :disabled="appStore.language === item.value"
          :command="item.value"
        >
          {{ item.label }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { useAppStore } from "@/store";
import type { SupportedLanguagesType } from "@/i18n/types";

defineProps({
  size: {
    type: String,
    required: false,
  },
});

const langOptions: Array<{ label: string; value: SupportedLanguagesType }> = [
  { label: "中文", value: "zh-cn" },
  { label: "English", value: "en" },
];

const appStore = useAppStore();
const { locale, t } = useI18n();

/**
 * 处理语言切换
 *
 * @param lang  语言（zh-cn、en）
 */
async function handleLanguageChange(lang: SupportedLanguagesType) {
  locale.value = lang;
  await appStore.changeLanguage(lang);

  ElMessage.success(t("common.langSelect.message.success"));
}
</script>
