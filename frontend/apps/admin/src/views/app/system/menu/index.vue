<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { IconifyIcon, LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { Icon } from '@iconify/vue';
import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type resourceservicev1_Menu as Menu } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import {
  menuTypeToColor,
  menuTypeToName,
  statusList,
  statusToColor,
  statusToName,
  useMenuStore,
} from '#/stores';
import { getRandomColor } from '#/utils/color';

import MenuDrawer from './menu-drawer.vue';

const menuStore = useMenuStore();

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
      fieldName: 'name',
      label: $t('page.menu.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('ui.table.status'),
      componentProps: {
        options: statusList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<Menu> = {
  toolbarConfig: {
    custom: true,
    export: true,
    // import: true,
    refresh: true,
    zoom: true,
  },
  exportConfig: {},
  pagerConfig: {
    enabled: false,
  },
  rowConfig: {
    isHover: true,
  },

  stripe: true,
  height: 'auto',

  treeConfig: {
    parentField: 'parentId',
    rowField: 'id',
    transform: true,
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        console.log('query:', formValues);

        return await menuStore.listMenu(
          {
            page: page.currentPage,
            pageSize: page.pageSize,
          },
          {
            'meta.title': formValues.name,
            status: formValues.status,
          },
          null,
          ['id'],
        );
      },
    },
  },

  columns: [
    {
      title: $t('page.menu.name'),
      field: 'meta.title',
      slots: { default: 'title' },
      width: 180,
      fixed: 'left',
      align: 'left',
      treeNode: true,
    },
    {
      title: $t('page.menu.type'),
      field: 'type',
      slots: { default: 'type' },
      width: 95,
    },
    {
      title: $t('page.menu.authority'),
      field: 'meta.authority',
      align: 'left',
      slots: { default: 'authority' },
    },
    { title: $t('page.menu.path'), field: 'path', align: 'left' },
    { title: $t('page.menu.component'), field: 'component', align: 'left' },
    {
      title: $t('ui.table.status'),
      field: 'status',
      slots: { default: 'status' },
      width: 95,
    },
    { title: $t('ui.table.sortOrder'), field: 'meta.order', width: 70 },
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
  connectedComponent: MenuDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      // 关闭时，重载表格数据
      gridApi.reload();
    }
  },
});

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
    await menuStore.deleteMenu(row.id);

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

const expandAll = () => {
  gridApi.grid?.setAllTreeExpand(true);
};

const collapseAll = () => {
  gridApi.grid?.setAllTreeExpand(false);
};
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.system.menu')">
      <template #toolbar-tools>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.menu.button.create') }}
        </a-button>
        <a-button class="mr-2" @click="expandAll">
          {{ $t('ui.tree.expand_all') }}
        </a-button>
        <a-button class="mr-2" @click="collapseAll">
          {{ $t('ui.tree.collapse_all') }}
        </a-button>
      </template>
      <template #title="{ row }">
        <div class="flex w-full items-center gap-1">
          <div class="size-5 flex-shrink-0">
            <IconifyIcon
              v-if="row.type === 'button'"
              icon="carbon:security"
              class="size-full"
            />
            <IconifyIcon
              v-else-if="row.meta?.icon"
              :icon="row.meta?.icon || 'carbon:circle-dash'"
              class="size-full"
            />
          </div>
          <span class="flex-auto">{{ $t(row.meta?.title) }}</span>
          <div class="items-center justify-end"></div>
        </div>
      </template>
      <template #icon="{ row }">
        <Icon
          v-if="row.meta.icon !== undefined"
          :icon="row.meta.icon"
          class="mr-1 size-4 flex-shrink-0"
        />
      </template>
      <template #status="{ row }">
        <a-tag :color="statusToColor(row.status)">
          {{ statusToName(row.status) }}
        </a-tag>
      </template>
      <template #type="{ row }">
        <a-tag :color="menuTypeToColor(row.type)">
          {{ menuTypeToName(row.type) }}
        </a-tag>
      </template>
      <template #authority="{ row }">
        <a-tag
          v-for="auth in row.meta.authority"
          :key="auth"
          class="mb-1 mr-1"
          :style="{
            backgroundColor: getRandomColor(auth), // 随机背景色
            color: '#333', // 深色文字（适配浅色背景）
            border: 'none', // 可选：去掉边框更美观
          }"
        >
          {{ auth }}
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
              moduleName: $t('page.menu.moduleName'),
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
