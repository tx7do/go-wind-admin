<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, type VbenFormProps } from '@vben/common-ui';
import { LucideLogOut } from '@vben/icons';

import { notification, type TagProps } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import TableExportButton from '#/components/TableExportButton.vue';
import {
  fetchListOnlineSessions,
  useForceLogoutSession,
} from '#/api';
import type {
  online_sessionservicev1_OnlineSession as OnlineSession,
} from '#/api/generated/admin/service/v1';
import { $t } from '#/locales';

const { mutateAsync: forceLogout } = useForceLogoutSession();

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'keyword',
      label: $t('page.onlineSession.keyword'),
      componentProps: {
        allowClear: true,
        placeholder: $t('page.onlineSession.keyword'),
      },
    },
  ],
};

const gridOptions: VxeGridProps<OnlineSession> = {
  toolbarConfig: {
    custom: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  pagerConfig: {},
  rowConfig: {
    isHover: true,
    keyField: 'jti',
  },
  stripe: true,

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        const values = (formValues ?? {}) as Record<string, any>;
        return await fetchListOnlineSessions({
          page: page.currentPage,
          pageSize: page.pageSize,
          keyword: values.keyword || undefined,
        });
      },
    },
  },

  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.onlineSession.username'), field: 'username', minWidth: 120 },
    {
      title: $t('page.onlineSession.tenant'),
      field: 'tenantId',
      width: 90,
      slots: { default: 'tenant' },
    },
    {
      title: $t('page.onlineSession.clientType'),
      field: 'clientType',
      width: 100,
      slots: { default: 'clientType' },
    },
    { title: $t('page.onlineSession.ipAddress'), field: 'ipAddress', minWidth: 130 },
    {
      title: $t('page.onlineSession.userAgent'),
      field: 'userAgent',
      minWidth: 220,
      showOverflow: true,
    },
    {
      title: $t('page.onlineSession.deviceId'),
      field: 'deviceId',
      width: 120,
      formatter: ({ cellValue }) => cellValue || '-',
    },
    {
      title: $t('page.onlineSession.loginAt'),
      field: 'loginAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 130,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  fetchListOnlineSessions({ page, pageSize });

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

function clientTagColor(clientType?: string): TagProps['color'] {
  return clientType === 'app' ? 'purple' : 'blue';
}

/* 强制下线：确认后吊销该会话的访问/刷新令牌 */
async function handleForceLogout(row: OnlineSession) {
  if (!row.jti || row.userId === undefined) return;

  try {
    await forceLogout({
      clientType: row.clientType,
      userId: row.userId,
      jti: row.jti,
    });
    notification.success({
      message: $t('page.onlineSession.forceLogoutSuccess'),
    });
    await gridApi.reload();
  } catch (error: any) {
    notification.error({
      message: error?.message || $t('page.onlineSession.forceLogoutFailed'),
    });
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.onlineSession.moduleName')">
      <template #toolbar-tools>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="export" />
      </template>

      <template #tenant="{ row }">
        <a-tag v-if="(row.tenantId ?? 0) === 0">
          {{ $t('page.onlineSession.platform') }}
        </a-tag>
        <span v-else>#{{ row.tenantId }}</span>
      </template>
      <template #clientType="{ row }">
        <a-tag :color="clientTagColor(row.clientType)">
          {{
            row.clientType === 'app'
              ? $t('page.onlineSession.clientApp')
              : $t('page.onlineSession.clientAdmin')
          }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('page.onlineSession.forceLogoutConfirmDesc', {
              user: row.username || String(row.userId),
            })
          "
          @confirm="handleForceLogout(row)"
        >
          <!-- Tailwind Preflight 把 svg 设为 display:block，块级 icon 必然独占一行，
               nowrap 拦不住，必须把按钮内 svg 收回行内并与文本垂直居中 -->
          <a-button
            danger
            type="link"
            size="small"
            class="whitespace-nowrap [&>svg]:inline-block [&>svg]:align-middle"
            :icon="h(LucideLogOut)"
          >
            {{ $t('page.onlineSession.forceLogout') }}
          </a-button>
        </a-popconfirm>
      </template>
    </Grid>
  </Page>
</template>
