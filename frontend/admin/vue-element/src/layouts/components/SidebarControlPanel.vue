<template>
  <div class="sidebar-control-panel">
    <!-- 折叠/展开按钮 -->
    <ElTooltip
      :content="collapsed ? t('common.sidebar.expandMenu') : t('common.sidebar.collapseMenu')"
      placement="right"
    >
      <div class="sidebar-control-panel__btn" @click="$emit('toggle-collapse')">
        <ElIcon :size="16">
          <DArrowLeft v-if="!collapsed" />
          <DArrowRight v-else />
        </ElIcon>
      </div>
    </ElTooltip>

    <!-- 固定/自动模式切换 -->
    <ElTooltip
      :content="
        expandOnHover ? t('common.sidebar.switchToFixed') : t('common.sidebar.switchToAuto')
      "
      placement="right"
    >
      <div class="sidebar-control-panel__btn" @click="$emit('toggle-expand-on-hover')">
        <ElIcon :size="16" :class="{ 'is-unpinned': expandOnHover }">
          <svg viewBox="0 0 24 24" fill="currentColor" width="1em" height="1em">
            <path d="M16 12V4h1V2H7v2h1v8l-2 2v2h5.2v6h1.6v-6H18v-2l-2-2z" />
          </svg>
        </ElIcon>
      </div>
    </ElTooltip>
  </div>
</template>

<script setup lang="ts">
import { DArrowLeft, DArrowRight } from "@element-plus/icons-vue";
import { ElIcon, ElTooltip } from "element-plus";
import { useI18n } from "@/i18n";

defineProps<{
  /** 侧边栏是否折叠 */
  collapsed: boolean;
  /** 是否启用鼠标悬停自动展开 */
  expandOnHover: boolean;
}>();

defineEmits<{
  "toggle-collapse": [];
  "toggle-expand-on-hover": [];
}>();

const { t } = useI18n();
</script>

<style lang="scss" scoped>
.sidebar-control-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  flex-shrink: 0;
  border-top: 1px solid var(--el-border-color-lighter);

  &__btn {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    cursor: pointer;
    border-radius: 4px;
    color: var(--el-text-color-secondary);
    transition: all 0.2s;

    &:hover {
      color: var(--el-color-primary);
      background-color: var(--el-fill-color-light);
    }

    // 自动模式：pin 旋转 45° 表示「未钉住」
    .is-unpinned {
      transform: rotate(45deg);
    }
  }
}
</style>
