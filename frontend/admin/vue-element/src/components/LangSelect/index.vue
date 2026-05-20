<template>
  <el-dropdown trigger="click" @command="handleLanguageChange">
    <div class="i-svg:language" :class="size" />
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="item in availableLanguages"
          :key="item.code"
          :disabled="locale === item.code"
          :command="item.code"
        >
          {{ item.name }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { preferencesManager } from "@/core/preferences";
import { loadLocaleMessages } from "@/i18n";

defineProps({
  size: {
    type: String,
    required: false,
  },
});

const { locale, t } = useI18n();

// 可用语言列表
const availableLanguages = [
  { code: "zh-CN" as SupportedLanguagesType, name: "简体中文", locale: "zh-CN" },
  { code: "en-US" as SupportedLanguagesType, name: "English", locale: "en-US" },
];

/**
 * 处理语言切换
 *
 * @param lang  语言（zh-cn、en-US）
 */
async function handleLanguageChange(lang: SupportedLanguagesType) {
  // 更新 i18n
  await loadLocaleMessages(lang);

  // 更新 HTML lang 属性
  document.documentElement.lang = lang;

  // 更新 preferences
  preferencesManager.updatePreferences({
    app: { locale: lang },
  });

  // 同步更新 vue-i18n
  locale.value = lang;

  // 等待下一个 tick 确保 locale 更新完成
  await nextTick();

  // 使用新的 locale 获取翻译文本
  ElMessage.success(t("common.langSelect.message.success"));
}
</script>
