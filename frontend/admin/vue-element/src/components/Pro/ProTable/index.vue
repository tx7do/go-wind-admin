<template>
  <div class="pro-table">
    <!-- ======================== vxe-table 引擎 ======================== -->
    <vxe-table
      v-if="engine === 'vxe'"
      :id="tableId"
      ref="tableRef"
      v-loading="loading ?? false"
      :row-config="{ keyField: rowKey, isHover: true, isCurrent: true }"
      :column-config="{ resizable: true }"
      :custom-config="{ storage: !!tableId, checkMethod }"
      :data="data"
      class="w-full"
      v-bind="tableAttrs"
      @checkbox-change="handleVxeSelectionChange"
      @checkbox-all="handleVxeSelectionChange"
      @current-change="handleVxeCurrentChange"
    >
      <template v-for="col in resolvedColumns" :key="col.prop ?? col.type">
        <!-- 复选框列 -->
        <vxe-column v-if="col.type === 'selection'" type="checkbox" width="50" align="center" />
        <!-- 序号列 -->
        <vxe-column v-else-if="col.type === 'index'" type="seq" width="60" align="center" />
        <!-- 展开列 -->
        <vxe-column v-else-if="col.type === 'expand'" type="expand">
          <template #default="scope">
            <slot name="expand" v-bind="scope" />
          </template>
        </vxe-column>
        <!-- 普通列 -->
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
          :sortable="col.sortable === true"
          :resizable="col.resizable !== false"
          :tree-node="col.treeNode"
          v-bind="col.attrs"
        >
          <template #default="scope">
            <!-- 自定义插槽 -->
            <slot
              v-if="col.cellType === 'custom' || col.slotName"
              :name="col.slotName ?? col.prop"
              :row="scope.row"
              :column="scope.column"
              :index="scope.rowIndex"
            />
            <!-- 共享单元格渲染 -->
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

    <!-- ======================== el-table 引擎 ======================== -->
    <ElTable
      v-else
      ref="tableRef"
      v-loading="loading ?? false"
      :data="data"
      :row-key="rowKey"
      border
      style="width: 100%"
      v-bind="tableAttrs"
      @selection-change="handleElSelectionChange"
      @current-change="handleElCurrentChange"
    >
      <template v-for="col in resolvedColumns" :key="col.prop ?? col.type">
        <!-- 复选框列 -->
        <ElTableColumn
          v-if="col.type === 'selection'"
          type="selection"
          width="50"
          :reserve-selection="col.reserveSelection ?? true"
        />
        <!-- 序号列 -->
        <ElTableColumn v-else-if="col.type === 'index'" type="index" width="60" />
        <!-- 展开列 -->
        <ElTableColumn v-else-if="col.type === 'expand'" type="expand">
          <template #default="scope">
            <slot name="expand" v-bind="scope" />
          </template>
        </ElTableColumn>
        <!-- 普通列 -->
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
          v-bind="col.attrs"
        >
          <template #default="scope">
            <!-- 自定义插槽 -->
            <slot
              v-if="col.cellType === 'custom' || col.slotName"
              :name="col.slotName ?? col.prop"
              :row="scope.row"
              :column="scope.column"
              :index="scope.$index"
            />
            <!-- 共享单元格渲染 -->
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

    <!-- 分页 -->
    <ProPagination
      v-if="pagination"
      :current-page="currentPage ?? 1"
      :page-size="pageSize ?? 20"
      :total="total ?? 0"
      :page-sizes="pageSizes"
      @current-change="(val: number) => emit('current-change', val)"
      @size-change="(val: number) => emit('size-change', val)"
    />
  </div>
</template>

<script setup lang="ts" generic="T extends Record<string, any>">
import { ref, computed } from "vue";
import { ElTable, ElTableColumn } from "element-plus";
import ProPagination from "../ProPagination/index.vue";
import ProTableCellContent from "./ProTableCellContent.vue";
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
}>();

const tableRef = ref<any>(null);
const engine = props.engine;
const tableId = props.tableId;

// 解析列（支持 initFn 和 show 默认值）
// 使用 computed 以响应外部 columns prop 的变化（如语言切换时 computed pageConfig 重新求值）
const resolvedColumns = computed(() =>
  [...props.columns].map((col) => {
    const resolved = { ...col };
    if (resolved.initFn) resolved.initFn(resolved as unknown as Record<string, any>);
    if (resolved.show === undefined) resolved.show = true;
    return resolved;
  })
);

