<template>
  <div class="pro-table">
    <ElTable
      ref="tableRef"
      v-loading="loading ?? false"
      :data="data"
      :row-key="rowKey"
      border
      style="width: 100%"
      v-bind="tableAttrs"
      @selection-change="handleSelectionChange"
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

            <!-- 图片模板 -->
            <template v-else-if="col.cellType === 'image'">
              <template v-if="col.prop">
                <template v-if="Array.isArray(scope.row[col.prop])">
                  <ElImage
                    v-for="(item, idx) in scope.row[col.prop]"
                    :key="idx"
                    :src="item"
                    :preview-src-list="scope.row[col.prop]"
                    :initial-index="Number(idx)"
                    :preview-teleported="true"
                    :style="`width: ${col.imageWidth ?? 40}px; height: ${col.imageHeight ?? 40}px`"
                  />
                </template>
                <template v-else>
                  <ElImage
                    :src="scope.row[col.prop]"
                    :preview-src-list="[scope.row[col.prop]]"
                    :preview-teleported="true"
                    :style="`width: ${col.imageWidth ?? 40}px; height: ${col.imageHeight ?? 40}px`"
                  />
                </template>
              </template>
            </template>

            <!-- 标签/列表模板 -->
            <template v-else-if="col.cellType === 'tag'">
              <ElTag :type="getTagType(scope.row[col.prop], col)">
                {{ (col.labelMap ?? {})[scope.row[col.prop]] ?? scope.row[col.prop] }}
              </ElTag>
            </template>

            <!-- 开关模板 -->
            <template v-else-if="col.cellType === 'switch'">
              <ElSwitch
                v-if="col.prop"
                v-model="scope.row[col.prop]"
                :active-value="col.activeValue ?? 1"
                :inactive-value="col.inactiveValue ?? 0"
                :inline-prompt="true"
                :active-text="col.activeText ?? ''"
                :inactive-text="col.inactiveText ?? ''"
                :validate-event="false"
                @change="
                  (val: any) => emit('modify', { row: scope.row, field: col.prop!, value: val })
                "
              />
            </template>

            <!-- 日期模板 -->
            <template v-else-if="col.cellType === 'date'">
              {{
                scope.row[col.prop]
                  ? useDateFormat(scope.row[col.prop], col.dateFormat ?? "YYYY-MM-DD HH:mm:ss")
                      .value
                  : ""
              }}
            </template>

            <!-- 链接模板 -->
            <template v-else-if="col.cellType === 'link'">
              <ElLink type="primary" :href="scope.row[col.prop]" target="_blank">
                {{ scope.row[col.prop] }}
              </ElLink>
            </template>

            <!-- 价格模板 -->
            <template v-else-if="col.cellType === 'price'">
              {{ `${col.pricePrefix ?? ""}${scope.row[col.prop]}` }}
            </template>

            <!-- 百分比模板 -->
            <template v-else-if="col.cellType === 'percent'">{{ scope.row[col.prop] }}%</template>

            <!-- 图标模板 -->
            <template v-else-if="col.cellType === 'icon'">
              <ElIcon><component :is="scope.row[col.prop]" /></ElIcon>
            </template>

            <!-- 操作列模板 -->
            <template v-else-if="col.cellType === 'tool'">
              <template v-for="(btn, idx) in col.buttons" :key="idx">
                <ElButton
                  v-if="btn.visible === undefined || btn.visible(scope.row)"
                  v-bind="{ link: true, size: 'small', ...btn.attrs }"
                  @click="emit('operate', { name: btn.name, row: scope.row, $index: scope.$index })"
                >
                  {{ btn.text ?? btn.name }}
                </ElButton>
              </template>
            </template>

            <!-- 默认显示 -->
            <template v-else>
              {{ col.prop ? scope.row[col.prop] : "" }}
            </template>
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
import { useDateFormat } from "@vueuse/core";
import {
  ElTable,
  ElTableColumn,
  ElImage,
  ElTag,
  ElSwitch,
  ElLink,
  ElButton,
  ElIcon,
} from "element-plus";
import ProPagination from "../ProPagination/index.vue";
import type { ProTableProps, ProTableColumn } from "./types";

const props = withDefaults(defineProps<ProTableProps<T>>(), {
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

const tableRef = ref<InstanceType<typeof ElTable>>();

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

// 选中处理
function handleSelectionChange(rows: T[]) {
  emit("selection-change", rows);
}

// Tag 类型判断
function getTagType(
  value: any,
  col: ProTableColumn<T>
): "primary" | "success" | "warning" | "danger" | "info" {
  if (col.tagType) return col.tagType as any;
  return value ? "success" : "danger";
}

defineExpose({
  tableRef,
  getSelectionRows: () => tableRef.value?.getSelectionRows(),
  clearSelection: () => tableRef.value?.clearSelection(),
  toggleRowSelection: (row: T, selected?: boolean) =>
    tableRef.value?.toggleRowSelection(row, selected),
});
</script>

<style scoped>
.pro-table {
  margin-top: 16px;
}
</style>
