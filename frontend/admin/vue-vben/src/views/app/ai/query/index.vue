<script lang="ts" setup>
import type { EchartsUIType } from '@vben/plugins/echarts';

import { computed, nextTick, ref, watch } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { message } from 'ant-design-vue';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import {
  EchartsUI,
  useEcharts,
} from '@vben/plugins/echarts';

import { apiClient } from '#/api';

// marked 单行换行按 GFM 处理
marked.setOptions({ gfm: true, breaks: true });

function md(content: string): string {
  // AI 回复经 DOMPurify 消毒后再 v-html；LLM 输出不可信，防注入
  return DOMPurify.sanitize(marked.parse(content, { async: false }) as string);
}

interface Round {
  question: string;
  sql?: string;
  columns?: string[];
  rows?: { cells: string[] }[];
  answer?: string;
  loading: boolean;
}

const sampleKeys = ['sample1', 'sample2', 'sample3'];

const rounds = ref<Round[]>([]);
const input = ref('');
const loading = ref(false);
const scrollRef = ref<HTMLElement>();

// ── 结果图表（确定性推断，零 LLM 参与） ───────────────────────────
const viewMode = ref<'chart' | 'table'>('chart');
const chartRef = ref<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

function numericCols(columns: string[], rows: Round['rows'] = []): number[] {
  return columns
    .map((_, i) => i)
    .filter(
      (i) =>
        i > 0 &&
        rows.length > 0 &&
        rows.every((r) => {
          const v = r.cells?.[i];
          return v !== undefined && v !== null && v !== '' && !Number.isNaN(Number(v));
        }),
    );
}

const currentRound = computed(() => rounds.value[rounds.value.length - 1]);

function renderChart() {
  const r = currentRound.value;
  if (!r) return;
  const rows = r.rows ?? [];
  if (viewMode.value !== 'chart') return;
  const cols = r.columns ?? [];
  const series = numericCols(cols, rows);
  if (series.length === 0) return;
  const kind = rows.length > 12 ? 'line' : 'bar';
  renderEcharts({
    grid: { left: 8, right: 8, top: 24, bottom: 8, containLabel: true },
    legend: { top: 0 },
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: rows.map((row) => row.cells?.[0] ?? ''),
      axisLabel: { rotate: rows.length > 6 ? 30 : 0 },
    },
    yAxis: { type: 'value' },
    series: series.map((colIdx) => ({
      name: cols[colIdx] ?? '',
      type: kind,
      barMaxWidth: 40,
      data: rows.map((row) => Number(row.cells?.[colIdx] ?? 0)),
    })),
  });
}

watch([currentRound, viewMode], renderChart, { deep: true });

function scrollToBottom() {
  nextTick(() => {
    scrollRef.value?.scrollTo({
      top: scrollRef.value?.scrollHeight ?? 0,
      behavior: 'smooth',
    });
  });
}

async function handleAsk(text?: string) {
  const question = (text ?? input.value).trim();
  if (!question || loading.value) return;
  // 携带最近 5 轮历史（问题+SQL+结果摘要），供模型消解追问里的指代
  const history = rounds.value
    .filter((r) => !r.loading && r.sql)
    .slice(-5)
    .map((r) => ({
      question: r.question,
      sql: r.sql || '',
      resultSummary: (r.rows || []).slice(0, 3).map((row) => row.cells.join(' | ')).join('；'),
    }));
  rounds.value.push({ question, loading: true });
  input.value = '';
  loading.value = true;
  scrollToBottom();
  try {
    const lang = $t('page.aiQuery.lang') || 'zh-CN';
    const resp = await apiClient.aiQueryService.Ask({
      question,
      lang,
      withAnswer: true,
      history,
    } as any);
    const last = rounds.value[rounds.value.length - 1];
    if (last) {
      last.sql = resp.sql;
      last.columns = (resp.columns || []) as string[];
      last.rows = (resp.rows || []).map((r) => ({
        cells: (r.values || []) as string[],
      }));
      last.answer = resp.answer || '';
      last.loading = false;
    }
    scrollToBottom();
  } catch (error) {
    console.error('ai query failed:', error);
    message.error(error instanceof Error ? error.message : 'query failed');
    const last = rounds.value[rounds.value.length - 1];
    if (last) last.loading = false;
  } finally {
    loading.value = false;
  }
}

