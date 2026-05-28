<template>
  <!-- 组件尺寸切换 -->
  <el-tooltip :content="t('core.sizeSelect.tooltip')" effect="dark" placement="bottom">
    <el-dropdown trigger="click" @command="handleSizeChange">
      <div class="size-trigger">
        <SvgIcon icon="size" />
      </div>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item
            v-for="item of sizeOptions"
            :key="item.value"
            :disabled="getCurrentSize() === item.value"
            :command="item.value"
          >
            {{ item.label }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </el-tooltip>
</template>

<script setup lang="ts">
import { preferences, preferencesManager } from "@/core/preferences";
import SvgIcon from "@/components/SvgIcon/index.vue";

const { t } = useI18n();

const sizeOptions = computed(() => [
  { label: t("core.sizeSelect.default"), value: "default" as const },
  { label: t("core.sizeSelect.small"), value: "small" as const },
]);

/**
 * 获取当前组件尺寸
 */
function getCurrentSize(): "default" | "small" {
  return preferences.app.compact ? "small" : "default";
}

function handleSizeChange(size: "default" | "small") {
  preferencesManager.updatePreferences({
    app: { compact: size === "small" },
  });
  ElMessage.success(t("core.sizeSelect.message.success"));
}
</script>

<style lang="scss" scoped>
.size-trigger {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}
</style>
