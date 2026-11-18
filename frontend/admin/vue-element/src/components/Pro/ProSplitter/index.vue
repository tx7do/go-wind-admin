<template>
  <div ref="hostRef" class="pro-splitter-host">
    <!-- 宽容器：左右分栏（可拖拽） -->
    <ElSplitter v-if="!isStacked">
      <!-- EP 2.14 起数字 size 按 px 解读且首帧容器未量宽时归一化产出 0，
           必须传百分比字符串（文档格式）才能稳定得到 40/60 分栏 -->
      <ElSplitterPanel
        :size="`${leftSize}%`"
        :min="leftMinSize != null ? `${leftMinSize}%` : undefined"
        :max="leftMaxSize != null ? `${leftMaxSize}%` : undefined"
      >
        <slot name="left" />
      </ElSplitterPanel>
      <ElSplitterPanel :size="`${rightSize}%`">
        <slot name="right" />
      </ElSplitterPanel>
    </ElSplitter>

    <!-- 窄容器：上下堆叠。百分比栏位在窄容器下放不下内嵌表格（横向溢出后
         vxe-table 固定操作列会悬浮遮盖中间列），堆叠后每栏独占整行 -->
    <div v-else class="pro-splitter--stacked">
      <div class="pro-splitter--stacked__panel" :style="{ minHeight: `${minPanelHeight}px` }">
        <slot name="left" />
      </div>
      <div class="pro-splitter--stacked__panel" :style="{ minHeight: `${minPanelHeight}px` }">
        <slot name="right" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { ElSplitter, ElSplitterPanel } from "element-plus";

/**
 * 三端统一的堆叠闸（权威值与推导见 docs/design-language.md §4）。
 *
 * 判据取「右栏按声明比例分到的宽度」而不是视口宽：各端 chrome（ele 侧栏 210 + 边距、
 * react/vben 侧栏 224 + 内容内边距）不同且侧栏可折叠/全屏，本端的 contentCompact
 * 还会把 .app-main 限到 1200px（视口 1920 时容器只有 1200），分栏比例又每页不同
 * （55% / 60% / 75%）——只有栏位的绝对宽度是三端同一、且不受这些开关影响的量。
 */
const RIGHT_PANE_MIN_WIDTH = 720;

const props = withDefaults(
  defineProps<{
    /** 左栏初始宽度（百分比） */
    leftSize?: number;
    /** 左栏最小宽度（百分比） */
    leftMinSize?: number;
    /** 左栏最大宽度（百分比） */
    leftMaxSize?: number;
    /** 右栏初始宽度（百分比） */
    rightSize?: number;
    /** 右栏分配宽低于该值（px）时改为上下堆叠 */
    stackBelowPaneWidth?: number;
    /** 堆叠模式下每栏最小高度（px），不足时整页滚动 */
    minPanelHeight?: number;
  }>(),
  {
    leftSize: 40,
    leftMinSize: undefined,
    leftMaxSize: undefined,
    rightSize: 60,
    stackBelowPaneWidth: RIGHT_PANE_MIN_WIDTH,
    minPanelHeight: 480,
  }
);

const hostRef = ref<HTMLDivElement>();
const containerWidth = ref(0);

const isStacked = computed(
  () =>
    containerWidth.value > 0 &&
    containerWidth.value * (props.rightSize / 100) < props.stackBelowPaneWidth
);

let observer: ResizeObserver | undefined;

function measure() {
  if (hostRef.value) containerWidth.value = hostRef.value.clientWidth;
}

onMounted(() => {
  // 挂载即量一次：containerWidth 初值 0 会让首帧按分栏渲染，量完再翻会闪一帧
  measure();
  if (typeof ResizeObserver === "undefined") return;
  observer = new ResizeObserver(measure);
  if (hostRef.value) observer.observe(hostRef.value);
});

onBeforeUnmount(() => {
  observer?.disconnect();
  observer = undefined;
});
</script>

<style lang="scss" scoped>
.pro-splitter-host {
  // .el-splitter 自带 width/height:100%，包一层不改变原有尺寸链
  width: 100%;
  height: 100%;
}

.pro-splitter--stacked {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;

  &__panel {
    flex: 1 1 50%;
    display: flex;
    flex-direction: column;

    > :deep(*) {
      flex: 1 1 auto;
      min-height: 0;
    }
  }
}
</style>