function sampleText(key: string): string {
  const map: Record<string, string> = {
    sample1: $t('page.aiQuery.sample1'),
    sample2: $t('page.aiQuery.sample2'),
    sample3: $t('page.aiQuery.sample3'),
  };
  return map[key] || key;
}
</script>

<template>
  <Page auto-content-height>
    <div class="mx-auto flex h-full max-w-4xl flex-col gap-4">
      <!-- 消息滚动区 -->
      <div
        ref="scrollRef"
        class="min-h-0 flex-1 overflow-y-auto rounded-xl border border-solid border-border bg-card p-4"
      >
        <!-- 标题 + 示例问题（仅首轮前展示） -->
        <div
          v-if="rounds.length === 0"
          class="flex flex-col items-center gap-3 py-10 text-center"
        >
          <span class="text-5xl">⚡</span>
          <div class="text-lg font-semibold">{{ $t('page.aiQuery.title') }}</div>
          <div class="text-sm text-muted-foreground">{{ $t('page.aiQuery.emptyDesc') }}</div>
          <div class="mt-2 flex flex-wrap justify-center gap-2">
            <a-button
              v-for="key in sampleKeys"
              :key="key"
              size="small"
              @click="input = sampleText(key)"
            >
              {{ sampleText(key) }}
            </a-button>
          </div>
        </div>

        <div class="flex flex-col gap-3">
          <div v-for="(round, i) in rounds" :key="i" class="flex flex-col gap-3">
            <!-- 用户问题：右侧气泡 -->
            <div class="flex flex-row-reverse items-start gap-3">
              <div
                class="max-w-[80%] whitespace-pre-wrap break-words rounded-xl rounded-tr-sm bg-primary px-4 py-2 text-primary-foreground"
              >
                {{ round.question }}
              </div>
            </div>

            <!-- 结果卡片：左侧 -->
            <div class="flex items-start gap-3">
              <div
                class="min-w-0 flex-1 rounded-xl border border-solid border-border bg-card p-4"
              >
                <div
                  v-if="round.loading"
                  class="flex items-center gap-2 text-sm text-muted-foreground"
                >
                  <span class="inline-block h-2 w-2 animate-pulse rounded-full bg-primary" />
                  {{ $t('page.aiQuery.thinking') }}
                </div>
                  <template v-else>
                  <details class="mb-3" open>
                    <summary class="cursor-pointer text-xs text-muted-foreground">
                      {{ $t('page.aiQuery.generatedSql') }}
                    </summary>
                    <pre
                      class="sql-block mt-2 overflow-x-auto p-3 text-xs leading-relaxed"
                      >{{ round.sql }}</pre
                    >
                  </details>

                  <template v-if="round.columns && round.columns.length > 0">
                    <div
                      class="mb-2 flex items-center justify-between"
                    >
                      <a-radio-group v-model:value="viewMode" size="small">
                        <a-radio-button value="chart">
                          {{ $t('page.aiQuery.viewChart') }}
                        </a-radio-button>
                        <a-radio-button value="table">
                          {{ $t('page.aiQuery.viewTable') }}
                        </a-radio-button>
                      </a-radio-group>
                      <span class="text-xs text-muted-foreground">
                        {{ $t('page.aiQuery.rowCount', { count: round.rows?.length ?? 0 }) }}
                      </span>
                    </div>
                    <div
                      class="mb-2 overflow-hidden rounded-lg border border-solid border-border p-2"
                    >
                      <EchartsUI ref="chartRef" height="280px" width="100%" />
                    </div>
                    <a-table
                      v-if="viewMode === 'table'"
                      :columns="round.columns.map((c, i) => ({ title: c, dataIndex: String(i) }))"
                      :data-source="(round.rows || []).map((r, ri) => ({ key: ri, cells: r.cells }))"
                      :pagination="
                        (round.rows?.length || 0) > 10
                          ? { pageSize: 10, showSizeChanger: false }
                          : false
                      "
                      size="small"
                    >
                      <template #bodyCell="{ column, record }">
                        {{ record.cells[Number(column.dataIndex)] }}
                      </template>
                    </a-table>
                  </template>
                  <div
                    v-if="round.sql && (round.rows?.length ?? 0) === 0"
                    class="mt-2 text-xs text-muted-foreground"
                  >
                    {{ $t('page.aiQuery.noRows') }}
                  </div>

                  <div
                    v-if="round.answer"
                    class="mt-3 rounded-xl border border-solid border-border bg-primary/5 p-3 text-sm leading-relaxed"
                  >
                    <a-tag color="processing">{{ $t('page.aiQuery.answerTag') }}</a-tag>
                    <div class="markdown-body break-words" v-html="md(round.answer)"></div>
                  </div>
                </template>
              </div>
            </div>
          </div>
        </div>
        <div ref="bottomRef"></div>
      </div>

      <!-- 输入区（吸底） -->
      <div
        class="shrink-0 rounded-xl border border-solid border-border bg-card p-3"
      >
        <div class="flex items-end gap-2">
          <a-textarea
            v-model:value="input"
            :auto-size="{ minRows: 1, maxRows: 4 }"
            :disabled="loading"
            :placeholder="$t('page.aiQuery.inputPlaceholder')"
            @keydown.enter.exact.prevent="handleAsk()"
          />
          <a-button
            :disabled="!input.trim()"
            :loading="loading"
            type="primary"
            @click="handleAsk()"
          >
            {{ $t('page.aiQuery.ask') }}
          </a-button>
        </div>
      </div>
    </div>
  </Page>
