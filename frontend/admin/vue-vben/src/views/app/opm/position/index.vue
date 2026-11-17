<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListOrgUnits,
  fetchListPositions,
  PaginationQuery,
  positionTypeList,
  positionTypeToColor,
  positionTypeToName,
  statusList,
  statusToColor,
  statusToName,
  useCreatePosition,
  useDeletePosition,
} from '#/api';
import { type identityservicev1_Position as Position } from '#/api';
import ImportModal from '#/components/ImportModal.vue';
import type { ImportField } from '#/utils/import';
import { $t } from '#/locales';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';

import PositionDrawer from './position-drawer.vue';

const { mutateAsync: deletePosition } = useDeletePosition();

const formOptions: VbenFormProps = {
  // 默认展开
  // 控制表单是否显示折叠按钮
  showCollapseButton: false,
  // 按下回车时是否提交表单
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.position.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.position.code'),
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
    {
      component: 'Select',
      fieldName: 'type',
      label: $t('page.position.type'),
      componentProps: {
        options: positionTypeList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'ApiTreeSelect',
      fieldName: 'orgUnitId',
      label: $t('page.position.orgUnit'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        numberToString: true,
        showSearch: true,
        treeDefaultExpandAll: true,
        allowClear: true,
        childrenField: 'children',
        labelField: 'name',
        valueField: 'id',
        treeNodeFilterProp: 'label',
        api: async () => {
          const result = await fetchListOrgUnits(
            new PaginationQuery({
              formValues: { status: 'ON' },
            }),
          );
          return result.items;
        },
      },
    },
  ],
};

const gridOptions: VxeGridProps<Position> = {
  toolbarConfig: {
    custom: true,
    export: true,
    // import: true,
    refresh: true,
    zoom: true,
  },
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },
  height: 'auto',
  stripe: true,

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {

        return await fetchListPositions(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },

  columns: [
    { title: $t('page.position.name'), field: 'name' },
    { title: $t('page.position.code'), field: 'code' },
    {
      title: $t('page.position.type'),
      field: 'type',
      slots: { default: 'type' },
      width: 95,
    },
    { title: $t('page.position.description'), field: 'description' },
    {
      title: $t('page.position.orgUnitName'),
      field: 'orgUnitName',
      width: 150,
    },
    { title: $t('page.position.headcount'), field: 'headcount', width: 80 },
    {
      title: $t('ui.table.status'),
      field: 'status',
      slots: { default: 'status' },
      width: 95,
    },
    { title: $t('ui.table.sortOrder'), field: 'sortOrder', width: 70 },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    { title: $t('ui.table.remark'), field: 'remark' },
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
  fetchListPositions(new PaginationQuery({ paging: { page, pageSize } }));

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

// Excel 导入：字段与创建表单一致，唯排除 orgUnitId（外键需名称解析，属后续演进）
const { mutateAsync: createPosition } = useCreatePosition();
const importOpen = ref(false);
const importFields: ImportField[] = [
  { label: $t('page.position.name'), prop: 'name' },
  { label: $t('page.position.code'), prop: 'code' },
  { label: $t('page.position.type'), prop: 'type' },
  { label: $t('ui.table.status'), prop: 'status' },
  { label: $t('page.position.headcount'), prop: 'headcount' },
  { label: $t('ui.table.sortOrder'), prop: 'sortOrder' },
  { label: $t('page.position.description'), prop: 'description' },
  { label: $t('ui.table.remark'), prop: 'remark' },
];

function handleImportSuccess() {
  gridApi.reload();
}

const [Drawer, drawerApi] = useVbenDrawer({
  // 连接抽离的组件
  connectedComponent: PositionDrawer,

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

  openDrawer(true);
}

/* 编辑 */
function handleEdit(row: any) {
  openDrawer(false, row);
}

/* 删除 */
async function handleDelete(row: any) {

  try {
    await deletePosition({ id: row.id });

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

/* 操作列描述符：编辑行内、删除 Popconfirm 确认（≤3 个全行内，定式由 TableActions 统一执行） */
function buildRowActions(): TableAction[] {
  return [
    {
      key: 'edit',
      label: $t('common.edit'),
      icon: h(LucideFilePenLine),
      onClick: handleEdit,
    },
    {
      key: 'delete',
      label: $t('common.delete'),
      icon: h(LucideTrash2),
      danger: true,
      confirm: {
        title: $t('ui.text.do_you_want_delete', {
          moduleName: $t('page.position.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.opm.position')">
      <template #toolbar-tools>
        <a-button class="mr-2" @click="importOpen = true">
          {{ $t('ui.import.title') }}
        </a-button>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="positions" />
        <TableAddButton @click="handleCreate" />
      </template>
      <template #status="{ row }">
        <a-tag :color="statusToColor(row.status)">
          {{ statusToName(row.status) }}
        </a-tag>
      </template>
      <template #type="{ row }">
        <a-tag :color="positionTypeToColor(row.type)">
          {{ positionTypeToName(row.type) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <TableActions :row="row" :actions="buildRowActions()" />
      </template>
    </Grid>
    <Drawer />
    <ImportModal
      :open="importOpen"
      :fields="importFields"
      :create-row="(values: Record<string, any>) => createPosition(values)"
      @close="importOpen = false"
      @success="handleImportSuccess"
    />
  </Page>
</template>
