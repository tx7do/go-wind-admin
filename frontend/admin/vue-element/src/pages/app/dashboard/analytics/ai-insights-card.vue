<template>
  <el-card shadow="hover" class="ai-insights-card mb-5">
    <template #header>
      <div class="card-header-tabs">
        <span class="card-title">
          <SvgIcon icon="svg:color_bell" :size="18" class="mr-1" />
          {{ t("pages.dashboard.aiInsights.title") }}
        </span>
        <el-button
          type="primary"
          size="small"
          :loading="insightsLoading"
          @click="handleScan"
        >
          {{ insights ? t("pages.dashboard.aiInsights.rescan") : t("pages.dashboard.aiInsights.scan") }}
        </el-button>
      </div>
    </template>

    <!-- 未扫描空态 -->
    <div v-if="!insights && !insightsLoading" class="empty-tip">
      {{ t("pages.dashboard.aiInsights.empty") }}
    </div>

    <template v-else>
      <!-- 指标带 -->
      <div class="metric-band">
        <div v-for="m in metricItems" :key="m.label" class="metric-block">
          <div class="metric-label">{{ m.label }}</div>
          <div class="metric-value" :style="{ color: m.color }">{{ m.value }}</div>
        </div>
      </div>

      <!-- 中排：登录趋势 mini 图 + 失败率 -->
      <div class="mid-grid">
        <div class="mid-block">
          <div class="block-label">{{ t("pages.dashboard.aiInsights.trend7d") }}</div>
          <AnalyticsTrends :data="trend" />
        </div>
        <div class="mid-block">
          <div class="block-label">{{ t("pages.dashboard.aiInsights.failRate") }}</div>
          <div class="fail-rate" :class="failRate > 20 ? 'is-high' : 'is-ok'">{{ failRate }}%</div>
          <div class="action-bars">
            <div v-for="a in actionBars" :key="a.label" class="action-bar">
              <span class="action-name">{{ a.label }}</span>
              <div class="action-track">
                <div class="action-fill" :style="{ width: a.pct + '%', background: a.color }" />
              </div>
              <span class="action-count">{{ a.count }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- AI 洞察要点（结构化告警，文案按界面语言由 i18n 模板渲染） -->
      <div v-if="alertItems.length > 0" class="alert-list">
        <div v-for="(a, i) in alertItems" :key="i" class="alert-item">
          <span class="alert-bar" :class="'sev-' + a.severity.toLowerCase()" />
          <div>
            <div class="alert-title">
              <el-tag :type="a.tagType" size="small">{{ a.severity }}</el-tag>
              {{ a.title }}
            </div>
            <div class="alert-detail">{{ a.detail }}</div>
          </div>
        </div>
      </div>

      <!-- LLM 总体评估（按请求语言生成） -->
      <div v-if="insights?.summary" class="assessment">
        <div class="block-label">{{ t("pages.dashboard.aiInsights.assessment") }}</div>
        <div class="assessment-text">{{ insights.summary }}</div>
      </div>
    </template>
  </el-card>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElButton, ElCard, ElMessage, ElTag } from "element-plus";
import { useI18n } from "@go-wind/admin-element";
import { apiClient } from "@/api/client";
import AnalyticsTrends from "./analytics-trends.vue";
import type {
  AiInsightAlert,
  AiInsightsResponse,
  LoginTrendResponse,
} from "@/api/generated/admin/service/v1";
import { i18n } from "@go-wind/admin-element";

// 指标/趋势/分布数据由父页面注入（复用页面已有的 dashboard 查询缓存，避免重复请求）
const props = defineProps<{
  overview?: {
    todayLoginCount?: number;
    todayOperationCount?: number;
    userCount?: number;
    roleCount?: number;
  };
  trend?: LoginTrendResponse;
  actions?: { label?: string; count?: number }[];
  statusItems?: { label?: string; count?: number }[];
}>();

const { t } = useI18n();

const insights = ref<AiInsightsResponse>();
const insightsLoading = ref(false);

const metricItems = computed(() => [
  // 语义色走 --el-* token（docs/design-language.md §2.1）；#3b82f6 是 §3.2 已退役的旧主色
  { label: t("pages.dashboard.todayLoginCount"), value: props.overview?.todayLoginCount ?? 0, color: "var(--el-color-primary)" },
  { label: t("pages.dashboard.todayOperationCount"), value: props.overview?.todayOperationCount ?? 0, color: "#22d3ee" },
  { label: t("pages.dashboard.userCount"), value: props.overview?.userCount ?? 0, color: "#a78bfa" },
  { label: t("pages.dashboard.roleCount"), value: props.overview?.roleCount ?? 0, color: "#34d399" },
]);

const failRate = computed(() => {
  const items = props.statusItems || [];
  const succ = items.find((s) => s.label === "SUCCESS")?.count || 0;
  const fail = items.find((s) => s.label === "FAILED")?.count || 0;
  return succ + fail > 0 ? Math.round((fail * 100) / (succ + fail)) : 0;
});

