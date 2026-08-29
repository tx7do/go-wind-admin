<template>
  <EchartsUI ref="chartRef" height="100%" />
</template>

<script lang="ts" setup>
import type { ActionDistributionResponse } from "@/api/generated/admin/service/v1";

import { EchartsUI, EchartsUIType, useEcharts } from "@/plugins/echarts";
import { $t } from "@/core/i18n";
import { usePreferences } from "@/core/preferences";

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
    case "CREATE":
      return $t("enum.operationAuditLog.action.CREATE");
    case "UPDATE":
      return $t("enum.operationAuditLog.action.UPDATE");
    case "DELETE":
      return $t("enum.operationAuditLog.action.DELETE");
    case "READ":
      return $t("enum.operationAuditLog.action.READ");
    case "ASSIGN":
      return $t("enum.operationAuditLog.action.ASSIGN");
    case "UNASSIGN":
      return $t("enum.operationAuditLog.action.UNASSIGN");
    case "EXPORT":
      return $t("enum.operationAuditLog.action.EXPORT");
    case "IMPORT":
      return $t("enum.operationAuditLog.action.IMPORT");
    case "OTHER":
      return $t("enum.operationAuditLog.action.OTHER");
    default:
      return label ?? "";
  }
};
const chartOptions = computed(() => {
  const items = props.data?.items ?? [];
  return {
    legend: {
      bottom: "2%",
      left: "center",
      textStyle: {
        color: isDark.value ? "#CFD3DC" : "#606266",
        fontSize: 12,
      },
    },
    series: [
      {
        animationDelay() {
          return Math.random() * 100;
        },
        animationEasing: "exponentialInOut",
        animationType: "scale",
        avoidLabelOverlap: false,
        color: ["#4080ff", "#36d399", "#f7ba1e", "#958ce2"],
        data: items.map((it) => ({
          name: actionLabel(it.label),
          value: it.count,
        })),
        emphasis: {
          label: {
            fontSize: "12",
            fontWeight: "bold",
            show: true,
          },
        },
        itemStyle: {
          borderRadius: 10,
          borderWidth: 2,
        },
        label: {
          position: "center",
          show: false,
        },
        labelLine: {
          show: false,
        },
        name: $t("pages.dashboard.operationActionDistribution"),
        radius: ["40%", "65%"],
        type: "pie",
      },
    ],
    tooltip: {
      backgroundColor: isDark.value ? "rgba(40,40,40,0.96)" : "rgba(255,255,255,0.96)",
      borderColor: isDark.value ? "#4c4d4f" : "#eee",
      borderRadius: 8,
      padding: [12, 16],
      textStyle: {
        color: isDark.value ? "#ffffff" : "#303133",
        fontSize: 13,
      },
      trigger: "item",
    },
  };
});

watch(
  () => chartOptions.value,
  (options) => {
    renderEcharts(options as any);
  },
  { immediate: true, deep: true }
);
</script>
