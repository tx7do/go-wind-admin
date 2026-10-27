<script lang="ts" setup>
import type { VxeGridListeners, VxeGridProps } from '#/adapter/vxe-table';

import { computed, h, ref, watch } from 'vue';

import { useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideInfo, LucidePrinter, LucideShieldOff, LucideTrash2 } from '@vben/icons';
import { isEqual } from '@vben/utils';

import { Modal, notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';
import { useAccessStore, useUserStore } from '@vben/stores';
import { type identityservicev1_User as User } from '#/api';
import {
  disableMfa,
  fetchListPositions,
  fetchListRoles,
  genderToColor,
  genderToName,
  PaginationQuery,
  useDeleteUser,
  userStatusList,
  userStatusToColor,
  userStatusToName,
} from '#/api';
import { $t } from '#/locales';
import TableActions from '#/components/TableActions.vue';
import PrintTemplatePrint from '#/components/PrintTemplatePrint.vue';
import type { TableAction } from '#/components/TableActions.vue';
import { router } from '#/router';
import { parseResourceHiddenFields } from '#/utils';
import { getRandomColor } from '#/utils/color';

import UserDrawer from './user-drawer.vue';
import { useUserViewStore } from './user-view.state';

const { mutateAsync: deleteUser } = useDeleteUser();
const userViewStore = useUserViewStore();

// 字段权限：当前用户在 User 资源上被隐藏的字段，命中的列整列不渲染。
// 隐藏集仅随登录聚合（重新登录后生效），setup 时取一次快照即可。
const userHiddenFields = parseResourceHiddenFields(
  useAccessStore().hiddenFields,
  'User',
);

const formOptions: VbenFormProps = {
  // 默认展开
  collapsed: true,
  // 控制表单是否显示折叠按钮
  showCollapseButton: true,
  // 按下回车时是否提交表单
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'username',
      label: $t('page.user.form.username'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'realname',
      label: $t('page.user.form.realname'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'mobile',
      label: $t('page.user.form.mobile'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('page.user.form.status'),
      componentProps: {
        options: userStatusList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'ApiSelect',
      fieldName: 'roleId',
      label: $t('page.user.form.role'),
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        afterFetch: (data: { name: string; path: string }[]) => {
          return data.map((item: any) => ({
            label: item.name,
            value: item.id,
          }));
        },
        api: async () => {
          const result = await fetchListRoles(
            new PaginationQuery({
              formValues: {
                status: 'ON',
                type__not: 'TEMPLATE',
                tenant_id: userViewStore.currentTenantId ?? 0,
              },
            }),
          );
          return result.items;
        },
      },
    },
    {
      component: 'ApiSelect',
      fieldName: 'positionId',
      label: $t('page.user.form.position'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
        showSearch: true,
        alwaysLoad: true,
        immediate: true,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        afterFetch: (data: { name: string; path: string }[]) => {
          return data.map((item: any) => ({
            label: item.name,
            value: item.id,
          }));
        },
        api: async () => {
          const result = await fetchListPositions(
            new PaginationQuery({
              formValues: {
                status: 'ON',
                org_unit_id: userViewStore.currentOrgUnitId,
                tenant_id: userViewStore.currentTenantId ?? 0,
              },
            }),
          );
          return result.items;
        },
      },
    },
  ],
};

const gridOptions: VxeGridProps<User> = {
  height: 'auto',
  stripe: true,
  autoResize: true,
  toolbarConfig: {
    custom: true,
    export: true,
    import: false,
    refresh: true,
    zoom: true,
  },
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
    resizable: true,
  },
  resizableConfig: {},
  tooltipConfig: {
    showAll: true,
    enterable: true,
    contentMethod: ({ column, row }) => {
      const { field } = column;
      if (field === 'roleNames') {
        return `${row[field]}`;
      }
      // 其余的单元格使用默认行为
      return null;
    },
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        return userViewStore.fetchUserList(
          page.currentPage,
          page.pageSize,
          formValues,
        );
      },
    },
  },

  columns: ([
    { type: 'checkbox', width: 44 },
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.user.table.username'), field: 'username', width: 120 },
    { title: $t('page.user.table.realname'), field: 'realname', width: 100 },
    { title: $t('page.user.table.nickname'), field: 'nickname', width: 100 },
    { title: $t('page.user.table.email'), field: 'email', width: 160 },
    { title: $t('page.user.table.mobile'), field: 'mobile', width: 130 },
    {
      title: $t('page.user.table.orgUnitId'),
      field: 'orgUnitNames',
      slots: { default: 'orgUnit' },
      width: 130,
    },
    {
      title: $t('page.user.table.positionId'),
      field: 'positionNames',
      slots: { default: 'position' },
      width: 130,
    },
    {
      title: $t('page.user.table.roleId'),
      field: 'roleNames',
      slots: { default: 'role' },
      width: 100,
      showOverflow: 'tooltip',
    },
    {
      title: $t('page.user.table.status'),
      field: 'status',
      width: 95,
      slots: { default: 'status' },
    },
    {
      title: $t('page.user.table.lastLoginAt'),
      field: 'lastLoginAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    { title: $t('ui.table.remark'), field: 'remark', width: 250 },

    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      // 平台侧最多 4 项（详情/编辑/重置MFA/更多），文本+图标实测约 250px
      width: 310,
    },
  ] satisfies VxeGridProps<User>['columns']).filter(
    (col) => !userHiddenFields.has(col.field as string),
  ),
};

