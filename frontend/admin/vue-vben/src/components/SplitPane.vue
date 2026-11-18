<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

import { ColPage, Page } from '@vben/common-ui';

defineOptions({ name: 'SplitPane' });

/**
 * 三端统一的堆叠闸（权威值与推导见 docs/design-language.md §4）。
 *
 * 判据取「右栏按声明比例分到的宽度」而不是视口宽：各端 chrome（react/vben 侧栏 224 + 内容内边距、
 * ele 侧栏 210 + 边距）不同且侧栏可折叠/全屏，分栏比例又每页不同（55% / 60% / 75%），
 * 只有栏位的绝对宽度是三端同一的量。
 */
const RIGHT_PANE_MIN_WIDTH = 720;

/** 堆叠模式下每栏最小高度（px），与 ele 端 ProSplitter 同值 */
const MIN_PANEL_HEIGHT = 480;

const props = withDefaults(
  defineProps<{
    autoContentHeight?: boolean;
    leftCollapsedWidth?: number;
    leftCollapsible?: boolean;
    leftMaxWidth?: number;
    leftMinWidth?: number;
    leftWidth?: number;
    minPanelHeight?: number;
    rightCollapsedWidth?: number;
    rightCollapsible?: boolean;
    rightMaxWidth?: number;
    rightMinWidth?: number;
    rightWidth?: number;
    resizable?: boolean;
    splitHandle?: boolean;
    splitLine?: boolean;
    /** 右栏分配宽低于该值时改为上下堆叠 */
    stackBelowPaneWidth?: number;
  }>(),
  {
    autoContentHeight: false,
    leftCollapsedWidth: 0,
    leftCollapsible: false,
    leftMaxWidth: undefined,
    leftMinWidth: undefined,
    leftWidth: 30,
    minPanelHeight: MIN_PANEL_HEIGHT,
    rightCollapsedWidth: 0,
    rightCollapsible: false,
    rightMaxWidth: undefined,
    rightMinWidth: undefined,
    rightWidth: 70,
    resizable: true,
    splitHandle: true,
    splitLine: true,
    stackBelowPaneWidth: RIGHT_PANE_MIN_WIDTH,
  },
);

const hostRef = ref<HTMLDivElement | null>(null);
const containerWidth = ref(0);

const stacked = computed(
  () =>
    containerWidth.value > 0 &&
    containerWidth.value * (props.rightWidth / 100) < props.stackBelowPaneWidth,
);

const colPageProps = computed(() => ({
  leftCollapsedWidth: props.leftCollapsedWidth,
  leftCollapsible: props.leftCollapsible,
  leftMaxWidth: props.leftMaxWidth,
  leftMinWidth: props.leftMinWidth,
  leftWidth: props.leftWidth,
  rightCollapsedWidth: props.rightCollapsedWidth,
  rightCollapsible: props.rightCollapsible,
  rightMaxWidth: props.rightMaxWidth,
  rightMinWidth: props.rightMinWidth,
  rightWidth: props.rightWidth,
  resizable: props.resizable,
  splitHandle: props.splitHandle,
  splitLine: props.splitLine,
}));

let observer: undefined | ResizeObserver;

function measure() {
  if (hostRef.value) {
    containerWidth.value = hostRef.value.clientWidth;
  }
}

onMounted(() => {
  // 挂载即量一次：containerWidth 初值 0 会让首帧按分栏渲染，量完再翻会闪一帧
  measure();
  if (typeof ResizeObserver === 'undefined') return;
  observer = new ResizeObserver(measure);
  if (hostRef.value) observer.observe(hostRef.value);
});

onBeforeUnmount(() => {
  observer?.disconnect();
  observer = undefined;
});
</script>

<template>
  <div ref="hostRef" class="flex h-full w-full flex-col">
    <!-- 宽容器：沿用 ColPage 的可拖拽左右分栏 -->
    <ColPage
      v-if="!stacked"
      v-bind="colPageProps"
      :auto-content-height="autoContentHeight"
    >
      <template #left>
        <slot name="left"></slot>
      </template>
      <slot></slot>
    </ColPage>

    <!-- 窄容器：上下堆叠。百分比栏位放不下内嵌表格（右栏 60% 时列宽合计已超栏宽），
         堆叠后每栏独占整行、表格各自横向滚动 -->
    <Page v-else :auto-content-height="autoContentHeight">
      <div class="flex h-full min-h-0 flex-col gap-3 overflow-auto">
        <div
          class="flex flex-1 flex-col"
          :style="{ minHeight: `${minPanelHeight}px` }"
        >
          <slot name="left"></slot>
        </div>
        <div
          class="flex flex-1 flex-col"
          :style="{ minHeight: `${minPanelHeight}px` }"
        >
          <slot></slot>
        </div>
      </div>
    </Page>
  </div>
</template>
