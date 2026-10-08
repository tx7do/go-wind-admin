<script lang="ts" setup>
import type { VxeGridPropTypes } from 'vxe-table';

import { computed, onMounted, ref, useTemplateRef } from 'vue';

defineOptions({ name: 'VbenVxeGridSkeleton' });

const props = withDefaults(
  defineProps<{
    columns?: VxeGridPropTypes.Columns;
    rows?: number;
  }>(),
  { columns: () => [], rows: 8 },
);

/** 单元格占位条宽度轮换表：整列等长会读成「数据已到位、内容恰好相同」，按 (行+列) 打散 */
const BAR_WIDTHS = ['78%', '62%', '84%', '54%', '70%'];

const DEFAULT_CELL_WIDTH = 120;
/** 与下方 CSS 里的 __head / __row 高度同源，用于按容器高度折算能铺几行 */
const HEAD_HEIGHT = 32;
const ROW_HEIGHT = 32;

function toPixels(width: unknown): null | number {
  const num = Number.parseInt(String(width ?? ''), 10);
  return Number.isFinite(num) && num > 0 ? num : null;
}

interface SkeletonCell {
  key: string;
  style: Record<string, string>;
}

/**
 * 骨架的列宽取页面真实 columns：定宽列（写了 width）不伸缩，自适应列（只有 minWidth 或都没写）
 * 按 minWidth 起底再吃掉剩余空间——vxe 就是这么分配宽度的，照抄才不会跟真实表格对不齐。
 */
const cells = computed<SkeletonCell[]>(() => {
  const result: SkeletonCell[] = [];
  const walk = (cols: any[], prefix: string) => {
    cols.forEach((column, index) => {
      if (column?.visible === false) {
        return;
      }
      if (Array.isArray(column?.columns)) {
        walk(column.columns, `${prefix}${index}-`);
        return;
      }
      const fixedWidth = toPixels(column?.width);
      const minWidth = toPixels(column?.minWidth) ?? DEFAULT_CELL_WIDTH;
      result.push({
        key: `${prefix}${column?.field ?? column?.type ?? index}`,
        style: {
          flex:
            fixedWidth === null ? `1 1 ${minWidth}px` : `0 0 ${fixedWidth}px`,
        },
      });
    });
  };
  walk(props.columns ?? [], '');
  return result;
});

const rootRef = useTemplateRef<HTMLElement>('root');
/** 实测能放下的行数（0 = 还没量到）：遮罩多高由 vxe 的表体决定，首屏空表只有一小条 */
const fittedRows = ref(0);

const visibleRows = computed(() => {
  if (fittedRows.value === 0) {
    return props.rows;
  }
  return Math.min(fittedRows.value, props.rows);
});

/**
 * 一次性同步测量，不用 ResizeObserver：RO 的回调挂在渲染帧上，页面切到后台就不跑，
 * 而骨架只在加载中的这几百毫秒里存在，等不到那一帧就会整块裁在遮罩里出不来。
 * 量不到高度（0）时宁可退回 props.rows，也不要铺成一行。
 */
onMounted(() => {
  const height = rootRef.value?.clientHeight ?? 0;
  if (height > HEAD_HEIGHT) {
    fittedRows.value = Math.max(
      1,
      Math.floor((height - HEAD_HEIGHT) / ROW_HEIGHT),
    );
  }
});
</script>

<template>
  <div ref="root" class="vben-grid-skeleton">
    <div class="vben-grid-skeleton__head">
      <span
        v-for="cell in cells"
        :key="`h-${cell.key}`"
        :style="cell.style"
        class="vben-grid-skeleton__cell"
      >
        <i class="vben-grid-skeleton__head-bar"></i>
      </span>
    </div>
    <div
      v-for="row in visibleRows"
      :key="row"
      class="vben-grid-skeleton__row"
    >
      <span
        v-for="(cell, cellIndex) in cells"
        :key="cell.key"
        :style="cell.style"
        class="vben-grid-skeleton__cell"
      >
        <i
          class="vben-grid-skeleton__bar"
          :style="{ width: BAR_WIDTHS[(row + cellIndex) % BAR_WIDTHS.length] }"
        ></i>
      </span>
    </div>
  </div>
</template>

<style scoped>
.vben-grid-skeleton {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  padding: 0 8px;
  overflow: hidden;
  background-color: hsl(var(--card));
}

.vben-grid-skeleton__cell {
  display: flex;
  align-items: center;
  min-width: 0;
  padding: 0 8px;
}

.vben-grid-skeleton__head,
.vben-grid-skeleton__row {
  display: flex;
  align-items: center;
}

.vben-grid-skeleton__head {
  height: 32px;
  background-color: hsl(var(--accent));
}

.vben-grid-skeleton__row {
  height: 32px;
  border-bottom: 1px solid hsl(var(--border));
}

.vben-grid-skeleton__head-bar {
  width: 60%;
  height: 10px;
  background-color: hsl(var(--accent-foreground) / 18%);
}

.vben-grid-skeleton__bar {
  position: relative;
  height: 12px;
  overflow: hidden;
  background-color: hsl(var(--foreground) / 10%);
  border-radius: 2px;
}

.vben-grid-skeleton__bar::after {
  position: absolute;
  inset: 0;
  content: '';
  background: linear-gradient(
    90deg,
    transparent 0%,
    hsl(var(--foreground) / 6%) 40%,
    hsl(var(--foreground) / 14%) 60%,
    transparent 100%
  );
  background-size: 300% 100%;
  animation: vben-grid-skeleton-shimmer 2.4s cubic-bezier(0.4, 0, 0.2, 1) infinite;
}

@keyframes vben-grid-skeleton-shimmer {
  0% {
    background-position: 300% 0;
  }

  100% {
    background-position: -300% 0;
  }
}
</style>
