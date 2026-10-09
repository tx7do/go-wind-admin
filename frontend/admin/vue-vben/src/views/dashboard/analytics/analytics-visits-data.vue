<script lang="ts" setup>
import type { ActionDistributionResponse } from '#/api/generated/admin/service/v1';

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
  data?: ActionDistributionResponse;
}>();

const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);
const { isDark } = usePreferences();

// 操作类型分布环形图。后端返回 action 枚举名（CREATE/UPDATE/...），
// 这里经 i18n 转成本地化文案展示，复用操作审计日志模块的 action.* 翻译。
const actionLabel = (label?: string): string => {
  switch (label) {
    case 'CREATE':
      return $t('enum.operationAuditLog.action.CREATE');
    case 'UPDATE':
      return $t('enum.operationAuditLog.action.UPDATE');
    case 'DELETE':
      return $t('enum.operationAuditLog.action.DELETE');
    case 'READ':
      return $t('enum.operationAuditLog.action.READ');
    case 'ASSIGN':
      return $t('enum.operationAuditLog.action.ASSIGN');
    case 'UNASSIGN':
      return $t('enum.operationAuditLog.action.UNASSIGN');
    case 'EXPORT':
      return $t('enum.operationAuditLog.action.EXPORT');
    case 'IMPORT':
      return $t('enum.operationAuditLog.action.IMPORT');
    case 'OTHER':
      return $t('enum.operationAuditLog.action.OTHER');
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
          name: actionLabel(it.label),
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
        name: $t('page.analytics.operationActionDistribution'),
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
