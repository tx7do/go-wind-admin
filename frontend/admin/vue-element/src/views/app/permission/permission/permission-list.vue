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
      <!-- 状态 -->
      <template #status="{ row }">
        <ElTag size="small" effect="dark" round :color="statusToColor(row.status)">
          {{ statusToName(row.status) }}
        </ElTag>
      </template>
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <PermissionDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElTag, ElMessage, ElMessageBox } from "element-plus";
import { watch, onMounted } from "vue";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import PermissionDrawer from "./permission-drawer.vue";

import { statusList, statusToColor, statusToName, useDeletePermission } from "@/api/composables";
import { $t } from "@/i18n";
import { usePermissionViewStore } from "@/views/app/permission/permission/permission-view.state";

const { mutateAsync: deletePermission } = useDeletePermission();
const permissionViewStore = usePermissionViewStore();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 抽屉引用
const drawerRef = ref();

// 监听分组切换,自动刷新权限列表
watch(
  () => permissionViewStore.needReloadPermissionList,
  (needReload) => {
    if (needReload && contentRef.value) {
      contentRef.value.fetchPageData({}, true);
      permissionViewStore.needReloadPermissionList = false;
    }
  }
);

// 初始化时加载一次
onMounted(() => {
  // 如果有初始分组ID,加载权限列表
  if (permissionViewStore.currentGroupId) {
    permissionViewStore.needReloadPermissionList = true;
  }
});

// 搜索配置
const searchConfig: ISearchConfig = {
  // grid: true,
  formItems: [
    {
      type: "input",
      label: $t("pages.permission.name"),
      prop: "name",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "input",
      label: $t("pages.permission.code"),
      prop: "code",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("common.table.status"),
      prop: "status",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
      },
      options: statusList.value,
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:permission",
  toolbarRight: ["add"],
  defaultToolbar: ["refresh", "filter"],
  table: {
    border: true,
    stripe: true,
    height: "auto",
  },
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await permissionViewStore.fetchPermissionList(
      permissionViewStore.currentGroupId,
      page || 1,
      pageSize || 10,
      queryParams
    );
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
  columns: [
    {
      prop: "name",
      label: $t("pages.permission.name"),
      minWidth: 120,
      align: "left",
    },
    {
      prop: "code",
      label: $t("pages.permission.code"),
      minWidth: 120,
      align: "left",
    },
    {
      prop: "groupName",
      label: $t("pages.permission.groupName"),
      minWidth: 120,
    },
    {
      prop: "status",
      label: $t("common.table.status"),
      width: 90,
      slotName: "status",
    },
    {
      prop: "action",
      label: $t("common.table.action"),
      fixed: "right",
      width: 150,
      template: "tool",
      action: [
        {
          name: "edit",
          text: $t("common.button.edit"),
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

// 新增按钮点击
function handleAddClick() {
  drawerRef.value?.open({ create: true });
}

// 操作按钮点击
async function handleOperateClick(data: IOperateData) {
  const { name, row } = data;

  if (name === "edit") {
    drawerRef.value?.open({ create: false, row });
  } else if (name === "delete") {
    try {
      await ElMessageBox.confirm(
        $t("common.message.confirmDelete", { moduleName: $t("pages.permission.moduleName") }),
        $t("common.title.warning"),
        {
          confirmButtonText: $t("common.button.confirm"),
          cancelButtonText: $t("common.button.cancel"),
          type: "warning",
        }
      );

      await deletePermission({ id: row.id });
      ElMessage.success($t("common.notification.deleteSuccess"));
      handleSuccess();
    } catch (error) {
      if (error !== "cancel") {
        ElMessage.error($t("common.notification.deleteFailed"));
      }
    }
  }
}

// 工具栏按钮点击（同步权限）
async function handleToolbarClick(name: string) {
  if (name === "syncPermissions") {
    try {
      await ElMessageBox.confirm(
        $t("common.message.confirmSyncPermissions", {
          moduleName: $t("pages.permission.moduleName"),
        }),
        $t("common.title.warning"),
        {
          confirmButtonText: $t("common.button.confirm"),
          cancelButtonText: $t("common.button.cancel"),
          type: "warning",
        }
      );

      // syncPermissions API not available yet
      permissionViewStore.reloadGroupList();
      ElMessage.success($t("common.notification.syncSuccess"));
    } catch (error) {
      if (error !== "cancel") {
        ElMessage.error($t("common.notification.syncFailed"));
      }
    }
  }
}

// 成功回调
function handleSuccess() {
  contentRef.value?.fetchPageData();
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