</template>



<style scoped>
/* SQL 块：放在 bg-card 的结果卡里，所以取 --accent（比表面亮一档）而不是再铺一层 --card */
.sql-block {
  color: hsl(var(--foreground));
  background: hsl(var(--accent));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
}

/* 本页 markdown 样式原先只有 p/table/th/td 四条，且两条是坏的：
   `border: 1px solid var(--border)` 拿到的是 HSL 三元组而非颜色 → 整条声明被丢弃；
   `rgb(128 128 128 / 10%)` 是硬编码中性灰。列表/代码/引用此前完全没样式
   （Tailwind 语境下 list-style 与缩进被 reset 掉了），故与 ai/chat 参照实现对齐。 */
.markdown-body :deep(p) {
  margin: 0 0 0.5em;
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4) {
  margin: 0.8em 0 0.4em;
  font-weight: 600;
  line-height: 1.4;
}

.markdown-body :deep(h1) {
  font-size: 1.25em;
}

.markdown-body :deep(h2) {
  font-size: 1.15em;
}

.markdown-body :deep(h3),
.markdown-body :deep(h4) {
  font-size: 1.05em;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 1.5em;
  margin: 0.4em 0;
}

.markdown-body :deep(ul:last-child),
.markdown-body :deep(ol:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(ul) {
  list-style: disc;
}

.markdown-body :deep(ul ul) {
  list-style: circle;
}

.markdown-body :deep(ol) {
  list-style: decimal;
}

.markdown-body :deep(li) {
  margin: 0.2em 0;
}

.markdown-body :deep(li > p) {
  margin: 0;
}

.markdown-body :deep(a) {
  color: hsl(var(--primary));
  text-decoration: none;
}

.markdown-body :deep(a:hover) {
  text-decoration: underline;
}

.markdown-body :deep(blockquote) {
  padding: 2px 0 2px 12px;
  margin: 0.5em 0;
  color: hsl(var(--muted-foreground));
  border-left: 3px solid hsl(var(--border));
}

.markdown-body :deep(code) {
  padding: 1px 5px;
  font-size: 0.92em;
  background: hsl(var(--muted));
  border: 1px solid hsl(var(--border));
  border-radius: 4px;
}

.markdown-body :deep(pre) {
  max-width: 100%;
  padding: 12px;
  margin: 0.5em 0;
  overflow-x: auto;
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
}

.markdown-body :deep(pre code) {
  padding: 0;
  font-size: 12.5px;
  line-height: 1.6;
  background: transparent;
  border: none;
}

.markdown-body :deep(table) {
  display: block;
  max-width: 100%;
  margin: 0.5em 0;
  overflow-x: auto;
  border-collapse: collapse;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  padding: 5px 10px;
  border: 1px solid hsl(var(--border));
}

.markdown-body :deep(th) {
  font-weight: 600;
  background: hsl(var(--muted));
}
</style>
