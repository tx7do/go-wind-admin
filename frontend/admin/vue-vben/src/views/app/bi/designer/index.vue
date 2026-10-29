<template>
  <div ref="rootRef" class="bi-designer-page">
    <!-- 页级工具条：编辑目标 + 保存状态 + 预览/全屏 -->
    <div class="bi-designer-page__bar">
      <span class="bi-designer-page__target">{{ targetLabel }}</span>
      <span v-if="dashboardId" class="bi-designer-page__status" :class="`is-${saveStatus}`">
        {{ statusText }}
      </span>
      <span class="bi-designer-page__spacer"></span>
      <a-button v-if="dashboardId" size="small" @click="goPreview">
        {{ $t('page.bi.preview') }}
      </a-button>
      <VbenTooltip side="bottom">
        <template #trigger>
          <VbenIconButton @click="toggleFullscreen">
            <!-- 图标与框架顶栏 FullScreen 同款：Maximize 进 / 手抄 Minimize 出 -->
            <svg
              v-if="isFullscreen"
              class="text-foreground size-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M8 3v3a2 2 0 0 1-2 2H3" />
              <path d="M21 8h-3a2 2 0 0 1-2-2V3" />
              <path d="M3 16v3a2 2 0 0 0 2 2h3" />
              <path d="M16 21h3a2 2 0 0 0 2-2v-3" />
            </svg>
            <Maximize v-else class="text-foreground size-4" />
          </VbenIconButton>
        </template>
        {{ isFullscreen ? $t('page.bi.exitFullscreen') : $t('page.bi.fullscreen') }}
      </VbenTooltip>
    </div>

    <div class="bi-designer-page__canvas">
      <!-- ready 门：取数完成前不挂画布——BiDesigner 不回灌运行期 props，先挂会以演示 schema
           落地且不随加载结果重挂（key 相同），自动保存会把演示内容覆盖进库 -->
      <!-- 不能写 display:block：会压过组件 scoped 的 display:flex，
           内部 __body flex:1 失效 → 画布塌成内容高、无法滚动、抽屉里不撑满 -->
      <BiDesigner
        v-if="ready"
        :key="canvasKey"
        :demo="!initialSchema"
        :initial-schema="initialSchema"
        style="height: 100%"
        @change="onSchemaChange"
        />

    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * BI 独立设计器页（vben 端）：
 * - ?id=N：加载库中仪表盘，画布改动防抖自动保存回写（version 乐观锁，409 后自动刷新版本）；
 *   「预览」跳仪表盘渲染页
 * - 无 id：本地草稿（localStorage），保存落草稿，不进服务端
 * - 全屏：@vueuse/core useFullscreen，进入时把页面底色刷成布局底色（fullscreen 元素透出黑底）
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useFullscreen } from '@vueuse/core';

import { notification } from 'ant-design-vue';

import { BiDesigner, provideBiEnv, setBiLocale } from '@go-wind/bi-vue';
import '@go-wind/bi-vue/style.css';
import type { BiDashboardSchema } from '@go-wind/bi-core';

import {
  fetchBiDashboard,
  useSaveBiDashboardPayload,
} from '#/api/composables/bi_dashboard';
import { usePreferences } from '@vben/preferences';
import { $t } from '#/locales';
import { VbenIconButton, VbenTooltip } from '@vben-core/shadcn-ui';
import { Maximize } from '@vben/icons';

defineOptions({ name: 'BiDesignerPage' });

/** 本地草稿键（无 id 模式），与仪表盘页共用 */
const DRAFT_KEY = 'bi:schema:default';
/** 自动保存防抖 */
const SAVE_DEBOUNCE_MS = 1200;

const route = useRoute();
const router = useRouter();
const { locale: appLocale } = usePreferences();
const biLocale = computed(() =>
  (appLocale.value ?? 'zh-CN').startsWith('zh') ? 'zh-CN' : 'en-US',
);
const { mutateAsync: savePayload } = useSaveBiDashboardPayload();

provideBiEnv({});

const rootRef = ref<HTMLElement | null>(null);
const { isFullscreen, toggle: toggleFs } = useFullscreen(rootRef);

const dashboardId = Number(route.query.id) > 0 ? Number(route.query.id) : undefined;

const initialSchema = ref<Partial<BiDashboardSchema> | undefined>(undefined);
const targetName = ref('');
/** 内容基线：首帧回显（normalizeSchema 会补默认值，与服务端原文永不字符串相等）
 *  定为画布首次 emit 的规范化串；此后内容与基线相同（如 select 类无内容变更）跳过保存 */
const baselinePayload = ref<string | null>(null);
/** 乐观锁本地持有版本：保存成功后 +1，409 后从服务端刷新 */
const versionRef = ref(1);
const saveStatus = ref<'idle' | 'saving' | 'saved' | 'conflict'>('idle');
/** 取数完成前不挂画布（见模板注释） */
const ready = ref(false);

const canvasKey = computed(() => `${dashboardId ?? 'draft'}`);

