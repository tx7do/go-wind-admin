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
    >
      <!-- 类型 -->
      <template #type="{ row }">
        <ElTag :color="tenantTypeToColor(row.type)">
          {{ tenantTypeToName(row.type) }}
        </ElTag>
      </template>

      <!-- 审核状态 -->
      <template #auditStatus="{ row }">
        <ElTag :color="tenantAuditStatusToColor(row.auditStatus)">
          {{ tenantAuditStatusToName(row.auditStatus) }}
        </ElTag>
      </template>

      <!-- 状态 -->
      <template #status="{ row }">
        <ElTag :color="tenantStatusToColor(row.status)">
          {{ tenantStatusToName(row.status) }}
        </ElTag>
      </template>
    </PageContent>

    <!-- 新增/编辑弹窗 -->
    <TenantDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElTag, ElMessage, ElMessageBox } from "element-plus";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import TenantDrawer from "./tenant-drawer.vue";

import {
  tenantAuditStatusList,
  tenantAuditStatusToColor,
  tenantAuditStatusToName,
  tenantStatusList,
  tenantStatusToColor,
  tenantStatusToName,
  tenantTypeList,
  tenantTypeToColor,
  tenantTypeToName,
  useTenantStore,
} from "@/stores";
import { $t } from "@/i18n";

const tenantStore = useTenantStore();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 抽屉引用
const drawerRef = ref();

// 搜索配置
const searchConfig: ISearchConfig = {
  formItems: [
    {
      type: "input",
      label: $t("pages.tenant.name"),
      prop: "name",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "input",
      label: $t("pages.tenant.code"),
      prop: "code",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("pages.tenant.type"),
      prop: "type",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
      },
      options: tenantTypeList.value,
    },
    {
      type: "select",
      label: $t("pages.tenant.auditStatus"),
      prop: "auditStatus",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
      },
      options: tenantAuditStatusList.value,
    },
    {
      type: "select",
      label: $t("common.table.status"),
      prop: "status",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
      },
      options: tenantStatusList.value,
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  table: {
    border: true,
    stripe: false,
  },
  indexAction: async (query: any) => {
    const result = await tenantStore.listTenant(
      {
        page: query.page || 1,
        pageSize: query.pageSize || 10,
      },
      query
    );
    // 转换数据格式：将 items 转换为 list
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
  cols: [
    { type: "index", label: $t("common.table.seq"), width: 60 },
    { prop: "name", label: $t("pages.tenant.name"), minWidth: 120 },
    { prop: "code", label: $t("pages.tenant.code"), minWidth: 120 },
    { prop: "adminUserName", label: $t("pages.tenant.adminUserName"), minWidth: 120 },
    {
      prop: "type",
      label: $t("pages.tenant.type"),
      minWidth: 100,
      slotName: "type",
    },
    {
      prop: "auditStatus",
      label: $t("pages.tenant.auditStatus"),
      minWidth: 100,
      slotName: "auditStatus",
    },
    {
      prop: "status",
      label: $t("common.table.status"),
      minWidth: 100,
      slotName: "status",
    },
    {
      prop: "createdAt",
      label: $t("common.table.createdAt"),
      minWidth: 160,
      formatter: (row: any) => {
        if (!row.createdAt) return "";
        return new Date(row.createdAt).toLocaleString("zh-CN");
      },
    },
    { prop: "remark", label: $t("common.table.remark"), minWidth: 150 },
    {
      label: $t("common.table.action"),
      fixed: "right",
      width: 150,
      operat: [
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

// 处理操作点击
const handleOperateClick = (data: IOperateData) => {
  const { name, row } = data;

  if (name === "edit") {
    // 编辑
    drawerRef.value?.open(row);
  } else if (name === "delete") {
    // 删除
    ElMessageBox.confirm(
      $t("common.text.do_you_want_delete", { moduleName: $t("pages.tenant.moduleName") }),
      $t("common.title.confirm"),
      {
        confirmButtonText: $t("common.button.confirm"),
        cancelButtonText: $t("common.button.cancel"),
        type: "warning",
      }
    ).then(async () => {
      try {
        await tenantStore.deleteTenant(row.id);
        ElMessage.success($t("common.notification.delete_success"));
        contentRef.value?.fetchPageData({}, true);
      } catch {
        ElMessage.error($t("common.notification.delete_failed"));
      }
    });
  }
};

// 处理新增点击
const handleAddClick = () => {
  drawerRef.value?.open();
};

// 处理成功回调
const handleSuccess = () => {
  contentRef.value?.fetchPageData({}, true);
};
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
}
</style>
