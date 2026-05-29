<template>
  <div ref="tableWrapperRef" class="pro-table" :style="tableStyle">
    <vxe-table
      v-if="engine === 'vxe'"
      :id="tableId"
      ref="tableRef"
      v-loading="loading ?? false"
      :row-config="{ keyField: rowKey, isHover: true, isCurrent: true }"
      :column-config="{ resizable: true }"
      :custom-config="{ storage: !!tableId, checkMethod }"
      :height="vxeTableHeight"
      :data="data"
      class="w-full"
      v-bind="tableAttrs"
      @checkbox-change="handleVxeSelectionChange"
      @checkbox-all="handleVxeSelectionChange"
      @current-change="handleVxeCurrentChange"
      @sort-change="handleSortChange"
    >
      <template v-for="(col, colIdx) in resolvedColumns" :key="col.prop ?? col.type ?? colIdx">
        <vxe-column v-if="col.type === 'selection'" type="checkbox" width="50" align="center" />
        <vxe-column v-else-if="col.type === 'index'" type="seq" width="60" align="center" />
        <vxe-column v-else-if="col.type === 'expand'" type="expand">
          <template #default="scope">
            <slot name="expand" v-bind="scope" />
          </template>
        </vxe-column>
        <vxe-column
          v-else-if="col.show !== false"
          :field="col.prop"
          :title="col.label"
          :width="col.width"
          :min-width="col.minWidth"
          :fixed="
            col.fixed === 'left' || col.fixed === 'right'
              ? col.fixed
              : col.fixed === true
                ? 'left'
                : undefined
          "
          :align="col.align || 'center'"
          :sortable="!!col.sortable"
          :show-overflow="col.showOverflowTooltip ? 'tooltip' : false"
          :resizable="col.resizable !== false"
          :tree-node="col.treeNode"
          :cell-style="col.cellStyle"
          :header-style="col.headerStyle"
          :class-name="col.className"
          :header-class-name="col.headerClassName"
          v-bind="col.attrs"
        >
          <template #default="scope">
            <slot
              v-if="col.cellType === 'custom' || col.slotName"
              :name="col.slotName ?? col.prop"
              :row="scope.row"
              :column="scope.column"
              :index="scope.rowIndex"
            />
            <ProTableCellContent
              v-else
              :col="col"
              :row="scope.row"
              :row-index="scope.rowIndex"
              @modify="(d: any) => emit('modify', d)"
              @operate="(d: any) => emit('operate', d)"
            />
          </template>
        </vxe-column>
      </template>
    </vxe-table>

    <ElTable
      v-else
      ref="tableRef"
      v-loading="loading ?? false"
      :data="data"
      :row-key="rowKey"
      style="width: 100%"
      v-bind="tableAttrs"
      @selection-change="handleElSelectionChange"
      @current-change="handleElCurrentChange"
      @sort-change="handleSortChange"
    >
      <template v-for="(col, colIdx) in resolvedColumns" :key="col.prop ?? col.type ?? colIdx">
        <ElTableColumn
          v-if="col.type === 'selection'"
          type="selection"
          width="50"
          :reserve-selection="col.reserveSelection ?? true"
        />
        <ElTableColumn v-else-if="col.type === 'index'" type="index" width="60" />
        <ElTableColumn v-else-if="col.type === 'expand'" type="expand">
          <template #default="scope">
            <slot name="expand" v-bind="scope" />
          </template>
        </ElTableColumn>
        <ElTableColumn
          v-else-if="col.show !== false"
          :prop="col.prop"
          :label="col.label"
          :width="col.width"
          :min-width="col.minWidth"
          :fixed="col.fixed"
          :sortable="col.sortable"
          :align="col.align || 'center'"
          :show-overflow-tooltip="col.showOverflowTooltip ?? true"
          :cell-style="col.cellStyle"
          :header-style="col.headerStyle"
          :class-name="col.className"
          :header-class-name="col.headerClassName"
          v-bind="col.attrs"
        >
          <template #default="scope">
            <slot
              v-if="col.cellType === 'custom' || col.slotName"
              :name="col.slotName ?? col.prop"
              :row="scope.row"
              :column="scope.column"
              :index="scope.$index"
            />
            <ProTableCellContent
              v-else
              :col="col"
              :row="scope.row"
              :row-index="scope.$index"
              @modify="(d: any) => emit('modify', d)"
              @operate="(d: any) => emit('operate', d)"
            />
          </template>
        </ElTableColumn>
      </template>
    </ElTable>

    <ProPagination
      v-if="paginationConfig.show && pagination !== false"
      :current-page="currentPage ?? 1"
      :page-size="pageSize ?? paginationConfig.pageSize"
      :total="total ?? 0"
      :page-sizes="pageSizes ?? paginationConfig.pageSizes"
      :layout="paginationConfig.layout"
      :background="paginationConfig.background"
      @current-change="(val: number) => emit('current-change', val)"
      @size-change="(val: number) => emit('size-change', val)"
    />
  </div>
