<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideCheck, LucideEye, LucideTrash2 } from '@vben/icons';
import { useUserStore } from '@vben/stores';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import TableExportButton from '#/components/TableExportButton.vue';
import { type internal_messageservicev1_InternalMessageRecipient as InternalMessageRecipient } from '#/api';
import {
  fetchListUserInbox,
  internalMessageRecipientStatusColor,
  internalMessageRecipientStatusLabel,
  internalMessageRecipientStatusList,
  PaginationQuery,
  useDeleteNotificationFromInbox,
  useMarkNotificationAsRead,
} from '#/api';
import { $t } from '#/locales';

import InboxDetailDrawer from './inbox-detail-drawer.vue';

const userStore = useUserStore();
const { mutateAsync: markNotificationAsRead } = useMarkNotificationAsRead();
const { mutateAsync: deleteNotificationFromInbox } =
  useDeleteNotificationFromInbox();

const formOptions: VbenFormProps = {
  // 默认展开
  collapsed: false,
  // 控制表单是否显示折叠按钮
  showCollapseButton: false,
  // 按下回车时是否提交表单
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'title',
      label: $t('page.internalMessage.title'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('page.internalMessage.status'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: internalMessageRecipientStatusList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<InternalMessageRecipient> = {
  toolbarConfig: {
    custom: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },
  stripe: true,

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        // 收件箱按当前登录用户过滤
        const userId = userStore.userInfo?.id;

        return await fetchListUserInbox(
          new PaginationQuery({
            paging: {
              page: page.currentPage,
              pageSize: page.pageSize,
            },
            formValues: {
              recipient_user_id: userId?.toString(),
              title: formValues.title,
              status: formValues.status,
            },
            orderBy: ['-created_at'],
          }),
        );
      },
    },
  },

  columns: [
    {
      title: $t('page.internalMessage.title'),
      field: 'title',
      minWidth: 200,
      slots: { default: 'title' },
    },
    {
      title: $t('page.internalMessage.status'),
      field: 'status',
      width: 120,
      slots: { default: 'status' },
    },
    {
      title: $t('page.internalMessage.readAt'),
      field: 'readAt',
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
      width: 140,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  fetchListUserInbox(new PaginationQuery({ paging: { page, pageSize } }));

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  // 连接详情抽屉组件
  connectedComponent: InboxDetailDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      // 关闭时重载表格（详情可能已标记为已读，状态需刷新）
      gridApi.reload();
    }
  },
});

/* 打开消息详情抽屉 */
function handleView(row: any) {
  drawerApi.setData({ row });
  drawerApi.open();
}

/* 标记为已读 */
async function handleMarkAsRead(row: any) {
  const userId = userStore.userInfo?.id ?? 0;

  try {
    await markNotificationAsRead({
      userId,
      recipientIds: [row.id],
    });

    notification.success({
      message: $t('ui.notification.operation_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  }
}

/* 删除消息 */
async function handleDelete(row: any) {
  const userId = userStore.userInfo?.id ?? 0;

  try {
    await deleteNotificationFromInbox({
      userId,
      recipientIds: [row.id],
    });

    notification.success({
      message: $t('ui.notification.delete_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.delete_failed'),
    });
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.user.detail.tab.internalMessage')">
      <template #toolbar-tools>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="export" />
      </template>

      <template #title="{ row }">
        <!-- 未读消息标题加粗，便于和已读区分 -->
        <span
          :class="{ 'font-semibold': row.status !== 'READ' }"
          class="cursor-pointer hover:text-primary"
          @click="handleView(row)"
        >
          {{ row.title }}
        </span>
      </template>
      <template #status="{ row }">
        <a-tag :color="internalMessageRecipientStatusColor(row.status)">
          {{ internalMessageRecipientStatusLabel(row.status) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucideEye)"
          @click="handleView(row)"
        />
        <a-button
          v-if="row.status !== 'READ'"
          type="link"
          :icon="h(LucideCheck)"
          @click="handleMarkAsRead(row)"
        />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', {
              moduleName: $t('page.internalMessage.moduleName'),
            })
          "
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
