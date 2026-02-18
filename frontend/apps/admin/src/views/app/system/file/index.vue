<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFileDownload, LucideTrash2 } from '@vben/icons';

import { notification, Upload } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { type storageservicev1_File as File } from '#/generated/api/admin/service/v1';
import { $t } from '#/locales';
import {
  ossProviderColor,
  ossProviderLabel,
  useFileStore,
  useFileTransferStore,
} from '#/stores';

import FileDrawer from './file-drawer.vue';

const fileStore = useFileStore();
const fileTransferStore = useFileTransferStore();

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
      fieldName: 'saveFileName',
      label: $t('page.file.saveFileName'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<File> = {
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

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        console.log('query:', formValues);

        return await fileStore.listFile(
          {
            page: page.currentPage,
            pageSize: page.pageSize,
          },
          formValues,
          null,
          ['-created_at'],
        );
      },
    },
  },

  columns: [
    { title: $t('ui.table.seq'), field: 'id', width: 50 },
    { title: $t('page.file.fileName'), field: 'fileName' },
    { title: $t('page.file.saveFileName'), field: 'saveFileName' },
    { title: $t('page.file.fileDirectory'), field: 'fileDirectory' },
    {
      title: $t('page.file.size'),
      field: 'sizeFormat',
    },
    {
      title: $t('page.file.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: $t('page.file.provider'),
      field: 'provider',
      fixed: 'right',
      slots: { default: 'provider' },
      width: 90,
    },
    {
      title: $t('page.file.createdAt'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 90,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [Drawer] = useVbenDrawer({
  // 连接抽离的组件
  connectedComponent: FileDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      // 关闭时，重载表格数据
      gridApi.reload();
    }
  },
});

async function handleUploadFile(options: any) {
  const { file, onSuccess, onError } = options;

  console.log('上传文件', options);

  try {
    const res = await fileTransferStore.uploadFile(
      'images',
      'temp',
      file,
      'post',
      (progressEvent: any) => {
        console.log(progressEvent);
        // ant-design-vue 要求的进度结构为 { percent: number }
        try {
          // onProgress?.({ percent });
        } catch {
          // 忽略回调内错误
        }
      },
    );

    onSuccess?.(res ?? {}, file);

    await gridApi.reload();

    notification.success({
      message: $t('ui.notification.upload_success'),
    });
  } catch (error) {
    console.error('上传文件失败', error);

    try {
      onError?.(error, file);
    } catch {}

    notification.success({
      message: $t('ui.notification.upload_failed'),
    });
  }
}

function handleDownloadFile(row: any) {
  console.log('下载文件', row);
  const objectName = row ? `${row.fileDirectory}/${row.saveFileName}` : '';
  fileTransferStore.downloadFile(row.bucketName, objectName, true);
}

/* 删除 */
async function handleDelete(row: any) {
  console.log('删除', row);

  try {
    await fileStore.deleteFile(row.id);

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
    <Grid :table-title="$t('menu.system.file')">
      <template #toolbar-tools>
        <Upload :multiple="false" :custom-request="handleUploadFile">
          <a-button class="mr-2" type="primary">
            {{ $t('page.file.button.upload') }}
          </a-button>
        </Upload>
      </template>
      <template #provider="{ row }">
        <a-tag :color="ossProviderColor(row.provider)">
          {{ ossProviderLabel(row.provider) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button
          type="link"
          :icon="h(LucideFileDownload)"
          @click.stop="handleDownloadFile(row)"
        />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', {
              moduleName: $t('page.file.moduleName'),
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
