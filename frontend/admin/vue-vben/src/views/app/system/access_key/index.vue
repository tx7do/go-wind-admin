<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideRefreshCw, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListAccessKeys,
  PaginationQuery,
  useDeleteAccessKey,
  useResetAccessKeySecret,
} from '#/api';
import { type access_keyservicev1_AccessKey as AccessKey } from '#/api';
import TableExportButton from '#/components/TableExportButton.vue';
import { $t } from '#/locales';

import AccessKeyDrawer from './access-key-drawer.vue';

const { mutateAsync: deleteAccessKey } = useDeleteAccessKey();

function copyResetSecret() {
  navigator.clipboard?.writeText(resetSecretValue.value).catch((err: unknown) => {
    console.error('copy secret failed:', err);
  });
}
const resetMutation = useResetAccessKeySecret();
const resetSecretVisible = ref(false);
const resetSecretValue = ref('');
const resettingId = ref(0);

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.accessKey.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps = {
  columns: [
    { title: $t('page.accessKey.name'), field: 'name', minWidth: 160 },
    {
      title: $t('page.accessKey.accessKeyLabel'),
      field: 'accessKey',
      minWidth: 220,
    },
    {
      title: $t('page.accessKey.status'),
      field: 'status',
      slots: { default: 'status' },
      width: 95,
    },
    {
      title: $t('page.accessKey.expiresAt'),
      field: 'expiresAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    {
      title: $t('page.accessKey.lastUsedAt'),
      field: 'lastUsedAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
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
  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        return await fetchListAccessKeys(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: AccessKeyDrawer,
  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.reload();
    }
  },
});

function openDrawer(create: boolean, row?: any) {
  drawerApi.setData({ create, row });
  drawerApi.open();
}

function handleCreate() {
  openDrawer(true);
}

function handleEdit(row: any) {
  openDrawer(false, row);
}

function handleResetSecret(row: AccessKey) {
  if (!row.id) return;
  resettingId.value = row.id;
  resetMutation.mutate(
    { id: row.id },
    {
      onSuccess: (resp) => {
        notification.success({ message: $t('page.accessKey.resetSuccess') });
        resetSecretValue.value = resp.secret ?? '';
        resetSecretVisible.value = true;
        gridApi.reload();
      },
      onError: () => {
        notification.error({ message: $t('page.accessKey.resetFailed') });
      },
    },
  );
}

function handleDelete(row: AccessKey) {
  if (!row.id) return;
  deleteAccessKey({ id: row.id }).then(
    () => {
      notification.success({
        message: $t('ui.notification.delete_success'),
      });
      gridApi.reload();
    },
    () => {
      notification.error({
        message: $t('ui.notification.delete_failed'),
      });
    },
  );
}

const exportFetcher = (
  page: number,
  pageSize: number,
) => fetchListAccessKeys(new PaginationQuery({ paging: { page, pageSize } }));
void exportFetcher;
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.accessKey.moduleName')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.accessKey.create') }}
        </a-button>
        <TableExportButton
          class="mr-2"
          :fetcher="exportFetcher"
          :columns="gridOptions.columns"
          filename="access-keys"
        />
      </template>
      <template #status="{ row }">
        <a-tag :color="row.status === 'ON' ? 'success' : 'default'">
          {{
            row.status === 'ON'
              ? $t('page.accessKey.statusOn')
              : $t('page.accessKey.statusOff')
          }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucideFilePenLine)"
          @click="handleEdit(row)"
        />
        <a-popconfirm
          :title="$t('page.accessKey.resetConfirm')"
          @confirm="handleResetSecret(row)"
        >
          <a-button type="link" :icon="h(LucideRefreshCw)" />
        </a-popconfirm>
        <a-popconfirm
          :title="
            $t('page.accessKey.deleteConfirm', {
              moduleName: $t('page.accessKey.moduleName'),
            })
          "
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />

    <a-modal
      v-model:open="resetSecretVisible"
      :title="$t('page.accessKey.resetSecret')"
      :cancel-button-props="{ style: { display: 'none' } }"
      :mask-closable="false"
      width="560px"
      @ok="resetSecretVisible = false"
    >
      <a-alert
        type="warning"
        :message="$t('page.accessKey.secretDialogHint')"
        class="mb-3"
      />
      <a-input :value="resetSecretValue" readonly>
        <template #addonAfter>
          <a @click="copyResetSecret">{{ $t('page.accessKey.secretCopied') }}</a>
        </template>
      </a-input>
    </a-modal>
  </Page>
</template>
