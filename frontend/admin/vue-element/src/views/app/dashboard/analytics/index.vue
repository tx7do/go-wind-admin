<template>
  <div class="p-5">
    <!-- Overview Cards -->
    <el-row :gutter="20" class="mb-6">
      <el-col v-for="(item, index) in overviewItems" :key="index" :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="overview-card">
          <div class="overview-header">
            <div class="title">{{ item.title }}</div>
            <div :class="item.icon" class="icon-svg" />
          </div>
          <div class="value">{{ item.value.toLocaleString() }}</div>
          <div class="total-row">
            <span class="total-label">{{ item.totalTitle }}</span>
            <span class="total-value">{{ item.totalValue.toLocaleString() }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Trends Chart -->
    <el-card shadow="hover" class="mb-6">
      <template #header>
        <div class="card-header-tabs">
          <el-radio-group v-model="activeTab" size="small">
            <el-radio-button value="trends">
              {{ $t("pages.dashboard.visitsTrend") }}
            </el-radio-button>
            <el-radio-button value="visits">
              {{ $t("pages.dashboard.monthVisits") }}
            </el-radio-button>
          </el-radio-group>
        </div>
      </template>
      <div class="chart-container chart-container-trend">
        <AnalyticsTrends v-if="activeTab === 'trends'" />
        <AnalyticsVisits v-else />
      </div>
    </el-card>

    <!-- Chart Cards Grid -->
    <el-row :gutter="20">
      <el-col :xs="24" :sm="24" :md="8">
        <el-card shadow="hover">
          <template #header>
            <span class="card-title">{{ $t("pages.dashboard.visitCount") }}</span>
          </template>
          <div class="chart-container chart-container-small">
            <AnalyticsVisitsData />
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="8">
        <el-card shadow="hover">
          <template #header>
            <span class="card-title">{{ $t("pages.dashboard.visitSource") }}</span>
          </template>
          <div class="chart-container chart-container-small">
            <AnalyticsVisitsSource />
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="8">
        <el-card shadow="hover">
          <template #header>
            <span class="card-title">{{ $t("pages.dashboard.salesDistribution") }}</span>
          </template>
          <div class="chart-container chart-container-small">
            <AnalyticsVisitsSales />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";

import { $t } from "@/i18n";

import AnalyticsTrends from "./analytics-trends.vue";
import AnalyticsVisits from "./analytics-visits.vue";
import AnalyticsVisitsData from "./analytics-visits-data.vue";
import AnalyticsVisitsSales from "./analytics-visits-sales.vue";
import AnalyticsVisitsSource from "./analytics-visits-source.vue";

// 定义 OverviewItem 接口
interface OverviewItem {
  icon: string;
  title: string;
  totalTitle: string;
  totalValue: number;
  value: number;
}

const overviewItems = ref<OverviewItem[]>([
  {
    icon: "i-svg:color_card",
    title: $t("pages.dashboard.currentUserCount"),
    totalTitle: $t("pages.dashboard.totalUserCount"),
    totalValue: 120_000,
    value: 2000,
  },
  {
    icon: "i-svg:color_cake",
    title: $t("pages.dashboard.currentAccessCount"),
    totalTitle: $t("pages.dashboard.totalAccessCount"),
    totalValue: 500_000,
    value: 20_000,
  },
  {
    icon: "i-svg:color_download",
    title: $t("pages.dashboard.currentDownloadCount"),
    totalTitle: $t("pages.dashboard.totalDownloadCount"),
    totalValue: 120_000,
    value: 8000,
  },
  {
    icon: "i-svg:color_bell",
    title: $t("pages.dashboard.currentUsageCount"),
    totalTitle: $t("pages.dashboard.totalUsageCount"),
    totalValue: 50_000,
    value: 5000,
  },
]);

// 当前激活的标签
const activeTab = ref<"trends" | "visits">("trends");
</script>

<style lang="scss" scoped>
.overview-card {
  height: 160px;
  border-radius: 4px;
  transition: all 0.3s;

  :deep(.el-card__body) {
    padding: 20px 24px;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 8px;
  }

  .overview-header {
    display: flex;
    align-items: center;
    justify-content: space-between;

    .title {
      font-size: 15px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .icon-svg {
      width: 28px;
      height: 28px;
      font-size: 28px;
    }
  }

  .value {
    font-size: 32px;
    font-weight: 700;
    color: var(--el-text-color-primary);
    line-height: 1;
  }

  .total-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-top: auto;

    .total-value {
      color: var(--el-text-color-regular);
    }
  }
}

.card-header-tabs {
  display: flex;
  align-items: center;
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.chart-container {
  width: 100%;
  height: 100%;
}

.chart-container-trend {
  height: 350px;
}

.chart-container-small {
  height: 280px;
}
</style>
