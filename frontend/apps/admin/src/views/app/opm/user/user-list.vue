<script lang="ts" setup>
import type { VxeGridListeners, VxeGridProps } from '#/adapter/vxe-table';

import { h, watch } from 'vue';

import { useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideInfo, LucideTrash2 } from '@vben/icons';
import { isEqual } from '@vben/utils';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type identityservicev1_User as User } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import { router } from '#/router';
import {
  genderToColor,
  genderToName,
  usePositionStore,
  useRoleStore,
  userStatusList,
  userStatusToColor,
  userStatusToName,
  useUserListStore,
} from '#/stores';
import { getRandomColor } from '#/utils/color';
import { useUserViewStore } from '#/views/app/opm/user/user-view.state';

import UserDrawer from './user-drawer.vue';

const userListStore = useUserListStore();
const roleStore = useRoleStore();
const positionStore = usePositionStore();
const userViewStore = useUserViewStore();

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
          const result = await roleStore.listRole(undefined, {
            status: 'ON',
            type__not: 'TEMPLATE',
            tenant_id: userViewStore.currentTenantId ?? 0,
          });
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
          const result = await positionStore.listPosition(undefined, {
            status: 'ON',
            org_unit_id: userViewStore.currentOrgUnitId,
            tenant_id: userViewStore.currentTenantId ?? 0,
          });
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
        // console.log('query:', filters, form, formValues);
        return userViewStore.fetchUserList(
          page.currentPage,
          page.pageSize,
          formValues,
        );
      },
    },
  },

  columns: [
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
      width: 120,
    },
  ],
};

const gridEvents: VxeGridListeners<User> = {
  cellDblclick: ({ row }) => {
    // console.log(`cell-click: ${row.id}`);
    handleDetail(row);
  },
};

const [Grid, gridApi] = useVbenVxeGrid({
  gridOptions,
  formOptions,
  gridEvents,
});

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
    await userListStore.deleteUser(row.id);

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

/* 详情 */
function handleDetail(row: any) {
  router.push(`/opm/users/detail/${row.id}`);
}

watch(
  () => [userViewStore.currentOrgUnitId, userViewStore.currentTenantId],
  (newValues, oldValue) => {
    if (isEqual(newValues, oldValue)) {
      return;
    }
    console.log(newValues, oldValue);
    gridApi.reload();
  },
);
</script>

<template>
  <Grid :table-title="$t('menu.opm.user')">
    <template #toolbar-tools>
      <a-button type="primary" @click="handleCreate">
        {{ $t('page.user.button.create') }}
      </a-button>
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
          :style="{
            backgroundColor: getRandomColor(role), // 随机背景色
            color: '#333', // 深色文字（适配浅色背景）
            border: 'none', // 可选：去掉边框更美观
          }"
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
          :style="{
            backgroundColor: getRandomColor(orgUnit), // 随机背景色
            color: '#333', // 深色文字（适配浅色背景）
            border: 'none', // 可选：去掉边框更美观
          }"
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
          :style="{
            backgroundColor: getRandomColor(position), // 随机背景色
            color: '#333', // 深色文字（适配浅色背景）
            border: 'none', // 可选：去掉边框更美观
          }"
        >
          {{ position }}
        </a-tag>
      </div>
    </template>
    <template #action="{ row }">
      <a-button
        type="link"
        :icon="h(LucideInfo)"
        @click.stop="handleDetail(row)"
      />

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
            moduleName: $t('page.user.moduleName'),
          })
        "
        @confirm="handleDelete(row)"
      >
        <a-button danger type="link" :icon="h(LucideTrash2)" />
      </a-popconfirm>
    </template>
  </Grid>
  <Drawer />
</template>

<style scoped></style>
