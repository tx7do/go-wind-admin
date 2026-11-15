<template>
  <div>
    <div class="mb-2 flex items-center justify-between">
      <!-- 不可画图（无数值列）时隐藏切换器并直接落表格，避免空图表容器 -->
      <ElRadioGroup v-if="chartable" v-model="viewMode" size="small">
        <ElRadioButton value="chart">{{ t("pages.ai_query.viewChart") }}</ElRadioButton>
        <ElRadioButton value="table">{{ t("pages.ai_query.viewTable") }}</ElRadioButton>
      </ElRadioGroup>
      <span v-else />
      <span class="row-count text-xs">{{ t("pages.ai_query.rowCount", { count: rows.length }) }}</span>
    </div>

    <EchartsUI v-if="chartable && viewMode === 'chart'" ref="chartRef" height="280px" width="100%" />
    <ElTable v-if="!chartable || viewMode === 'table'" :data="tableRows" size="small" border>
      <ElTableColumn
        v-for="(col, ci) in columns"
        :key="col"
        :label="col"
        :prop="String(ci)"
        show-overflow-tooltip
      >
        <template #default="{ row }">{{ row.cells[ci] }}</template>
      </ElTableColumn>
    </ElTable>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import {
  ElRadioButton,
  ElRadioGroup,
  ElTable,
  ElTableColumn,
} from "element-plus";
import { EchartsUI, type EchartsUIType, useEcharts } from "@/plugins/echarts";
import { useI18n } from "@go-wind/admin-element";

interface RowT {
  cells: string[];
}

const props = defineProps<{
  columns: string[];
  rows: RowT[];
}>();

const { t } = useI18n();

const viewMode = ref<"chart" | "table">("chart");
const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

/** 数值列：该列全部行均可解析为数字（跳过首列分类列）。 */
const numericCols = computed(() =>
  props.columns
    .map((_, i) => i)
    .filter(
      (i) =>
        i > 0 &&
        props.rows.length > 0 &&
        props.rows.every((r) => {
          const v = r.cells?.[i];
          return v !== undefined && v !== null && v !== "" && !Number.isNaN(Number(v));
        }),
    ),
);

/** 可画图：≥2 行、≥2 列、且有数值列；行数多时折线比柱状更可读。 */
const chartable = computed(
  () => props.rows.length >= 2 && props.columns.length >= 2 && numericCols.value.length > 0,
);

const chartOption = computed(() => {
  const kind = props.rows.length > 12 ? "line" : "bar";
  return {
    // echarts 6：containLabel 已废弃，等价写法为 outerBoundsMode:'same' + outerBoundsContain:'axisLabel'
    grid: { left: 8, right: 8, top: 24, bottom: 8, outerBoundsMode: "same", outerBoundsContain: "axisLabel" },
    legend: { top: 0 },
    tooltip: { trigger: "axis" },
    xAxis: {
      type: "category",
      data: props.rows.map((r) => r.cells[0] ?? ""),
      axisLabel: { rotate: props.rows.length > 6 ? 30 : 0 },
    },
    yAxis: { type: "value" },
    series: numericCols.value.map((colIdx) => ({
      name: props.columns[colIdx],
      type: kind,
      barMaxWidth: 40,
      data: props.rows.map((r) => Number(r.cells[colIdx] ?? 0)),
    })),
  };
});

function render() {
  if (!chartable.value) return;
  renderEcharts(chartOption.value as any);
}

const tableRows = computed(() =>
  props.rows.map((r, i) => ({ id: i, cells: r.cells || [] })),
);

watch([chartable, chartOption, viewMode], render, { immediate: true, deep: true });
</script>

<style lang="scss" scoped>
// tailwind gray-400 在浅色白底上实测仅 2.54:1，改走 EP 次要文字 token（§5.2 禁止硬编码色值）
.row-count {
  color: var(--el-text-color-secondary);
}
</style>
