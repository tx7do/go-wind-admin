<script lang="ts" setup>
import type { VxeGridProps } from '@/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '@/adapter/vxe-table';
import { type internal_messageservicev1_InternalMessage as InternalMessage } from '@/api/generated/admin/service/v1';
import { $t } from '@/locales';
import {
  internalMessageStatusColor,
  internalMessageStatusLabel,
  internalMessageStatusList,
  internalMessageTypeColor,
  internalMessageTypeLabel,
  internalMessageTypeList,
  useInternalMessageCategoryStore,
  useInternalMessageStore,
} from '@/stores';

import InternalMessageDrawer from './internal-message-drawer.vue';

const internalMessageStore = useInternalMessageStore();
const internalMessageCategoryStore = useInternalMessageCategoryStore();

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
      label: t('pages.internalMessage.title'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: t('pages.internalMessage.status'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: internalMessageStatusList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        showSearch: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'type',
      label: t('pages.internalMessage.type'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: internalMessageTypeList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        showSearch: true,
      },
    },
    {
      component: 'ApiTreeSelect',
      fieldName: 'category_id',
      label: t('pages.internalMessage.categoryId'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        numberToString: true,
        showSearch: true,
        treeDefaultExpandAll: true,
        childrenField: 'children',
        labelField: 'name',
        valueField: 'id',
        treeNodeFilterProp: 'label',
        api: async () => {
          const result =
            await internalMessageCategoryStore.listInternalMessageCategory(
              undefined,
              {
                is_enabled: 'true',
              },
            );
          return result.items;
        },
      },
    },
  ],
};

const gridOptions: VxeGridProps<InternalMessage> = {
  toolbarConfig: {
    custom: true,
    export: true,
    // import: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  exportConfig: {},
  pagerConfig: {
    enabled: false,
  },
  rowConfig: {
    isHover: true,
  },
  stripe: true,

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        console.log('query:', formValues);

        return await internalMessageStore.listMessage(
          {
            page: page.currentPage,
            pageSize: page.pageSize,
          },
          formValues,
        );
      },
    },
  },

  columns: [
    {
      title: t('pages.internalMessage.title'),
      field: 'title',
    },
    {
      title: t('pages.internalMessage.categoryName'),
      field: 'categoryName',
    },
    {
      title: t('pages.internalMessage.status'),
      field: 'status',
      slots: { default: 'status' },
    },
    {
      title: t('pages.internalMessage.type'),
      field: 'type',
      slots: { default: 'type' },
    },
    {
      title: t('pages.internalMessage.senderName'),
      field: 'senderName',
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 90,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  // 连接抽离的组件
  connectedComponent: InternalMessageDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      // 关闭时，重载表格数据
      gridApi.reload();
    }
  },
});

/* 打开模态窗口 */
function openDrawer(create: boolean, row?: any) {
  drawerApi.setData({
    create,
    row,
  });

  drawerApi.open();
}

/* 创建 */
function handleCreate() {
  console.log('创建');
  openDrawer(true);
}

/* 编辑 */
function handleEdit(row: any) {
  console.log('编辑', row);
  openDrawer(false, row);
}

/* 删除 */
async function handleDelete(row: any) {
  console.log('删除', row);

  try {
    await internalMessageStore.deleteMessage(row.id);

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
    <Grid :table-title="$t('menu.internalMessage.internalMessage')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ t('pages.internalMessage.button.create') }}
        </a-button>
      </template>
      <template #status="{ row }">
        <a-tag :color="internalMessageStatusColor(row.status)">
          {{ internalMessageStatusLabel(row.status) }}
        </a-tag>
      </template>
      <template #type="{ row }">
        <a-tag :color="internalMessageTypeColor(row.type)">
          {{ internalMessageTypeLabel(row.type) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucideFilePenLine)"
          @click.stop="handleEdit(row)"
        />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', {
              moduleName: t('pages.internalMessage.moduleName'),
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
