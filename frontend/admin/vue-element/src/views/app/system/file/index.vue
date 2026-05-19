<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <!-- 搜索 -->
    <PageSearch
      ref="searchRef"
      :search-config="searchConfig"
      @query-click="handleQueryClick"
      @reset-click="handleResetClick"
    />

    <!-- 列表 -->
    <PageContent
      ref="contentRef"
      :content-config="contentConfig"
      @add-click="handleAddClick"
      @operate-click="handleOperateClick"
      @toolbar-click="handleToolbarClick"
    >
      <!-- 存储提供商 -->
      <template #provider="{ row }">
        <ElTag size="small" effect="dark" round :color="ossProviderColor(row.provider)">
          {{ ossProviderLabel(row.provider) }}
        </ElTag>
      </template>
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <FileDrawer ref="drawerRef" @success="handleSuccess" />

    <!-- 隐藏的文件上传组件 -->
    <ElUpload
      ref="uploadRef"
      :auto-upload="false"
      :show-file-list="false"
      :on-change="handleFileChange"
      class="upload-trigger"
    >
      <el-button style="display: none">上传</el-button>
    </ElUpload>
  </div>
</template>

<script lang="ts" setup>
import { ElMessage, ElMessageBox, ElTag, type UploadFile } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import FileDrawer from "./file-drawer.vue";

import { ossProviderColor, ossProviderLabel, useFileStore, useFileTransferStore } from "@/stores";
import { $t } from "@/i18n";

const fileStore = useFileStore();
const fileTransferStore = useFileTransferStore();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 抽屉引用
const drawerRef = ref();
// 上传组件引用
const uploadRef = ref();

// 搜索配置
const searchConfig: ISearchConfig = {
  grid: true, // 启用 Grid 布局
  formItems: [
    {
      type: "input",
      label: $t("pages.file.saveFileName"),
      prop: "saveFileName",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:file", // 文件管理权限前缀
  toolbarRight: [
    {
      name: "upload",
      text: $t("pages.file.button.upload"),
      attrs: {
        type: "primary",
        icon: "upload",
      },
    },
  ], // 右侧自定义按钮（在defaultToolbar左侧）
  defaultToolbar: ["refresh", "exports", "filter"], // 右侧默认工具栏
  table: {
    border: true,
    stripe: false,
  },
  pagination: false, // 禁用分页
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await fileStore.listFile(
      {
        page: page || 1,
        pageSize: pageSize || 10,
      },
      queryParams,
      null,
      ["-created_at"] // 按创建时间倒序排序
    );
    // 转换数据格式
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
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
      template: "date",
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
      template: "tool",
      action: [
        {
          name: "download",
          text: $t("common.button.download"),
        },
        {
          name: "delete",
          text: $t("common.button.delete"),
          attrs: {
            type: "danger",
          },
        },
      ],
    },
  ],
};

// 处理操作点击
const handleOperateClick = (data: IOperateData) => {
  const { name, row } = data;

  if (name === "download") {
    // 下载文件
    handleDownloadFile(row);
  } else if (name === "delete") {
    // 删除
    ElMessageBox.confirm(
      $t("common.confirm.do_you_want_delete", { moduleName: $t("pages.file.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await fileStore.deleteFile(row.id);
        ElMessage.success($t("common.notification.delete_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("common.notification.delete_failed"));
      }
    });
  }
};

// 处理新增点击（上传文件）
const handleAddClick = () => {
  // 触发文件上传
  uploadRef.value?.handleClick();
};

// 处理成功回调
const handleSuccess = () => {
  contentRef.value?.fetchPageData({}, true);
};

// 处理工具栏点击
const handleToolbarClick = (name: string) => {
  if (name === "upload") {
    // 触发上传按钮点击
    uploadRef.value?.handleClick();
  }
};

// 文件选择变化
const handleFileChange = async (file: UploadFile) => {
  if (!file.raw) return;

  try {
    await fileTransferStore.uploadFile("images", "temp", file.raw, "post", () => {
      // 可以在这里处理进度
    });

    ElMessage.success($t("pages.file.notification.upload_success"));
    contentRef.value?.fetchPageData({}, true);
  } catch (error) {
    console.error("上传文件失败", error);
    ElMessage.error($t("pages.file.notification.upload_failed"));
  }
};

// 下载文件
const handleDownloadFile = (row: any) => {
  const objectName = row ? `${row.fileDirectory}/${row.saveFileName}` : "";
  fileTransferStore.downloadFile(row.bucketName, objectName, true);
};
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}

// 隐藏默认的上传按钮，使用自定义工具栏按钮
:deep(.upload-trigger) {
  display: none;
}
</style>
