<template>
  <EchartsUI ref="chartRef" height="100%" />
</template>

<script lang="ts" setup>
import type { LoginTrendResponse } from "@/api/generated/admin/service/v1";

import { EchartsUI, EchartsUIType, useEcharts } from "@/plugins/echarts";
import { usePreferences } from "@/core/preferences";
import { getChartPrimary, hexToRgba } from "@/utils/chart-palette";

const props = defineProps<{
  data?: LoginTrendResponse;
}>();

const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);
const { isDark } = usePreferences();

// 登录趋势单折线。数据由父组件从后端 GetLoginTrend 拉取后下发；
// 后端已按日补零、升序返回 points。
const chartOptions = computed(() => {
  const points = props.data?.points ?? [];
  // 主色跟随运行时主题（对齐 react 端 LineChart 的 token.colorPrimary）
  const primary = getChartPrimary();
  return {
    color: [primary],
    grid: {
      bottom: 24,
      left: 40,
      right: 16,
      top: 16,
    },
    series: [
      {
        areaStyle: {
          color: {
            colorStops: [
              { offset: 0, color: hexToRgba(primary, 0.25) },
              { offset: 1, color: hexToRgba(primary, 0.02) },
            ],
            x: 0,
            x2: 0,
            y: 0,
            y2: 1,
          },
        },
        data: points.map((p) => p.count),
        lineStyle: {
          width: 2,
        },
        smooth: 0.4,
        symbol: "none",
        type: "line",
      },
    ],
    tooltip: {
      backgroundColor: isDark.value ? "rgba(40,40,40,0.96)" : "rgba(255,255,255,0.96)",
      borderColor: isDark.value ? "#4c4d4f" : "#eee",
      borderRadius: 8,
      padding: [12, 16],
      textStyle: {
        color: isDark.value ? "#ffffff" : "#303133",
        fontSize: 13,
      },
      axisPointer: {
        lineStyle: {
          color: primary,
          opacity: 0.3,
          width: 1,
        },
      },
      trigger: "axis",
    },
    xAxis: {
      axisLine: {
        show: false,
      },
      axisLabel: {
        color: isDark.value ? "#CFD3DC" : "#606266",
        fontSize: 11,
      },
      axisTick: {
        show: false,
      },
      boundaryGap: false,
      data: points.map((p) => p.date),
      splitLine: {
        show: false,
      },
      type: "category",
    },
    yAxis: [
      {
        axisLine: {
          show: false,
        },
        axisLabel: {
          color: isDark.value ? "#8c8c8c" : "#909399",
          fontSize: 11,
          formatter: (val: number) => {
            if (val >= 1000) return `${Math.round(val / 1000)}k`;
            return `${val}`;
          },
        },
        axisTick: {
          show: false,
        },
        splitArea: {
          show: false,
        },
        splitLine: {
          lineStyle: {
            color: isDark.value ? "rgba(255,255,255,0.06)" : "rgba(0,0,0,0.05)",
            type: "solid",
          },
          show: true,
        },
        splitNumber: 4,
        type: "value",
      },
    ],
  };
});

watch(
  () => chartOptions.value,
  (options) => {
    renderEcharts(options as any);
  },
  { immediate: true, deep: true }
);
</script>
