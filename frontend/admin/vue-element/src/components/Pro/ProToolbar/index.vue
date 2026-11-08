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
              <!-- 文本必须包 span：EP 的图标间距规则是 .el-button [class*=el-icon]+span，裸文本节点吃不到 margin -->
              <span v-if="btn.text">{{ btn.text }}</span>
            </ElButton>
          </AccessControl>
        </template>
        <slot name="left" />
      </div>

      <!-- 右侧工具按钮 -->
      <!-- 窄屏（<md）列间距 0.5：保证筛选圆钮不因 1px 差孤行悬在第三行（2026-11-08 实测） -->
      <div class="toolbar-right flex items-center gap-y-2.5 gap-x-0.5 md:gap-x-1.5 flex-wrap">
        <!-- 工具栏图标按钮区前插槽（导出全部下拉在此落地，位于页面配置按钮左侧，与 react/vben 同位） -->
        <slot name="before-tools" />

        <!-- 页面配置按钮（toolbarRight，"add" 惯例收尾）：排在导出下拉之后、内置图标之前 -->
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
              <span v-if="btn.text">{{ btn.text }}</span>
            </ElButton>
          </AccessControl>
        </template>

        <!-- 默认工具栏图标按钮（全部带 Tooltip，对齐 react ProTable options 惯例） -->
        <template v-for="(tool, idx) in defaultToolbar" :key="'tool-' + idx">
          <!-- 刷新 -->
          <ElTooltip
            v-if="tool === 'refresh'"
            :content="t('pages.curd.toolbar.refresh')"
            placement="top"
          >
            <ElButton circle :icon="RefreshLeft" @click="handleDefaultTool('refresh')" />
          </ElTooltip>
          <!-- 筛选/列设置 -->
          <ElTooltip
            v-else-if="tool === 'filter' && hasFilterContent"
            :content="t('pages.curd.toolbar.columnSetting')"
            placement="top"
          >
            <span class="inline-flex">
              <ElPopover
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
            </span>
          </ElTooltip>
          <!-- 搜索 -->
          <ElTooltip
            v-else-if="tool === 'search'"
            :content="t('pages.curd.toolbar.search')"
            placement="top"
          >
            <ElButton circle :icon="Search" @click="handleDefaultTool('search')" />
          </ElTooltip>
          <!-- 导入 -->
          <ElTooltip
            v-else-if="tool === 'imports'"
            :content="t('pages.curd.toolbar.batchImport')"
            placement="top"
          >
            <ElButton circle :icon="Upload" @click="handleDefaultTool('import')" />
          </ElTooltip>
          <!-- 全屏（图标用本地 fullscreen/fullscreen-exit SVG：与 react FullscreenOutlined
               同款字形、与布局层全屏钮同资源，EP 图标集没有退出全屏的对偶字形） -->
          <ElTooltip
            v-else-if="tool === 'zoom'"
            :content="t('pages.curd.toolbar.fullscreen')"
            placement="top"
          >
            <ElButton circle @click="handleZoom">
              <SvgIcon :icon="isFullscreen ? 'fullscreen-exit' : 'fullscreen'" />
            </ElButton>
          </ElTooltip>
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
              <span v-if="tool.text">{{ tool.text }}</span>
            </ElButton>
          </AccessControl>
        </template>

        <slot name="right" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElButton, ElIcon, ElPopover, ElTooltip } from "element-plus";
import {
  RefreshLeft,
  Operation,
  Search,
  Upload,
} from "@element-plus/icons-vue";
import ColumnFilter from "./ColumnFilter.vue";
import { AccessControl } from "@/core/access";
import { useI18n } from "@/core/i18n";
import type {
  ProToolbarProps,
  ProToolbarEmits,
  ToolbarButton,
  ToolbarCustomButton,
  ToolbarRightType,
} from "./types";
import { computed, ref, useSlots, onMounted, onBeforeUnmount } from "vue";
import { ProTableColumn } from "@/components/Pro";

defineOptions({ inheritAttrs: false });

const props = withDefaults(defineProps<ProToolbarProps>(), {
  visible: true,
  defaultToolbar: () => ["refresh", "zoom", "filter", "search"] as ToolbarRightType[],
  leftButtons: () => [],
  rightButtons: () => [],
});

const emit = defineEmits<ProToolbarEmits>();
const slots = useSlots();
const { t } = useI18n();

