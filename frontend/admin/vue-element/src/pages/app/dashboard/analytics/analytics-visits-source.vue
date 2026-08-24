<template>
  <EchartsUI ref="chartRef" height="100%" />
</template>

<script lang="ts" setup>
import type { StatusDistributionResponse } from "@/api/generated/admin/service/v1";

import { EchartsUI, EchartsUIType, useEcharts } from "@/plugins/echarts";
import { $t } from "@/core/i18n";
import { usePreferences } from "@/core/preferences";

const props = defineProps<{
  data?: StatusDistributionResponse;
}>();

const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);
const { isDark } = usePreferences();

// 登录成功/失败占比环形图。后端返回 status 枚举名（SUCCESS/FAILED/...），这里经 i18n 转成本地化文案展示。
const statusLabel = (label?: string): string => {
  switch (label) {
    case "SUCCESS":
      return $t("enum.loginAuditLog.status.SUCCESS");
    case "FAILED":
      return $t("enum.loginAuditLog.status.FAILED");
    case "PARTIAL":
      return $t("enum.loginAuditLog.status.PARTIAL");
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
          name: statusLabel(it.label),
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
        name: $t("pages.dashboard.loginStatusDistribution"),
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
