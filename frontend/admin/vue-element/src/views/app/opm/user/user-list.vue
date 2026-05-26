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
        <ElTag size="small" effect="dark" round :color="userStatusToColor(row.status)">
          {{ userStatusToName(row.status) }}
        </ElTag>
      </template>
      <!-- 角色 -->
      <template #roleNames="{ row }">
        <div>
          <ElTag
            v-for="role in row.roleNames"
            :key="role"
            class="mb-1 mr-1"
            :style="{
              backgroundColor: getRandomColor(role),
              color: '#333',
              border: 'none',
            }"
          >
            {{ role }}
          </ElTag>
        </div>
      </template>
      <!-- 组织 -->
      <template #orgUnitNames="{ row }">
        <div>
          <ElTag
            v-for="orgUnit in row.orgUnitNames"
            :key="orgUnit"
            class="mb-1 mr-1"
            :style="{
              backgroundColor: getRandomColor(orgUnit),
              color: '#333',
              border: 'none',
            }"
          >
            {{ orgUnit }}
          </ElTag>
        </div>
      </template>
      <!-- 职位 -->
      <template #positionNames="{ row }">
        <div>
          <ElTag
            v-for="position in row.positionNames"
            :key="position"
            class="mb-1 mr-1"
            :style="{
              backgroundColor: getRandomColor(position),
              color: '#333',
              border: 'none',
            }"
          >
            {{ position }}
          </ElTag>
        </div>
      </template>
    </PageContent>

    <!-- 新增/编辑抽屉 -->
    <UserDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ElTag, ElMessage, ElMessageBox } from "element-plus";
import { watch } from "vue";

import PageContent from "@/components/CURD/PageContent.vue";
import PageSearch from "@/components/CURD/PageSearch.vue";
import usePage from "@/components/CURD/usePage";
import type { IOperateData, ISearchConfig, IContentConfig } from "@/components/CURD/types";
import UserDrawer from "./user-drawer.vue";

import {
  fetchListPositions,
  fetchListRoles,
  userStatusList,
  userStatusToColor,
  userStatusToName,
  useDeleteUser,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";
import { router } from "@/router";
import { getRandomColor } from "@/utils/color";
import { useUserViewStore } from "@/views/app/opm/user/user-view.state";

const { mutateAsync: deleteUser } = useDeleteUser();
const userViewStore = useUserViewStore();

// 使用 CURD hook
const { searchRef, contentRef, handleQueryClick, handleResetClick } = usePage();

// 抽屉引用
const drawerRef = ref();

// 搜索配置
const searchConfig: ISearchConfig = {
  grid: true,
  // collapsed: true,
  // showCollapseButton: true,
  formItems: [
    {
      type: "input",
      label: $t("pages.user.form.username"),
      prop: "username",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "input",
      label: $t("pages.user.form.realname"),
      prop: "realname",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "input",
      label: $t("pages.user.form.mobile"),
      prop: "mobile",
      attrs: {
        placeholder: $t("common.placeholder.input"),
        clearable: true,
      },
    },
    {
      type: "select",
      label: $t("pages.user.form.status"),
      prop: "status",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
      },
      options: userStatusList.value,
    },
    {
      type: "tree-select",
      label: $t("pages.user.form.role"),
      prop: "roleId",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
        nodeKey: "id",
        props: {
          label: "name",
          value: "id",
          children: "children",
        },
      },
      initFn: async (item: any) => {
        try {
          const result = await fetchListRoles(
            new PaginationQuery({
              formValues: {
                status: "ON",
                type__not: "TEMPLATE",
                tenant_id: userViewStore.currentTenantId ?? 0,
              },
            })
          );
          item.attrs.data = result.items || [];
        } catch (error) {
          console.error("Failed to load roles:", error);
        }
      },
    },
    {
      type: "tree-select",
      label: $t("pages.user.form.position"),
      prop: "positionId",
      attrs: {
        placeholder: $t("common.placeholder.select"),
        clearable: true,
        filterable: true,
        nodeKey: "id",
        props: {
          label: "name",
          value: "id",
          children: "children",
        },
      },
      initFn: async (item: any) => {
        try {
          const result = await fetchListPositions(
            new PaginationQuery({
              formValues: {
                status: "ON",
                org_unit_id: userViewStore.currentOrgUnitId,
                tenant_id: userViewStore.currentTenantId ?? 0,
              },
            })
          );
          item.attrs.data = result.items || [];
        } catch (error) {
          console.error("Failed to load positions:", error);
        }
      },
    },
  ],
};

