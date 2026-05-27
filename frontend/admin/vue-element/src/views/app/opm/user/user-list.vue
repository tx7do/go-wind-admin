<template>
  <div class="app-container h-full flex flex-1 flex-col">
    <ProPage
      ref="pageRef"
      :config="pageConfig"
      @add="handleAdd"
      @edit="handleEdit"
      @operate="handleOperate"
    >
      <!-- 状态 -->
      <template #status="scope: any">
        <ElTag size="small" effect="dark" round :color="userStatusToColor(scope.row.status)">
          {{ userStatusToName(scope.row.status) }}
        </ElTag>
      </template>
      <!-- 角色 -->
      <template #roleNames="scope: any">
        <div>
          <ElTag
            v-for="role in scope.row.roleNames"
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
      <template #orgUnitNames="scope: any">
        <div>
          <ElTag
            v-for="orgUnit in scope.row.orgUnitNames"
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
      <template #positionNames="scope: any">
        <div>
          <ElTag
            v-for="position in scope.row.positionNames"
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
    </ProPage>

    <!-- 新增/编辑抽屉 -->
    <UserDrawer ref="drawerRef" @success="handleSuccess" />
  </div>
</template>

<script lang="ts" setup>
import { ref, watch } from "vue";
import { ElTag } from "element-plus";

import ProPage from "@/components/Pro/ProPage/index.vue";
import type { ProPageConfig } from "@/components/Pro/ProPage/types";
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

const pageRef = ref();
const drawerRef = ref();

// 监听组织和租户变化，重新加载数据
watch(
  () => [userViewStore.currentOrgUnitId, userViewStore.currentTenantId],
  () => {
    pageRef.value?.refresh();
  }
);

const pageConfig = computed<ProPageConfig>(() => ({

  search: {
    grid: true,
    fields: [
      {
        type: "input",
        label: $t("pages.user.form.username"),
        field: "username",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.user.form.realname"),
        field: "realname",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "input",
        label: $t("pages.user.form.mobile"),
        field: "mobile",
        attrs: { placeholder: $t("common.placeholder.input"), clearable: true },
      },
      {
        type: "select",
        label: $t("pages.user.form.status"),
        field: "status",
        attrs: { placeholder: $t("common.placeholder.select"), clearable: true },
        options: userStatusList.value,
      },
      {
        type: "tree-select",
        label: $t("pages.user.form.role"),
        field: "roleId",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
          nodeKey: "id",
          props: { label: "name", value: "id", children: "children" },
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
        field: "positionId",
        attrs: {
          placeholder: $t("common.placeholder.select"),
          clearable: true,
          filterable: true,
          nodeKey: "id",
          props: { label: "name", value: "id", children: "children" },
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
  },

  table: {
    listAction: async (query: any) => {
      const { page, pageSize, ...queryParams } = query;
      const result = await userViewStore.fetchUserList(page || 1, pageSize || 10, queryParams);
      return { items: result.items || [], total: result.total || 0 };
    },
    deleteAction: async (ids: string) => {
      await deleteUser(ids as any);
    },
    toolbar: [],
    toolbarRight: ["add"],
    defaultToolbar: ["refresh", "filter"],
    tableAttrs: { border: true, stripe: true, height: "auto" },
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
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      {
        prop: "createdAt",
        label: $t("common.table.createdAt"),
        width: 160,
        cellType: "date",
        dateFormat: "YYYY-MM-DD HH:mm:ss",
      },
      { prop: "remark", label: $t("common.table.remark"), width: 250 },
      {
        prop: "action",
        label: $t("common.table.action"),
        fixed: "right",
        width: 150,
        cellType: "tool",
        buttons: [
          { name: "detail", label: $t("common.button.detail") },
          { name: "edit", label: $t("common.button.edit") },
          { name: "delete", label: $t("common.button.delete"), attrs: { type: "danger" } },
        ],
      },
    ],
  },
}));

function handleAdd() {
  drawerRef.value?.open({ create: true });
}

function handleEdit(row: any) {
  drawerRef.value?.open({ create: false, row });
}

function handleOperate(data: { name: string; row: any }) {
  if (data.name === "detail") {
    router.push(`/opm/users/detail/${data.row.id}`);
  }
}

function handleSuccess() {
  pageRef.value?.refresh();
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
