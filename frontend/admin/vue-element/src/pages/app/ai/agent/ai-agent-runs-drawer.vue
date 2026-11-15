<template>
  <ProModal
    v-model:visible="visible"
    :title="runsTitle"
    :loading="loading"
    :config="{
      component: 'drawer',
      drawer: { size: 640, closeOnClickModal: false },
    }"
  >
    <div v-if="runs.length === 0 && !loading" class="field-tip">
      {{ t("pages.ai_agent.runs.empty") }}
    </div>

    <div v-for="run in runs" :key="run.id" class="run-card">
      <div class="run-meta">
        <ElTag v-if="run.status === 'SUCCEEDED'" type="success" size="small">
          {{ t("pages.ai_agent.runs.succeeded") }}
        </ElTag>
        <ElTag v-else-if="run.status === 'FAILED'" type="danger" size="small">
          {{ t("pages.ai_agent.runs.failed") }}
        </ElTag>
        <ElTag v-else size="small">{{ t("pages.ai_agent.runs.running") }}</ElTag>
        <span class="run-id">#{{ run.id }}</span>
        <ElTag v-if="(run.totalTokens ?? 0) > 0" size="small">
          tokens: {{ run.totalTokens }}
        </ElTag>
        <ElTag v-if="run.durationMs != null" size="small">{{ run.durationMs }}ms</ElTag>
      </div>
      <div v-if="run.output" class="run-output">{{ run.output }}</div>
      <div v-if="run.error" class="run-error">{{ run.error }}</div>
      <ElTimeline v-if="parseTrace(run.trace).length > 0" style="margin-top: 10px">
        <ElTimelineItem
          v-for="(s, i) in parseTrace(run.trace)"
          :key="i"
          :type="s.status === 'ok' ? 'success' : 'danger'"
        >
          <div class="trace-step">
            <code>{{ s.nodeId }}</code>
            <ElTag size="small">{{ s.type }}</ElTag>
            <span>{{ s.ms }}ms</span>
          </div>
        </ElTimelineItem>
      </ElTimeline>
    </div>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElTag, ElTimeline, ElTimelineItem } from "element-plus";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { useI18n } from "@go-wind/admin-element";
import { PaginationQuery } from "@go-wind/admin-core";
import { fetchListAiAgentRuns } from "@/api/composables";

const { t } = useI18n();

const visible = ref(false);

const loading = ref(false);
const app = ref<any>();
const runs = ref<any[]>([]);

const runsTitle = computed(() => `${t("pages.ai_agent.runsTitle")} · ${app.value?.name || ""}`);

function parseTrace(raw?: string): Array<Record<string, any>> {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function open(row: any) {
  app.value = row;
  visible.value = true;
  loading.value = true;
  fetchListAiAgentRuns(new PaginationQuery({ paging: { page: 1, pageSize: 20 }, formValues: { appId: String(row.id) } }))
    .then((resp) => {
      runs.value = (resp as any).items ?? [];
    })
    .catch((error: any) => {
      console.error("list agent runs failed:", error);
    })
    .finally(() => {
      loading.value = false;
    });
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.field-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.run-card {
  margin-bottom: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}

.run-meta {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.run-id {
  color: var(--el-text-color-secondary);
}

.run-output {
  margin-top: 6px;
  font-size: 12px;
  white-space: pre-wrap;
  max-height: 72px;
  overflow: hidden;
}

.run-error {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-color-danger);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.trace-step {
  font-size: 12px;
  display: flex;
  gap: 6px;
  align-items: center;
}
</style>
