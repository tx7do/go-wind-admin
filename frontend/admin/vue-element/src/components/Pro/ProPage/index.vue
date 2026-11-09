<template>
  <div ref="rootRef" class="pro-page">
    <!-- 搜索区 -->
    <ProSearch
      v-if="!showSkeleton && searchVisible && config.search?.fields?.length"
      ref="searchRef"
      :fields="config.search.fields"
      :colon="config.search.colon"
      :is-expandable="config.search.isExpandable"
      :show-number="config.search.showNumber"
      :grid="config.search.grid"
      @search="handleSearch"
      @reset="handleReset"
    />

    <!-- 表格区 -->
    <div ref="contentRef" class="pro-page__content">
      <!-- 首次加载骨架屏 -->
      <TableSkeleton
        v-if="showSkeleton"
        v-bind="skeletonProps"
        :show-search="skeletonProps.showSearch ?? !!config.search?.fields?.length"
        :column-count="
          config.table.columns.filter(
            (c: any) => c.prop && c.type !== 'selection' && c.type !== 'index'
          ).length
        "
        :show-pagination="skeletonProps.showPagination ?? config.table.pagination !== false"
      />

      <!-- 首屏加载失败（没有可展示的数据）：整块替换为错误态 + 重试 -->
      <div
        v-else-if="tableState.hasError.value && tableData.length === 0"
        class="pro-page__error"
      >
        <ElEmpty :description="errorText" :image-size="120">
          <ElButton type="primary" @click="handleRetry">{{ t("common.button.retry") }}</ElButton>
        </ElEmpty>
      </div>

      <!-- 正常内容 -->
      <template v-else>
        <!-- 工具栏 -->
        <ProToolbar
          :left-buttons="leftButtons"
          :right-buttons="rightButtons"
          :default-toolbar="defaultToolbarButtons"
          :columns="filterColumns"
          @button-click="handleToolbarClick"
          @zoom="handleZoom"
          @filter-change="handleFilterChange"
        >
          <template #left><slot name="toolbar-left" /></template>
          <template #before-tools>
            <slot name="toolbar-before-tools" />
            <!-- 导出全部（exports 工具的落地形态，置于图标工具区之前，与 react/vben 同位）：
                 CSV/Excel 按当前搜索条件经 listAction 聚合（上限 1 万行）；
                 服务端全量项由页面经 table.exportServer 提供 -->
            <ExportAllDropdown
              v-if="hasExportAllDropdown"
              :fetcher="exportFetcher"
              :columns="config.table.columns"
              :params="searchParams"
              :filename="config.exportFilename ?? 'export'"
              :server-export="config.table.exportServer"
            />
          </template>
          <template v-if="!!$slots['toolbar-filter']" #filter>
            <slot name="toolbar-filter" />
          </template>
          <template #right><slot name="toolbar-right" /></template>
        </ProToolbar>

        <!-- 翻页/搜索失败但仍有旧数据：保留表格，只在表上方挂一条错误横幅 -->
        <ElAlert
          v-if="tableState.hasError.value"
          class="pro-page__error-alert"
          type="error"
          :title="errorText"
          :closable="false"
          show-icon
        />

        <!-- 表格 -->
        <ProTable
          ref="tableRef"
          class="flex-1 min-h-0"
          :engine="config.engine"
          :table-id="tableId"
          :columns="config.table.columns"
          :data="tableData"
          :loading="delayedLoading"
          :row-key="rowKey"
          :table="config.table.tableAttrs"
          :empty-action-text="config.table.emptyActionText"
          :pagination="tableState.showPagination"
          :total="tableState.pagination.total"
          :current-page="tableState.pagination.currentPage"
          :page-size="tableState.pagination.pageSize"
          :page-sizes="tableState.pagination.pageSizes"
          @selection-change="tableState.handleSelectionChange"
          @modify="handleModify"
          @operate="handleOperate"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
          @row-click="handleRowClick"
          @empty-action="handleEmptyAction"
        >
          <!-- 透传自定义列插槽 -->
          <template v-for="(_, name) in $slots" :key="name" #[name]="slotProps">
            <slot :name="name" v-bind="slotProps" />
          </template>
        </ProTable>
      </template>
    </div>

    <!-- 导入弹窗 -->
    <ImportModal
      ref="importModalRef"
      :imports-action="config.table.importsAction"
      :import-action="config.table.importAction"
      :import-template="config.table.importTemplate"
      @success="handleRefresh"
    />

    <!-- 弹窗 -->
    <ProModal
      v-if="config.modal"
      v-model:visible="modalState.visible.value"
      :mode="modalState.mode.value"
      :config="modalConfig"
      :form-data="modalState.formData as any"
      @submit="handleModalSubmit"
    >
      <template v-for="(_, name) in modalSlots" :key="name" #[name]="slotProps">
        <slot :name="name" v-bind="slotProps" />
      </template>
    </ProModal>
  </div>
