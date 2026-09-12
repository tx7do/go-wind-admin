<template>
  <div class="analytics-page">
    <!-- Overview Cards -->
    <el-row :gutter="16" class="mb-5">
      <el-col v-for="(item, index) in overviewItems" :key="index" :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="overview-card">
          <div class="overview-header">
            <div class="overview-header__text">
              <div class="title">{{ item.title }}</div>
              <div class="value-row">
                <span class="value">{{ item.value.toLocaleString() }}</span>
              </div>
            </div>
            <div class="overview-header__icon">
              <SvgIcon :icon="item.icon" :size="32" />
            </div>
          </div>
          <div class="overview-footer">
            <span class="footer-label">{{ $t("pages.dashboard.total") }}</span>
            <span class="footer-total">
              {{ $t("pages.dashboard.total") }}
              <strong>{{ item.totalValue.toLocaleString() }}</strong>
            </span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Login Trend Chart -->
    <el-card shadow="hover" class="mb-5">
      <template #header>
        <div class="card-header-tabs">
          <span class="card-title">{{ $t("pages.dashboard.loginTrend") }}</span>
        </div>
      </template>
      <div class="chart-container chart-container-trend">
        <AnalyticsTrends :data="trendQuery.data.value" />
      </div>
    </el-card>

    <!-- Distribution Cards Grid -->
    <el-row :gutter="16">
      <el-col :xs="24" :sm="24" :md="12">
        <el-card shadow="hover">
          <template #header>
            <span class="card-title">{{ $t("pages.dashboard.operationActionDistribution") }}</span>
          </template>
          <div class="chart-container chart-container-small">
            <AnalyticsVisitsData :data="actionDistQuery.data.value" />
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="24" :md="12">
        <el-card shadow="hover">
          <template #header>
            <span class="card-title">{{ $t("pages.dashboard.loginStatusDistribution") }}</span>
          </template>
          <div class="chart-container chart-container-small">
            <AnalyticsVisitsSource :data="statusDistQuery.data.value" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script lang="ts" setup>
import SvgIcon from "@/components/SvgIcon/index.vue";
import { $t } from "@/core/i18n";
import {
  useDashboardOverview,
  useLoginStatusDistribution,
  useLoginTrend,
  useOperationActionDistribution,
} from "@/api/composables/dashboard";
import AnalyticsTrends from "./analytics-trends.vue";
import AnalyticsVisitsData from "./analytics-visits-data.vue";
import AnalyticsVisitsSource from "./analytics-visits-source.vue";

// 概览卡：图标在前端按固定顺序映射，数值来自后端 GetOverview。
// totalValue 字段在本场景无独立"总量"语义，与 value 同值以避免渲染空。
// 数据未就绪（加载/出错）时返回空数组，待数据到达后由 computed 自动填充。
const overviewQuery = useDashboardOverview();

const overviewItems = computed(() => {
  const d = overviewQuery.data.value;
  if (!d) {
    return [];
  }
  const iconPool = ["svg:color_card", "svg:color_cake", "svg:color_download", "svg:color_bell"];
  const values = [
    d.userCount ?? 0,
    d.roleCount ?? 0,
    d.todayLoginCount ?? 0,
    d.todayOperationCount ?? 0,
  ];
  const titleKeys = [
    "pages.dashboard.userCount",
    "pages.dashboard.roleCount",
    "pages.dashboard.todayLoginCount",
    "pages.dashboard.todayOperationCount",
  ];
  return values.map((v, i) => ({
    icon: iconPool[i]!,
    title: $t(titleKeys[i]!),
    totalValue: v,
    value: v,
  }));
});

const trendQuery = useLoginTrend(7);
const actionDistQuery = useOperationActionDistribution();
const statusDistQuery = useLoginStatusDistribution();
</script>

<style lang="scss" scoped>
.analytics-page {
  padding: 20px;
}

.overview-card {
  border-radius: 12px;
  transition: all 0.3s ease;
  border: 1px solid var(--el-border-color-lighter);

  &:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.03);
  }

  // 暗黑模式 hover 阴影
  html.dark & {
    &:hover {
      border-color: var(--el-color-primary-light-3);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
    }

    .overview-header__icon {
      background: rgba(0, 107, 230, 0.15); // 主色 15% α（design-language.md §2.3）
    }
  }

  :deep(.el-card__body) {
    padding: 20px;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 12px;
  }

  .overview-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;

    &__text {
      flex: 1;
      min-width: 0;
    }

    &__icon {
      flex-shrink: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      width: 48px;
      height: 48px;
      border-radius: 12px;
      background: var(--el-color-primary-light-9);
    }
  }

  .title {
    font-size: 14px;
    font-weight: 500;
    color: var(--el-text-color-regular);
    margin-bottom: 8px;
  }

  .value-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .value {
    font-size: 28px;
    font-weight: 700;
    color: var(--el-text-color-primary);
    line-height: 1;
    letter-spacing: -0.5px;
  }

  .overview-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 10px;
    border-top: 1px solid var(--el-border-color-lighter);
    font-size: 12px;

    .footer-label {
      color: var(--el-text-color-regular);
    }

    .footer-total {
      color: var(--el-text-color-regular);

      strong {
        color: var(--el-text-color-primary);
        font-weight: 600;
      }
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
  display: block;
  padding-top: 2px;
}

.chart-container {
  width: 100%;
  height: 100%;
}

.chart-container-trend {
  height: 380px;
}

.chart-container-small {
  height: 300px;
}
</style>
