<script lang="ts" setup>
import type { AnalysisOverviewItem } from '@vben/common-ui';

import {
  AnalysisChartCard,
  AnalysisChartsTabs,
  AnalysisOverview,
} from '@vben/common-ui';
import { SvgBellIcon, SvgCakeIcon, SvgCardIcon, SvgDownloadIcon } from '@vben/icons';
import { $t } from '@vben/locales';

import { computed } from 'vue';

import {
  useDashboardOverview,
  useLoginStatusDistribution,
  useLoginTrend,
  useOperationActionDistribution,
} from '#/api/composables/dashboard';

import AnalyticsTrends from './analytics-trends.vue';
import AnalyticsAiInsights from './analytics-ai-insights.vue';
import AnalyticsVisitsData from './analytics-visits-data.vue';
import AnalyticsVisitsSource from './analytics-visits-source.vue';

// 概览卡：图标在前端按固定顺序映射，数值来自后端 GetOverview。
// totalValue 字段在本场景无独立"总量"语义，与 value 同值以避免通用组件渲染空。
// 数据未就绪（加载/出错）时返回空数组，待数据到达后由 computed 自动填充。
const overviewQuery = useDashboardOverview();

const overviewItems = computed<AnalysisOverviewItem[]>(() => {
  const d = overviewQuery.data.value;
  if (!d) {
    return [];
  }
  const items: AnalysisOverviewItem[] = [
    {
      icon: SvgCardIcon,
      title: $t('page.analytics.userCount'),
      totalTitle: $t('page.analytics.userCount'),
      totalValue: d.userCount ?? 0,
      value: d.userCount ?? 0,
    },
    {
      icon: SvgCakeIcon,
      title: $t('page.analytics.roleCount'),
      totalTitle: $t('page.analytics.roleCount'),
      totalValue: d.roleCount ?? 0,
      value: d.roleCount ?? 0,
    },
    {
      icon: SvgDownloadIcon,
      title: $t('page.analytics.todayLoginCount'),
      totalTitle: $t('page.analytics.todayLoginCount'),
      totalValue: d.todayLoginCount ?? 0,
      value: d.todayLoginCount ?? 0,
    },
    {
      icon: SvgBellIcon,
      title: $t('page.analytics.todayOperationCount'),
      totalTitle: $t('page.analytics.todayOperationCount'),
      totalValue: d.todayOperationCount ?? 0,
      value: d.todayOperationCount ?? 0,
    },
  ];
  return items;
});

// 登录趋势：近 7 天每日登录次数。后端已按日补零、升序返回。
const trendQuery = useLoginTrend(7);

// 操作审计按 action 分布。
const actionDistQuery = useOperationActionDistribution();

// 登录审计按 status 分布。
const statusDistQuery = useLoginStatusDistribution();

const chartTabs = [
  {
    label: $t('page.analytics.loginTrend'),
    value: 'trends',
  },
];
</script>

<template>
  <div class="p-5">
    <AnalysisOverview :items="overviewItems" />

    <!-- AI 安全与异常洞察（行为模式挖掘，平台用户专属） -->
    <AnalyticsAiInsights
      :actions="actionDistQuery.data.value?.items"
      :overview="overviewQuery.data.value"
      :status-items="statusDistQuery.data.value?.items"
      :trend="trendQuery.data.value"
      class="mt-5"
    />
    <AnalysisChartsTabs :tabs="chartTabs" class="mt-5">
      <template #trends>
        <AnalyticsTrends :data="trendQuery.data.value" />
      </template>
    </AnalysisChartsTabs>

    <div class="mt-5 w-full md:flex">
      <AnalysisChartCard
        class="mt-5 md:mr-4 md:mt-0 md:w-1/2"
        :title="$t('page.analytics.operationActionDistribution')"
      >
        <AnalyticsVisitsData :data="actionDistQuery.data.value" />
      </AnalysisChartCard>
      <AnalysisChartCard
        class="mt-5 md:mt-0 md:w-1/2"
        :title="$t('page.analytics.loginStatusDistribution')"
      >
        <AnalyticsVisitsSource :data="statusDistQuery.data.value" />
      </AnalysisChartCard>
    </div>
  </div>
</template>