// 表格配置
const contentConfig: IContentConfig = {
  permPrefix: "sys:user",
  toolbarRight: ["add"],
  defaultToolbar: ["refresh", "filter"],
  table: {
    border: true,
    stripe: true,
    height: "auto",
  },
  indexAction: async (query: any) => {
    const { page, pageSize, ...queryParams } = query;
    const result = await userViewStore.fetchUserList(page || 1, pageSize || 10, queryParams);
    return {
      items: result.items || [],
      total: result.total || 0,
    };
  },
  columns: [
    { type: "index", label: $t("common.table.seq"), width: 60 },
    { prop: "username", label: $t("pages.user.table.username"), width: 120 },
    { prop: "realname", label: $t("pages.user.table.realname"), width: 100 },
    { prop: "nickname", label: $t("pages.user.table.nickname"), width: 100 },
    { prop: "email", label: $t("pages.user.table.email"), width: 160 },
    { prop: "mobile", label: $t("pages.user.table.mobile"), width: 130 },
    {
      prop: "orgUnitNames",
      label: $t("pages.user.table.orgUnitId"),
      width: 130,
      slotName: "orgUnitNames",
    },
    {
      prop: "positionNames",
      label: $t("pages.user.table.positionId"),
      width: 130,
      slotName: "positionNames",
    },
    {
      prop: "roleNames",
      label: $t("pages.user.table.roleId"),
      width: 100,
      slotName: "roleNames",
      showOverflow: "tooltip",
    },
    {
      prop: "status",
      label: $t("pages.user.table.status"),
      width: 95,
      slotName: "status",
    },
    {
      prop: "lastLoginAt",
      label: $t("pages.user.table.lastLoginAt"),
      width: 160,
      template: "date",
      dateFormat: "YYYY-MM-DD HH:mm:ss",
    },
    {
      prop: "createdAt",
      label: $t("common.table.createdAt"),
      width: 160,
      template: "date",
      dateFormat: "YYYY-MM-DD HH:mm:ss",
    },
    { prop: "remark", label: $t("common.table.remark"), width: 250 },
    {
      prop: "action",
      label: $t("common.table.action"),
      fixed: "right",
      width: 150,
      template: "tool",
      action: [
        {
          name: "detail",
          text: $t("common.button.detail"),
        },
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

  if (name === "detail") {
    router.push(`/opm/users/detail/${row.id}`);
  } else if (name === "edit") {
    drawerRef.value?.open({ create: false, row });
  } else if (name === "delete") {
    try {
      await ElMessageBox.confirm(
        $t("common.message.confirmDelete", { moduleName: $t("pages.user.moduleName") }),
        $t("common.title.warning"),
        {
          confirmButtonText: $t("common.button.confirm"),
          cancelButtonText: $t("common.button.cancel"),
          type: "warning",
        }
      );

      await deleteUser(row.id);
      ElMessage.success($t("common.notification.deleteSuccess"));
      handleSuccess();
    } catch (error) {
      if (error !== "cancel") {
        ElMessage.error($t("common.notification.deleteFailed"));
      }
    }
  }
}

// 工具栏按钮点击
function handleToolbarClick(name: string) {
  console.log("toolbar click:", name);
}

// 成功回调
function handleSuccess() {
  contentRef.value?.fetchPageData();
}

// 监听组织和租户变化，重新加载数据
watch(
  () => [userViewStore.currentOrgUnitId, userViewStore.currentTenantId],
  () => {
    contentRef.value?.fetchPageData();
  }
);
</script>

<style lang="scss" scoped>
.app-container {
  padding: 20px;
  width: 100%;
  min-width: 0;
  flex-shrink: 0;
}
</style>
