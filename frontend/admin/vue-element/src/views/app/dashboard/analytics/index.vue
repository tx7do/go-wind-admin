<template>
  <div class="p-5">
    <!-- Overview Cards -->
    <el-row :gutter="20" class="mb-5">
      <el-col v-for="(item, index) in overviewItems" :key="index" :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="overview-card">
          <div class="overview-content">
            <div class="overview-icon">
              <span class="icon-placeholder">{{ item.icon }}</span>
            </div>
            <div class="overview-info">
              <div class="title">{{ item.title }}</div>
              <div class="value">{{ item.value.toLocaleString() }}</div>
              <div class="total">{{ item.totalTitle }}: {{ item.totalValue.toLocaleString() }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Charts Tabs -->
    <el-card shadow="hover" class="mb-5">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="流量趋势" name="trends">
          <AnalyticsTrends />
        </el-tab-pane>
        <el-tab-pane label="月度访问量" name="visits">
          <AnalyticsVisits />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- Chart Cards Grid -->
    <el-row :gutter="20">
      <el-col :xs="24" :sm="24" :md="8" class="mb-5">
        <el-card shadow="hover" :title="t('pages.dashboard.accessCount')">
          <AnalyticsVisitsData />
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="8" class="mb-5">
        <el-card shadow="hover" :title="t('pages.dashboard.accessSource')">
          <AnalyticsVisitsSource />
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="8" class="mb-5">
        <el-card shadow="hover" :title="t('pages.dashboard.salesDistribution')">
          <AnalyticsVisitsSales />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { useI18n } from "vue-i18n";

import AnalyticsTrends from "./analytics-trends.vue";
import AnalyticsVisits from "./analytics-visits.vue";
import AnalyticsVisitsData from "./analytics-visits-data.vue";
import AnalyticsVisitsSales from "./analytics-visits-sales.vue";
import AnalyticsVisitsSource from "./analytics-visits-source.vue";

const { t } = useI18n();

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
    icon: "svg-card",
    title: t("pages.dashboard.currentUserCount"),
    totalTitle: t("pages.dashboard.totalUserCount"),
    totalValue: 120_000,
    value: 2000,
  },
  {
    icon: "svg-cake",
    title: t("pages.dashboard.currentAccessCount"),
    totalTitle: t("pages.dashboard.totalAccessCount"),
    totalValue: 500_000,
    value: 20_000,
  },
  {
    icon: "svg-download",
    title: t("pages.dashboard.currentDownloadCount"),
    totalTitle: t("pages.dashboard.totalDownloadCount"),
    totalValue: 120_000,
    value: 8000,
  },
  {
    icon: "svg-bell",
    title: t("pages.dashboard.currentUsageCount"),
    totalTitle: t("pages.dashboard.totalUsageCount"),
    totalValue: 50_000,
    value: 5000,
  },
]);

// 当前激活的标签
const activeTab = ref("trends");
</script>

<style lang="scss" scoped>
.overview-card {
  :deep(.el-card__body) {
    padding: 20px;
  }
}

.overview-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.overview-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;

  .icon-placeholder {
    font-size: 20px;
    color: white;
  }
}

.overview-info {
  flex: 1;
}

.title {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}

.value {
  font-size: 24px;
  font-weight: bold;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}

.total {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