const ACTION_COLORS: Record<string, string> = {
  // 语义可对应的动作用 §2.1 的语义 token；EXPORT/UPDATE 的紫/青是纯分类装饰色，
  // 本仓 token 表里没有对应项，要收口得先扩 §2.1 再三端同补，故此处保留原值。
  ASSIGN: "var(--el-color-success)",
  CREATE: "var(--el-color-primary)",
  DELETE: "var(--el-color-danger)",
  EXPORT: "#a78bfa",
  IMPORT: "var(--el-color-warning)",
  OTHER: "var(--el-text-color-secondary)",
  UPDATE: "#22d3ee",
};

const actionBars = computed(() => {
  const items = props.actions || [];
  const total = items.reduce((sum, a) => sum + (a.count || 0), 0) || 1;
  return items
    .slice()
    .sort((a, b) => (b.count || 0) - (a.count || 0))
    .map((a) => ({
      label: a.label || "-",
      count: a.count || 0,
      pct: Math.round(((a.count || 0) * 100) / total),
      color: ACTION_COLORS[a.label || ""] || "var(--el-text-color-secondary)",
    }));
});

const alertItems = computed(() => {
  const alerts = (insights.value?.alerts || []) as AiInsightAlert[];
  return alerts.map((a) => ({
    severity: a.severity || "LOW",
    tagType: (a.severity === "HIGH" ? "danger" : a.severity === "MEDIUM" ? "warning" : "info") as "danger" | "warning" | "info",
    title: renderTemplate(a),
    detail: detailText(a),
  }));
});

/** 告警文案：后端只回结构化事实（type+facts），标题/明细由前端 i18n 模板插值渲染。 */
function renderTemplate(alert: AiInsightAlert): string {
  const templates: Record<string, string> = {
    BRUTE_FORCE: t("pages.dashboard.aiInsights.bruteForceTitle", alert.facts || {}),
    NIGHT_OPS: t("pages.dashboard.aiInsights.nightOpsTitle", alert.facts || {}),
    FAILED_OPS: t("pages.dashboard.aiInsights.failedOpsTitle", alert.facts || {}),
    SENSITIVE_OPS: t("pages.dashboard.aiInsights.sensitiveOpsTitle", alert.facts || {}),
  };
  return templates[alert.type || ""] || alert.type || "";
}

function detailText(alert: AiInsightAlert): string {
  const templates: Record<string, string> = {
    BRUTE_FORCE: t("pages.dashboard.aiInsights.bruteForceDetail", alert.facts || {}),
    NIGHT_OPS: t("pages.dashboard.aiInsights.nightOpsDetail", alert.facts || {}),
    FAILED_OPS: t("pages.dashboard.aiInsights.failedOpsDetail", alert.facts || {}),
  };
  return templates[alert.type || ""] || "";
}

async function handleScan() {
  insightsLoading.value = true;
  try {
    const locale = i18n.global.locale as unknown as { value?: string };
    const lang = locale?.value || "zh-CN";
    insights.value = await apiClient.dashboardService.GetAiInsights({ lang } as any);
  } catch (error: any) {
    console.error("dashboard ai insights failed:", error);
    ElMessage.error(error?.message || t("pages.dashboard.aiInsights.failed"));
  } finally {
    insightsLoading.value = false;
  }
}
</script>

<style lang="scss" scoped>
.card-header-tabs {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.metric-band {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 16px;
}

.metric-block {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 8px 12px;
}

.metric-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.metric-value {
  font-size: 22px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.mid-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 16px;
}

.mid-block {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 8px 12px;
}

.block-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.fail-rate {
  font-size: 26px;
  font-weight: 600;

  &.is-high {
    color: var(--gowind-danger-text);
  }

  &.is-ok {
    color: var(--gowind-success-text);
  }
}

.action-bars {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.action-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.action-name {
  width: 64px;
  color: var(--el-text-color-secondary);
}

.action-track {
  flex: 1;
  height: 6px;
  border-radius: 3px;
  overflow: hidden;
  background: rgba(128, 128, 128, 0.15);
}

.action-fill {
  height: 100%;
  border-radius: 3px;
}

.action-count {
  width: 32px;
  text-align: right;
  color: var(--el-text-color-secondary);
}

.empty-tip {
  color: var(--el-text-color-secondary);
  padding: 8px 0;
}

.alert-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.alert-item {
  display: flex;
  gap: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 10px 12px;
}

.alert-bar {
  width: 4px;
  border-radius: 2px;
  flex-shrink: 0;

  &.sev-high {
    background: var(--el-color-danger);
  }

  &.sev-medium {
    background: var(--el-color-warning);
  }

  &.sev-low {
    background: var(--el-color-primary);
  }
}

.alert-title {
  font-weight: 500;
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.alert-detail {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.assessment {
  margin-top: 12px;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.assessment-text {
  margin-top: 4px;
  line-height: 1.7;
}
</style>
