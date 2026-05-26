<template>
  <div v-if="visible" class="pro-toolbar" :class="props.class">
    <!-- 左侧按钮 -->
    <div class="pro-toolbar__left flex gap-2">
      <template v-for="btn in leftButtons" :key="btn.name">
        <ElButton
          v-if="shouldShow(btn)"
          v-bind="btn.attrs"
          :disabled="btn.disabled"
          :loading="btn.loading"
          @click="handleButtonClick(btn)"
        >
          <ElIcon v-if="btn.icon"><component :is="btn.icon" /></ElIcon>
          {{ btn.text }}
        </ElButton>
      </template>

      <!-- 插槽：左侧自定义 -->
      <slot name="left" />
    </div>

    <!-- 右侧按钮 -->
    <div class="pro-toolbar__right flex gap-2">
      <!-- 自定义右侧按钮 -->
      <template v-for="btn in rightButtons" :key="'right-' + btn.name">
        <ElButton
          v-if="shouldShow(btn)"
          v-bind="btn.attrs"
          :disabled="btn.disabled"
          :loading="btn.loading"
          @click="handleButtonClick(btn)"
        >
          <ElIcon v-if="btn.icon"><component :is="btn.icon" /></ElIcon>
          {{ btn.text }}
        </ElButton>
      </template>

      <!-- 默认工具栏 -->
      <template v-for="tool in defaultToolbar" :key="tool">
        <!-- 刷新 -->
        <ElButton
          v-if="tool === 'refresh'"
          circle
          :icon="Refresh"
          @click="handleDefaultTool('refresh')"
        />

        <!-- 筛选 -->
        <ElPopover v-if="tool === 'filter'" placement="bottom" trigger="click" :width="200">
          <template #reference>
            <ElButton circle :icon="Operation" />
          </template>
          <div class="pro-toolbar__filter-popover">
            <slot name="filter" />
          </div>
        </ElPopover>

        <!-- 搜索 -->
        <ElButton
          v-if="tool === 'search'"
          circle
          :icon="Search"
          @click="handleDefaultTool('search')"
        />

        <!-- 导出 -->
        <ElButton
          v-if="tool === 'exports'"
          circle
          :icon="Download"
          @click="handleDefaultTool('export')"
        />

        <!-- 导入 -->
        <ElButton
          v-if="tool === 'imports'"
          circle
          :icon="Upload"
          @click="handleDefaultTool('import')"
        />
      </template>

      <!-- 插槽：右侧自定义 -->
      <slot name="right" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElButton, ElIcon, ElPopover } from "element-plus";
import { Refresh, Operation, Search, Download, Upload } from "@element-plus/icons-vue";
import type { ProToolbarProps, ProToolbarEmits, ToolbarButton } from "./types";
import { useAccess } from "@/core/access"; // 假设你有权限 Hook

defineOptions({ inheritAttrs: false });

const props = withDefaults(defineProps<ProToolbarProps>(), {
  visible: true,
  defaultToolbar: () => ["refresh", "filter", "search"],
  leftButtons: () => [],
  rightButtons: () => [],
});

const emit = defineEmits<ProToolbarEmits>();

const { hasAccessByCodes } = useAccess();

// 检查按钮是否应该显示
function shouldShow(btn: ToolbarButton): boolean {
  if (btn.hidden) return false;
  if (btn.visible && !btn.visible()) return false;

  // 权限检查
  if (btn.perm) {
    const perms = Array.isArray(btn.perm) ? btn.perm : [btn.perm];
    const fullPerms = perms.map((p) => (p.includes(":") ? p : `${props.permPrefix ?? ""}:${p}`));
    if (!hasAccessByCodes(fullPerms)) return false;
  }

  return true;
}

// 处理按钮点击
function handleButtonClick(btn: ToolbarButton) {
  emit("button-click", btn.name, btn);

  // 内置按钮类型处理
  switch (btn.type) {
    case "refresh":
      emit("refresh");
      break;
    case "export":
      emit("export");
      break;
    case "import":
      emit("import");
      break;
    case "search":
      emit("search");
      break;
    case "filter":
      emit("filter");
      break;
  }
}

// 处理默认工具栏点击
function handleDefaultTool(tool: string) {
  emit("button-click", tool);

  switch (tool) {
    case "refresh":
      emit("refresh");
      break;
    case "filter":
      emit("filter");
      break;
    case "search":
      emit("search");
      break;
    case "export":
      emit("export");
      break;
    case "import":
      emit("import");
      break;
  }
}

// 暴露方法
defineExpose({
  trigger: (name: string) => {
    const btn = [...props.leftButtons, ...props.rightButtons].find((b) => b.name === name);
    if (btn) handleButtonClick(btn);
  },
});
</script>

<style scoped lang="scss">
.pro-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  margin-bottom: 12px;

  &__left,
  &__right {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  &__filter-popover {
    padding: 12px;
    max-height: 300px;
    overflow-y: auto;
  }
}

@media (max-width: 768px) {
  .pro-toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;

    &__left,
    &__right {
      justify-content: flex-start;
    }
  }
}
</style>
