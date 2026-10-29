<template>
  <div ref="rootRef" class="bi-dashboard-page">
    <div class="bi-dashboard-page__toolbar">
      <span class="bi-dashboard-page__title">{{ $t('page.bi.dashboardTitle') }}</span>
      <span class="bi-dashboard-page__spacer"></span>
      <a-button v-if="dashboardId" size="small" @click="goDesigner">
        {{ $t('common.edit') }}
      </a-button>
      <a-button size="small" @click="toggleFullscreen">
        {{ isFullscreen ? $t('page.bi.exitFullscreen') : $t('page.bi.fullscreen') }}
      </a-button>
    </div>
    <div ref="bodyRef" class="bi-dashboard-page__body">
      <BiDashboard
        :schema="schema"
        :view="{ designWidth, fit: 'width' }"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * BI 仪表盘页面（vben 端）：@go-wind/bi-vue 渲染器。
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { BiDashboard, setBiLocale } from '@go-wind/bi-vue';
import '@go-wind/bi-vue/style.css';
import { layoutCharts } from '@go-wind/bi-core';
import type { BiDashboardSchema } from '@go-wind/bi-core';
import { useRoute, useRouter } from 'vue-router';
import { useFullscreen } from '@vueuse/core';
import { fetchBiDashboard } from '#/api/composables/bi_dashboard';
import { usePreferences } from '@vben/preferences';

defineOptions({ name: 'BiDashboardPage' });

/** 与 /bi/designer 共用：设计器落盘、仪表盘读取 */
const BI_SCHEMA_STORAGE_KEY = 'bi:schema:default';

const route = useRoute();
const router = useRouter();
const { locale: appLocale } = usePreferences();

const dashboardId = computed(() =>
  Number(route.query.id) > 0 ? Number(route.query.id) : undefined,
);

// 全屏：进入时补页面底色（fullscreen 元素透出 UA 黑底）
const rootRef = ref<HTMLElement | null>(null);
const { isFullscreen, toggle: toggleFs } = useFullscreen(rootRef);

function toggleFullscreen() {
  if (!rootRef.value) return;
  if (!isFullscreen.value) {
    rootRef.value.style.background = getComputedStyle(document.documentElement).backgroundColor;
  } else {
    rootRef.value.style.background = '';
  }
  void toggleFs();
}

function goDesigner() {
  void router.push({ path: '/bi/designer', query: { id: String(dashboardId.value) } });
}
const biLocale = computed(() =>
  (appLocale.value ?? 'zh-CN').startsWith('zh') ? 'zh-CN' : 'en-US',
);

// ============ 视口填充 ============
// BiDashboard 的 fit=width 只按容器宽等比缩放，高度随内容（8 行网格 ≈ 644 设计px），
// 容器偏矮时会缩成半屏、下方留白。按容器宽高比反推设计宽：
// 缩放比 = 容器宽/设计宽，令 内容高×缩放比 = 容器高 → designWidth = 内容高 × W/H。
const bodyRef = ref<HTMLElement | null>(null);
const designWidth = ref(1920);
let resizeObserver: ResizeObserver | null = null;

// schema 来源优先级：?id= 后端行 > localStorage 草稿 > 内置演示
async function loadSchema() {
  const id = Number(route.query.id);
  if (id > 0) {
    try {
      const row = await fetchBiDashboard(id);
      if (row.payload) {
        schema.value = JSON.parse(row.payload);
        syncDesignWidth();
        return;
      }
    } catch (error) {
      console.error('[bi-dashboard] 拉取仪表盘失败，改用演示 schema', error);
    }
  }
  // 设计器保存过的 schema 优先于内置演示（两页共用一个键）
  const saved = localStorage.getItem(BI_SCHEMA_STORAGE_KEY);
  if (saved) {
    try {
      schema.value = JSON.parse(saved);
    } catch (error) {
      console.error('[bi-dashboard] 本地 schema 解析失败，改用演示 schema', error);
    }
  }
  syncDesignWidth();
}

