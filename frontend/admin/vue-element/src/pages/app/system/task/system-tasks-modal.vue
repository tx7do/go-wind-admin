<template>
  <ElDialog
    v-model="visible"
    :title="t('pages.task.sysTasksTitle')"
    width="860px"
    align-center
    :close-on-click-modal="false"
    @open="loadOnce"
  >
    <div v-if="loading" class="inspect-loading">
      {{ t("pages.task.sysTaskLoading") }}
    </div>
    <div v-else-if="loadError" class="inspect-error">{{ loadError }}</div>
    <template v-else-if="data">
      <!-- 调度计划 -->
      <h3 class="section-title">{{ t("pages.task.sysTaskSchedules") }}</h3>
      <div class="section-hint">
        {{ t("pages.task.sysTaskQueueHint", { queue: data.queue ?? "" }) }}
      </div>
      <ElTable :data="data.schedules ?? []" size="small" border row-key="taskType">
        <ElTableColumn
          prop="taskType"
          :label="t('pages.task.sysTaskType')"
          min-width="180"
        >
          <template #default="{ row }">
            <ElTag size="small">{{ row.taskType }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="cronSpec"
          :label="t('pages.task.sysTaskCron')"
          width="130"
        />
        <ElTableColumn
          prop="nextEnqueueAt"
          :label="t('pages.task.sysTaskNext')"
          width="170"
        />
        <ElTableColumn
          prop="prevEnqueueAt"
          :label="t('pages.task.sysTaskPrev')"
          width="170"
        >
          <template #default="{ row }">
            <span v-if="row.prevEnqueueAt">{{ row.prevEnqueueAt }}</span>
            <span v-else class="text-secondary">
              {{ t("pages.task.sysTaskNeverRun") }}
            </span>
          </template>
        </ElTableColumn>
      </ElTable>

      <!-- 运行状态 -->
      <h3 class="section-title mt">{{ t("pages.task.sysTaskStates") }}</h3>
      <ElTable :data="data.summaries ?? []" size="small" border row-key="taskType">
        <ElTableColumn
          prop="taskType"
          :label="t('pages.task.sysTaskType')"
          min-width="180"
        >
          <template #default="{ row }">
            <ElTag size="small">{{ row.taskType }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="active"
          :label="t('pages.task.sysTaskActive')"
          width="80"
          align="center"
        />
        <ElTableColumn
          prop="pending"
          :label="t('pages.task.sysTaskPending')"
          width="80"
          align="center"
        />
        <ElTableColumn
          prop="retry"
          :label="t('pages.task.sysTaskRetry')"
          width="80"
          align="center"
        >
          <template #default="{ row }">
            <ElTag v-if="row.retry > 0" size="small" type="warning">
              {{ row.retry }}
            </ElTag>
            <span v-else>{{ row.retry }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="archived"
          :label="t('pages.task.sysTaskArchived')"
          width="80"
          align="center"
        >
          <template #default="{ row }">
            <ElTag v-if="row.archived > 0" size="small" type="danger">
              {{ row.archived }}
            </ElTag>
            <span v-else>{{ row.archived }}</span>
          </template>
        </ElTableColumn>
      </ElTable>

      <!-- 最近失败 -->
      <h3 class="section-title mt">{{ t("pages.task.sysTaskFailures") }}</h3>
      <div v-if="failures.length === 0" class="section-hint">
        {{ t("pages.task.sysTaskNoFailures") }}
      </div>
      <ElTable v-else :data="failures" size="small" border>
        <ElTableColumn
          prop="taskType"
          :label="t('pages.task.sysTaskType')"
          width="170"
        />
        <ElTableColumn
          prop="state"
          :label="t('pages.task.sysTaskState')"
          width="90"
        >
          <template #default="{ row }">
            <ElTag size="small" :type="row.state === 'archived' ? 'danger' : 'warning'">
              {{ row.state }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="lastError"
          :label="t('pages.task.sysTaskLastError')"
          min-width="220"
          show-overflow-tooltip
        />
        <ElTableColumn
          prop="lastFailedAt"
          :label="t('pages.task.sysTaskFailedAt')"
          width="170"
        />
        <ElTableColumn :label="t('pages.task.sysTaskRetried')" width="90" align="center">
          <template #default="{ row }">
            {{ row.retried ?? 0 }}/{{ row.maxRetry ?? 0 }}
          </template>
        </ElTableColumn>
      </ElTable>
    </template>
    <template #footer>
      <ElButton @click="visible = false">{{ $t("common.button.cancel") }}</ElButton>
      <ElButton type="primary" :loading="loading" @click="loadOnce">
        {{ t("pages.task.sysTaskRefresh") }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { ElButton, ElDialog, ElMessage, ElTable, ElTableColumn, ElTag } from "element-plus";

import { fetchInspectSystemTasks } from "@/api/composables";
import { useI18n } from "@go-wind/admin-element";

/**
 * 系统级常驻任务弹窗（方案 C'：只读 asynq Inspector，零新表）。
 * 数据来自 asynq 本身：调度计划（cron + 上次/下次入队，上次为空 =
 * 本次运行会话内从未触发）+ 各任务类型的队列状态与最近失败明细。
 */

const { t } = useI18n();

const visible = ref(false);
const loading = ref(false);
const loadError = ref("");
const data = ref<Record<string, any> | null>(null);

const failures = computed(() => {
  const summaries = (data.value?.summaries ?? []) as any[];
  return summaries.flatMap((s) => s.recentFailures ?? []);
});

async function loadOnce() {
  loading.value = true;
  loadError.value = "";
  try {
    data.value = (await fetchInspectSystemTasks()) as any;
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error("inspect system tasks failed", error);
    loadError.value = error?.message || t("pages.task.sysTaskLoadFailed");
    ElMessage.error(error?.message || t("pages.task.sysTaskLoadFailed"));
  } finally {
    loading.value = false;
  }
}

function open() {
  visible.value = true;
}

defineExpose({ open });
</script>

<style lang="scss" scoped>
.section-title {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 600;

  &.mt {
    margin-top: 20px;
  }
}

.section-hint {
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.inspect-loading {
  padding: 48px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}

.inspect-error {
  padding: 24px 0;
  color: var(--el-color-danger);
}

.text-secondary {
  color: var(--el-text-color-secondary);
}
</style>
