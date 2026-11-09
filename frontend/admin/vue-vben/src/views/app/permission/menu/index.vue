<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { IconifyIcon, LucideFilePenLine, LucideTrash2 } from '@vben/icons';

import { Icon } from '@iconify/vue';
import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  buildSyncMenusRequest,
  fetchListMenus,
  menuTypeToColor,
  menuTypeToName,
  PaginationQuery,
  statusList,
  statusToColor,
  statusToName,
  useDeleteMenu,
  useSyncMenus,
} from '#/api';
import { type permissionservicev1_Menu as Menu } from '#/api';
import { accessRoutes } from '#/router/routes';
import { $t } from '#/locales';
import TableActions from '#/components/TableActions.vue';
import type { TableAction } from '#/components/TableActions.vue';
import TableAddButton from '#/components/TableAddButton.vue';
import TableExportButton from '#/components/TableExportButton.vue';
import { getRandomColor } from '#/utils/color';

import MenuDrawer from './menu-drawer.vue';

const { mutateAsync: deleteMenu } = useDeleteMenu();
const { mutateAsync: syncMenus, isPending: isSyncing } = useSyncMenus();

/**
 * 菜单标题关键词的前端过滤。
 * 标题没法在服务端搜：库里 meta.title 存的是 i18n key 或源语言文本，用户看到的
 * 译文只有前端知道；且后端 go-crud 对 meta.* JSONB 路径的过滤条件值绑定有缺陷
 * （值被丢成空串，CONTAINS 恒真/EQ 恒假）。因此按「所见即所搜」在前端匹配：
 * 译文 + 库里原文 + 路由名三路 contains；命中行连同祖先链一起保留，
 * vxe 的 treeConfig.transform 才能还原树形。
 */
function filterMenusByTitle(items: Menu[], keyword?: string): Menu[] {
  const kw = (keyword ?? '').trim().toLowerCase();
  if (!kw) return items;
  const byId = new Map<string, Menu>(
    items.map((item) => [item.id as string, item]),
  );
  const keep = new Set<string>();
  for (const item of items) {
    const title = item.meta?.title ?? '';
    const haystack = [$t(title), title, item.name ?? '']
      .join('\n')
      .toLowerCase();
    if (!haystack.includes(kw)) continue;
    keep.add(item.id as string);
    let parentId = item.parentId;
    while (parentId != null && parentId !== '0' && !keep.has(parentId)) {
      keep.add(parentId);
      parentId = byId.get(parentId)?.parentId;
    }
  }
  return items.filter((item) => keep.has(item.id as string));
}

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
        // 标题走前端过滤（见 filterMenusByTitle），服务端只收 status；
        // pageSize 拉满确保全量行都在前端过滤范围内
        const resp = await fetchListMenus(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: 1000 },
            formValues: {
              status: formValues.status,
            },
            orderBy: ['id'],
          }),
        );
        const items = filterMenusByTitle(resp.items ?? [], formValues.name);
        return { ...resp, items, total: items.length };
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
      width: 140,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  fetchListMenus(new PaginationQuery({ paging: { page, pageSize } }));

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

  openDrawer(true);
}

/* 编辑 */
function handleEdit(row: any) {
  openDrawer(false, row);
}

/* 删除 */
async function handleDelete(row: any) {

  try {
    await deleteMenu({ id: row.id });

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

/* 同步菜单：将本地静态路由全部推送到后端 */
async function handleSyncMenus() {
  try {
    await syncMenus(buildSyncMenusRequest(accessRoutes));
    notification.success({ message: $t('ui.notification.update_success') });
    await gridApi.reload();
  } catch {
    notification.error({ message: $t('ui.notification.update_failed') });
  }
}

function normalizeAuthority(authority: unknown): string[] {
  if (Array.isArray(authority)) return authority;
  if (typeof authority === 'string') {
    return authority
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
  }
  return [];
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
          moduleName: $t('page.menu.moduleName'),
        }),
      },
      onClick: handleDelete,
    },
  ];
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.permission.menu')">
      <template #toolbar-tools>
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="$t('page.menu.text.do_you_want_sync_menus')"
          @confirm="handleSyncMenus"
        >
          <a-button class="gowind-btn-warning-outline mr-2" :loading="isSyncing">
            {{ $t('page.menu.button.sync') }}
          </a-button>
        </a-popconfirm>
        <a-button class="mr-2" @click="expandAll">
          {{ $t('ui.tree.expand_all') }}
        </a-button>
        <a-button class="mr-2" @click="collapseAll">
          {{ $t('ui.tree.collapse_all') }}
        </a-button>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="menus" />
        <TableAddButton @click="handleCreate" />
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
          <!-- meta 缺失（直接 API 创建的菜单）时回退 name；$t(undefined) 会抛
               Invalid arguments 使整个 grid 渲染崩溃（__vnode 污染） -->
          <span class="flex-auto">
            {{ row.meta?.title ? $t(row.meta.title) : (row.name ?? '-') }}
          </span>
          <div class="items-center justify-end"></div>
        </div>
      </template>
      <template #icon="{ row }">
        <Icon
          v-if="row.meta?.icon !== undefined"
          :icon="row.meta?.icon"
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
          v-for="auth in normalizeAuthority(row.meta?.authority)"
          :key="auth"
          class="mb-1 mr-1"
          :color="getRandomColor(auth)"
        >
          {{ auth }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <TableActions :row="row" :actions="buildRowActions()" />
      </template>
    </Grid>
    <Drawer />
  </Page>
</template>
