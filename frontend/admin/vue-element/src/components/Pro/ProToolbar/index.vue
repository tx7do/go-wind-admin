<template>
  <div v-if="visible" class="pro-toolbar" :class="props.class">
    <div class="flex flex-col md:flex-row justify-between gap-y-2">
      <!-- 左侧按钮 -->
      <div class="toolbar-left flex items-center gap-y-2.5 gap-x-2 md:gap-x-3 flex-wrap">
        <template v-for="btn in leftButtons" :key="btn.name">
          <AccessControl
            :codes="btn.auth ? (Array.isArray(btn.auth) ? btn.auth : [btn.auth]) : undefined"
          >
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
          </AccessControl>
        </template>
        <slot name="left" />
      </div>

      <!-- 右侧工具按钮 -->
      <div class="toolbar-right flex items-center gap-y-2.5 gap-x-1 md:gap-x-1.5 flex-wrap">
        <template v-for="btn in rightButtons" :key="'right-' + btn.name">
          <AccessControl
            :codes="btn.auth ? (Array.isArray(btn.auth) ? btn.auth : [btn.auth]) : undefined"
          >
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
          </AccessControl>
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
            :width="350"
          >
            <template #reference>
              <ElButton circle :icon="Operation" />
            </template>
            <ColumnFilter
              :columns="filterableColumns"
              @confirm="handleFilterConfirm"
              @cancel="handleFilterCancel"
            />
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
          <AccessControl
            v-else-if="typeof tool === 'object'"
            :codes="tool.auth ? (Array.isArray(tool.auth) ? tool.auth : [tool.auth]) : undefined"
          >
            <ElButton
              v-if="shouldShow(tool)"
              v-bind="tool.attrs"
              :disabled="tool.disabled"
              :loading="tool.loading"
              circle
              @click="handleCustomToolClick(tool)"
            >
              <ElIcon v-if="tool.icon"><component :is="tool.icon" /></ElIcon>
              <template v-if="tool.text">{{ tool.text }}</template>
            </ElButton>
          </AccessControl>
        </template>

        <slot name="right" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElButton, ElIcon, ElPopover } from "element-plus";
import {
  Refresh,
  Operation,
  Search,
  Download,
  Upload,
  FullScreen,
  Aim,
} from "@element-plus/icons-vue";
import ColumnFilter from "./ColumnFilter.vue";
import { AccessControl } from "@/core/access";
import type {
  ProToolbarProps,
  ProToolbarEmits,
  ToolbarButton,
  ToolbarCustomButton,
  ToolbarRightType,
} from "./types";
import { computed, ref, useSlots } from "vue";
import { ProTableColumn } from "@/components/Pro";

defineOptions({ inheritAttrs: false });

const props = withDefaults(defineProps<ProToolbarProps>(), {
  visible: true,
  defaultToolbar: () => ["refresh", "filter", "search"] as ToolbarRightType[],
  leftButtons: () => [],
  rightButtons: () => [],
});

const emit = defineEmits<ProToolbarEmits>();
const slots = useSlots();

// === 全屏状态 ===
const isFullscreen = ref(false);

// 可筛选的列（有 prop 和 label，仅 el-table 引擎使用）
const filterableColumns = computed(() =>
  (props.columns ?? []).filter((col) => col.prop && col.label)
);

// filter 按钮是否有内容
const hasFilterContent = computed(() => filterableColumns.value.length > 0 || !!slots.filter);

