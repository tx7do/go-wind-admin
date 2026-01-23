<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, watch } from 'vue';

import { useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideTrash2 } from '@vben/icons';
import { preferences } from '@vben/preferences';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type dictservicev1_DictEntry as DictEntry } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import { enableBoolToColor, enableBoolToName, useDictStore } from '#/stores';
import { useDictViewStore } from '#/views/app/system/dict/dict-view.state';

import DictEntryDrawer from './dict-entry-drawer.vue';

const dictStore = useDictStore();
const dictViewStore = useDictViewStore();

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
      fieldName: 'entry_value',
      label: $t('page.dict.entryValue'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<DictEntry> = {
  stripe: true,
  height: 'auto',

  toolbarConfig: {
    custom: false,
    export: true,
    import: true,
    refresh: true,
    zoom: false,
  },
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
    isCurrent: false,
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        // console.log('query:', filters, form, formValues);
        return await dictViewStore.fetchEntryList(
          dictViewStore.currentTypeId,
          page.currentPage,
          page.pageSize,
          formValues,
        );
      },
    },
  },

  columns: [
    {
      title: $t('page.dict.entryLabel'),
      field: 'entryLabel',
      slots: { default: 'entryLabel' },
    },
    { title: $t('page.dict.entryValue'), field: 'entryValue' },
    { title: $t('page.dict.numericValue'), field: 'numericValue' },
    { title: $t('ui.table.sortOrder'), field: 'sortOrder' },
    {
      title: $t('ui.table.status'),
      field: 'isEnabled',
      slots: { default: 'isEnabled' },
      width: 95,
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
  connectedComponent: DictEntryDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
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
    await dictStore.deleteDictEntry([row.id]);

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

function getEntryLabel(row: DictEntry) {
  const currentI18n = row.i18n?.[preferences.app.locale];
  if (currentI18n === undefined) {
    return '';
  }
  return currentI18n.entryLabel;
}

watch(
  () => dictViewStore.currentTypeId,
  () => {
    gridApi.reload();
  },
);
</script>

<template>
  <Grid :table-title="$t('page.dict.dictEntryList')">
    <template #toolbar-tools>
      <a-button type="primary" @click="handleCreate">
        {{ $t('page.dict.button.create') }}
      </a-button>
    </template>
    <template #isEnabled="{ row }">
      <a-tag :color="enableBoolToColor(row.isEnabled)">
        {{ enableBoolToName(row.isEnabled) }}
      </a-tag>
    </template>
    <template #entryLabel="{ row }">
      {{ getEntryLabel(row) }}
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
            moduleName: $t('page.dict.moduleName'),
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
