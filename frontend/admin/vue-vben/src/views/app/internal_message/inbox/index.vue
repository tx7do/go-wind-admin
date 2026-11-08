<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import dayjs from 'dayjs';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideCheck, LucideTrash2 } from '@vben/icons';
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
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';

import InboxDetailDrawer from './inbox-detail-drawer.vue';

const userStore = useUserStore();
const { mutateAsync: markNotificationAsRead } = useMarkNotificationAsRead();
const { mutateAsync: deleteNotificationFromInbox } =
  useDeleteNotificationFromInbox();

// 当前搜索条件（导出与 Grid proxy 查询同源——"导出的就是当前搜索看到的"，
// 定式与审计日志页一致）
const latestFormValuesRef = ref<Record<string, unknown>>({});

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
    {
      // 接收时间区间搜索（对齐 react/ele 端）：query 里映射为 go-crud 的
      // created_at__gte/lte（操作符后缀表内，不会被误加 __contains）
      component: 'RangePicker',
      fieldName: 'createdAt',
      label: $t('page.internalMessage.createdAt'),
      componentProps: {
        showTime: true,
        allowClear: true,
        presets: [
          {
            label: $t('ui.dateRange.today'),
            value: [dayjs().startOf('day'), dayjs().endOf('day')],
          },
          {
            label: $t('ui.dateRange.yesterday'),
            value: [
              dayjs().subtract(1, 'day').startOf('day'),
              dayjs().subtract(1, 'day').endOf('day'),
            ],
          },
          {
            label: $t('ui.dateRange.thisWeek'),
            value: [dayjs().startOf('week'), dayjs().endOf('week')],
          },
          {
            label: $t('ui.dateRange.lastWeek'),
            value: [
              dayjs().subtract(1, 'week').startOf('week'),
              dayjs().subtract(1, 'week').endOf('week'),
            ],
          },
          {
            label: $t('ui.dateRange.thisMonth'),
            value: [dayjs().startOf('month'), dayjs().endOf('month')],
          },
          {
            label: $t('ui.dateRange.lastMonth'),
            value: [
              dayjs().subtract(1, 'month').startOf('month'),
              dayjs().subtract(1, 'month').endOf('month'),
            ],
          },
        ],
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
        // 收件箱按当前登录用户过滤。userId 未就绪（会话静默恢复期 profile 拉取失败等）
        // 必须空返回而不是省略过滤——PaginationQuery 会剥掉 undefined 条目，请求变成
        // 无 recipient 过滤，按租户列出其他用户的收件记录（静默越权）。
        const userId = userStore.userInfo?.id;
        if (!userId) return { items: [], total: 0 };

        let startTime: string | undefined;
        let endTime: string | undefined;
        if (
          Array.isArray(formValues.createdAt) &&
          formValues.createdAt.length === 2
        ) {
          startTime = dayjs(formValues.createdAt[0]).format(
            'YYYY-MM-DD HH:mm:ss',
          );
          endTime = dayjs(formValues.createdAt[1]).format(
            'YYYY-MM-DD HH:mm:ss',
          );
        }

        // 列表与导出共用同一份搜索条件
        const conditions: Record<string, unknown> = {
          recipient_user_id: userId.toString(),
          title: formValues.title,
          status: formValues.status,
          created_at__gte: startTime,
          created_at__lte: endTime,
        };
        latestFormValuesRef.value = conditions;

        return await fetchListUserInbox(
          new PaginationQuery({
            paging: {
              page: page.currentPage,
              pageSize: page.pageSize,
            },
            formValues: conditions,
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
      align: 'left',
      slots: { default: 'title' },
    },
    {
      title: $t('page.internalMessage.status'),
      field: 'status',
      width: 120,
      align: 'left',
      slots: { default: 'status' },
    },
    {
      title: $t('page.internalMessage.readAt'),
      field: 'readAt',
      formatter: 'formatDateTime',
      width: 160,
      align: 'left',
    },
    {
      title: $t('page.internalMessage.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 160,
      align: 'left',
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 175,
    },
  ],
};

// 导出与列表同源同一份条件；recipient_user_id 在调用时再强制注入一次——
// 它是隐私闸门，缺失时 PaginationQuery 会静默剥掉该条件导致租户级越权，
// 绝不能只依赖 latestFormValuesRef 的写入时序
const exportFetcher = (page: number, pageSize: number) => {
  const userId = userStore.userInfo?.id;
  if (!userId) return Promise.resolve({ items: [], total: 0 });
  return fetchListUserInbox(
    new PaginationQuery({
      paging: { page, pageSize },
      formValues: {
        ...latestFormValuesRef.value,
        recipient_user_id: userId.toString(),
      },
    }),
  );
};

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

/* 全部已读（工具栏）：recipientIds 传空 = 标记该用户全部未读 */
const markAllReadLoading = ref(false);
async function handleMarkAllRead() {
  const userId = userStore.userInfo?.id;
  if (!userId) return;
  markAllReadLoading.value = true;
  try {
    await markNotificationAsRead({ userId, recipientIds: [] });
    notification.success({
      message: $t('ui.notification.operation_success'),
    });
    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.operation_failed'),
    });
  } finally {
    markAllReadLoading.value = false;
  }
}
/* 操作列描述符：标记已读（未读才显示，hidden 计溢出前剔除）+ 删除 Popconfirm */
function buildRowActions(row: any): TableAction[] {
  return [
    {
      key: 'mark-read',
      label: $t('page.internalMessage.button.markAsRead'),
      icon: h(LucideCheck),
      hidden: row.status === 'READ',
      onClick: handleMarkAsRead,
    },
    {
      key: 'delete',
      label: $t('common.delete'),
      icon: h(LucideTrash2),
      danger: true,
      confirm: {
        title: $t('ui.text.do_you_want_delete', {
          moduleName: $t('page.internalMessage.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.user.detail.tab.internalMessage')">
      <template #toolbar-tools>
        <!-- 全部已读：recipientIds 传空 = 标记该用户全部未读 -->
        <a-button
          class="mr-2"
          :loading="markAllReadLoading"
          :disabled="!userStore.userInfo?.id"
          @click="handleMarkAllRead"
        >
          {{ $t('page.internalMessage.button.markAllRead') }}
        </a-button>
        <!-- userId 未就绪不提供导出入口（同 react 端口径）：隐私过滤无法注入时宁可不导 -->
        <TableExportButton
          v-if="userStore.userInfo?.id"
          :fetcher="exportFetcher"
          :columns="gridOptions.columns"
          filename="export"
        />
      </template>

      <template #title="{ row }">
        <!-- 链接形态：常显主题色，悬停下划线；未读消息加粗，便于和已读区分 -->
        <span
          :class="{ 'font-semibold': row.status !== 'READ' }"
          class="cursor-pointer text-primary hover:underline"
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
        <TableActions :row="row" :actions="buildRowActions(row)" />
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
