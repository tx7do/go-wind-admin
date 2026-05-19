<template>
  <EchartsUI ref="chartRef" height="100%" />
</template>

<script lang="ts" setup>
import type { EChartsOption } from "echarts";

import { EchartsUI, EchartsUIType, useEcharts } from "@/plugins/echarts";
import { $t } from "@/i18n";

const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

const chartOptions = computed<EChartsOption>(() => ({
  grid: {
    bottom: 0,
    left: "1%",
    right: "1%",
    top: "2%",
  },
  series: [
    {
      barMaxWidth: 80,
      data: [3000, 2000, 3333, 5000, 3200, 4200, 3200, 2100, 3000, 5100, 6000, 3200, 4800],
      type: "bar",
    },
  ],
  tooltip: {
    axisPointer: {
      lineStyle: {
        width: 1,
      },
    },
    trigger: "axis",
  },
  xAxis: {
    data: Array.from({ length: 12 }).map(
      (_item, index) => `${index + 1}${$t("pages.dashboard.month")}`
    ),
    type: "category",
  },
  yAxis: {
    max: 8000,
    splitNumber: 4,
    type: "value",
  },
}));

watch(
  () => chartOptions.value,
  (options) => {
    renderEcharts(options);
  },
  { immediate: true, deep: true }
);
</script>
