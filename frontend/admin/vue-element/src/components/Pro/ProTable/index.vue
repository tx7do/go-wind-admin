<template>
  <div class="pro-table">
    <!-- ======================== vxe-table 引擎 ======================== -->
    <vxe-table
      v-if="engine === 'vxe'"
      ref="tableRef"
      v-loading="loading ?? false"
      :row-config="{ keyField: rowKey, isHover: true, isCurrent: true }"
      :column-config="{ resizable: true }"
      :custom-config="{ storage: true, checkMethod }"
      :data="data"
      class="w-full"
      v-bind="tableAttrs"
      @checkbox-change="handleVxeSelectionChange"
      @checkbox-all="handleVxeSelectionChange"
    >
      <template v-for="col in resolvedColumns" :key="col.prop ?? col.type">
        <!-- 复选框列 -->
        <vxe-column
          v-if="col.type === 'selection'"
          type="checkbox"
          width="50"
          align="center"
        />
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
          :fixed="col.fixed === 'left' || col.fixed === 'right' ? col.fixed : col.fixed === true ? 'left' : undefined"
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
import { ref } from "vue";
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
}>();

const tableRef = ref<any>(null);
const engine = props.engine;

// 解析列（支持 initFn 和 show 默认值）
const resolvedColumns = ref(
  [...props.columns].map((col) => {
    if (col.initFn) col.initFn(col as unknown as Record<string, any>);
    if (col.show === undefined) col.show = true;
    return col;
  })
);

// 透传 table 属性
const tableAttrs = props.table ?? {};
const rowKey = props.rowKey;

// vxe-table customConfig checkMethod: 控制 selection/index 列不可被自定义隐藏
function checkMethod({ column }: { column: any }) {
  const field = column.field ?? column.type;
  return field !== 'checkbox' && field !== 'seq' && column.type !== 'checkbox';
}

// === vxe-table 选中处理 ===
function handleVxeSelectionChange({ records }: { records: T[] }) {
  emit("selection-change", records);
}

// === el-table 选中处理 ===
function handleElSelectionChange(rows: T[]) {
  emit("selection-change", rows);
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
  overflow: auto;
}

// vxe-table 样式优化（对齐 CURD PageContent）
:deep(.vxe-table) {
  border-radius: 4px;
  overflow: hidden;

  // 表头样式
  .vxe-table--header-wrapper {
    background-color: var(--el-fill-color-light);

    .vxe-header--column {
      height: 45px !important;
      line-height: 45px !important;

      .vxe-cell {
        height: 45px !important;
        line-height: 45px !important;
      }
    }

    .vxe-cell {
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  // 表格主体样式
  .vxe-table--body-wrapper {
    background-color: var(--el-bg-color);
  }

  // 去掉列之间的分割线
  .vxe-body--column,
  .vxe-header--column,
  .vxe-footer--column {
    border-right: none !important;
  }

  // 表格行样式
  .vxe-body--row {
    background-color: var(--el-bg-color);

    .vxe-body--column {
      height: 40px !important;
      line-height: 40px !important;

      .vxe-cell {
        height: 40px !important;
        line-height: 40px !important;
      }
    }

    &.row--hover {
      background-color: var(--el-fill-color-light);
    }

    &.row--current {
      background-color: var(--el-fill-color);
    }
  }

  // 暗黑模式下的悬停效果增强
  html.dark & .vxe-body--row {
    background-color: #1a1a1a !important;
    transition: background-color 0.2s ease !important;

    &:hover,
    &.row--hover {
      background-color: #2a2a2a !important;

      > td {
        background-color: transparent !important;
      }
    }

    &.row--current {
      background-color: #333333 !important;

      > td {
        background-color: transparent !important;
      }
    }

    &.row--hover.row--current {
      background-color: #333333 !important;
    }
  }

  // 表格单元格
  .vxe-cell {
    color: var(--el-text-color-regular);
  }

  // 表格边框
  &.vxe-table--border-line--inner {
    border-color: var(--el-border-color);
  }
}
</style>
