<script lang="ts" setup>
import type { TableColumnsType } from 'ant-design-vue';

import { computed, ref } from 'vue';

import { formatDateTime } from '@vben/utils';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message, Popconfirm, Tag } from 'ant-design-vue';

import {
  useListMyOnlineSessions,
  useRevokeMyOnlineSession,
} from '#/api';
import type {
  online_sessionservicev1_OnlineSession as OnlineSession,
} from '#/api/generated/admin/service/v1';

const { data, isLoading } = useListMyOnlineSessions();
const { mutateAsync: revoke } = useRevokeMyOnlineSession();

const submitting = ref(false);

// 通过 computed 派生，保证 useQuery 数据返回后视图响应式更新
const items = computed(() => (data.value?.items ?? []) as OnlineSession[]);

const columns: TableColumnsType = [
  { title: $t('page.onlineSession.clientType'), dataIndex: 'clientType', width: 110 },
  { title: $t('page.onlineSession.ipAddress'), dataIndex: 'ipAddress', width: 150 },
  { title: $t('page.onlineSession.userAgent'), dataIndex: 'userAgent', ellipsis: true },
  { title: $t('page.onlineSession.loginAt'), dataIndex: 'loginAt', width: 180 },
  { title: $t('page.onlineSession.status'), key: 'status', width: 150 },
];

async function handleRevoke(row: OnlineSession) {
  if (!row.jti) return;
  submitting.value = true;
  try {
    await revoke({ clientType: row.clientType, jti: row.jti });
    message.success($t('page.onlineSession.revokeSuccess'));
  } catch (error: any) {
    message.error(error?.message || $t('page.onlineSession.revokeFailed'));
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <Page auto-content-height>
    <div class="my-sessions-page">
      <div class="page-hint">{{ $t('page.onlineSession.mySessionsHint') }}</div>
      <a-table
        :columns="columns"
        :data-source="items"
        :loading="isLoading"
        :pagination="false"
        row-key="jti"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'clientType'">
            <a-tag :color="record.clientType === 'app' ? 'purple' : 'blue'">
              {{
                record.clientType === 'app'
                  ? $t('page.onlineSession.clientApp')
                  : $t('page.onlineSession.clientAdmin')
              }}
            </a-tag>
          </template>
          <template v-else-if="column.dataIndex === 'ipAddress'">
            {{ record.ipAddress || '-' }}
          </template>
          <template v-else-if="column.dataIndex === 'userAgent'">
            {{ record.userAgent || '-' }}
          </template>
          <template v-else-if="column.dataIndex === 'loginAt'">
            {{ formatDateTime(record.loginAt ?? '') }}
          </template>
          <template v-else-if="column.key === 'status'">
            <Tag v-if="record.current" color="success">
              {{ $t('page.onlineSession.currentSession') }}
            </Tag>
            <Popconfirm
              :cancel-text="$t('ui.button.cancel')"
              :ok-text="$t('ui.button.ok')"
              :title="$t('page.onlineSession.revokeConfirm')"
              @confirm="handleRevoke(record)"
            >
              <a-button danger type="link" size="small" :loading="submitting">
                {{ $t('page.onlineSession.revoke') }}
              </a-button>
            </Popconfirm>
          </template>
        </template>
        <template #emptyText>
          {{ $t('page.onlineSession.noSessions') }}
        </template>
      </a-table>
    </div>
  </Page>
</template>

<style scoped>
.my-sessions-page {
  width: 100%;
}

.page-hint {
  margin-bottom: 12px;
  font-size: 12px;
  opacity: 0.65;
}
</style>
