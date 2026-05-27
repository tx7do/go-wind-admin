<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage ref="pageRef" :config="pageConfig" @toolbar="handleToolbar" @operate="handleOperate">
      <!-- 存储提供商 -->
      <template #provider="scope: any">
        <ElTag size="small" effect="dark" round :color="ossProviderColor(scope.row.provider)">
          {{ ossProviderLabel(scope.row.provider) }}
        </ElTag>
      </template>
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <FileDrawer ref="drawerRef" @success="handleSuccess" />

    <ProFileSelect ref="fileSelectRef" @select="handleFileSelect" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { ElMessage, ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import ProFileSelect from "@/components/Pro/ProFileSelect/index.vue";
import type { ProPageConfig, ToolsButton } from "@/components/Pro/ProPage/types";
import FileDrawer from "./file-drawer.vue";

import {
  ossProviderColor,
  ossProviderLabel,
  fetchListFiles,
  useDeleteFile,
  useUploadFile,
  useDownloadFile,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const { mutateAsync: deleteFile } = useDeleteFile();
const { mutateAsync: uploadFileAction } = useUploadFile();
const { mutateAsync: downloadFileAction } = useDownloadFile();

const pageRef = ref();
const drawerRef = ref();
const fileSelectRef = ref();

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.file.saveFileName"),
        field: "saveFileName",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
    ],
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await fetchListFiles(
        new PaginationQuery({
          paging: { page: page || 1, pageSize: pageSize || 10 },
          formValues: queryParams,
          orderBy: ["-created_at"],
        })
      );
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteFile({ id: ids as any });
    },
    toolbar: [],
    toolbarRight: [
      {
        name: "upload",
        label: $t("pages.file.button.upload"),
        attrs: { type: "primary", icon: "upload" },
      } as ToolsButton,
    ],
    defaultToolbar: ["refresh", "exports", "filter"],
    pagination: false,
    tableAttrs: { border: true, stripe: false },
    columns: [
      { type: "index", label: $t("common.table.seq"), width: 60 },
      { prop: "fileName", label: $t("pages.file.fileName"), minWidth: 150 },
      { prop: "saveFileName", label: $t("pages.file.saveFileName"), minWidth: 150 },
      { prop: "fileDirectory", label: $t("pages.file.fileDirectory"), minWidth: 150 },
      { prop: "sizeFormat", label: $t("pages.file.size"), width: 100 },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        minWidth: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "provider",
        label: $t("pages.file.provider"),
        width: 120,
        slotName: "provider",
      },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "download", label: $t("common.button.download") },
          { name: "delete", label: $t("common.button.delete"), attrs: { type: "danger" } },
        ],
      },
    ],
  },
}));

function handleToolbar(name: string) {
  if (name === "upload") {
    fileSelectRef.value?.open();
  }
}

function handleOperate(data: { name: string; row: any }) {
  if (data.name === "download") {
    handleDownloadFile(data.row);
  }
}

function handleSuccess() {
  pageRef.value?.refresh();
}

function handleFileSelect(files: File[]) {
  if (files.length > 0) {
    handleUploadFile(files[0]);
  }
}

async function handleUploadFile(file: File) {
  try {
    await uploadFileAction({
      bucketName: "images",
      fileDirectory: "temp",
      file,
      method: "post",
    });
    ElMessage.success($t("pages.file.notification.upload_success"));
    pageRef.value?.refresh();
  } catch (error) {
    console.error("上传文件失败", error);
    ElMessage.error($t("pages.file.notification.upload_failed"));
  }
}

function handleDownloadFile(row: any) {
  const objectName = row ? `${row.fileDirectory}/${row.saveFileName}` : "";
  downloadFileAction({ bucketName: row.bucketName, objectName, preferPresignedUrl: true });
}
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