</template>

<script setup lang="ts" generic="T extends Record<string, any>, Q extends Record<string, any>">
import { reactive, computed, ref, useSlots, watch, onBeforeUnmount } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { useRoute } from "vue-router";
import { useI18n } from "@/core/i18n";
import { preferences } from "@/core/preferences";
import { PaginationQuery } from "@/core/transport/rest";

import ProSearch from "../ProSearch/index.vue";
import ProTable from "../ProTable/index.vue";
import ProModal from "../ProModal/index.vue";
import ProToolbar from "../ProToolbar/index.vue";
import ExportAllDropdown from "../ExportAllDropdown.vue";
import ImportModal from "./ImportModal.vue";
import TableSkeleton from "./TableSkeleton.vue";

import { useTableState } from "../composables/useTableState";
import { useModalState } from "../composables/useModalState";

import type { ToolbarButton, ToolbarCustomButton, ToolbarRightType } from "../ProToolbar/types";
import type { ProPageConfig, SkeletonConfig, ToolsButton } from "./types";

const props = defineProps<{ config: ProPageConfig<T, Q> }>();
const emit = defineEmits<{
  search: [params: Q];
  reset: [params: Q];
  add: [];
  edit: [row: T];
  view: [row: T];
  delete: [ids: string];
  operate: [data: { name: string; row: T; $index: number }];
  toolbar: [name: string];
  "row-click": [row: T];
}>();

const { t } = useI18n();
const slots = useSlots();
const route = useRoute();

