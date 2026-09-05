<template>
  <!-- 宽屏：左右分栏（可拖拽） -->
  <ElSplitter v-if="!isStacked">
    <ElSplitterPanel :size="leftSize" :min-size="leftMinSize" :max-size="leftMaxSize">
      <slot name="left" />
    </ElSplitterPanel>
    <ElSplitterPanel :size="rightSize">
      <slot name="right" />
    </ElSplitterPanel>
  </ElSplitter>

  <!-- 窄视口：上下堆叠。百分比栏位在窄视口下放不下内嵌表格（横向溢出后
       vxe-table 固定操作列会悬浮遮盖中间列），堆叠后每栏独占整行 -->
  <div v-else class="pro-splitter--stacked">
    <div class="pro-splitter--stacked__panel" :style="{ minHeight: `${minPanelHeight}px` }">
      <slot name="left" />
    </div>
    <div class="pro-splitter--stacked__panel" :style="{ minHeight: `${minPanelHeight}px` }">
      <slot name="right" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { ElSplitter, ElSplitterPanel } from "element-plus";
import { useWindowSize } from "@vueuse/core";

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
    /** 视口宽度低于该值时改为上下堆叠 */
    stackBelow?: number;
    /** 堆叠模式下每栏最小高度（px），不足时整页滚动 */
    minPanelHeight?: number;
  }>(),
  {
    leftSize: 40,
    leftMinSize: undefined,
    leftMaxSize: undefined,
    rightSize: 60,
    stackBelow: 1600,
    minPanelHeight: 480,
  }
);

const { width } = useWindowSize();
const isStacked = computed(() => width.value < props.stackBelow);
</script>

<style lang="scss" scoped>
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
