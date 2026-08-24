<template>
  <div class="table-skeleton">
    <!-- 工具栏骨架 -->
    <div class="ts-toolbar">
      <div class="ts-toolbar__left">
        <div class="ts-btn ts-btn--primary" />
        <div class="ts-btn" />
      </div>
      <div class="ts-toolbar__right">
        <div class="ts-btn-circle" />
        <div class="ts-btn-circle" />
        <div class="ts-btn-circle" />
      </div>
    </div>

    <!-- 搜索栏骨架 -->
    <div v-if="showSearch" class="ts-search">
      <div v-for="i in searchFields" :key="i" class="ts-field" :style="stagger(i * 0.06)">
        <div class="ts-field__label" />
        <div class="ts-field__input" />
      </div>
      <div class="ts-field__btns">
        <div class="ts-btn ts-btn--primary ts-btn--sm" />
        <div class="ts-btn ts-btn--ghost ts-btn--sm" />
      </div>
    </div>

    <!-- 表格卡片骨架 -->
    <div class="ts-table-card">
      <!-- 表头 -->
      <div class="ts-thead">
        <div v-if="showSelection" class="ts-checkbox" />
        <div v-if="showIndex" class="ts-th" style="width: 50px">#</div>
        <div v-for="col in columns" :key="col.key" class="ts-th" :style="{ width: col.width }">
          {{ col.label }}
        </div>
        <div class="ts-th ts-th--end" />
      </div>
      <!-- 行 -->
      <div v-for="i in rows" :key="i" class="ts-trow" :style="stagger(i * 0.04)">
        <div v-if="showSelection" class="ts-checkbox" />
        <div v-if="showIndex" class="ts-td">
          <div class="ts-bar ts-bar--xs ts-bar--subtle" style="width: 24px" />
        </div>
        <div v-for="col in columns" :key="col.key" class="ts-td">
          <div
            class="ts-bar"
            :class="col.barClass"
            :style="{ width: col.barWidths[(i - 1) % col.barWidths.length] }"
          />
        </div>
        <div class="ts-td ts-td--end">
          <div class="ts-link-bar" />
          <div class="ts-link-sep" />
          <div class="ts-link-bar" />
        </div>
      </div>
    </div>

    <!-- 分页器骨架 -->
    <div v-if="showPagination" class="ts-pagination">
      <div class="ts-bar ts-bar--xs ts-bar--subtle" style="width: 120px" />
      <div class="ts-pagi-pages">
        <div
          v-for="p in 5"
          :key="p"
          class="ts-pagi-btn"
          :class="{ 'ts-pagi-btn--active': p === 1 }"
        />
        <div class="ts-pagi-btn ts-pagi-btn--nav" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    /** 列数（不含序号/选择/操作列） */
    columnCount?: number;
    /** 行数 */
    rowCount?: number;
    /** 是否显示搜索区 */
    showSearch?: boolean;
    /** 搜索字段数 */
    searchFieldCount?: number;
    /** 是否显示选择列 */
    showSelection?: boolean;
    /** 是否显示序号列 */
    showIndex?: boolean;
    /** 是否显示分页 */
    showPagination?: boolean;
  }>(),
  {
    columnCount: 5,
    rowCount: 8,
    showSearch: true,
    searchFieldCount: 3,
    showSelection: true,
    showIndex: true,
    showPagination: true,
  }
);

const rows = computed(() => props.rowCount);
const searchFields = computed(() => props.searchFieldCount);

// 根据列数生成伪列定义
const columns = computed(() => {
  const widthOptions = ["18%", "20%", "12%", "16%", "18%", "14%", "22%", "10%"];
  const result: Array<{
    key: number;
    label: string;
    width: string;
    barClass: string;
    barWidths: string[];
  }> = [];

  for (let i = 0; i < props.columnCount; i++) {
    const isStatusCol = i === Math.floor(props.columnCount / 2) - 1;
    const isNameCol = i === 0;

    const barWidths = isNameCol
      ? ["52%", "64%", "48%", "72%", "56%", "60%", "44%", "68%"]
      : isStatusCol
        ? ["48px", "56px", "40px", "64px", "44px", "52px", "48px", "60px"]
        : ["60%", "45%", "70%", "50%", "65%", "55%", "40%", "58%"];

    result.push({
      key: i,
      label: "",
      width: widthOptions[i % widthOptions.length],
      barClass: isStatusCol
        ? "ts-bar--tag"
        : isNameCol
          ? "ts-bar--name"
          : "ts-bar--s ts-bar--subtle",
      barWidths,
    });
  }
  return result;
});

// 交错延迟
function stagger(delay: number) {
  return { animationDelay: `${delay}s` };
}
</script>

