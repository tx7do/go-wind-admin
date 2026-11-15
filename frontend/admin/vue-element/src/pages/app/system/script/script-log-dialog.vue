<template>
  <ElDrawer
    v-model="visible"
    :title="$t('pages.script.viewLogTitle')"
    size="960px"
    :close-on-click-modal="false"
    append-to-body
  >
    <div class="flex h-full flex-col gap-2">
      <div class="flex items-center gap-2">
        <ElSelect
          v-model="filterTrigger"
          :placeholder="$t('pages.script.logTrigger')"
          clearable
          style="width: 140px"
        >
          <ElOption label="hook" value="hook" />
          <ElOption label="task" value="task" />
          <ElOption label="test_run" value="test_run" />
        </ElSelect>
        <ElSelect
          v-model="filterSuccess"
          :placeholder="$t('pages.script.logSuccess')"
          clearable
          style="width: 140px"
        >
          <ElOption :label="$t('pages.script.testRunSuccess')" value="true" />
          <ElOption :label="$t('pages.script.testRunFailed')" value="false" />
        </ElSelect>
        <ElButton :loading="loading" @click="loadLogs(1)">
          {{ $t("common.button.search") }}
        </ElButton>
        <ElPopconfirm
          :title="$t('pages.script.logPurgeConfirmDesc')"
          @confirm="handlePurge"
        >
          <template #reference>
            <ElButton type="danger" plain :loading="purging">
              {{ $t("pages.script.logPurge") }}
            </ElButton>
          </template>
        </ElPopconfirm>
      </div>

      <div class="min-h-0 flex-1">
        <ElTable
          :data="logs"
          size="small"
          border
          stripe
          height="100%"
        >
          <ElTableColumn prop="scriptName" :label="$t('pages.script.logScriptName')" min-width="150" show-overflow-tooltip />
          <ElTableColumn prop="language" :label="$t('pages.script.language')" width="90" />
          <ElTableColumn prop="triggerType" :label="$t('pages.script.logTrigger')" width="100">
            <template #default="{ row }">
              <ElTag size="small">{{ row.triggerType }}</ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="hookPoint" :label="$t('pages.script.hookPoint')" min-width="130" show-overflow-tooltip />
          <ElTableColumn :label="$t('pages.script.logSuccess')" width="90">
            <template #default="{ row }">
              <ElTag size="small" :type="row.success ? 'success' : 'danger'">
                {{ row.success ? $t("pages.script.testRunSuccess") : $t("pages.script.testRunFailed") }}
              </ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn :label="$t('pages.script.testRunDuration')" width="90">
            <template #default="{ row }">{{ row.durationMs ?? 0 }}ms</template>
          </ElTableColumn>
          <ElTableColumn prop="error" :label="$t('pages.script.logError')" min-width="180" show-overflow-tooltip />
          <ElTableColumn :label="$t('pages.script.logCreatedAt')" width="170">
            <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
          </ElTableColumn>
        </ElTable>
      </div>

      <ElPagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="loadLogs"
      />
    </div>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import {
  ElButton,
  ElDrawer,
  ElMessage,
  ElOption,
  ElPagination,
  ElPopconfirm,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
} from "element-plus";

import { fetchListScriptLogs } from "@/api/composables";
import { usePurgeScriptLog } from "@/api/composables";
import { $t } from "@go-wind/admin-element";

const visible = ref(false);
const loading = ref(false);
const purging = ref(false);
const logs = ref<any[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 20;
const filterTrigger = ref<string>("");
const filterSuccess = ref<string>("");

const { mutateAsync: purgeLogs } = usePurgeScriptLog();

function formatTime(value: any): string {
  if (!value) return "-";
  const d = new Date(value as unknown as string);
  return Number.isNaN(d.getTime()) ? String(value) : d.toLocaleString();
}

async function loadLogs(targetPage?: number) {
  if (targetPage) page.value = targetPage;
  loading.value = true;
  try {
    const formValues: Record<string, any> = {};
    if (filterTrigger.value) formValues.triggerType = filterTrigger.value;
    if (filterSuccess.value) formValues.success = filterSuccess.value;

    // fetchListScriptLogs 接收 PaginationQuery：以 raw 参数构造等价查询
    const result = await fetchListScriptLogs({
      toRawParams: () => ({
        page: page.value,
        pageSize,
        query: JSON.stringify(formValues),
      }),
    } as any);
    logs.value = result.items || [];
    total.value = Number(result.total || 0);
  } catch (error) {
    console.error("加载脚本执行日志失败", error);
    ElMessage.error($t("pages.script.fetchFailed"));
  } finally {
    loading.value = false;
  }
}

async function handlePurge() {
  purging.value = true;
  try {
    const before = new Date(Date.now() - 90 * 24 * 3600 * 1000).toISOString();
    const resp = await purgeLogs({ before });
    ElMessage.success($t("pages.script.logPurgeSuccess", { count: Number(resp?.deleted ?? 0) }));
    await loadLogs(1);
  } catch (error) {
    console.error("清理脚本执行日志失败", error);
    ElMessage.error($t("pages.script.fetchFailed"));
  } finally {
    purging.value = false;
  }
}

function open() {
  visible.value = true;
  loadLogs(1);
}

defineExpose({ open });
</script>
