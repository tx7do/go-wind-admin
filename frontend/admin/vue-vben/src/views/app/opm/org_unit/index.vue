<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import { ColPage, Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type identityservicev1_OrgUnit as OrgUnit } from '#/api';
import {
  fetchListOrgUnits,
  orgUnitStatusToColor,
  orgUnitStatusToName,
  orgUnitTypeListForQuery,
  orgUnitTypeToColor,
  orgUnitTypeToName,
  PaginationQuery,
  statusList,
  useDeleteOrgUnit,
} from '#/api';
import { $t } from '#/locales';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';

import OrgDrawer from './org-drawer.vue';
import OrgTreePanel from '../user/list/org-list.vue';

// 左树选中机构（issue #5：左树右表布局）；undefined = 全部
const selectedOrgId = ref<string | undefined>(undefined);
const selectedOrgKey = ref<(number | string)[]>([]);

function handleOrgSelect(node: any) {
  // 左树用于导航聚焦：选中后按该机构过滤右表（parentId 精确匹配——
  // _id 后缀键在 makeQueryString 里不加 __contains，走 exact）。
  selectedOrgId.value = node?.id ? String(node.id) : undefined;
  selectedOrgKey.value = node?.id ? [`${node.parentId ?? ''}-${node.id}`] : [];
  // gridApi 在本文件后段才解构（模板事件晚于装配，运行时必然就绪）
  gridApi?.reload();
}

const { mutateAsync: deleteOrgUnit } = useDeleteOrgUnit();

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
      label: $t('page.orgUnit.name'),
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
      label: $t('page.orgUnit.type'),
      componentProps: {
        options: orgUnitTypeListForQuery,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<OrgUnit> = {
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

  treeConfig: {
    childrenField: 'children',
    rowField: 'id',
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        // 左树选中机构注入 parentId 过滤（_id 后缀走精确匹配，issue #5）
        const merged = { ...formValues };
        if (selectedOrgId.value) {
          merged.parentId = selectedOrgId.value;
        }
        return await fetchListOrgUnits(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues: merged,
          }),
        );
      },
    },
  },

  columns: [
    { title: $t('page.orgUnit.name'), field: 'name', treeNode: true },
    { title: $t('page.orgUnit.code'), field: 'code' },
    {
      title: $t('page.orgUnit.type'),
      field: 'type',
      slots: { default: 'orgUnitType' },
      width: 95,
    },
    { title: $t('page.orgUnit.description'), field: 'description' },
    { title: $t('page.orgUnit.leaderName'), field: 'leaderName' },
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
  fetchListOrgUnits(new PaginationQuery({ paging: { page, pageSize } }));

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer, drawerApi] = useVbenDrawer({
  // 连接抽离的组件
  connectedComponent: OrgDrawer,

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

  openDrawer(true);
}

/* 编辑 */
function handleEdit(row: any) {
  openDrawer(false, row);
}

/* 删除 */
async function handleDelete(row: any) {

  try {
    await deleteOrgUnit({ id: row.id });

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
          moduleName: $t('page.orgUnit.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}
</script>

<template>
  <Page auto-content-height>
    <ColPage
      auto-content-height
      :left-collapsed-width="0"
      :left-collapsible="false"
      :left-max-width="40"
      :left-min-width="20"
      :left-width="25"
      :right-width="75"
      :resizable="true"
      :split-handle="true"
      :split-line="true"
    >
      <template #left>
        <OrgTreePanel
          v-model:selected-keys="selectedOrgKey"
          @org-select="handleOrgSelect"
        />
      </template>
      <div class="ml-3 h-full">
        <Grid :table-title="$t('menu.opm.orgUnit')">
      <template #toolbar-tools>
        <a-button class="mr-2" @click="expandAll">
          {{ $t('ui.tree.expand_all') }}
        </a-button>
        <a-button class="mr-2" @click="collapseAll">
          {{ $t('ui.tree.collapse_all') }}
        </a-button>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="org-units" />
        <TableAddButton @click="handleCreate" />
      </template>
      <template #status="{ row }">
        <a-tag :color="orgUnitStatusToColor(row.status)">
          {{ orgUnitStatusToName(row.status) }}
        </a-tag>
      </template>
      <template #orgUnitType="{ row }">
        <a-tag :color="orgUnitTypeToColor(row.type)">
          {{ orgUnitTypeToName(row.type) }}
        </a-tag>
      </template>
          <template #action="{ row }">
            <TableActions :row="row" :actions="buildRowActions()" />
          </template>
        </Grid>
      </div>
    </ColPage>
    <Drawer />
  </Page>
</template>
