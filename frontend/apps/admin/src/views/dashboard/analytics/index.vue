<script lang="ts" setup>
import type { AnalysisOverviewItem } from '@vben/common-ui';
import type { TabOption } from '@vben/types';

import {
  AnalysisChartCard,
  AnalysisChartsTabs,
  AnalysisOverview,
} from '@vben/common-ui';
import {
  SvgBellIcon,
  SvgCakeIcon,
  SvgCardIcon,
  SvgDownloadIcon,
} from '@vben/icons';
import { $t } from '@vben/locales';

import AnalyticsTrends from './analytics-trends.vue';
import AnalyticsVisits from './analytics-visits.vue';
import AnalyticsVisitsData from './analytics-visits-data.vue';
import AnalyticsVisitsSales from './analytics-visits-sales.vue';
import AnalyticsVisitsSource from './analytics-visits-source.vue';

const overviewItems: AnalysisOverviewItem[] = [
  {
    icon: SvgCardIcon,
    title: $t('page.analytics.currentUserCount'),
    totalTitle: $t('page.analytics.totalUserCount'),
    totalValue: 120_000,
    value: 2000,
  },
  {
    icon: SvgCakeIcon,
    title: $t('page.analytics.currentAccessCount'),
    totalTitle: $t('page.analytics.totalAccessCount'),
    totalValue: 500_000,
    value: 20_000,
  },
  {
    icon: SvgDownloadIcon,
    title: $t('page.analytics.currentDownloadCount'),
    totalTitle: $t('page.analytics.totalDownloadCount'),
    totalValue: 120_000,
    value: 8000,
  },
  {
    icon: SvgBellIcon,
    title: $t('page.analytics.currentUsageCount'),
    totalTitle: $t('page.analytics.totalUsageCount'),
    totalValue: 50_000,
    value: 5000,
  },
];

const chartTabs: TabOption[] = [
  {
    label: $t('page.analytics.trafficTrend'),
    value: 'trends',
  },
  {
    label: $t('page.analytics.monthAccessCount'),
    value: 'visits',
  },
];
</script>

<template>
  <div class="p-5">
    <AnalysisOverview :items="overviewItems" />
    <AnalysisChartsTabs :tabs="chartTabs" class="mt-5">
      <template #trends>
        <AnalyticsTrends />
      </template>
      <template #visits>
        <AnalyticsVisits />
      </template>
    </AnalysisChartsTabs>

    <div class="mt-5 w-full md:flex">
      <AnalysisChartCard
        class="mt-5 md:mr-4 md:mt-0 md:w-1/3"
        :title="$t('page.analytics.accessCount')"
      >
        <AnalyticsVisitsData />
      </AnalysisChartCard>
      <AnalysisChartCard
        class="mt-5 md:mr-4 md:mt-0 md:w-1/3"
        :title="$t('page.analytics.accessSource')"
      >
        <AnalyticsVisitsSource />
      </AnalysisChartCard>
      <AnalysisChartCard
        class="mt-5 md:mt-0 md:w-1/3"
        :title="$t('page.analytics.accessSource')"
      >
        <AnalyticsVisitsSales />
      </AnalysisChartCard>
    </div>
  </div>
</template>
