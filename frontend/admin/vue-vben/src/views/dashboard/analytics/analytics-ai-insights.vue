<script lang="ts" setup>
import type {
  AiInsightAlert,
  AiInsightsResponse,
  LoginTrendResponse,
} from '#/api/generated/admin/service/v1';

import { computed, ref } from 'vue';

import { $t } from '@vben/locales';

import { message } from 'ant-design-vue';

import { apiClient } from '#/api';

import AnalyticsTrends from './analytics-trends.vue';

// 指标/趋势/分布数据由父页面注入（复用页面已有的 dashboard 查询缓存，避免重复请求）
const props = defineProps<{
  actions?: { label?: string; count?: number }[];
  overview?: {
    todayLoginCount?: number;
    todayOperationCount?: number;
    userCount?: number;
    roleCount?: number;
  };
  statusItems?: { label?: string; count?: number }[];
  trend?: LoginTrendResponse;
}>();

const insights = ref<AiInsightsResponse>();
const insightsLoading = ref(false);

const metricItems = computed(() => [
  {
    // 语义色一律走 design token（§2.1）；#3b82f6 是 §3.2 已退役的旧主色
    color: 'hsl(var(--primary))',
    label: $t('page.aiInsights.todayLoginCount'),
    value: props.overview?.todayLoginCount ?? 0,
  },
  {
    color: '#22d3ee',
    label: $t('page.aiInsights.todayOperationCount'),
    value: props.overview?.todayOperationCount ?? 0,
  },
  {
    color: '#a78bfa',
    label: $t('page.aiInsights.userCount'),
    value: props.overview?.userCount ?? 0,
  },
  {
    color: '#34d399',
    label: $t('page.aiInsights.roleCount'),
    value: props.overview?.roleCount ?? 0,
  },
]);

const failRate = computed(() => {
  const items = props.statusItems || [];
  const succ = items.find((s) => s.label === 'SUCCESS')?.count || 0;
  const fail = items.find((s) => s.label === 'FAILED')?.count || 0;
  return succ + fail > 0 ? Math.round((fail * 100) / (succ + fail)) : 0;
});

const ACTION_COLORS: Record<string, string> = {
  // 语义可对应的动作用 §2.1 的语义 token；EXPORT/UPDATE 的紫/青是纯分类装饰色，
  // 本仓 token 表里没有对应项，要收口得先扩 §2.1 再三端同补，故此处保留原值。
  ASSIGN: 'hsl(var(--success))',
  CREATE: 'hsl(var(--primary))',
  DELETE: 'hsl(var(--destructive))',
  EXPORT: '#a78bfa',
  IMPORT: 'hsl(var(--warning))',
  OTHER: 'hsl(var(--muted-foreground))',
  UPDATE: '#22d3ee',
};

const actionBars = computed(() => {
  const items = props.actions || [];
  const total = items.reduce((sum, a) => sum + (a.count || 0), 0) || 1;
  return items
    .slice()
    .sort((a, b) => (b.count || 0) - (a.count || 0))
    .map((a) => ({
      color: ACTION_COLORS[a.label || ''] || 'hsl(var(--muted-foreground))',
      count: a.count || 0,
      label: a.label || '-',
      pct: Math.round(((a.count || 0) * 100) / total),
    }));
});

const alertItems = computed(() => {
  const alerts = (insights.value?.alerts || []) as AiInsightAlert[];
  return alerts.map((a) => ({
    severity: a.severity || 'LOW',
    tagColor:
      a.severity === 'HIGH' ? 'error' : a.severity === 'MEDIUM' ? 'warning' : 'processing',
    title: renderTemplate(a),
  }));
});

/** 告警文案：后端只回结构化事实（type+facts），标题由前端 i18n 模板插值渲染。 */
function renderTemplate(alert: AiInsightAlert): string {
  const templates: Record<string, string> = {
    BRUTE_FORCE: $t('page.aiInsights.bruteForceTitle'),
    FAILED_OPS: $t('page.aiInsights.failedOpsTitle'),
    NIGHT_OPS: $t('page.aiInsights.nightOpsTitle'),
    SENSITIVE_OPS: $t('page.aiInsights.sensitiveOpsTitle'),
  };
  const facts = alert.facts || {};
  let text = templates[alert.type || ''] || alert.type || '';
  for (const [key, val] of Object.entries(facts)) {
    text = text.split(`{{${key}}}`).join(val);
  }
  return text;
}