// === 基础状态 ===
const rowKey = props.config.rowKey ?? "id";
// 自动生成 tableId：优先用 config.tableId，否则用路由路径
const tableId =
  props.config.tableId ?? (route.path.replace(/\//g, "_").replace(/^_/, "") || undefined);
const searchParams = reactive<Q>({} as Q);
const searchVisible = ref(true);

const tableState = useTableState<T, Q>({
  indexAction: props.config.table.listAction as any,
  rowKey,
  pagination: props.config.table.pagination,
  request: props.config.table.request,
});

const modalState = useModalState<T>(rowKey);
const tableRef = ref<any>(null);
const searchRef = ref<any>(null);
const rootRef = ref<HTMLElement | null>(null);
const contentRef = ref<HTMLElement | null>(null);
const importModalRef = ref<InstanceType<typeof ImportModal>>();

const tableData = computed(() => tableState.data.value);

// === 加载态：250ms 阈值，避免快响应下 spinner 一闪而过 ===
// 与 react 端 ListTable 的 loading.delay 同值，见 docs/design-language.md §4「列表加载态」。
const LOADING_DELAY = 250;
const delayedLoading = ref(false);
let loadingDelayTimer: ReturnType<typeof setTimeout> | null = null;

function clearLoadingDelayTimer() {
  if (loadingDelayTimer !== null) {
    clearTimeout(loadingDelayTimer);
    loadingDelayTimer = null;
  }
}

watch(
  () => tableState.loading.value,
  (isLoading) => {
    clearLoadingDelayTimer();
    if (isLoading) {
      // 已在显示则不重新计时，否则慢响应会让 spinner 反复消失又出现。
      if (!delayedLoading.value) {
        loadingDelayTimer = setTimeout(() => {
          delayedLoading.value = true;
          loadingDelayTimer = null;
        }, LOADING_DELAY);
      }
    } else {
      delayedLoading.value = false;
    }
  }
);

const errorText = computed(
  () => tableState.error.value || t("common.message.loadDataFailed")
);

function handleRetry() {
  tableState.fetch(searchParams);
}

// === filter 列显隐 ===
// vxe-table 引擎由 customConfig 接管列显隐；el-table 引擎使用 checkbox
const filterColumns = computed(() => {
  const cols = tableRef.value?.resolvedColumns;
  if (!cols) return [];
  return cols.filter(
    (col: any) => col.prop && col.label && col.type !== "selection" && col.type !== "index"
  );
});

// === 导出全部：exports 工具的落地形态 ===
// defaultToolbar 含 "exports" 即渲染「导出全部」下拉（图标弹窗形态已退役）。
const hasExportAllDropdown = computed(() =>
  (props.config.table.defaultToolbar ?? []).some((item) => item === "exports")
);

// 聚合 fetcher：复用页面 listAction（与表格加载同一入口、同一参数映射），
// 分页字段摊平 + 搜索条件原样透传——导出的即当前搜索看到的。
const exportFetcher = async (query: PaginationQuery) => {
  const listAction = props.config.table.listAction as unknown as (
    q: Record<string, any>
  ) => Promise<{ items?: any[]; total?: number | string }>;
  return await listAction({
    page: query.paging?.page ?? 1,
    pageSize: query.paging?.pageSize ?? 1000,
    ...(query.formValues ?? {}),
  });
};

// === 区分表格插槽和弹窗插槽 ===
const tableColumnProps = computed(() =>
  props.config.table.columns
    .filter((c) => c.cellType === "custom" || c.slotName)
    .map((c) => c.slotName ?? c.prop)
);
const modalSlots = computed(() => {
  const result: Record<string, any> = {};
  for (const name of Object.keys(slots)) {
    if (!tableColumnProps.value.includes(name as any)) {
      result[name] = slots[name];
    }
  }
  return result;
});

// === 工具栏按钮转换 ===
const builtinButtons: Record<string, { label: string; icon: string; attrs: Record<string, any> }> =
  {
    add: { label: "common.button.add", icon: "plus", attrs: { type: "primary", plain: true } },
    delete: {
      label: "common.button.delete",
      icon: "delete",
      attrs: { type: "danger" },
    },
    import: { label: "common.button.import", icon: "upload", attrs: {} },
    export: { label: "common.button.export", icon: "download", attrs: {} },
  };

function toToolbarButtons(
  items: Array<string | ToolsButton> | undefined,
  defaultAttrs: Record<string, any> = {}
): ToolbarButton[] {
  if (!items?.length) return [];
  return items.map((item) => {
    if (typeof item === "string") {
      const cfg = builtinButtons[item];
      return {
        name: item,
        text: cfg ? t(cfg.label) : item,
        type: item as ToolbarButton["type"],
        icon: cfg?.icon,
        attrs: { ...defaultAttrs, ...cfg?.attrs },
      };
    }
    return {
      name: item.name,
      text: item.label ? t(item.label) : item.name,
      attrs: { ...defaultAttrs, ...item.attrs },
      auth: item.auth,
      visible: item.visible,
    };
  });
}

const leftButtons = computed(() => toToolbarButtons(props.config.table.toolbar));
const rightButtons = computed(() => toToolbarButtons(props.config.table.toolbarRight));
const defaultToolbarButtons = computed(() => {
  const dt = props.config.table.defaultToolbar;
  if (!dt?.length)
    return ["refresh", "filter", "search", "zoom"] as Array<
      ToolbarRightType | ToolbarCustomButton
    >;
  // "exports" 已由 before-tools 区的「导出全部」下拉承载，不再渲染图标钮
  return dt
    .filter((item) => item !== "exports")
    .map((item) => {
      if (typeof item === "string") return item as ToolbarRightType;
      // ToolsButton (label) -> ToolbarCustomButton (text)
      return {
        name: item.name,
        text: item.label,
        icon: item.icon,
        auth: item.auth,
        attrs: item.attrs,
        hidden: item.hidden,
        disabled: item.disabled,
        loading: item.loading,
        visible: item.visible,
      } as ToolbarCustomButton;
    });
});

// === 搜索处理 ===
function handleSearch(params: Record<string, any>) {
  Object.keys(searchParams).forEach((k) => delete (searchParams as any)[k]);
  Object.assign(searchParams, params);
  tableState.fetch(searchParams, true);
  emit("search", { ...searchParams } as Q);
}

function handleReset(params: Record<string, any>) {
  Object.keys(searchParams).forEach((k) => delete (searchParams as any)[k]);
  Object.assign(searchParams, params);
  tableState.fetch(searchParams, true);
  emit("reset", { ...searchParams } as Q);
}

function toggleSearch() {
  searchVisible.value = !searchVisible.value;
}

// === 筛选处理（filter 由 ProToolbar 内部 ElPopover + columns checkbox 处理） ===
function handleFilterChange() {
  // 列配置已通过响应式更新，这里可以做额外处理
  // 比如保存到 localStorage 或后端
}

// === 空态引导按钮 ===
// ProTable 空态的「立即新增」按钮触发，复用工具栏 add 路径。
function handleEmptyAction() {
  handleToolbarClick("add");
}

// === 工具栏事件 ===
function handleToolbarClick(name: string) {
  switch (name) {
    case "add":
      modalState.open("add");
      emit("add");
      break;
    case "delete":
      handleBatchDelete();
      break;
    case "import":
    case "imports":
      openImportsModal();
      break;
    case "refresh":
      handleRefresh();
      break;
    case "search":
      toggleSearch();
      break;
    default:
      emit("toolbar", name);
      break;
  }
}

function handleRefresh() {
  tableState.fetch(searchParams);
}

// === 表格事件 ===
function handleModify(data: { row: T; field: string; value: any }) {
  if (props.config.table.modifyAction) {
    props.config.table.modifyAction({
      [rowKey]: (data.row as any)[rowKey],
      field: data.field,
      value: data.value,
    });
  }
}

function handleOperate(data: { name: string; row: T; $index: number }) {
  switch (data.name) {
    case "edit":
      modalState.open("edit", data.row);
      emit("edit", data.row);
      break;
    case "view":
      modalState.open("view", data.row);
      emit("view", data.row);
      break;
    case "delete":
      handleDelete(data.row);
      break;
    default:
      emit("operate", data);
      break;
  }
}

// === 删除 ===
async function handleBatchDelete() {
  const ids = tableState.getSelectionIds().join(",");
  if (!ids) {
    ElMessage.warning(t("pages.curd.message.selectDeleteItems"));
    return;
  }
  await ElMessageBox.confirm(t("pages.curd.message.confirmDelete"), t("common.title.confirm"), {
    confirmButtonText: t("common.button.confirm"),
    cancelButtonText: t("common.button.cancel"),
    type: "warning",
    lockScroll: false,
  });
  await props.config.table.deleteAction?.(ids);
  ElMessage.success(t("pages.curd.message.deleteSuccess"));
  // 清空选中行：否则 selection 仍含已删 id，UI 勾选也不会自动清除，
  // 再次批量删除会带上不存在的 id。
  tableState.clearSelection();
  tableRef.value?.clearSelection();
  tableState.fetch(searchParams, true);
}

async function handleDelete(row: T) {
  const id = String((row as any)[rowKey]);
  await ElMessageBox.confirm(t("pages.curd.message.confirmDelete"), t("common.title.confirm"), {
    confirmButtonText: t("common.button.confirm"),
    cancelButtonText: t("common.button.cancel"),
    type: "warning",
    lockScroll: false,
  });
  await props.config.table.deleteAction?.(id);
  ElMessage.success(t("pages.curd.message.deleteSuccess"));
  tableState.clearSelection();
  tableRef.value?.clearSelection();
  tableState.fetch(searchParams, true);
}

// === 全屏缩放 ===
// 全屏目标是整页根节点（搜索卡 + 工具栏 + 表格），对齐 react ProTable fullScreen /
// vben vxe zoom 的范围（两端都把搜索区一起带进全屏）。
function handleZoom(isFullscreen: boolean) {
  const el = rootRef.value;
  if (!el) return;
  if (isFullscreen) {
    // requestFullscreen 可能被拒绝（iframe 未授权 / 用户拒绝权限提示），拒绝是正常路径
    el.requestFullscreen?.()?.catch((error: unknown) => {
      console.error("[ProPage] 进入全屏失败", error);
    });
  } else {
    document.exitFullscreen?.()?.catch((error: unknown) => {
      console.error("[ProPage] 退出全屏失败", error);
    });
  }
}

function openImportsModal() {
  importModalRef.value?.open();
}

// === 分页 ===
function handlePageChange(page: number) {
  tableState.pagination.currentPage = page;
  tableState.fetch(searchParams);
}

function handleSizeChange(size: number) {
  tableState.pagination.pageSize = size;
  tableState.fetch(searchParams, true);
}

// === 行点击 ===
function handleRowClick(row: T) {
  emit("row-click", row);
}

// === 弹窗配置映射 ===
const modalConfig = computed(() => {
  if (!props.config.modal) return {} as any;
  return {
    component: props.config.modal.component,
    dialog: props.config.modal.dialog,
    drawer: props.config.modal.drawer,
    form: props.config.modal.form,
    colon: props.config.modal.colon,
    rowKey,
    fields: props.config.modal.fields ?? [],
    submitAction: props.config.modal.submitAction,
    beforeSubmit: props.config.modal.beforeSubmit,
  };
});

function handleModalSubmit() {
  tableState.fetch(searchParams, true);
}

// 初始化加载数据
tableState.fetch(searchParams);

// === 骨架屏：追踪首次加载 ===
// 页面 config.skeleton 决定「这页允不允许上骨架」，偏好设置 transition.loading 决定
// 「全局要不要用骨架代替 spinner」——两者都为真才渲染骨架。关掉开关后回落到表内 delayed Spin，
// 这条开关从此名副其实（此前它只有 schema 和抽屉开关，0 个消费者）。
const skeletonEnabled = computed(
  () => props.config.skeleton !== undefined && props.config.skeleton !== false
);
const globalLoadingSkeleton = computed(() => preferences.transition.loading);
const skeletonProps = computed<SkeletonConfig>(() => {
  const s = props.config.skeleton;
  if (s === true || s === undefined || s === false) return {} as SkeletonConfig;
  return s;
});
const hasLoaded = ref(false);
const showSkeleton = computed(
  () => skeletonEnabled.value && globalLoadingSkeleton.value && !hasLoaded.value
);

watch(
  () => tableState.loading.value,
  (loading) => {
    if (!loading) hasLoaded.value = true;
  }
);

onBeforeUnmount(clearLoadingDelayTimer);

defineExpose({
  tableRef,
  tableState,
  modalState,
  searchParams,
  refresh: () => tableState.fetch(searchParams),
  // 透传 ProSearch.reloadFieldOptions：按字段名重跑 initFn 刷新下拉选项。
  // 用于外部响应式数据（租户/组织）变化后刷新 role/position 等选项。
  reloadFieldOptions: (fieldNames: string | string[]) =>
    searchRef.value?.reloadFieldOptions(fieldNames),
});
</script>

<style scoped lang="scss">
.pro-page {
  display: flex;
  flex-direction: column;
  /* 搜索卡 → 列表卡 12px：8px 太挤，两块"贴一块"；16px 又让列表数据显示空间被压缩。
     取中间值，既保留两张卡的独立感，又不浪费垂直空间。 */
  gap: 12px;
  height: 100%;

  &__content {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
    /* 卡片内缩 20px（§2.2 p-5）：原先 12px 让列表内容（工具条/表头）落在 13px，
       而搜索卡字段在 21px——盒子对齐了、内容没对齐。改成 20 后两块内容同一条竖线，
       分页器下方留白也由这一条提供（上下不再各写一份）。 */
    padding: 20px;
    background: var(--el-bg-color);
    /* 描边与投影走 §2.2/§2.3 的卡片档（`--gowind-card-*`，定义见 _element-plus.scss）：
     * `--el-box-shadow-light` 是浮层的 12px 全向晕，静止卡片只要 1px/2px 浅落影。 */
    border: 1px solid var(--gowind-card-border);
    /* 卡片圆角 12（§2.4 大容器用 LG）：与 ProSearch 同档，见那里的注；
       内层表格面仍按 proTableConfig 的 6px（外 > 内），骨架卡 .ts-table-card 是这里的镜像。 */
    border-radius: 12px;
    box-shadow: var(--gowind-card-shadow);
  }

  &__error {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 0;
  }

  &__error-alert {
    margin-bottom: 8px;
  }

  // 全屏态：铺满视口时去掉卡片圆角，露出根节点背景色的只有搜索卡与列表卡之间的 12px 缝，
  // 用页面底色填充；小视口下内容超出时允许滚动兜底。
  &:fullscreen {
    background: var(--el-bg-color-page);
    border-radius: 0;
    overflow: auto;
  }
}
</style>