// 透传 table 属性
const tableAttrs = props.table ?? {};
const rowKey = props.rowKey;

// vxe-table customConfig checkMethod: 控制 selection/index 列不可被自定义隐藏
function checkMethod({ column }: { column: any }) {
  const field = column.field ?? column.type;
  return field !== "checkbox" && field !== "seq" && column.type !== "checkbox";
}

// === vxe-table 选中处理 ===
function handleVxeSelectionChange({ records }: { records: T[] }) {
  emit("selection-change", records);
}

// === el-table 选中处理 ===
function handleElSelectionChange(rows: T[]) {
  emit("selection-change", rows);
}

// === vxe-table 行点击（高亮行切换） ===
function handleVxeCurrentChange({ row }: { row: T }) {
  emit("row-click", row);
}

// === el-table 行点击 ===
function handleElCurrentChange(row: T) {
  emit("row-click", row);
}

// === 统一 expose API ===
defineExpose({
  tableRef,
  resolvedColumns,
  getSelectionRows: () => {
    if (engine === "vxe") {
      return tableRef.value?.getCheckboxRecords?.();
    }
    return tableRef.value?.getSelectionRows?.();
  },
  clearSelection: () => {
    if (engine === "vxe") {
      tableRef.value?.clearCheckboxRow?.();
    } else {
      tableRef.value?.clearSelection?.();
    }
  },
  toggleRowSelection: (row: T, selected?: boolean) => {
    if (engine === "vxe") {
      tableRef.value?.setCheckboxRow?.(row, selected ?? true);
    } else {
      tableRef.value?.toggleRowSelection?.(row, selected);
    }
  },
});
</script>

<style lang="scss" scoped>
.pro-table {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

// ======================== vxe-table 样式 ========================
:deep(.vxe-table) {
  border-radius: 6px;
  overflow: hidden;
  font-size: 13px;

  // --- 表头 ---
  .vxe-table--header-wrapper {
    background-color: var(--el-fill-color-light);

    .vxe-header--column {
      height: 44px !important;

      .vxe-cell {
        height: 44px !important;
        line-height: 44px !important;
        font-weight: 600;
        font-size: 13px;
        color: var(--el-text-color-primary);
      }
    }
  }

  // --- 去掉列间竖线 ---
  .vxe-body--column,
  .vxe-header--column,
  .vxe-footer--column {
    border-right: none !important;
  }

  // --- 表体行 ---
  .vxe-table--body-wrapper {
    background-color: var(--el-bg-color);
  }

  .vxe-body--row {
    background-color: var(--el-bg-color);
    transition: background-color 0.15s ease;

    .vxe-body--column {
      height: 42px !important;

      .vxe-cell {
        height: 42px !important;
        line-height: 42px !important;
      }
    }

    &.row--hover {
      background-color: var(--el-fill-color-light);
    }

    &.row--current {
      background-color: var(--el-color-primary-light-9);
    }
  }

  // --- 单元格文本 ---
  .vxe-cell {
    color: var(--el-text-color-regular);
  }

  // --- 复选框/序号列居中 ---
  .vxe-checkbox--icon,
  .vxe-table--checkbox-column {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  // --- 表格底部边框 ---
  &.border--default .vxe-table--border-line {
    border-color: var(--el-border-color-lighter);
  }

  // --- 空数据 ---
  .vxe-table--empty-content {
    color: var(--el-text-color-placeholder);
    padding: 32px 0;
  }
}

// ======================== el-table 样式 ========================
:deep(.el-table) {
  border-radius: 6px;
  overflow: hidden;
  font-size: 13px;

  // 表头
  .el-table__header-wrapper {
    .el-table__header th {
      height: 44px;
      font-weight: 600;
      font-size: 13px;
      color: var(--el-text-color-primary);
      background-color: var(--el-fill-color-light);
    }
  }

  // 表体行
  .el-table__body-wrapper {
    .el-table__row td {
      height: 42px;
      padding: 0;
    }
  }

  // 空数据
  .el-table__empty-block {
    color: var(--el-text-color-placeholder);
    padding: 32px 0;
  }
}
</style>