</template>

<script setup lang="ts" generic="T extends Record<string, any>">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from "vue";
import { ElTable, ElTableColumn } from "element-plus";
import ProPagination from "../ProPagination/index.vue";
import ProTableCellContent from "./ProTableCellContent.vue";
import { proTableGlobalConfig, mergeTableConfig } from "./proTableConfig";
import type { ProTableProps } from "./types";

const props = withDefaults(defineProps<ProTableProps<T>>(), {
  engine: "vxe",
  rowKey: "id",
  total: 0,
  currentPage: 1,
  pageSize: 20,
});

const emit = defineEmits<{
  "selection-change": [rows: T[]];
  modify: [data: { row: T; field: string; value: any }];
  operate: [data: { name: string; row: T; $index: number }];
  "current-change": [currentPage: number];
  "size-change": [pageSize: number];
  "row-click": [row: T];
  "sort-change": [data: { prop: string; order: string | null }];
}>();

const tableRef = ref<any>(null);
const tableWrapperRef = ref<HTMLElement | null>(null);
const vxeTableHeight = ref<number | undefined>(undefined);
const engine = props.engine;
const tableId = props.tableId;
const rowKey = props.rowKey;

// ResizeObserver: 动态计算 vxe-table 的像素高度
let resizeObserver: ResizeObserver | null = null;

function updateTableHeight() {
  if (!tableWrapperRef.value) return;
  const wrapperHeight = tableWrapperRef.value.clientHeight;
  if (wrapperHeight <= 0) return;
  // 减去分页器高度（如果存在）
  const pagerEl = tableWrapperRef.value.querySelector(".pro-pagination") as HTMLElement | null;
  const pagerHeight = pagerEl ? pagerEl.offsetHeight + 12 : 0; // 12 = margin/gap
  vxeTableHeight.value = wrapperHeight - pagerHeight;
}

onMounted(() => {
  nextTick(() => {
    if (tableWrapperRef.value) {
      resizeObserver = new ResizeObserver(() => {
        updateTableHeight();
      });
      resizeObserver.observe(tableWrapperRef.value);
      updateTableHeight();
    }
  });
});

// 分页器显隐变化时重新计算
watch(
  () => [props.pagination, props.total],
  () => nextTick(updateTableHeight)
);

onUnmounted(() => {
  resizeObserver?.disconnect();
  resizeObserver = null;
});

const tableConfig = computed(() => mergeTableConfig(proTableGlobalConfig, props.config));
const paginationConfig = computed(() => tableConfig.value.pagination!);

const tableStyle = computed(() => ({
  "--pro-table-radius": `${tableConfig.value.borderRadius}px`,
  "--pro-table-font-size": `${tableConfig.value.fontSize}px`,
}));

const tableAttrs = computed(() => ({
  stripe: tableConfig.value.stripe,
  border: tableConfig.value.border,
  emptyText: tableConfig.value.emptyText,
  ...props.table,
}));

