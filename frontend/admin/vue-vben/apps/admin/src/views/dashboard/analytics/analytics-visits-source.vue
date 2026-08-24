<script lang="ts" setup>
import type { StatusDistributionResponse } from '#/api/generated/admin/service/v1';

import { computed, ref, watch } from 'vue';

import { $t } from '@vben/locales';
import { usePreferences } from '@vben/preferences';
import {
  EchartsUI,
  type EchartsUIType,
  useEcharts,
} from '@vben/plugins/echarts';

import { getSeriesColors } from './chart-theme';

const props = defineProps<{
  data?: StatusDistributionResponse;
}>();

const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);
const { isDark } = usePreferences();

// 登录成功/失败占比环形图。后端返回 status 枚举名（SUCCESS/FAILED/...），
// 这里经 i18n 转成本地化文案展示。
const statusLabel = (label?: string): string => {
  switch (label) {
    case 'SUCCESS':
      return $t('enum.loginAuditLog.status.SUCCESS');
    case 'FAILED':
      return $t('enum.loginAuditLog.status.FAILED');
    case 'PARTIAL':
      return $t('enum.loginAuditLog.status.PARTIAL');
    default:
      return label ?? '';
  }
};

const buildOption = (): any => {
  const items = props.data?.items ?? [];
  return {
    legend: {
      bottom: '2%',
      left: 'center',
    },
    series: [
      {
        animationDelay() {
          return Math.random() * 100;
        },
        animationEasing: 'exponentialInOut',
        animationType: 'scale',
        avoidLabelOverlap: false,
        color: getSeriesColors(),
        data: items.map((it) => ({
          name: statusLabel(it.label),
          value: it.count,
        })),
        emphasis: {
          label: {
            fontSize: '12',
            fontWeight: 'bold',
            show: true,
          },
        },
        itemStyle: {
          borderRadius: 10,
          borderWidth: 2,
        },
        label: {
          position: 'center',
          show: false,
        },
        labelLine: {
          show: false,
        },
        name: $t('page.analytics.loginStatusDistribution'),
        radius: ['40%', '65%'],
        type: 'pie',
      },
    ],
    tooltip: {
      trigger: 'item',
    },
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