const targetLabel = computed(() =>
  dashboardId ? targetName.value || `#${dashboardId}` : $t('page.bi.localDraft'),
);

const statusText = computed(() => {
  switch (saveStatus.value) {
    case 'saving': { return $t('page.bi.statusSaving'); }
    case 'saved': { return $t('page.bi.statusSaved'); }
    case 'conflict': { return $t('page.bi.statusConflict'); }
    default: { return ''; }
  }
});

// ============ 全屏：进入时补页面底色（fullscreen 元素透出 UA 黑底） ============
function toggleFullscreen() {
  if (!rootRef.value) return;
  if (!isFullscreen.value) {
    rootRef.value.style.background = getComputedStyle(document.documentElement).backgroundColor;
  } else {
    rootRef.value.style.background = '';
  }
  void toggleFs();
}

// ============ 加载 ============
onMounted(async () => {
  setBiLocale(biLocale.value);
  void loadSchema();
});

watch(biLocale, (locale) => setBiLocale(locale));

async function loadSchema() {
  if (dashboardId) {
    try {
      const row = await fetchBiDashboard(dashboardId);
      targetName.value = row.name || '';
      versionRef.value = row.version ?? 1;
      if (row.payload) {
        initialSchema.value = JSON.parse(row.payload);
      }
    } catch (error) {
      console.error('[bi-designer] 加载仪表盘失败，切换为本地草稿', error);
      notification.error({ message: $t('page.bi.loadFailed') });
      loadDraft();
    }
    ready.value = true;
  } else {
    loadDraft();
    ready.value = true;
  }
}

function loadDraft() {
  const saved = localStorage.getItem(DRAFT_KEY);
  if (saved) {
    try {
      initialSchema.value = JSON.parse(saved);
    } catch (error) {
      console.error('[bi-designer] 本地草稿解析失败', error);
      initialSchema.value = undefined;
    }
  }
}

// ============ 自动保存 ============
let saveTimer: ReturnType<typeof setTimeout> | null = null;
let conflictToasted = false;

function onSchemaChange(schema: BiDashboardSchema) {
  const payloadStr = JSON.stringify(schema);

  if (dashboardId) {
    // 首帧 emit = 加载内容的规范化形态，定为基线（原文与规范化串字符串永不相等，不能当基线）
    if (baselinePayload.value === null) {
      baselinePayload.value = payloadStr;
      return;
    }
    if (payloadStr === baselinePayload.value) return;
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = setTimeout(() => void saveToServer(schema, payloadStr), SAVE_DEBOUNCE_MS);
  } else {
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = setTimeout(() => {
      localStorage.setItem(DRAFT_KEY, payloadStr);
    }, SAVE_DEBOUNCE_MS);
  }
}

async function saveToServer(schema: BiDashboardSchema, payloadStr: string) {
  saveStatus.value = 'saving';
  try {
    await savePayload({
      id: dashboardId!,
      version: versionRef.value,
      payload: payloadStr,
      chartCount: schema.charts?.length ?? 0,
      themeKey: schema.theme?.key ?? '',
    });
    versionRef.value += 1;
    baselinePayload.value = payloadStr;
    saveStatus.value = 'saved';
  } catch (error: any) {
    const msg = String(error?.message || '');
    if (msg.includes('modified by another session') || msg.includes('CONFLICT')) {
      // 他人已保存：刷新版本后继续自动保存（后续编辑会用新版本落盘）
      saveStatus.value = 'conflict';
      if (!conflictToasted) {
        conflictToasted = true;
        notification.warning({ message: $t('page.bi.saveConflict') });
      }
      try {
        const row = await fetchBiDashboard(dashboardId!);
        versionRef.value = row.version ?? versionRef.value;
        saveStatus.value = 'idle';
      } catch (refreshError) {
        console.error('[bi-designer] 冲突后刷新版本失败', refreshError);
      }
    } else {
      console.error('[bi-designer] 自动保存失败', error);
      saveStatus.value = 'idle';
    }
  }
}

function goPreview() {
  void router.push({ path: '/bi/dashboard', query: { id: String(dashboardId) } });
}

onBeforeUnmount(() => {
  if (saveTimer) clearTimeout(saveTimer);
  saveTimer = null;
});
</script>

<style scoped>
.bi-designer-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 12px 16px;
  box-sizing: border-box;

  /* 全屏元素默认透出 UA 黑底，进入时由脚本补布局底色 */
  &:fullscreen {
    background: hsl(var(--background));
  }

  &__bar {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-shrink: 0;
    padding-bottom: 8px;
  }

  &__target {
    font-size: 15px;
    font-weight: 600;
  }

  &__status {
    font-size: 12px;

    &.is-saving {
      color: hsl(var(--secondary-foreground));
    }

    &.is-saved {
      color: hsl(var(--success));
    }

    &.is-conflict {
      color: hsl(var(--destructive));
    }
  }

  &__spacer {
    flex: 1;
  }

  &__canvas {
    flex: 1;
    min-height: 0;
  }
}
</style>
