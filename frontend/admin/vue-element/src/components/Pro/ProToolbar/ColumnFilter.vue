<template>
  <div class="column-filter">
    <ElScrollbar max-height="350px">
      <!-- 头部全选 -->
      <div class="column-filter__header">
        <ElCheckbox
          :model-value="isAllSelected"
          :indeterminate="isIndeterminate"
          @change="handleToggleAll"
        >
          {{ t("pages.curd.toolbar.filterAll") }}
        </ElCheckbox>
      </div>

      <!-- 列列表 -->
      <div ref="listRef" class="column-filter__list">
        <div
          v-for="col in columns"
          :key="col.prop"
          class="column-filter__item"
          :class="{ 'column-filter__item--fixed': col.fixed }"
          :data-prop="col.prop"
        >
          <!-- 拖拽手柄 -->
          <ElIcon class="column-filter__drag-handle" :size="16">
            <Rank />
          </ElIcon>

          <!-- 列名 -->
          <ElCheckbox v-model="col.show" :label="col.label" class="column-filter__label" />

          <!-- 操作按钮 -->
          <div class="column-filter__actions">
            <!-- 固定按钮 -->
            <ElIcon
              class="column-filter__action-icon"
              :class="{ 'column-filter__action-icon--active': col.fixed === 'left' }"
              :size="16"
              @click="handleToggleFixed(col, 'left')"
            >
              <ArrowLeft />
            </ElIcon>
            <ElIcon
              class="column-filter__action-icon"
              :class="{ 'column-filter__action-icon--active': col.fixed === 'right' }"
              :size="16"
              @click="handleToggleFixed(col, 'right')"
            >
              <ArrowRight />
            </ElIcon>
          </div>
        </div>
      </div>
    </ElScrollbar>

    <!-- 底部按钮 -->
    <div class="column-filter__footer">
      <ElButton text @click="handleReset">{{ t("pages.curd.toolbar.filterReset") }}</ElButton>
      <ElButton text @click="handleCancel">{{ t("common.button.cancel") }}</ElButton>
      <ElButton type="primary" text @click="handleConfirm">
        {{ t("common.button.confirm") }}
      </ElButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from "vue";
import { ElCheckbox, ElButton, ElIcon, ElScrollbar } from "element-plus";
import { Rank, ArrowLeft, ArrowRight } from "@element-plus/icons-vue";
import { useI18n } from "@go-wind/admin-element";
import Sortable from "sortablejs";
import type { ProTableColumn } from "../ProTable/types";

const { t } = useI18n();

const props = defineProps<{
  columns: ProTableColumn[];
}>();

const emit = defineEmits<{
  confirm: [columns: ProTableColumn[]];
  cancel: [];
}>();

const listRef = ref<HTMLElement>();

let sortable: Sortable | null = null;

// 可筛选的列
const filterableColumns = computed(() => props.columns.filter((col) => col.prop && col.label));

// 是否全选
const isAllSelected = computed(
  () => filterableColumns.value.length > 0 && filterableColumns.value.every((col) => col.show)
);

// 是否部分选中
const isIndeterminate = computed(() => {
  const selectedCount = filterableColumns.value.filter((col) => col.show).length;
  return selectedCount > 0 && selectedCount < filterableColumns.value.length;
});

// 初始化拖拽排序
onMounted(() => {
  if (listRef.value) {
    sortable = Sortable.create(listRef.value, {
      handle: ".column-filter__drag-handle",
      animation: 150,
      onEnd: () => {
        // 拖拽结束后，更新列顺序
        const items = listRef.value?.querySelectorAll(".column-filter__item");
        if (!items) return;

        const newOrder: string[] = [];
        items.forEach((item) => {
          const prop = item.getAttribute("data-prop");
          if (prop) newOrder.push(prop);
        });

        // 按照新顺序重新排列 columns
        const newColumns = [...props.columns].sort((a, b) => {
          const indexA = newOrder.indexOf(a.prop as string);
          const indexB = newOrder.indexOf(b.prop as string);
          return indexA - indexB;
        });

        emit("confirm", newColumns);
      },
    });
  }
});

// 组件卸载时销毁 Sortable 实例
onBeforeUnmount(() => {
  sortable?.destroy();
  sortable = null;
});

// 全选/取消全选
function handleToggleAll(checked: boolean | string | number) {
  filterableColumns.value.forEach((col) => {
    col.show = !!checked;
  });
}

// 切换固定状态
function handleToggleFixed(col: ProTableColumn, position: "left" | "right") {
  if (col.fixed === position) {
    col.fixed = false;
  } else {
    col.fixed = position;
  }
}

// 恢复默认
function handleReset() {
  props.columns.forEach((col) => {
    col.show = true;
    col.fixed = false;
  });
  emit("confirm", [...props.columns]);
}

// 取消
function handleCancel() {
  emit("cancel");
}

// 确认
function handleConfirm() {
  emit("confirm", [...props.columns]);
}
</script>

<style scoped lang="scss">
.column-filter {
  min-width: 280px;
  max-width: 400px;
}

.column-filter__header {
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color);
  font-weight: 500;
}

.column-filter__list {
  padding: 8px 0;
}

.column-filter__item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  gap: 8px;
  cursor: move;
  transition: background-color 0.2s;

  &:hover {
    background: var(--el-fill-color-light);
  }
}

.column-filter__drag-handle {
  color: var(--el-text-color-secondary);
  cursor: grab;
  flex-shrink: 0;

  &:active {
    cursor: grabbing;
  }
}

.column-filter__label {
  flex: 1;
  min-width: 0;

  :deep(.el-checkbox__label) {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.column-filter__actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.column-filter__action-icon {
  color: var(--el-text-color-secondary);
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s;

  &:hover {
    color: var(--gowind-primary-text);
    background: var(--el-fill-color);
  }
}

.column-filter__action-icon--active {
  color: var(--gowind-primary-text);
}

.column-filter__footer {
  display: flex;
  justify-content: space-between;
  padding: 12px;
  border-top: 1px solid var(--el-border-color);
  gap: 8px;
}
</style>