// === 全屏状态 ===
// 图标态完全事件驱动：只跟随浏览器真实全屏状态（fullscreenchange），点击不在这里翻
// isFullscreen——requestFullscreen 被拒（iframe 未授权）或用户 ESC 退出时，前者不发事件、
// 后者发事件，乐观翻转在这两种情况下都会让图标停在错误的一侧。全屏元素由 ProPage 挂，
// 这里只消费 document 级事件（对布局层全屏同样成立）。
const isFullscreen = ref(false);

function syncFullscreen() {
  isFullscreen.value = document.fullscreenElement != null;
}

onMounted(() => document.addEventListener("fullscreenchange", syncFullscreen));
onBeforeUnmount(() => document.removeEventListener("fullscreenchange", syncFullscreen));

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

// 全屏切换：只上抛意图（取当前真实态的反面），图标等 fullscreenchange 回报后翻转
function handleZoom() {
  emit("zoom", !isFullscreen.value);
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
    padding: 0 16px;
    font-size: 15px; // 从 14px 增大到 15px
    line-height: normal;
    transition: all 0.2s ease;
    cursor: pointer;

    // ======== 主按钮（primary）—— 标准 Element Plus 主色 ========
    // 注意：plain+primary（镂空次要主按钮）由 Element Plus 内置样式接管，
    // 此处用 :not(.is-plain) 排除，避免实心样式覆盖镂空语义。
    &.el-button--primary:not(.is-plain) {
      background-color: var(--el-color-primary);
      border-color: var(--el-color-primary);
      color: var(--el-color-white);

      &:hover,
      &:focus {
        background-color: var(--el-color-primary-light-3);
        border-color: var(--el-color-primary-light-3);
        color: var(--el-color-white);
      }

      &:active {
        background-color: var(--el-color-primary-dark-2);
        border-color: var(--el-color-primary-dark-2);
        color: var(--el-color-white);
      }
    }

    // ======== 危险按钮（danger）—— 柔和红色，使用 Element Plus 主题 danger 色阶（亮/暗自动切换） ========
    &.el-button--danger {
      background-color: var(--el-color-danger-light-9);
      border-color: var(--el-color-danger-light-5);
      color: var(--gowind-danger-text);

      &:hover,
      &:focus {
        background-color: var(--el-color-danger-light-7);
        border-color: var(--el-color-danger-light-3);
        color: var(--gowind-danger-text);
      }

      &:active {
        background-color: var(--el-color-danger-light-5);
      }
    }

    // ======== 成功按钮（success）—— 柔和绿色，使用 Element Plus 主题 success 色阶（亮/暗自动切换） ========
    &.el-button--success {
      background-color: var(--el-color-success-light-9);
      border-color: var(--el-color-success-light-5);
      color: var(--gowind-success-text);

      &:hover,
      &:focus {
        background-color: var(--el-color-success-light-7);
        border-color: var(--el-color-success-light-3);
        color: var(--gowind-success-text);
      }

      &:active {
        background-color: var(--el-color-success-light-5);
      }
    }

    // ======== 普通按钮（default）—— 中性灰，使用 Element Plus 按钮主题变量（亮/暗自动切换） ========
    // stylelint-disable-next-line selector-max-universal
    &:not(.el-button--primary):not(.el-button--danger):not(.el-button--success):not(
        .el-button--warning
      ):not(.el-button--info):not(.is-circle) {
      background-color: var(--el-button-bg-color);
      border-color: var(--el-button-border-color);
      color: var(--el-button-text-color);

      &:hover,
      &:focus {
        background-color: var(--el-button-hover-bg-color);
        border-color: var(--el-button-hover-border-color);
        color: var(--el-button-hover-text-color);
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
    // 颜色用主文本色阶对齐 react（antd colorText）：EP 图标字形墨迹仅占
    // viewBox 75%（antd ≈90%），16px 名义尺寸下实际观感比 react 小一截，
    // 故图标放大到 18px 补偿 + 主文本色提精神（暗 #F8FAFC / 亮 #1F2937）。
    color: var(--el-text-color-primary);
    transition: all 0.2s ease;
    font-size: 18px;

    // 增大图标尺寸
    .el-icon {
      font-size: 18px !important;
    }

    &:hover {
      background-color: var(--el-fill-color-light);
    }

    &:active {
      background-color: var(--el-fill-color);
    }
  }
}

// 注：亮色分支已全部使用 Element Plus 主题变量（var(--el-*)），这些变量在
// 暗黑模式下由 element-plus/theme-chalk/dark/css-vars.css 自动提供对应暗色值，
// 因此不再需要 :global(html.dark) 覆盖块，避免双分支硬编码色值不一致。
</style>
