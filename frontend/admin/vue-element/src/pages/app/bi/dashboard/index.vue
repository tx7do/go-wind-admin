<template>
  <div class="bi-dashboard-page">
    <!-- 工具栏：主题切换 + 数据集刷新 -->
    <div class="bi-dashboard-page__toolbar">
      <span class="bi-dashboard-page__title">{{ $t("pages.bi.dashboardTitle") }}</span>
      <ElButton size="small" @click="refresh">
        {{ $t("common.button.retry") }}
      </ElButton>
    </div>

    <!-- 渲染器：schema 驱动（数据集 static 内嵌 / api 走注入的 resolver） -->
    <div class="bi-dashboard-page__body">
      <BiDashboard
        ref="dashboardRef"
        :schema="schema"
        :view="{ designWidth: 1920, fit: 'width' }"
        @chart-click="onChartClick"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * BI 仪表盘页面：@go-wind/bi-vue 渲染器消费。
 * schema 落 localStorage（MVP 存储；后续接后端 dashboard CRUD）。
 */
import { onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { BiDashboard, provideBiEnv, setBiLocale } from "@go-wind/bi-vue";
import type { BiDashboardSchema } from "@go-wind/bi-core";
import { useI18n } from "@/core/i18n";


defineOptions({ name: "BiDashboardPage" });

const route = useRoute();
const { t, locale } = useI18n();
const dashboardRef = ref<InstanceType<typeof BiDashboard>>();

// env 注入：api 型数据集 resolver（演示接 IoT 设备列表）
// api 数据集 resolver 待接后端 CRUD 后注入
provideBiEnv({});

onMounted(() => {
  setBiLocale(locale.value.startsWith("zh") ? "zh-CN" : "en-US");
});

// schema：从 localStorage 读设计器保存的仪表盘（或用内置演示）
const schema = ref<Partial<BiDashboardSchema>>({});

onMounted(() => {
  const saved = localStorage.getItem(`bi:schema:${route.query.id ?? "default"}`);
  if (saved) {
    try {
      schema.value = JSON.parse(saved);
    } catch {
      schema.value = DEMO_SCHEMA;
    }
  } else {
    schema.value = DEMO_SCHEMA;
  }
});

function refresh() {
  void dashboardRef.value?.refresh();
}

function onChartClick(event: unknown) {
  console.log("[bi-dashboard] chart click:", event);
}

const DEMO_SCHEMA: Partial<BiDashboardSchema> = {
  name: "default",
  linkage: { enabled: true },
  theme: { key: "tech-blue" },
  datasets: [
    {
      id: "ds_month",
      name: "月度汇总",
      type: "static",
      rows: Array.from({ length: 12 }, (_, i) => ({
        month: `${i + 1}月`,
        sales: Math.round(80 + Math.sin(i / 2) * 60 + i * 8),
        target: 160,
      })),
    },
    {
      id: "ds_orders",
      name: "订单明细",
      type: "static",
      rows: Array.from({ length: 48 }, (_, i) => ({
        city: ["北京", "上海", "广州", "深圳", "杭州", "成都"][i % 6],
        category: ["硬件", "软件", "服务"][i % 3],
        amount: Math.round(50 + Math.random() * 450),
      })),
    },
  ],
  charts: [
    {
      id: "c_bar",
      title: "城市订单 Top6",
      datasetId: "ds_orders",
      type: "bar",
      x: 0, y: 0, w: 6, h: 4,
      encoding: { dimension: "city", measure: "amount" },
      aggregation: "sum",
      sort: { by: "measure", order: "desc", topN: 6 },
    },
    {
      id: "c_wide",
      title: "销售 vs 目标",
      datasetId: "ds_month",
      type: "bar",
      x: 6, y: 0, w: 6, h: 4,
      encoding: { dimension: "month", measures: ["sales", "target"] },
      style: { showLegend: true },
    },
    {
      id: "c_pie",
      title: "品类占比",
      datasetId: "ds_orders",
      type: "donut",
      x: 0, y: 4, w: 6, h: 4,
      encoding: { dimension: "category", measure: "amount" },
      aggregation: "sum",
    },
    {
      id: "c_area",
      title: "销售趋势",
      datasetId: "ds_month",
      type: "area",
      x: 6, y: 4, w: 6, h: 4,
      encoding: { dimension: "month", measure: "sales" },
    },
  ],
};
</script>

<style scoped lang="scss">
.bi-dashboard-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 12px 16px;
  box-sizing: border-box;

  &__toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-shrink: 0;
    padding-bottom: 10px;
  }

  &__title {
    font-size: 15px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__body {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
}
</style>