<style lang="scss" scoped>
.table-skeleton {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

// =============== 基础骨架条 ===============
@mixin skeleton-pulse {
  position: relative;
  overflow: hidden;
  background: var(--el-fill-color-light);
  border-radius: 3px;

  &::after {
    content: "";
    position: absolute;
    inset: 0;
    background: linear-gradient(
      90deg,
      transparent 0%,
      var(--el-fill-color) 40%,
      var(--el-fill-color-lighter) 60%,
      transparent 100%
    );
    background-size: 300% 100%;
    animation: ts-shimmer 2.4s cubic-bezier(0.4, 0, 0.2, 1) infinite;
  }
}

// --- 骨架条 ---
.ts-bar {
  @include skeleton-pulse;
  height: 12px;
  flex-shrink: 0;
  animation: ts-appear 0.5s ease both;

  &--xs {
    height: 10px;
  }
  &--s {
    height: 11px;
  }
  &--xl {
    height: 26px;
  }
  &--subtle {
    opacity: 0.65;
  }
  &--name {
    height: 14px;
  }
  &--tag {
    height: 22px;
    border-radius: 4px;
    background: var(--el-color-primary-light-8);
  }
}

// --- 复选框 ---
.ts-checkbox {
  width: 15px;
  height: 15px;
  border-radius: 3px;
  border: 1.5px solid var(--el-border-color);
  flex-shrink: 0;
  margin: 0 14px 0 16px;
}

// --- 按钮 ---
.ts-btn {
  @include skeleton-pulse;
  height: 32px;
  border-radius: 6px;
  width: 72px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  animation: ts-appear 0.4s ease both;

  &--primary {
    background: var(--el-color-primary-light-7);
  }
  &--ghost {
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
  }
  &--sm {
    width: 64px;
    height: 32px;
  }
}

.ts-btn-circle {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  @include skeleton-pulse;
  flex-shrink: 0;
  background: transparent;
  border: 1px solid var(--el-border-color-lighter);
}

// --- 链接（操作列） ---
.ts-link-bar {
  width: 28px;
  height: 8px;
  border-radius: 2px;
  background: var(--el-color-primary-light-5);
  opacity: 0.4;
  flex-shrink: 0;
}

.ts-link-sep {
  width: 1px;
  height: 12px;
  background: var(--el-border-color-lighter);
  margin: 0 8px;
  flex-shrink: 0;
}

// --- 动画 ---
@keyframes ts-shimmer {
  0% {
    background-position: 300% 0;
  }
  100% {
    background-position: -300% 0;
  }
}

@keyframes ts-appear {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

// =============== 布局 ===============

// 工具栏
.ts-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;

  &__left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__right {
    display: flex;
    align-items: center;
    gap: 6px;
  }
}

// 搜索栏
.ts-search {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 0 16px;
  padding: 16px 20px;
  background: var(--el-bg-color);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: var(--el-box-shadow-light);
}

.ts-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
  min-width: 160px;
  animation: ts-appear 0.5s ease both;

  &__label {
    width: 48px;
    height: 10px;
    border-radius: 2px;
    background: var(--el-fill-color-lighter);
    @include skeleton-pulse;
  }

  &__input {
    height: 32px;
    border-radius: 6px;
    @include skeleton-pulse;
    animation: ts-appear 0.5s ease both;
  }

  &__btns {
    display: flex;
    gap: 8px;
    align-items: center;
    padding-top: 16px;
  }
}

// 表格卡片
.ts-table-card {
  background: var(--el-bg-color);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  overflow: hidden;
  box-shadow: var(--el-box-shadow-light);
  flex: 1;
}

// 表头
.ts-thead {
  display: flex;
  align-items: center;
  padding: 12px 0;
  background: var(--el-fill-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.ts-th {
  height: 12px;
  border-radius: 2px;
  background: var(--el-fill-color);
  margin: 0 12px;
  flex: 1;

  &--end {
    flex: 0 0 auto;
    width: 100px;
    margin-right: 16px;
  }
}

// 行
.ts-trow {
  display: flex;
  align-items: center;
  padding: 13px 0;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  animation: ts-appear 0.5s ease both;

  &:last-child {
    border-bottom: none;
  }
}

.ts-td {
  flex: 1;
  margin: 0 12px;
  display: flex;
  align-items: center;

  &--end {
    flex: 0 0 auto;
    width: 100px;
    margin-right: 16px;
    justify-content: flex-end;
  }
}

// 分页器
.ts-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 2px;
}

.ts-pagi-pages {
  display: flex;
  gap: 4px;
}

.ts-pagi-btn {
  width: 30px;
  height: 30px;
  border-radius: 4px;
  @include skeleton-pulse;

  &--active {
    background: var(--el-color-primary-light-7);
  }

  &--nav {
    width: 30px;
    height: 30px;
    border-radius: 4px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
  }
}
</style>