const resolvedColumns = computed(() =>
  [...props.columns].map((col) => {
    const resolved = { ...col };
    const colDef = tableConfig.value.column!;

    if (resolved.align === undefined) resolved.align = colDef.align;
    if (resolved.resizable === undefined) resolved.resizable = colDef.resizable;
    if (resolved.showOverflowTooltip === undefined) {
      resolved.showOverflowTooltip = colDef.showOverflowTooltip;
    }
    if (resolved.sortable === undefined) resolved.sortable = colDef.sortable;

    if (resolved.initFn) resolved.initFn(resolved);
    if (resolved.show === undefined) resolved.show = true;
    return resolved;
  })
);

function checkMethod({ column }: { column: any }) {
  const field = column.field ?? column.type;
  return field !== "checkbox" && field !== "seq" && column.type !== "checkbox";
}

function handleVxeSelectionChange({ records }: { records: T[] }) {
  emit("selection-change", records);
}

function handleElSelectionChange(rows: T[]) {
  emit("selection-change", rows);
}

function handleVxeCurrentChange({ row }: { row: T }) {
  emit("row-click", row);
}

function handleElCurrentChange(row: T) {
  emit("row-click", row);
}

function handleSortChange(data: any) {
  const prop = data.prop ?? data.property ?? data.field;
  const order = data.order;
  emit("sort-change", { prop, order });
}

defineExpose({
  tableRef,
  resolvedColumns,
  getSelectionRows: () => {
    if (props.engine === "vxe") {
      return tableRef.value?.getCheckboxRecords?.();
    }
    return tableRef.value?.getSelectionRows?.();
  },
  clearSelection: () => {
    if (props.engine === "vxe") {
      tableRef.value?.clearCheckboxRow?.();
    } else {
      tableRef.value?.clearSelection?.();
    }
  },
  toggleRowSelection: (row: T, selected?: boolean) => {
    if (props.engine === "vxe") {
      tableRef.value?.setCheckboxRow?.(row, selected ?? true);
    } else {
      tableRef.value?.toggleRowSelection?.(row, selected);
    }
  },
});
</script>

<style lang="scss" scoped>
.pro-table {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

:deep(.vxe-table) {
  border-radius: var(--pro-table-radius);
  font-size: var(--pro-table-font-size);

  // 暗色模式去掉列分割线（border:outer 仍会在列间渲染细线）
  [data-vxe-ui-theme="dark"] & {
    .vxe-body--column,
    .vxe-header--column,
    .vxe-footer--column {
      border-right: none !important;
    }
  }

  .vxe-table--header-wrapper .vxe-header--column .vxe-cell {
    display: flex;
    align-items: center;
    color: var(--el-text-color-primary);
  }

  // 表项单元格默认文本色
  .vxe-body--column .vxe-cell {
    color: var(--el-text-color-regular);
  }

  // 操作列按钮容器居中（针对 cellType: "tool"）
  .vxe-body--column .flex.items-center.justify-center {
    justify-content: center !important;
    width: 100%;
  }

  .vxe-checkbox--icon,
  .vxe-table--checkbox-column {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  &.border--outer .vxe-table--border-line {
    border-color: var(--el-border-color-lighter);
  }

  .vxe-table--empty-content {
    color: var(--el-text-color-placeholder);
    padding: 32px 0;
  }
}

:deep(.el-table) {
  border-radius: var(--pro-table-radius);
  overflow: auto;
  font-size: var(--pro-table-font-size);
  max-height: 100%;

  .el-table__header-wrapper th {
    height: var(--pro-table-header-height);
    font-weight: 600;
    color: var(--el-text-color-primary);
    background-color: var(--el-fill-color-light);
  }

  .el-table__row td {
    height: var(--pro-table-row-height);
    padding: 0;
  }

  .el-table__empty-block {
    color: var(--el-text-color-placeholder);
    padding: 32px 0;
  }
}
</style>
