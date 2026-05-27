<template>
  <div v-if="visible" class="pro-toolbar" :class="props.class">
    <div class="flex flex-col md:flex-row justify-between gap-y-2">
      <!-- 左侧按钮 -->
      <div class="toolbar-left flex gap-y-2.5 gap-x-2 md:gap-x-3 flex-wrap">
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
        <slot name="left" />
      </div>

      <!-- 右侧按钮 -->
      <div class="toolbar-right flex items-center gap-y-2.5 gap-x-2 md:gap-x-3 flex-wrap">
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

        <!-- 工具栏图标按钮区前插槽 -->
        <slot name="before-tools" />

        <!-- 默认工具栏图标按钮 -->
        <template v-for="(tool, idx) in defaultToolbar" :key="'tool-' + idx">
          <!-- 刷新 -->
          <ElButton
            v-if="tool === 'refresh'"
            circle
            :icon="Refresh"
            @click="handleDefaultTool('refresh')"
          />
          <!-- 筛选 -->
          <ElPopover
            v-else-if="tool === 'filter' && hasFilterContent"
            placement="bottom"
            trigger="click"
            :width="200"
          >
            <template #reference>
              <ElButton circle :icon="Operation" />
            </template>
            <ElScrollbar max-height="350px">
              <template v-if="filterableColumns.length">
                <ElCheckbox
                  v-for="col in filterableColumns"
                  :key="col.prop"
                  v-model="col.show"
                  :label="col.label"
                />
              </template>
              <slot v-else name="filter" />
            </ElScrollbar>
          </ElPopover>
          <!-- 搜索 -->
          <ElButton
            v-else-if="tool === 'search'"
            circle
            :icon="Search"
            @click="handleDefaultTool('search')"
          />
          <!-- 导出 -->
          <ElButton
            v-else-if="tool === 'exports'"
            circle
            :icon="Download"
            @click="handleDefaultTool('export')"
          />
          <!-- 导入 -->
          <ElButton
            v-else-if="tool === 'imports'"
            circle
            :icon="Upload"
            @click="handleDefaultTool('import')"
          />
          <!-- 全屏 -->
          <ElButton
            v-else-if="tool === 'zoom'"
            circle
            :icon="isFullscreen ? Aim : FullScreen"
            @click="handleZoom"
          />
          <!-- 自定义工具栏按钮 -->
          <ElButton
            v-else-if="typeof tool === 'object' && shouldShow(tool)"
            v-bind="tool.attrs"
            :disabled="tool.disabled"
            :loading="tool.loading"
            circle
            @click="handleCustomToolClick(tool)"
          >
            <ElIcon v-if="tool.icon"><component :is="tool.icon" /></ElIcon>
            <template v-if="tool.text">{{ tool.text }}</template>
          </ElButton>
        </template>

        <slot name="right" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElButton, ElCheckbox, ElIcon, ElPopover, ElScrollbar } from "element-plus";
import {
  Refresh,
  Operation,
  Search,
  Download,
  Upload,
  FullScreen,
  Aim,
} from "@element-plus/icons-vue";
import type {
  ProToolbarProps,
  ProToolbarEmits,
  ToolbarButton,
  ToolbarCustomButton,
  ToolbarRightType,
} from "./types";
import { useAccess } from "@/core/access";
import { computed, ref, useSlots } from "vue";

defineOptions({ inheritAttrs: false });

const props = withDefaults(defineProps<ProToolbarProps>(), {
  visible: true,
  defaultToolbar: () => ["refresh", "filter", "search"] as ToolbarRightType[],
  leftButtons: () => [],
  rightButtons: () => [],
});

const emit = defineEmits<ProToolbarEmits>();
const slots = useSlots();

const { hasAccessByCodes } = useAccess();

// === 全屏状态 ===
const isFullscreen = ref(false);

// 可筛选的列（有 prop 和 label，仅 el-table 引擎使用）
const filterableColumns = computed(() =>
  (props.columns ?? []).filter((col) => col.prop && col.label),
);

// filter 按钮是否有内容
const hasFilterContent = computed(
  () => filterableColumns.value.length > 0 || !!slots.filter,
);

// 检查按钮是否应该显示
function shouldShow(btn: ToolbarButton | ToolbarCustomButton): boolean {
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

// 处理自定义工具栏按钮点击
function handleCustomToolClick(btn: ToolbarCustomButton) {
  emit("button-click", btn.name, btn);
}

// 全屏切换
function handleZoom() {
  isFullscreen.value = !isFullscreen.value;
  emit("zoom", isFullscreen.value);
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
  margin-bottom: 8px;
}

.toolbar-left,
.toolbar-right {
  :deep(.el-button) {
    margin-right: 0 !important;
    margin-left: 0 !important;
  }
}
</style>
