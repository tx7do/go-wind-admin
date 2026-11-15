<template>
  <div class="app-container ai-query-page h-full flex flex-1 flex-col">
    <div class="mx-auto flex w-full max-w-4xl flex-1 flex-col gap-4 overflow-y-auto pb-2">
      <!-- 标题 + 示例问题（仅首轮前展示） -->
      <div v-if="rounds.length === 0" class="flex flex-col items-center gap-3 py-10 text-center">
        <Icon icon="lucide:zap" class="text-5xl query-empty__icon" />
        <div class="text-lg font-semibold">{{ t("pages.ai_query.title") }}</div>
        <div class="text-sm query-hint">{{ t("pages.ai_query.emptyDesc") }}</div>
        <div class="mt-2 flex flex-wrap justify-center gap-2">
          <ElButton
            v-for="key in sampleKeys"
            :key="key"
            size="small"
            @click="input = t(`pages.ai_query.${key}`)"
          >
            {{ t(`pages.ai_query.${key}`) }}
          </ElButton>
        </div>
      </div>

      <!-- 问答轮次 -->
      <div v-for="(round, i) in rounds" :key="i" class="flex flex-col gap-3">
        <!-- 用户问题：右侧气泡 -->
        <div class="flex flex-row-reverse items-start gap-3">
          <div class="query-bubble max-w-[80%] whitespace-pre-wrap break-words rounded-2xl rounded-tr-sm px-4 py-2">
            {{ round.question }}
          </div>
        </div>

        <!-- 结果卡片：左侧 -->
        <div class="flex items-start gap-3">
          <div class="query-card min-w-0 flex-1 rounded-xl p-4">
            <div v-if="round.loading" class="flex items-center gap-2 text-sm query-hint">
              <span class="query-dot inline-block h-2 w-2 animate-pulse rounded-full" />
              {{ t("pages.ai_query.thinking") }}
            </div>
            <template v-else>
              <details v-if="round.sql" class="mb-3" open>
                <summary class="cursor-pointer text-xs query-hint">
                  {{ t("pages.ai_query.generatedSql") }}
                </summary>
                <pre class="query-sql mt-2 overflow-x-auto rounded-lg p-3 text-xs leading-relaxed">{{ round.sql }}</pre>
              </details>

              <QueryResultCard
                v-if="round.columns && round.columns.length > 0"
                :columns="round.columns"
                :rows="round.rows || []"
              />
              <div v-if="round.errorMessage" class="query-error mt-2 text-xs">
                {{ round.errorMessage }}
              </div>
              <div v-else-if="round.sql && (round.rows?.length ?? 0) === 0" class="query-hint mt-2 text-xs">
                {{ t("pages.ai_query.noRows") }}
              </div>

              <div
                v-if="round.answer"
                class="query-answer mt-3 rounded-lg p-3 text-sm leading-relaxed"
              >
                <ElTag type="primary" size="small" class="mb-1">{{ t("pages.ai_query.answerTag") }}</ElTag>
                <div class="markdown-body break-words" v-html="renderMarkdown(round.answer)" />
              </div>
            </template>
          </div>
        </div>
      </div>
    </div>

    <!-- 输入区（吸底） -->
    <div class="query-composer sticky bottom-0 mt-4 p-4">
      <div class="mx-auto flex w-full max-w-4xl items-end gap-2">
        <ElInput
          v-model="input"
          type="textarea"
          :autosize="{ minRows: 1, maxRows: 4 }"
          :placeholder="t('pages.ai_query.inputPlaceholder')"
          :disabled="loading"
          @keydown.enter.exact.prevent="handleAsk()"
        />
        <ElButton type="primary" :loading="loading" :disabled="!input.trim()" @click="handleAsk()">
          {{ loading ? t("pages.ai_query.thinking") : t("pages.ai_query.ask") }}
        </ElButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { nextTick, ref } from "vue";
import { ElButton, ElInput, ElMessage, ElTable, ElTableColumn, ElTag } from "element-plus";
import { Icon } from "@iconify/vue";
import { marked } from "marked";
import DOMPurify from "dompurify";
import { i18n, useI18n } from "@go-wind/admin-element";
import { apiClient } from "@/api/client";
import QueryResultCard from "./query-result-card.vue";