const gridEvents: VxeGridListeners<User> = {
  cellDblclick: ({ row }) => {
    handleDetail(row);
  },
};

const exportFetcher = (page: number, pageSize: number) =>
  userViewStore.fetchUserList(page, pageSize, {});

const printRef = ref();

const [Grid, gridApi] = useVbenVxeGrid({

  gridOptions,
  formOptions,
  gridEvents,
});

/** 批量打印：勾选行合并渲染为单文档（内核份数分页），走行打印同一定式入口 */
function handleBatchPrint() {
  const records: any[] = (gridApi.grid.getCheckboxRecords() || []) as any[];
  if (records.length === 0) {
    notification.warning({ message: $t('page.printTemplate.noSelection') });
    return;
  }
  printRef.value?.print('system/user', records);
}

const [Drawer, drawerApi] = useVbenDrawer({
  // 连接抽离的组件
  connectedComponent: UserDrawer,

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
// 仅平台侧操作者（tenantId=0）显示救援重置；非平台用户由后端 403 兜底
const isPlatformSide = computed(() => Number(useUserStore().userInfo?.tenantId ?? 0) === 0);

// 管理端救援重置：清空目标用户 TOTP 因子（仅平台管理员生效，后端强制校验）
async function handleResetMfa(row: any) {
  try {
    await disableMfa({ userId: row.id, method: 'TOTP' } as any);
    notification.success({ message: $t('page.user.resetMfaSuccess') });
  } catch (err: any) {
    notification.error({ message: err?.message || $t('page.user.resetMfaFailed') });
  }
}

async function handleDelete(row: any) {

  try {
    await deleteUser(row.id);

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

/* 「更多」下拉里的删除走 Modal 确认（下拉里放不下 Popconfirm） */
function confirmDelete(row: any) {
  Modal.confirm({
    title: $t('ui.text.do_you_want_delete', {
      moduleName: $t('page.user.moduleName'),
    }),
    okText: $t('ui.button.ok'),
    cancelText: $t('ui.button.cancel'),
    okType: 'danger',
    onOk: () => handleDelete(row),
  });
}

/* 详情 */
function handleDetail(row: any) {
  router.push(`/opm/users/detail/${row.id}`);
}

/* 操作列描述符：详情/编辑 + 平台侧才有重置 MFA（hidden 计溢出前剔除），
   删除溢出进「更多」由组件降级 Modal.confirm（confirmDelete） */
function buildRowActions(): TableAction[] {
  return [
    {
      key: 'detail',
      label: $t('ui.button.detail'),
      icon: h(LucideInfo),
      onClick: handleDetail,
    },
    {
      key: 'print',
      label: $t('page.printTemplate.print'),
      icon: h(LucidePrinter),
      onClick: (row: any) => printRef.value?.print('system/user', row),
    },
    {
      key: 'edit',
      label: $t('common.edit'),
      icon: h(LucideFilePenLine),
      onClick: handleEdit,
    },
    {
      key: 'reset-mfa',
      label: $t('page.user.button.resetMfa'),
      icon: h(LucideShieldOff),
      hidden: !isPlatformSide.value,
      confirm: { title: $t('page.user.resetMfaConfirmTitle') },
      onClick: handleResetMfa,
    },
    {
      key: 'delete',
      label: $t('common.delete'),
      icon: h(LucideTrash2),
      danger: true,
      // confirmDelete 内部已弹 Modal.confirm（含 okType danger），不再挂 confirm 免双重确认
      onClick: confirmDelete,
    },
  ];
}

watch(
  () => [userViewStore.currentOrgUnitId, userViewStore.currentTenantId],
  (newValues, oldValue) => {
    if (isEqual(newValues, oldValue)) {
      return;
    }
    gridApi.reload();
  },
);
</script>

<template>
  <Grid :table-title="$t('menu.opm.user')">
    <template #toolbar-tools>
        <a-button
          class="mr-2"
          @click="handleBatchPrint"
        >
          {{ $t('page.printTemplate.batchPrint') }}
        </a-button>
          <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="export" />
          <TableAddButton @click="handleCreate" />
    </template>
    <template #status="{ row }">
      <a-tag :color="userStatusToColor(row.status)">
        {{ userStatusToName(row.status) }}
      </a-tag>
    </template>
    <template #gender="{ row }">
      <a-tag :color="genderToColor(row.gender)">
        {{ genderToName(row.gender) }}
      </a-tag>
    </template>
    <template #role="{ row }">
      <div>
        <a-tag
          v-for="role in row.roleNames"
          :key="role"
          class="mb-1 mr-1"
          :color="getRandomColor(role)"
        >
          {{ role }}
        </a-tag>
      </div>
    </template>
    <template #orgUnit="{ row }">
      <div>
        <a-tag
          v-for="orgUnit in row.orgUnitNames"
          :key="orgUnit"
          class="mb-1 mr-1"
          :color="getRandomColor(orgUnit)"
        >
          {{ orgUnit }}
        </a-tag>
      </div>
    </template>
    <template #position="{ row }">
      <div>
        <a-tag
          v-for="position in row.positionNames"
          :key="position"
          class="mb-1 mr-1"
          :color="getRandomColor(position)"
        >
          {{ position }}
        </a-tag>
      </div>
    </template>
    <template #action="{ row }">
      <TableActions :row="row" :actions="buildRowActions()" />
    </template>
  </Grid>
  <PrintTemplatePrint ref="printRef" />
  <Drawer />
</template>

<style scoped></style>
