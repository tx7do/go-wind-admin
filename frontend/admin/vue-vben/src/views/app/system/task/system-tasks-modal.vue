<script lang="ts" setup>
import { computed, watch } from 'vue';

import { $t } from '@vben/locales';

import {
  message,
  Modal,
  Table,
  Tag,
} from 'ant-design-vue';

import { useInspectSystemTasks } from '#/api';

/**
 * 系统级常驻任务弹窗（方案 C'：只读 asynq Inspector，零新表）。
 * 数据来自 asynq 本身：调度计划（cron + 上次/下次入队，上次为空 =
 * 本次运行会话内从未触发）+ 各任务类型的队列状态与最近失败明细。
 */

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ 'update:open': [value: boolean] }>();

const { mutateAsync: inspect, data: inspectData, isPending } =
  useInspectSystemTasks();

const visible = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value),
});

// 打开即拉取一次
watch(
  () => props.open,
  (open) => {
    if (open && !inspectData.value) {
      handleInspect();
    }
  },
);

const scheduleColumns = ([
  {
    title: $t('page.task.sysTaskType'),
    dataIndex: 'taskType',
    key: 'taskType',
    minWidth: 180,
  },
  { title: $t('page.task.sysTaskCron'), dataIndex: 'cronSpec', width: 130 },
  { title: $t('page.task.sysTaskNext'), dataIndex: 'nextEnqueueAt', width: 170 },
  { title: $t('page.task.sysTaskPrev'), dataIndex: 'prevEnqueueAt', width: 170 },
]) as any[];

const summaryColumns = ([
  {
    title: $t('page.task.sysTaskType'),
    dataIndex: 'taskType',
    key: 'taskType',
    minWidth: 180,
  },
  { title: $t('page.task.sysTaskActive'), dataIndex: 'active', width: 80, align: 'center' },
  { title: $t('page.task.sysTaskPending'), dataIndex: 'pending', width: 80, align: 'center' },
  { title: $t('page.task.sysTaskRetry'), dataIndex: 'retry', key: 'retry', width: 80, align: 'center' },
  { title: $t('page.task.sysTaskArchived'), dataIndex: 'archived', key: 'archived', width: 90, align: 'center' },
]) as any[];

const failureColumns = ([
  { title: $t('page.task.sysTaskType'), dataIndex: 'taskType', width: 170 },
  { title: $t('page.task.sysTaskState'), dataIndex: 'state', key: 'state', width: 90 },
  { title: $t('page.task.sysTaskLastError'), dataIndex: 'lastError', ellipsis: true },
  { title: $t('page.task.sysTaskFailedAt'), dataIndex: 'lastFailedAt', width: 170 },
  { title: $t('page.task.sysTaskRetried'), key: 'retried', width: 90, align: 'center' },
]) as any[];

const schedules = computed(() => (inspectData.value?.schedules ?? []) as any[]);
const summaries = computed(() => (inspectData.value?.summaries ?? []) as any[]);
const failures = computed(() =>
  summaries.value.flatMap((s) => s.recentFailures ?? []),
);

async function handleInspect() {
  try {
    await inspect({});
  } catch (error: any) {
    // 原始错误必须留在控制台：用户可见的那句翻译不包含服务端的原因
    console.error('[task-monitor] inspect failed', error);
    message.error(error?.message || $t('page.task.sysTaskLoadFailed'));
  }
}

function formatTs(value: any) {
  return value ? String(value).replace('T', ' ').slice(0, 19) : '-';
}
</script>

<template>
  <Modal
    :open="visible"
    :title="$t('page.task.sysTasksTitle')"
    :confirm-loading="isPending"
    :footer="null"
    width="880px"
    destroy-on-close
    @update:open="visible = $event"
    @ok="handleInspect"
  >
    <h3 class="section-title">{{ $t('page.task.sysTaskSchedules') }}</h3>
    <div class="section-hint">
      {{
        $t('page.task.sysTaskQueueHint', {
          queue: inspectData?.queue ?? '',
        })
      }}
    </div>
    <Table
      :columns="scheduleColumns"
      :data-source="schedules"
      :loading="isPending"
      :pagination="false"
      row-key="taskType"
      size="small"
      bordered
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'taskType'">
          <Tag color="geekblue">{{ record.taskType }}</Tag>
        </template>
        <template v-else-if="column.dataIndex === 'prevEnqueueAt'">
          <span v-if="record.prevEnqueueAt">{{ formatTs(record.prevEnqueueAt) }}</span>
          <span v-else class="text-secondary">
            {{ $t('page.task.sysTaskNeverRun') }}
          </span>
        </template>
        <template v-else-if="column.dataIndex === 'nextEnqueueAt'">
          {{ formatTs(record.nextEnqueueAt) }}
        </template>
      </template>
    </Table>

    <h3 class="section-title mt">{{ $t('page.task.sysTaskStates') }}</h3>
    <Table
      :columns="summaryColumns"
      :data-source="summaries"
      :loading="isPending"
      :pagination="false"
      row-key="taskType"
      size="small"
      bordered
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'taskType'">
          <Tag color="geekblue">{{ record.taskType }}</Tag>
        </template>
        <template v-else-if="column.key === 'retry'">
          <Tag v-if="record.retry > 0" color="warning">{{ record.retry }}</Tag>
          <span v-else>{{ record.retry }}</span>
        </template>
        <template v-else-if="column.key === 'archived'">
          <Tag v-if="record.archived > 0" color="error">{{ record.archived }}</Tag>
          <span v-else>{{ record.archived }}</span>
        </template>
      </template>
    </Table>

    <h3 class="section-title mt">{{ $t('page.task.sysTaskFailures') }}</h3>
    <div v-if="failures.length === 0" class="section-hint">
      {{ $t('page.task.sysTaskNoFailures') }}
    </div>
    <Table
      v-else
      :columns="failureColumns"
      :data-source="failures"
      :pagination="false"
      row-key="lastFailedAt"
      size="small"
      bordered
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'state'">
          <Tag :color="record.state === 'archived' ? 'error' : 'warning'">
            {{ record.state }}
          </Tag>
        </template>
        <template v-else-if="column.key === 'retried'">
          {{ record.retried ?? 0 }}/{{ record.maxRetry ?? 0 }}
        </template>
        <template v-else-if="column.dataIndex === 'lastFailedAt'">
          {{ formatTs(record.lastFailedAt) }}
        </template>
      </template>
    </Table>
  </Modal>
</template>

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
  opacity: 0.7;
}

.text-secondary {
  opacity: 0.7;
}
</style>
