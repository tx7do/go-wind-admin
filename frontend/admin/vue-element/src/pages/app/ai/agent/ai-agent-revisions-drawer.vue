<template>
  <ProModal
    v-model:visible="visible"
    :title="revisionsTitle"
    :loading="loading"
    :config="{
      component: 'drawer',
      drawer: { size: 640, closeOnClickModal: false },
    }"
  >
    <div v-if="revisions.length === 0 && !loading" class="field-tip">
      {{ t("pages.ai_agent.revisions.empty") }}
    </div>

    <div v-for="rev in revisions" :key="rev.id" class="revision-card">
      <div class="revision-meta">
        <ElTag v-if="rev.source === 'RESTORE'" type="primary" size="small">
          {{ t("pages.ai_agent.revisions.sourceRestore") }}
        </ElTag>
        <ElTag v-else size="small" type="info" effect="plain">
          {{ t("pages.ai_agent.revisions.sourceSave") }}
        </ElTag>
        <span class="revision-id">#{{ rev.id }}</span>
        <span class="revision-time">{{ rev.createdAt }}</span>
        <span class="revision-actions">
          <ElButton text size="small" @click="expandedId = expandedId === rev.id ? undefined : rev.id">
            {{ expandedId === rev.id ? t("pages.ai_agent.revisions.collapse") : t("pages.ai_agent.revisions.viewGraph") }}
          </ElButton>
          <ElPopconfirm
            :title="t('pages.ai_agent.revisions.restoreConfirm')"
            @confirm="handleRestore(rev.id)"
          >
            <template #reference>
              <ElButton size="small" type="warning" plain :icon="refreshIcon">
                {{ t("pages.ai_agent.revisions.restore") }}
              </ElButton>
            </template>
          </ElPopconfirm>
        </span>
      </div>
      <pre v-if="expandedId === rev.id" class="revision-graph">{{ rev.graph }}</pre>
    </div>

    <div v-if="total > 0" class="field-tip" style="margin-top: 8px">
      {{ t("pages.ai_agent.revisions.total", { total }) }}
    </div>
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElButton, ElMessage, ElPopconfirm, ElTag } from "element-plus";
import { RefreshLeft } from "@element-plus/icons-vue";

import ProModal from "@/components/Pro/ProModal/index.vue";
import { useI18n } from "@/core/i18n";
import { listAiAgentRevisions, restoreAiAgentRevision } from "@/api/composables";

const { t } = useI18n();

const visible = ref(false);

const loading = ref(false);
const app = ref<any>();
const revisions = ref<any[]>([]);
const total = ref(0);
const expandedId = ref<string | undefined>();

const refreshIcon = RefreshLeft;

const revisionsTitle = computed(
  () => `${t("pages.ai_agent.openRevisions")} · ${app.value?.name || ""}`,
);

function open(row: any) {
  app.value = row;
  visible.value = true;
  loading.value = true;
  expandedId.value = undefined;
  listAiAgentRevisions(row.id, { page: 1, pageSize: 50 })
    .then((resp: any) => {
      revisions.value = resp.items ?? [];
      total.value = resp.total ?? 0;
    })
    .catch((error: any) => {
      console.error("list agent revisions failed:", error);
    })
    .finally(() => {
      loading.value = false;
    });
}

function handleRestore(id: string) {
  restoreAiAgentRevision(id)
    .then(() => {
      ElMessage.success(t("pages.ai_agent.revisions.restored"));
      // 重拉快照列表：恢复动作本身会以 RESTORE 来源留底当前版本
      if (app.value) open(app.value);
    })
    .catch((error: any) => {
      console.error("restore agent revision failed:", error);
      ElMessage.error(error.message || t("pages.ai_agent.revisions.restoreFailed"));
    });
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.field-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.revision-card {
  margin-bottom: 12px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.revision-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.revision-id,
.revision-time {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.revision-actions {
  display: flex;
  gap: 4px;
  margin-left: auto;
  align-items: center;
}

.revision-graph {
  margin: 8px 0 0;
  padding: 8px;
  max-height: 240px;
  overflow: auto;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  font-family: monospace;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