// marked 单行换行按 GFM 处理
marked.setOptions({ gfm: true, breaks: true });

function renderMarkdown(content: string): string {
  // AI 回复经 DOMPurify 消毒后再 v-html；LLM 输出不可信，防注入
  return DOMPurify.sanitize(marked.parse(content, { async: false }) as string);
}

const { t } = useI18n();

const sampleKeys = ["sample1", "sample2", "sample3"];

interface Round {
  question: string;
  sql?: string;
  columns?: string[];
  rows?: { cells: string[] }[];
  answer?: string;
  errorMessage?: string;
  loading: boolean;
}

const rounds = ref<Round[]>([]);
const input = ref("");
const loading = ref(false);

/** 滚到底部（新轮次追加后）。 */
function scrollToBottom() {
  nextTick(() => {
    window.scrollTo({ top: document.body.scrollHeight, behavior: "smooth" });
  });
}

/** 告警文案：后端只回结构化事实，LLM 结论走 markdown 渲染，无需模板。 */
async function handleAsk() {
  const question = input.value.trim();
  if (!question || loading.value) return;
  // 携带最近 5 轮历史（问题+SQL+结果摘要），供模型消解追问里的指代
  const history = rounds.value
    .filter((r) => !r.loading && r.sql)
    .slice(-5)
    .map((r) => ({
      question: r.question,
      sql: r.sql || "",
      resultSummary: (r.rows || []).slice(0, 3).map((row) => row.cells.join(" | ")).join("；"),
    }));
  rounds.value.push({ question, loading: true });
  input.value = "";
  loading.value = true;
  scrollToBottom();
  try {
    const locale = i18n.global.locale as unknown as { value?: string };
    const lang = locale?.value || "zh-CN";
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
      last.rows = (resp.rows || []).map((r) => ({ cells: (r.values || []) as string[] }));
      last.answer = resp.answer || "";
      last.errorMessage = resp.errorMessage || "";
      last.loading = false;
    }
    scrollToBottom();
  } catch (error: any) {
    console.error("ai query failed:", error);
    ElMessage.error(error?.message || t("pages.ai_query.failed"));
    const last = rounds.value[rounds.value.length - 1];
    if (last) last.loading = false;
  } finally {
    // 全局 loading 必须复位：否则首次提问后按钮永久 loading，且后续提问被入口守卫拦截
    loading.value = false;
  }
}
</script>

<style lang="scss" scoped>
// 颜色全部走 Element Plus 主题变量（与 ai/chat 同一约定，见 docs/design-language.md §5.2）：
// 原先写死的 tailwind 调色板（bg-blue-500 = #3B82F6 是 §3.2 已退役旧主色、bg-white /
// border-gray-200 / text-gray-400）在换主题色时不跟随，且 gray-400 (#9CA3AF) 在浅色
// 白底上实测仅 2.54:1。
.ai-query-page {
  min-height: 0;
}

.query-empty__icon {
  color: var(--gowind-primary-text);
}

.query-hint {
  color: var(--el-text-color-secondary);
}

.query-bubble {
  color: var(--el-color-white);
  background: var(--el-color-primary);
}

.query-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
}

.query-dot {
  background: var(--el-color-primary);
}

.query-sql {
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
}

.query-error {
  color: var(--gowind-danger-text);
}

// 答案卡：主色浅底用 EP 派生 token（§2.1 派生规则：light-9 由脚本生成，不手写）
.query-answer {
  background: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-7);
}

.query-composer {
  background: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color-lighter);
}

.markdown-body {
  :deep(p) {
    margin: 0 0 0.5em;
    &:last-child {
      margin-bottom: 0;
    }
  }
  :deep(table) {
    border-collapse: collapse;
    margin: 0.5em 0;
    width: 100%;
  }
  :deep(th),
  :deep(td) {
    border: 1px solid var(--el-border-color);
    padding: 4px 8px;
  }
  :deep(th) {
    background: var(--el-fill-color-light);
  }
}
</style>