// 检查按钮是否应该显示（仅处理 hidden/visible，权限由 AccessControl 组件处理）
function shouldShow(btn: ToolbarButton | ToolbarCustomButton): boolean {
  if (btn.hidden) return false;
  return !(btn.visible && !btn.visible());
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

// 筛选确认
function handleFilterConfirm(columns: ProTableColumn[]) {
  emit("filter-change", columns);
}

// 筛选取消
function handleFilterCancel() {
  // 不做处理，Popover 会自动关闭
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

    // ======== 统一基础规范 ========
    height: 32px;
    border-radius: 6px;
    padding: 0 14px;
    font-size: 13px;
    font-weight: 500;
    transition: all 0.2s ease;
    cursor: pointer;

    // ======== 主按钮（primary）—— 标准 Element Plus 主色 ========
    &.el-button--primary {
      background-color: var(--el-color-primary);
      border-color: var(--el-color-primary);
      color: #fff;

      &:hover,
      &:focus {
        background-color: var(--el-color-primary-light-3);
        border-color: var(--el-color-primary-light-3);
        color: #fff;
      }

      &:active {
        background-color: var(--el-color-primary-dark-2);
        border-color: var(--el-color-primary-dark-2);
        color: #fff;
      }
    }

    // ======== 危险按钮（danger）—— 柔和红色 ========
    &.el-button--danger {
      background-color: rgba(245, 63, 63, 0.1);
      border-color: rgba(245, 63, 63, 0.25);
      color: #f53f3f;

      &:hover,
      &:focus {
        background-color: rgba(245, 63, 63, 0.18);
        border-color: rgba(245, 63, 63, 0.4);
        color: #f53f3f;
      }

      &:active {
        background-color: rgba(245, 63, 63, 0.22);
      }
    }

    // ======== 成功按钮（success）—— 柔和绿色 ========
    &.el-button--success {
      background-color: rgba(82, 196, 26, 0.1);
      border-color: rgba(82, 196, 26, 0.3);
      color: #52c41a;

      &:hover,
      &:focus {
        background-color: rgba(82, 196, 26, 0.18);
        border-color: rgba(82, 196, 26, 0.5);
        color: #52c41a;
      }

      &:active {
        background-color: rgba(82, 196, 26, 0.22);
      }
    }

    // ======== 普通按钮（default）—— 中性灰 ========
    // stylelint-disable-next-line selector-max-universal
    &:not(.el-button--primary):not(.el-button--danger):not(.el-button--success):not(.el-button--warning):not(.el-button--info):not(.is-circle) {
      background-color: rgba(0, 0, 0, 0.04);
      border-color: #dcdfe6;
      color: var(--el-text-color-regular);

      &:hover,
      &:focus {
        background-color: rgba(0, 0, 0, 0.08);
        border-color: #c0c4cc;
        color: var(--el-text-color-primary);
      }
    }
  }
}

// ======== 右侧圆形工具按钮 ========
.toolbar-right {
  flex-shrink: 0;

  // 圆形图标按钮：32px，与左侧文字按钮高度匹配
  :deep(.el-button.is-circle) {
    width: 32px;
    height: 32px;
    padding: 0;
    border-radius: 50%;
    background-color: transparent;
    border: none;
    color: var(--el-text-color-regular);
    transition: all 0.2s ease;
    font-size: 15px;

    &:hover {
      background-color: var(--el-fill-color-light);
      color: var(--el-text-color-primary);
    }

    &:active {
      background-color: var(--el-fill-color);
    }
  }
}

// ======== 暗色模式适配 ========
:global(html.dark) {
  .toolbar-left,
  .toolbar-right {
    :deep(.el-button) {
      // 主按钮（暗色模式：柔和风格，与其他按钮统一）
      &.el-button--primary {
        background-color: var(--el-color-primary-light-9);
        border-color: var(--el-color-primary-light-5);
        color: var(--el-color-primary-light-3);

        &:hover,
        &:focus {
          background-color: var(--el-color-primary-light-8);
          border-color: var(--el-color-primary-light-3);
        }
      }

      // 危险按钮
      &.el-button--danger {
        background-color: rgba(245, 63, 63, 0.15);
        border-color: rgba(245, 63, 63, 0.35);

        &:hover,
        &:focus {
          background-color: rgba(245, 63, 63, 0.25);
          border-color: rgba(245, 63, 63, 0.5);
        }
      }

      // 成功按钮
      &.el-button--success {
        background-color: rgba(82, 196, 26, 0.15);
        border-color: rgba(82, 196, 26, 0.4);

        &:hover,
        &:focus {
          background-color: rgba(82, 196, 26, 0.25);
          border-color: rgba(82, 196, 26, 0.6);
        }
      }

      // 普通按钮
      // 同时覆盖 EP CSS 变量 + 直接属性，确保不被全局暗色主题覆盖
      // stylelint-disable-next-line selector-max-universal
      &:not(.el-button--primary):not(.el-button--danger):not(.el-button--success):not(.el-button--warning):not(.el-button--info):not(.is-circle) {
        --el-button-bg-color: rgba(255, 255, 255, 0.06);
        --el-button-border-color: rgba(255, 255, 255, 0.12);
        --el-button-text-color: var(--el-text-color-regular);
        --el-button-hover-bg-color: rgba(255, 255, 255, 0.1);
        --el-button-hover-border-color: rgba(255, 255, 255, 0.2);
        --el-button-hover-text-color: var(--el-text-color-primary);

        background-color: rgba(255, 255, 255, 0.06) !important;
        border-color: rgba(255, 255, 255, 0.12) !important;
        color: var(--el-text-color-regular) !important;

        &:hover,
        &:focus {
          background-color: rgba(255, 255, 255, 0.1) !important;
          border-color: rgba(255, 255, 255, 0.2) !important;
          color: var(--el-text-color-primary) !important;
        }
      }
    }
  }

  .toolbar-right {
    :deep(.el-button.is-circle) {
      color: var(--el-text-color-secondary);

      &:hover {
        background-color: rgba(255, 255, 255, 0.08);
        color: var(--el-text-color-primary);
      }

      &:active {
        background-color: rgba(255, 255, 255, 0.12);
      }
    }
  }
}
</style>
