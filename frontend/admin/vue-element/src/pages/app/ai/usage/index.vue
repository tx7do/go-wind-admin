<template>
  <div class="app-container ai-usage-page h-full flex flex-1 flex-col">
    <!-- 汇总卡 -->
    <el-row :gutter="16" class="mb-4">
      <el-col v-for="m in metricItems" :key="m.label" :xs="24" :sm="8">
        <el-card shadow="hover">
          <div class="metric-label">{{ m.label }}</div>
          <div class="metric-value" :style="{ color: m.color }">{{ m.value }}</div>
          <el-progress
            v-if="m.progress !== undefined"
            :percentage="m.progress"
            :status="m.progress > 80 ? 'exception' : 'success'"
          />
        </el-card>
      </el-col>
    </el-row>

    <!-- 流水列表：ProPage 统一搜索/分页/导出约定（与其余管理页一致） -->
    <div class="flex-1 min-h-0">
      <ProPage ref="pageRef" :config="pageConfig" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { ElCard, ElCol, ElProgress, ElRow, ElMessage } from "element-plus";
import { useI18n } from "@/core/i18n";
import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
import { PaginationQuery } from "@/core/transport/rest";
import { apiClient } from "@/api/client";
import { serverExportFile } from "@/api/composables/server-export";

const { t } = useI18n();

const pageRef = ref();

// 当前搜索条件（服务端导出透传用，与列表 listAction 同源——"导出的就是当前搜索看到的"）
const latestFormValuesRef = ref<Record<string, unknown>>({});

const monthTokens = ref(0);
const monthCalls = ref(0);
const quotaConfigured = ref(false);
const quotaLimit = ref(0);

const metricItems = computed(() => {
  const pct = quotaConfigured.value && quotaLimit.value > 0
    ? Math.min(100, Math.round((monthTokens.value * 100) / quotaLimit.value))
    : undefined;
  // 汇总数字是「文字」不是色块，取值随主题切换（见底部 --metric-* 定义）：
  // 亮色下 #22d3ee/#a78bfa 对白底只有 1.81/2.72:1（24px 大字号的下限是 3.0）
  return [
    { label: t("pages.ai_usage.monthTokens"), value: monthTokens.value.toLocaleString(), color: "var(--metric-blue)", progress: pct },
    { label: t("pages.ai_usage.monthCalls"), value: monthCalls.value.toLocaleString(), color: "var(--metric-cyan)", progress: undefined },
    { label: t("pages.ai_usage.quota"), value: quotaConfigured.value ? quotaLimit.value.toLocaleString() : "∞", color: "var(--metric-violet)", progress: undefined },
  ];
});

async function loadSummary() {
  try {
    const resp = await apiClient.aiUsageLogService.GetUsageSummary({});
    monthTokens.value = resp.monthTokens ?? 0;
    monthCalls.value = resp.monthCalls ?? 0;
    quotaConfigured.value = resp.quotaConfigured ?? false;
    // uint64 经 protojson 是字符串，参与算术前转回数字
    quotaLimit.value = Number(resp.quotaLimit ?? 0);
  } catch (error: any) {
    console.error("load usage summary failed:", error);
  }
}

/** 流水列表 fetcher：List + 分页参数（列表与导出共用） */
async function fetchUsageLogs(query: PaginationQuery) {
  return apiClient.aiUsageLogService.List(query.toRawParams());
}

// 服务端全量导出：当前搜索条件经 PaginationQuery 同源序列化透传，
// 导出的即当前搜索看到的；行范围由后端 viewer 语义决定（租户=本租户）。
const exportingServer = ref(false);
async function handleServerExport() {
  if (exportingServer.value) return;
  exportingServer.value = true;
  try {
    await serverExportFile(
      "admin/v1/ai/usage-logs:export",
      new PaginationQuery({ formValues: latestFormValuesRef.value }),
    );
    ElMessage.success(t("pages.ai_usage.exportServerSuccess"));
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error("server-side export failed", error);
    ElMessage.error(error?.message || t("pages.ai_usage.exportServerFailed"));
  } finally {
    exportingServer.value = false;
  }
}

const pageConfig = computed<ProPageConfig>(() => ({
  skeleton: true,
  exportFilename: "ai-usage-logs",
  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: t("pages.ai_usage.model"),
        field: "modelName",
        attrs: { placeholder: t("common.placeholder.input"), clearable: true },
      },
    ],
  },
  table: {
    // 服务端全量导出：导出全部下拉的第三项，错误处理在本页 handleServerExport
    exportServer: handleServerExport,
    listAction: async (query: any) => {
      const { page, pageSize, ...rest } = query;
      latestFormValuesRef.value = rest;
      // formValues 字符串值由 PaginationQuery 统一转 __contains（搜索铁律）
      const result = await fetchUsageLogs(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 20 },
          formValues: Object.keys(rest).length > 0 ? rest : undefined,
        }),
      );
      return { items: result.items || [], total: Number(result.total || 0) };
    },
    toolbar: [],
    toolbarRight: [],
    defaultToolbar: ["exports", "refresh", "zoom", "filter"],
    tableAttrs: { border: true, stripe: true },
    columns: [
      { type: "index", label: t("common.table.seq"), width: 60 },
      { prop: "modelName", label: t("pages.ai_usage.model"), minWidth: 180 },
      {
        prop: "promptTokens",
        label: t("pages.ai_usage.promptTokens"),
        width: 130,
        formatter: (row: any) => Number(row.promptTokens ?? 0).toLocaleString(),
      },
      {
        prop: "completionTokens",
        label: t("pages.ai_usage.completionTokens"),
        width: 140,
        formatter: (row: any) => Number(row.completionTokens ?? 0).toLocaleString(),
      },
      {
        prop: "totalTokens",
        label: t("pages.ai_usage.totalTokens"),
        width: 110,
        formatter: (row: any) => Number(row.totalTokens ?? 0).toLocaleString(),
      },
      {
        prop: "durationMs",
        label: t("pages.ai_usage.duration"),
        width: 110,
        formatter: (row: any) => `${Number(row.durationMs ?? 0)} ms`,
      },
      {
        prop: "createdAt",
        label: t("pages.ai_usage.time"),
        width: 180,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
    ],
  },
}));

onMounted(() => {
  loadSummary();
});
</script>

<style lang="scss" scoped>
// 汇总数字的三个强调色（24px/600 → WCAG 大字号门槛 3.0:1）。
// 亮色侧原值实测不可读：#22d3ee 1.81、#a78bfa 2.72（卡片是白底），
// 故亮色降到 cyan-700 / violet-600（5.36 / 5.70）；暗色侧原值达标（9.82 / 6.52）保留。
.ai-usage-page {
  --metric-blue: #3b82f6; // 白底 3.68
  --metric-cyan: #0e7490;
  --metric-violet: #7c3aed;
}

html.dark .ai-usage-page {
  --metric-blue: #3b82f6; // L1 #111827 上 4.82
  --metric-cyan: #22d3ee;
  --metric-violet: #a78bfa;
}

.metric-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}

.metric-value {
  font-size: 24px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
</style>