async function handleScan() {
  insightsLoading.value = true;
  try {
    const lang = $t('page.aiInsights.lang') || 'zh-CN';
    insights.value = await apiClient.dashboardService.GetAiInsights({
      lang,
    } as any);
  } catch (error) {
    console.error('dashboard ai insights failed:', error);
    message.error(error instanceof Error ? error.message : 'scan failed');
  } finally {
    insightsLoading.value = false;
  }
}
</script>

<template>
  <div class="rounded-xl border border-solid border-border bg-card p-5">
    <!-- 头部 -->
    <div class="mb-4 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div
          class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/12 text-primary"
        >
          🤖
        </div>
        <div>
          <div class="text-sm font-semibold">{{ $t('page.aiInsights.title') }}</div>
          <div class="text-xs text-muted-foreground">{{ $t('page.aiInsights.subtitle') }}</div>
        </div>
      </div>
      <a-button
        :loading="insightsLoading"
        type="primary"
        @click="handleScan"
      >
        ⚡ {{ insights ? $t('page.aiInsights.rescan') : $t('page.aiInsights.scan') }}
      </a-button>
    </div>

    <!-- 未扫描空态 -->
    <div v-if="!insights && !insightsLoading" class="py-2 text-sm text-muted-foreground">
      {{ $t('page.aiInsights.empty') }}
    </div>

    <template v-else>
      <!-- 指标带 -->
      <div class="mb-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div
          v-for="m in metricItems"
          :key="m.label"
          class="rounded-lg border border-solid border-border px-3 py-2"
        >
          <div class="text-xs text-muted-foreground">{{ m.label }}</div>
          <div class="text-xl font-semibold tabular-nums" :style="{ color: m.color }">
            {{ m.value }}
          </div>
        </div>
      </div>

      <!-- 中排：登录趋势 mini 图 + 失败率/动作分布 -->
      <div class="mb-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div class="rounded-lg border border-solid border-border p-3">
          <div class="mb-1 text-xs text-muted-foreground">{{ $t('page.aiInsights.trend7d') }}</div>
          <AnalyticsTrends :data="trend" />
        </div>
        <div class="rounded-lg border border-solid border-border p-3">
          <div class="mb-2 flex items-baseline justify-between">
            <span class="text-xs text-muted-foreground">{{ $t('page.aiInsights.failRate') }}</span>
            <span
              class="text-lg font-semibold tabular-nums"
              :style="{
                color: failRate > 20 ? 'hsl(var(--destructive))' : 'hsl(var(--success))',
              }"
            >
              {{ failRate }}%
            </span>
          </div>
          <div class="space-y-2">
            <div v-for="a in actionBars" :key="a.label" class="flex items-center gap-2">
              <span class="w-16 shrink-0 text-xs text-muted-foreground">{{ a.label }}</span>
              <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-accent">
                <div
                  :style="{ width: a.pct + '%', background: a.color }"
                  class="h-full rounded-full"
                />
              </div>
              <span class="w-10 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
                {{ a.count }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- AI 洞察要点（结构化告警，文案按界面语言由 i18n 模板插值） -->
      <div>
        <div class="mb-2 text-xs text-muted-foreground">{{ $t('page.aiInsights.points') }}</div>
        <div class="space-y-1.5">
          <div
            v-for="(a, i) in alertItems"
            :key="i"
            class="flex items-start gap-2 rounded-lg border border-solid border-border p-3 text-sm"
          >
            <span
              :class="a.severity === 'HIGH' ? 'bg-destructive' : a.severity === 'MEDIUM' ? 'bg-warning' : 'bg-primary'"
              class="mt-1.5 inline-block h-1.5 w-1.5 shrink-0 rounded-full"
            />
            <span>{{ a.title }}</span>
          </div>
        </div>
      </div>

      <!-- LLM 总体评估（按界面语言生成） -->
      <div
        v-if="insights?.summary"
        class="mt-3 border-t border-solid border-border pt-3"
      >
        <div class="mb-1 text-xs text-muted-foreground">{{ $t('page.aiInsights.assessment') }}</div>
        <div class="text-sm leading-relaxed">{{ insights.summary }}</div>
      </div>
    </template>
  </div>
</template>