/** 内容高度（设计单位）：直接复用渲染器的布局函数，避免公式重复走样 */
function contentHeightOf(value: Partial<BiDashboardSchema>): number {
  return layoutCharts(value.charts ?? [], value.grid, 1920).height;
}

function syncDesignWidth() {
  const el = bodyRef.value;
  if (!el) return;
  const width = el.clientWidth;
  // 留 2px 余量，避免取整误差把内容顶出滚动条
  const height = el.clientHeight - 2;
  const contentHeight = contentHeightOf(schema.value);
  if (width <= 0 || height <= 0 || contentHeight <= 0) return;
  designWidth.value = Math.min(3840, Math.max(960, Math.round((contentHeight * width) / height)));
}

onMounted(() => {
  setBiLocale(biLocale.value);
  void loadSchema();
  if (!bodyRef.value) return;
  // 容器尺寸变化（窗口缩放 / 侧栏折叠）都要重算设计宽
  resizeObserver = new ResizeObserver(syncDesignWidth);
  resizeObserver.observe(bodyRef.value);
  syncDesignWidth();
});

watch(biLocale, (locale) => setBiLocale(locale));

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  resizeObserver = null;
});

const schema = ref<Partial<BiDashboardSchema>>({
  name: 'default',
  linkage: { enabled: true },
  // 不钉主题预设：跟随宿主亮暗（大屏需要钉死时在设计器主题下拉里选预设）
  datasets: [
    {
      id: 'ds_month',
      name: '月度汇总',
      type: 'static',
      rows: Array.from({ length: 12 }, (_, i) => ({
        month: `${i + 1}月`,
        sales: Math.round(80 + Math.sin(i / 2) * 60 + i * 8),
        target: 160,
      })),
    },
    {
      id: 'ds_orders',
      name: '订单明细',
      type: 'static',
      rows: Array.from({ length: 48 }, (_, i) => ({
        city: ['北京', '上海', '广州', '深圳', '杭州', '成都'][i % 6],
        category: ['硬件', '软件', '服务'][i % 3],
        amount: Math.round(50 + Math.random() * 450),
      })),
    },
  ],
  charts: [
    {
      id: 'c_bar', title: '城市订单 Top6', datasetId: 'ds_orders', type: 'bar',
      x: 0, y: 0, w: 6, h: 4,
      encoding: { dimension: 'city', measure: 'amount' },
      aggregation: 'sum', sort: { by: 'measure', order: 'desc', topN: 6 },
    },
    {
      id: 'c_wide', title: '销售 vs 目标', datasetId: 'ds_month', type: 'bar',
      x: 6, y: 0, w: 6, h: 4,
      encoding: { dimension: 'month', measures: ['sales', 'target'] },
      style: { showLegend: true },
    },
    {
      id: 'c_pie', title: '品类占比', datasetId: 'ds_orders', type: 'donut',
      x: 0, y: 4, w: 6, h: 4,
      encoding: { dimension: 'category', measure: 'amount' }, aggregation: 'sum',
    },
    {
      id: 'c_area', title: '销售趋势', datasetId: 'ds_month', type: 'area',
      x: 6, y: 4, w: 6, h: 4,
      encoding: { dimension: 'month', measure: 'sales' },
    },
  ],
});
</script>

<style scoped lang="scss">
.bi-dashboard-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 12px 16px;
  box-sizing: border-box;

  &:fullscreen {
    background: hsl(var(--background));
  }

  &__toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-shrink: 0;
    padding-bottom: 10px;
  }

  &__spacer {
    flex: 1;
  }

  &__title {
    font-size: 15px;
    font-weight: 600;
  }

  &__body {
    flex: 1;
    min-height: 0;
    // 断掉"出滚动条→容器变窄→重算设计宽"的反馈环：设计宽已按可视区精确反推
    overflow: hidden;
  }
}
</style>
