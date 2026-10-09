<script lang="ts" setup>
import { h, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { fetchListScriptLogs, usePurgeScriptLogs } from '#/api';
import { $t } from '#/locales';

const { mutateAsync: purgeLogs } = usePurgeScriptLogs();

const logs = ref<any[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 20;
const loading = ref(false);

const columns = [
  { title: $t('page.script.logScriptName'), field: 'scriptName', minWidth: 150 },
  { title: $t('page.script.language'), field: 'language', width: 90 },
  { title: $t('page.script.logTrigger'), field: 'triggerType', width: 100 },
  { title: $t('page.script.hookPoint'), field: 'hookPoint', minWidth: 130 },
  {
    title: $t('page.script.logSuccess'),
    field: 'success',
    width: 90,
    slots: { default: 'success' },
  },
  { title: $t('page.script.testRunDuration'), field: 'durationMs', width: 90 },
  { title: $t('page.script.logError'), field: 'error', minWidth: 180 },
  {
    title: $t('page.script.logCreatedAt'),
    field: 'createdAt',
    width: 160,
    formatter: 'formatDateTime',
  },
];

async function loadLogs() {
  loading.value = true;
  try {
    const result = await fetchListScriptLogs({
      page: page.value,
      pageSize,
      orderBy: JSON.stringify(['-created_at']),
    });
    logs.value = result.items ?? [];
    total.value = Number(result.total ?? 0);
  } catch (error) {
    console.error('加载脚本执行日志失败', error);
    notification.error({ message: $t('page.script.fetchFailed') });
  } finally {
    loading.value = false;
  }
}

async function handlePurge() {
  try {
    const before = new Date(Date.now() - 90 * 24 * 3600 * 1000).toISOString();
    const resp = await purgeLogs({ before });
    notification.success({
      message: $t('page.script.logPurgeSuccess', { count: Number(resp?.deleted ?? 0) }),
    });
    await loadLogs();
  } catch (error) {
    console.error('清理脚本执行日志失败', error);
    notification.error({ message: $t('page.script.fetchFailed') });
  }
}

const [Drawer] = useVbenDrawer({
  title: $t('page.script.logTitle'),
  class: 'w-[960px]',
  footer: false,
  onOpenChange(isOpen) {
    if (isOpen) {
      loadLogs();
    }
  },
});
</script>

<template>
  <Drawer>
    <div class="mb-2">
      <a-popconfirm
        :cancel-text="$t('ui.button.cancel')"
        :ok-text="$t('ui.button.ok')"
        :title="$t('page.script.logPurgeConfirmDesc')"
        @confirm="handlePurge"
      >
        <a-button danger :icon="h(LucideTrash2)">
          {{ $t('page.script.logPurge') }}
        </a-button>
      </a-popconfirm>
    </div>
    <a-table
      :columns="columns"
      :data-source="logs"
      :loading="loading"
      :pagination="{
        current: page,
        pageSize,
        total,
        showSizeChanger: false,
        onChange: (p: number) => {
          page = p;
          loadLogs();
        },
      }"
      row-key="id"
      size="small"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.field === 'success'">
          <a-tag :color="record.success ? 'success' : 'error'">
            {{
              record.success
                ? $t('page.script.testRunSuccess')
                : $t('page.script.testRunFailed')
            }}
          </a-tag>
        </template>
        <template v-else-if="column.field === 'triggerType'">
          <a-tag>{{ record.triggerType }}</a-tag>
        </template>
      </template>
    </a-table>
  </Drawer>
</template>
