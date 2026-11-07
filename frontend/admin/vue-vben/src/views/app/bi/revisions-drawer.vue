<script lang="ts" setup>
import { computed, ref, watch } from 'vue';

import { $t } from '@vben/locales';

import { Button, message, Modal, Table, Tag } from 'ant-design-vue';
import { useQueryClient } from '@tanstack/vue-query';

import type { biservicev1_BiDashboardRevision } from '#/api/generated/admin/service/v1';
import {
  useBiDashboardRevisions,
  useRestoreBiDashboardRevision,
} from '#/api/composables/bi_dashboard';

/**
 * 版本历史弹窗：每次保存留旧版快照（新快照在前），恢复 = 快照写回主表。
 * 恢复前服务端会先以 RESTORE 来源留底当前版本——误恢复可再恢复回来（可逆）。
 */

const visible = ref(false);
const dashboardId = ref<number | undefined>(undefined);
const queryClient = useQueryClient();

const revisions = useBiDashboardRevisions(
  () => dashboardId.value,
  () => visible.value && !!dashboardId.value,
);
const loading = computed(() => revisions.isLoading.value);
const rows = computed<biservicev1_BiDashboardRevision[]>(
  () => (revisions.data.value?.items as any) || [],
);

const { mutateAsync: restoreRevision } = useRestoreBiDashboardRevision();

const columns = computed(() => [
  {
    title: $t('page.bi.revisionVersion'),
    dataIndex: 'version',
    width: 80,
  },
  { title: $t('page.bi.name'), dataIndex: 'name', ellipsis: true },
  { title: $t('page.bi.revisionSource'), dataIndex: 'source', width: 100 },
  { title: $t('common.table.createdAt'), dataIndex: 'createdAt', width: 160 },
  { title: $t('page.bi.action'), width: 90 },
]);

async function handleRestore(row: biservicev1_BiDashboardRevision) {
  try {
    await restoreRevision({ id: row.id });
    message.success($t('page.bi.revisionRestoreSuccess'));
    // 恢复会改主表（version+1）：刷新快照列表与列表页
    await queryClient.invalidateQueries({ queryKey: ['biDashboardRevisions', dashboardId.value] });
    await queryClient.invalidateQueries({ queryKey: ['listBiDashboards'] });
  } catch (e) {
    // 错误提示由请求层统一弹出；此处带出原始错误对象防静默吞错
    console.error('restore bi dashboard revision failed:', e);
  }
}

watch(visible, (v) => {
  if (v && dashboardId.value) {
    queryClient.invalidateQueries({ queryKey: ['biDashboardRevisions', dashboardId.value] });
  }
});

function open(id: number) {
  dashboardId.value = id;
  visible.value = true;
}

defineExpose({ open });
</script>

<template>
  <Modal
    v-model:open="visible"
    :title="$t('page.bi.revisionTitle')"
    :footer="null"
    :width="720"
  >
    <Table
      :columns="columns as any"
      :data-source="rows"
      :loading="loading"
      row-key="id"
      size="small"
      :pagination="false"
      :scroll="{ y: 420 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'version'">
          <Tag color="blue">v{{ record.version }}</Tag>
        </template>
        <template v-else-if="column.dataIndex === 'source'">
          <Tag v-if="record.source === 'RESTORE'" color="orange">
            {{ $t('page.bi.revisionSourceRestore') }}
          </Tag>
          <Tag v-else>{{ $t('page.bi.revisionSourceSave') }}</Tag>
        </template>
        <template v-else-if="column.key === 'action'">
          <Button type="link" size="small" @click="handleRestore(record as biservicev1_BiDashboardRevision)">
            {{ $t('page.bi.revisionRestore') }}
          </Button>
        </template>
      </template>
    </Table>
  </Modal>
</template>
