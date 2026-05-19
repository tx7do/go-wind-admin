<template>
  <EchartsUI ref="chartRef" height="100%" />
</template>

<script lang="ts" setup>
import type { EChartsOption } from "echarts";

import { EchartsUI, EchartsUIType, useEcharts } from "@/plugins/echarts";

const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

const chartOptions = computed<EChartsOption>(() => ({
  series: [
    {
      animationDelay() {
        return Math.random() * 400;
      },
      animationEasing: "exponentialInOut",
      animationType: "scale",
      center: ["50%", "50%"],
      color: ["#5ab1ef", "#b6a2de", "#67e0e3", "#2ec7c9"],
      data: [
        { name: "外包", value: 500 },
        { name: "定制", value: 310 },
        { name: "技术支持", value: 274 },
        { name: "远程", value: 400 },
      ].sort((a, b) => a.value - b.value),
      name: "商业占比",
      radius: "80%",
      roseType: "radius",
      type: "pie",
    },
  ],
  tooltip: {
    trigger: "item",
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
