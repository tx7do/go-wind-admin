<script lang="ts" setup>
import type { LoginTrendResponse } from '#/api/generated/admin/service/v1';

import { computed, ref, watch } from 'vue';

import { usePreferences } from '@vben/preferences';
import {
  EchartsUI,
  type EchartsUIType,
  useEcharts,
} from '@vben/plugins/echarts';

import { getAccentColor } from './chart-theme';

const props = defineProps<{
  data?: LoginTrendResponse;
}>();

const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);
const { isDark } = usePreferences();

// 登录趋势：单条折线。x 轴为日期（YYYY-MM-DD），y 轴为当日登录次数。
// 数据由父组件从后端拉取后通过 prop 下发；缺日已在后端补零。
const buildOption = (): any => {
  const points = props.data?.points ?? [];
  return {
    grid: {
      bottom: 0,
      containLabel: true,
      left: '1%',
      right: '1%',
      top: '2 %',
    },
    series: [
      {
        areaStyle: {},
        data: points.map((p) => p.count),
        itemStyle: {
          color: getAccentColor(0),
        },
        smooth: true,
        type: 'line',
      },
    ],
    tooltip: {
      axisPointer: {
        lineStyle: {
          color: getAccentColor(1),
          width: 1,
        },
      },
      trigger: 'axis',
    },
    xAxis: {
      axisTick: {
        show: false,
      },
      boundaryGap: false,
      data: points.map((p) => p.date),
      splitLine: {
        lineStyle: {
          type: 'solid',
          width: 1,
        },
        show: true,
      },
      type: 'category',
    },
    yAxis: [
      {
        axisTick: {
          show: false,
        },
        splitArea: {
          show: true,
        },
        splitNumber: 4,
        type: 'value',
      },
    ],
  };
};

const option = computed(() => buildOption());

watch(
  option,
  (val) => {
    renderEcharts(val);
  },
  { immediate: true, deep: true },
);

watch(isDark, () => renderEcharts(option.value));
</script>

<template>
  <EchartsUI ref="chartRef" />
</template>
